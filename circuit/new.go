package circuitbreaker

import (
	"context"
	"time"
)

func NewCircuitBreaker(config Config) CircuitBreaker {
	if config.MaxRequests == 0 {
		config.MaxRequests = 1
	}

	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}

	if config.ReadyToTrip == nil {
		config.ReadyToTrip = func(m Metrics) bool {
			return m.ConsecutiveFailures >= 5
		}
	}

	return &circuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

func (cb *circuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

func (cb *circuitBreaker) GetMetrics() Metrics {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.metrics
}

func (cb *circuitBreaker) cancelHalfOpenRequest() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen && cb.halfOpenRequests > 0 {
		cb.halfOpenRequests--
	}
}

func (cb *circuitBreaker) Call(
	ctx context.Context,
	operation func() (interface{}, error),
) (interface{}, error) {

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	beforeNotify, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}
	if beforeNotify != nil {
		beforeNotify()
	}

	if err := ctx.Err(); err != nil {
		cb.cancelHalfOpenRequest()
		return nil, err
	}

	// Execute outside lock (VERY IMPORTANT)
	result, err := operation()

	afterNotify := cb.afterRequest(err)
	if afterNotify != nil {
		afterNotify()
	}

	return result, err
}
