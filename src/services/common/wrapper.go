package circuitbreaker

import (
	"context"
	"fmt"
	"net/http"
)

// Result represents a result from a service call
type Result struct {
	Data  interface{}
	Error error
}

// ServiceCall represents a function that can be protected by a circuit breaker
type ServiceCall func() (interface{}, error)

// Breaker provides a convenient way to use circuit breakers in services
type Breaker struct {
	cb *CircuitBreaker
}

// NewBreaker creates a new Breaker with the given name and default settings
func NewBreaker(name string) *Breaker {
	settings := DefaultSettings(name)
	settings.OnStateChange = func(name string, from State, to State) {
		fmt.Printf("Circuit Breaker '%s' state changed from %s to %s\n", name, from, to)
	}
	return &Breaker{
		cb: NewCircuitBreaker(settings),
	}
}

// NewBreakerWithSettings creates a new Breaker with custom settings
func NewBreakerWithSettings(settings *Settings) *Breaker {
	return &Breaker{
		cb: NewCircuitBreaker(settings),
	}
}

// Execute runs the given function with circuit breaker protection
func (b *Breaker) Execute(call ServiceCall) Result {
	var result interface{}
	err := b.cb.Execute(func() error {
		var execErr error
		result, execErr = call()
		return execErr
	})

	return Result{
		Data:  result,
		Error: err,
	}
}

// ExecuteContext runs the given function with circuit breaker protection and context
func (b *Breaker) ExecuteContext(ctx context.Context, call ServiceCall) Result {
	var result interface{}
	err := b.cb.Execute(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var execErr error
			result, execErr = call()
			return execErr
		}
	})

	return Result{
		Data:  result,
		Error: err,
	}
}

// IsCircuitBreakerError checks if the error is from the circuit breaker
func IsCircuitBreakerError(err error) bool {
	return err == ErrCircuitBreakerOpen || err == ErrTooManyRequests
}

// HandleCircuitBreakerError returns appropriate HTTP status code for circuit breaker errors
func HandleCircuitBreakerError(err error) (int, string) {
	if err == ErrCircuitBreakerOpen {
		return http.StatusServiceUnavailable, "Service is temporarily unavailable"
	}
	if err == ErrTooManyRequests {
		return http.StatusTooManyRequests, "Too many requests"
	}
	return http.StatusInternalServerError, "Internal server error"
}
