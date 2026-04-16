package circuitbreaker

import "time"
// Pre request logic
func (cb *circuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// OPEN STATE
	if cb.state == StateOpen {
		if time.Since(cb.openedAt) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
		} else {
			return ErrCircuitBreakerOpen
		}
	}

	// HALF-OPEN STATE
	if cb.state == StateHalfOpen {
		if cb.halfOpenRequests >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
	}

	return nil
}

// Post request logic
func (cb *circuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.Requests++

	if err == nil {
		cb.metrics.Successes++
		cb.metrics.ConsecutiveFailures = 0

		if cb.state == StateHalfOpen {
			// success in half-open → close circuit
			cb.resetMetrics()
			cb.transitionTo(StateClosed)
		}
		return
	}
	// Failure
	cb.metrics.Failures++
	cb.metrics.ConsecutiveFailures++
	cb.metrics.LastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.transitionTo(StateOpen)
		return
	}

	if cb.state == StateClosed {
		if cb.config.ReadyToTrip(cb.metrics) {
			cb.transitionTo(StateOpen)
		}
	}
}