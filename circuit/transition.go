package circuitbreaker

import "time"
//Transition
func (cb *circuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
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
		cb.config.OnStateChange("circuit-breaker", old, newState)
	}
}

// Reset metrics
func (cb *circuitBreaker) resetMetrics() {
	cb.metrics = Metrics{}
	cb.halfOpenRequests = 0
}