package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests    = errors.New("too many requests in half-open state")
)

type State int

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

func (m Metrics) String() string {
	return fmt.Sprintf("{Requests:%d Successes:%d Failures:%d ConsecutiveFailures:%d LastFailureTime:%v}",
		m.Requests,
		m.Successes,
		m.Failures,
		m.ConsecutiveFailures,
		m.LastFailureTime)
}

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type Metrics struct {
	Requests            int64
	Successes           int64
	Failures            int64
	ConsecutiveFailures int64
	LastFailureTime     time.Time
}

type Config struct {
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(Metrics) bool
	OnStateChange func(name string, from State, to State)
}

type CircuitBreaker interface {
	Call(ctx context.Context, operation func() (interface{}, error)) (interface{}, error)
	GetState() State
	GetMetrics() Metrics
}

type circuitBreaker struct {
	mu sync.Mutex

	config Config

	state   State
	metrics Metrics

	halfOpenRequests uint32
	openedAt         time.Time
}
