package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	circuitbreaker "circuit-breaker-Go/circuit"
)

type statusResponse struct {
	State   string                 `json:"state"`
	Metrics circuitbreaker.Metrics `json:"metrics"`
}

type callResponse struct {
	Result  string                 `json:"result,omitempty"`
	Error   string                 `json:"error,omitempty"`
	State   string                 `json:"state"`
	Metrics circuitbreaker.Metrics `json:"metrics"`
}

func main() {
	var cb circuitbreaker.CircuitBreaker
	cb = circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxRequests: 3,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(m circuitbreaker.Metrics) bool {
			return m.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			fmt.Printf("STATE CHANGE: %v → %v\n", from, to)
		},
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statusResponse{
			State:   cb.GetState().String(),
			Metrics: cb.GetMetrics(),
		})
	})

	http.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mode := r.URL.Query().Get("mode")
		ctx := context.Background()

		operation := func() (interface{}, error) {
			if mode == "fail" {
				return nil, fmt.Errorf("service failure")
			}
			return "SUCCESS", nil
		}

		result, err := cb.Call(ctx, operation)
		resp := callResponse{
			State:   cb.GetState().String(),
			Metrics: cb.GetMetrics(),
		}
		if err != nil {
			resp.Error = err.Error()
		} else if result != nil {
			resp.Result = fmt.Sprint(result)
		}

		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/forcestate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		target := r.URL.Query().Get("state")
		var s circuitbreaker.State
		switch target {
		case "CLOSED":
			s = circuitbreaker.StateClosed
		case "OPEN":
			s = circuitbreaker.StateOpen
		case "HALF-OPEN":
			s = circuitbreaker.StateHalfOpen
		default:
			http.Error(w, `{"error":"unknown state"}`, http.StatusBadRequest)
			return
		}
		cb.ForceState(s)
		fmt.Printf("FORCED STATE → %v\n", target)
		json.NewEncoder(w).Encode(statusResponse{
			State:   cb.GetState().String(),
			Metrics: cb.GetMetrics(),
		})
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cb = circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
			MaxRequests: 3,
			Timeout:     5 * time.Second,
			ReadyToTrip: func(m circuitbreaker.Metrics) bool {
				return m.ConsecutiveFailures >= 3
			},
			OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
				fmt.Printf("STATE CHANGE: %v → %v\n", from, to)
			},
		})
		json.NewEncoder(w).Encode(statusResponse{
			State:   cb.GetState().String(),
			Metrics: cb.GetMetrics(),
		})
	})

	http.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir("ui"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ui/index.html")
	})

	fmt.Println("Serving UI at http://localhost:8080")
	fmt.Println("Use /call?mode=fail or /call?mode=success to exercise the breaker")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
