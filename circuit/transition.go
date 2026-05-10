package circuitbreaker

import "time"

// Transition
func (cb *circuitBreaker) transitionTo(newState State) func() {
	if cb.state == newState {
		return nil
	}

	old := cb.state
	cb.state = newState

	switch newState {
	case StateOpen:
		cb.openedAt = time.Now()
	case StateHalfOpen:
		cb.halfOpenRequests = 0
	case StateClosed:
		cb.resetMetrics()
	}

	if cb.config.OnStateChange != nil {
		name := "circuit-breaker"
		from := old
		to := newState
		return func() {
			cb.config.OnStateChange(name, from, to)
		}
	}

	return nil
}

// Reset metrics
func (cb *circuitBreaker) resetMetrics() {
	cb.metrics = Metrics{}
	cb.halfOpenRequests = 0
}
