// internal/circuitbreaker/circuit_breaker.go
package circuitbreaker

import (
	"errors"
	"log"
	"sync"
	"time"
)

// State represents the state of the circuit breaker
type State int

const (
	// Closed means the circuit breaker is allowing requests
	Closed State = iota
	// Open means the circuit breaker is not allowing requests
	Open
	// HalfOpen means the circuit breaker is allowing a test request
	HalfOpen
)

var (
	// ErrCircuitOpen is returned when the circuit is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name               string
	failureThreshold   int
	resetTimeout       time.Duration
	halfOpenMaxRetries int
	state              State
	failureCount       int
	lastFailure        time.Time
	mutex              sync.Mutex
	retryCount         int
}

// New creates a new circuit breaker
func New(name string, failureThreshold int, resetTimeout time.Duration, halfOpenMaxRetries int) *CircuitBreaker {
	return &CircuitBreaker{
		name:               name,
		failureThreshold:   failureThreshold,
		resetTimeout:       resetTimeout,
		halfOpenMaxRetries: halfOpenMaxRetries,
		state:              Closed,
	}
}

// Execute runs the given function protected by the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mutex.Lock()

	// Check if the circuit is open
	if cb.state == Open {
		// Check if it's time to try half-open state
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = HalfOpen
			cb.retryCount = 0
			log.Printf("Circuit %s changed from Open to HalfOpen", cb.name)
		} else {
			cb.mutex.Unlock()
			return ErrCircuitOpen
		}
	}

	// If we're in half-open state, increment retry counter
	if cb.state == HalfOpen {
		cb.retryCount++
		if cb.retryCount > cb.halfOpenMaxRetries {
			// Too many retries in half-open state, go back to open
			cb.state = Open
			cb.lastFailure = time.Now()
			cb.mutex.Unlock()
			log.Printf("Circuit %s exceeded half-open retries, returning to Open", cb.name)
			return ErrCircuitOpen
		}
	}

	cb.mutex.Unlock()

	// Execute the protected function
	err := fn()

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Handle the result
	if err != nil {
		// Function call failed
		cb.failureCount++
		cb.lastFailure = time.Now()

		if cb.state == HalfOpen || cb.failureCount >= cb.failureThreshold {
			cb.state = Open
			log.Printf("Circuit %s opened due to failure: %v", cb.name, err)
		}

		return err
	}

	// Function call succeeded
	if cb.state == HalfOpen {
		// Success in half-open state, reset to closed
		cb.state = Closed
		log.Printf("Circuit %s closed after successful retry", cb.name)
	}

	// Reset failure count on success
	cb.failureCount = 0
	return nil
}

// IsOpen returns whether the circuit breaker is currently open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state == Open
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state
}
