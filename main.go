package main

import (
	"context"
	"fmt"
	"time"

	"circuit-breaker-Go/circuit"
)

func main() {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxRequests: 3,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(m circuitbreaker.Metrics) bool {
			return m.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			fmt.Printf("STATE CHANGE: %v → %v\n", from, to)
		},
	})

	ctx := context.Background()

	// Simulated external service
	failMode := true

	operation := func() (interface{}, error) {
		if failMode {
			return nil, fmt.Errorf("service failure")
		}
		return "SUCCESS", nil
	}

	// Trigger failures → OPEN
	for i := 0; i < 5; i++ {
		res, err := cb.Call(ctx, operation)
		fmt.Printf("Call %d | result: %v | err: %v\n", i+1, res, err)
		time.Sleep(500 * time.Millisecond)
	}
	// Circuit is OPEN → fast fail
	fmt.Println("\n--- Circuit should be OPEN now ---")
	res, err := cb.Call(ctx, operation)
	fmt.Printf("Fast fail | result: %v | err: %v\n", res, err)

	// ---------------------------
	// Wait for HALF-OPEN
	fmt.Println("\n--- Waiting for recovery ---")
	time.Sleep(6 * time.Second)

	// Switch service to success
	failMode = false

	
	//HALF-OPEN → test requests
	
	for i := 0; i < 3; i++ {
		res, err := cb.Call(ctx, operation)
		fmt.Printf("Recovery Call %d | result: %v | err: %v\n", i+1, res, err)
		time.Sleep(500 * time.Millisecond)
	}

	// Final state check

	fmt.Println("\nFinal State:", cb.GetState())
	fmt.Println("Metrics:", cb.GetMetrics())
}