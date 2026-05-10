package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_OpenThenRecover(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		MaxRequests: 3,
		Timeout:     40 * time.Millisecond,
		ReadyToTrip: func(m Metrics) bool { return m.ConsecutiveFailures >= 3 },
	})

	ctx := context.Background()
	failOp := func() (interface{}, error) { return nil, errors.New("service failure") }

	for i := 0; i < 3; i++ {
		_, err := cb.Call(ctx, failOp)
		if err == nil {
			t.Fatalf("expected failure on call %d", i+1)
		}
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("expected open state after failures, got %v", cb.GetState())
	}

	_, err := cb.Call(ctx, failOp)
	if err != ErrCircuitBreakerOpen {
		t.Fatalf("expected open error while open, got %v", err)
	}

	time.Sleep(60 * time.Millisecond)

	successOp := func() (interface{}, error) { return "ok", nil }
	result, err := cb.Call(ctx, successOp)
	if err != nil {
		t.Fatalf("expected success after recovery, got %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected result ok, got %v", result)
	}

	if cb.GetState() != StateClosed {
		t.Fatalf("expected closed state after recovery, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_CancelledHalfOpenDoesNotConsumeSlot(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		MaxRequests: 1,
		Timeout:     30 * time.Millisecond,
		ReadyToTrip: func(m Metrics) bool { return m.ConsecutiveFailures >= 1 },
	})

	ctx := context.Background()
	_, err := cb.Call(ctx, func() (interface{}, error) { return nil, errors.New("fail") })
	if err == nil {
		t.Fatal("expected failure")
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("expected open state, got %v", cb.GetState())
	}

	time.Sleep(40 * time.Millisecond)

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = cb.Call(cancelCtx, func() (interface{}, error) { return "ok", nil })
	if err != context.Canceled {
		t.Fatalf("expected context canceled, got %v", err)
	}

	result, err := cb.Call(ctx, func() (interface{}, error) { return "ok", nil })
	if err != nil {
		t.Fatalf("expected success after cancelled half-open attempt, got %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected ok result, got %v", result)
	}
}

func TestCircuitBreaker_IntervalResetsConsecutiveFailures(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		MaxRequests: 1,
		Timeout:     40 * time.Millisecond,
		Interval:    20 * time.Millisecond,
		ReadyToTrip: func(m Metrics) bool { return m.ConsecutiveFailures >= 2 },
	})

	ctx := context.Background()
	failOp := func() (interface{}, error) { return nil, errors.New("fail") }

	_, err := cb.Call(ctx, failOp)
	if err == nil {
		t.Fatal("expected first failure")
	}

	time.Sleep(30 * time.Millisecond)

	_, err = cb.Call(ctx, failOp)
	if err == nil {
		t.Fatal("expected second failure")
	}

	if cb.GetState() != StateClosed {
		t.Fatalf("expected still closed after interval reset, got %v", cb.GetState())
	}

	_, err = cb.Call(ctx, failOp)
	if err == nil {
		t.Fatal("expected third failure to trip breaker")
	}
	if cb.GetState() != StateOpen {
		t.Fatalf("expected open after second consecutive failure, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OnStateChangeCallback(t *testing.T) {
	var events []string
	cb := NewCircuitBreaker(Config{
		MaxRequests: 3,
		Timeout:     30 * time.Millisecond,
		ReadyToTrip: func(m Metrics) bool { return m.ConsecutiveFailures >= 2 },
		OnStateChange: func(name string, from State, to State) {
			events = append(events, from.String()+"->"+to.String())
		},
	})

	ctx := context.Background()
	failOp := func() (interface{}, error) { return nil, errors.New("fail") }

	_, err := cb.Call(ctx, failOp)
	if err == nil {
		t.Fatal("expected failure")
	}
	_, err = cb.Call(ctx, failOp)
	if err == nil {
		t.Fatal("expected second failure")
	}

	if len(events) != 1 || events[0] != "CLOSED->OPEN" {
		t.Fatalf("expected state change CLOSED->OPEN, got %v", events)
	}

	time.Sleep(40 * time.Millisecond)

	_, err = cb.Call(ctx, func() (interface{}, error) { return "ok", nil })
	if err != nil {
		t.Fatalf("expected recovery success, got %v", err)
	}

	expected := []string{"CLOSED->OPEN", "OPEN->HALF-OPEN", "HALF-OPEN->CLOSED"}
	if len(events) != len(expected) {
		t.Fatalf("expected state change sequence %v, got %v", expected, events)
	}
	for i := range expected {
		if events[i] != expected[i] {
			t.Fatalf("expected state change sequence %v, got %v", expected, events)
		}
	}
}

func TestCircuitBreaker_HalfOpenLimitsConcurrentRequests(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		MaxRequests: 1,
		Timeout:     30 * time.Millisecond,
		ReadyToTrip: func(m Metrics) bool { return m.ConsecutiveFailures >= 1 },
	})

	ctx := context.Background()
	_, err := cb.Call(ctx, func() (interface{}, error) { return nil, errors.New("fail") })
	if err == nil {
		t.Fatal("expected failure")
	}

	time.Sleep(40 * time.Millisecond)

	start := make(chan struct{})
	results := make(chan error, 2)

	worker := func() {
		<-start
		_, err := cb.Call(ctx, func() (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "ok", nil
		})
		results <- err
	}

	go worker()
	go worker()
	close(start)

	var succeeded, rejected int
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			succeeded++
		} else if err == ErrTooManyRequests {
			rejected++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if succeeded != 1 || rejected != 1 {
		t.Fatalf("expected one success and one rejection, got %d success and %d reject", succeeded, rejected)
	}

	if cb.GetState() != StateClosed {
		t.Fatalf("expected closed state after half-open success, got %v", cb.GetState())
	}
}
