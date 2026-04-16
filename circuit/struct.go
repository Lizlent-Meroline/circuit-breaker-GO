package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests    = errors.New("too many requests in half-open state")
)

type State int

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