package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the current state of the circuit breaker
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

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

// Settings holds the settings for the circuit breaker
type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
}

var (
	// ErrTooManyRequests is returned when the CB state is half open and the requests count is over the cb maxRequests
	ErrTooManyRequests = errors.New("too many requests")
	// ErrCircuitBreakerOpen is returned when the CB state is open
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

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

// Execute runs the given request if the circuit breaker accepts it
func (cb *CircuitBreaker) Execute(req func() error) error {
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

	err = req()
	cb.afterRequest(generation, err == nil)
	return err
}

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

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.generation++

	if cb.state == StateOpen {
		cb.expiry = now.Add(cb.timeout)
	}

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}

	// Reset counts
	cb.counts = Counts{}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Counts returns the current counts of the circuit breaker
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.counts
}
