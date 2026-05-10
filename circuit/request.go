package circuitbreaker

import "time"

// Pre request logic
func (cb *circuitBreaker) beforeRequest() (func(), error) {
	cb.mu.Lock()
	var notify func()

	// CLOSED STATE: expire stale failures before any request.
	if cb.state == StateClosed {
		cb.clearExpiredFailuresLocked()
	}

	// OPEN STATE
	if cb.state == StateOpen {
		if time.Since(cb.openedAt) >= cb.config.Timeout {
			notify = cb.transitionTo(StateHalfOpen)
		} else {
			cb.mu.Unlock()
			return nil, ErrCircuitBreakerOpen
		}
	}

	// HALF-OPEN STATE
	if cb.state == StateHalfOpen {
		if cb.halfOpenRequests >= cb.config.MaxRequests {
			cb.mu.Unlock()
			return nil, ErrTooManyRequests
		}
		cb.halfOpenRequests++
	}

	cb.mu.Unlock()
	return notify, nil
}

func (cb *circuitBreaker) clearExpiredFailuresLocked() {
	if cb.config.Interval <= 0 || cb.metrics.LastFailureTime.IsZero() {
		return
	}

	if time.Since(cb.metrics.LastFailureTime) > cb.config.Interval {
		cb.metrics.ConsecutiveFailures = 0
	}
}

// Post request logic
func (cb *circuitBreaker) afterRequest(err error) func() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.Requests++
	var notify func()

	if err == nil {
		cb.metrics.Successes++
		cb.metrics.ConsecutiveFailures = 0

		if cb.state == StateHalfOpen {
			// success in half-open → close circuit
			cb.resetMetrics()
			notify = cb.transitionTo(StateClosed)
		}
		return notify
	}

	if cb.state == StateClosed {
		cb.clearExpiredFailuresLocked()
	}

	// Failure
	cb.metrics.Failures++
	cb.metrics.ConsecutiveFailures++
	cb.metrics.LastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		notify = cb.transitionTo(StateOpen)
		return notify
	}

	if cb.state == StateClosed {
		if cb.config.ReadyToTrip(cb.metrics) {
			notify = cb.transitionTo(StateOpen)
		}
	}

	return notify
}
