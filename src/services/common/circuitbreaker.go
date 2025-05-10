package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Circuit breaker errors
var (
	ErrTooManyRequests    = errors.New("too many requests")
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

// Result represents a result from a service call
type Result struct {
	Data  interface{}
	Error error
}

// ServiceCall represents a function that can be protected by a circuit breaker
type ServiceCall func() (interface{}, error)

// State represents the current state of the circuit breaker
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return fmt.Sprintf("unknown state: %d", s)
	}
}

// Settings holds the settings for the circuit breaker
type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
}

// Counts holds the counts of requests and their results
type Counts struct {
	Requests      uint32
	TotalFailures uint32
	Failures      uint32
	Successes     uint32
}

// CircuitBreaker represents our circuit breaker implementation
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.RWMutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

// Breaker provides a convenient way to use circuit breakers in services
type Breaker struct {
	cb *CircuitBreaker
}

// DefaultSettings returns the default settings for a circuit breaker
func DefaultSettings(name string) *Settings {
	return &Settings{
		Name:        name,
		MaxRequests: 5,
		Interval:    60 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			failureRatio := float64(counts.Failures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: nil,
	}
}

// NewCircuitBreaker creates a new circuit breaker with the given settings
func NewCircuitBreaker(settings *Settings) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          settings.Name,
		maxRequests:   settings.MaxRequests,
		interval:      settings.Interval,
		timeout:       settings.Timeout,
		readyToTrip:   settings.ReadyToTrip,
		onStateChange: settings.OnStateChange,
		state:         StateClosed,
		generation:    0,
	}

	if cb.interval == 0 {
		cb.interval = 60 * time.Second
	}
	if cb.timeout == 0 {
		cb.timeout = 60 * time.Second
	}

	return cb
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

// Execute method for CircuitBreaker to be used by the Breaker wrapper
func (cb *CircuitBreaker) Execute(fn func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		if e := recover(); e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	err = fn()
	cb.afterRequest(generation, err == nil)
	return err
}

// beforeRequest prepares the circuit breaker for a request
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state := cb.state

	switch state {
	case StateClosed:
		cb.counts.Requests++
		return cb.generation, nil

	case StateOpen:
		if now.After(cb.expiry) {
			cb.setState(StateHalfOpen, now)
			cb.counts.Requests++
			return cb.generation, nil
		}
		return 0, ErrCircuitBreakerOpen

	case StateHalfOpen:
		if cb.counts.Requests >= cb.maxRequests {
			return 0, ErrTooManyRequests
		}
		cb.counts.Requests++
		return cb.generation, nil

	default:
		return 0, errors.New("invalid circuit breaker state")
	}
}

// afterRequest updates the circuit breaker state after a request
func (cb *CircuitBreaker) afterRequest(generation uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.generation != generation {
		return
	}

	if success {
		cb.onSuccess(time.Now())
	} else {
		cb.onFailure(time.Now())
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess(now time.Time) {
	switch cb.state {
	case StateClosed:
		cb.counts.Successes++
		if cb.counts.Failures > 0 {
			cb.counts.Failures--
		}

	case StateHalfOpen:
		cb.counts.Successes++
		if cb.counts.Successes >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure(now time.Time) {
	switch cb.state {
	case StateClosed:
		cb.counts.Failures++
		cb.counts.TotalFailures++
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}

	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.generation++

	if state == StateOpen {
		cb.expiry = now.Add(cb.timeout)
	}

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// Execute runs the given function with circuit breaker protection
func (b *Breaker) Execute(call ServiceCall) Result {
	var result interface{}
	var err error

	err = b.cb.Execute(func() error {
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
	var err error

	select {
	case <-ctx.Done():
		return Result{Data: nil, Error: ctx.Err()}
	default:
		err = b.cb.Execute(func() error {
			var execErr error
			result, execErr = call()
			return execErr
		})
	}

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
