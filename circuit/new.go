package circuitbreaker

import "context"

func NewCircuitBreaker(config Config) CircuitBreaker {
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

func (cb *circuitBreaker) Call(
	ctx context.Context,
	operation func() (interface{}, error),
) (interface{}, error) {

	if err := cb.beforeRequest(); err != nil {
		return nil, err
	}

	// Execute outside lock (VERY IMPORTANT)
	result, err := operation()

	cb.afterRequest(err)

	return result, err
}