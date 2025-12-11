package models

import (
	"sync"
	"time"
)

// StepInput contains the input data for a step
type StepInput struct {
	Data            map[string]map[string]*Data // Data from dependencies
	EventID         string                      // Unique event ID (propagated through pipeline)
	Timestamp       time.Time                   // Event timestamp
	GlobalVariables map[string]any              // Global pipeline variables
	GlobalSecrets   map[string]any              // Global pipeline secrets
	mu              sync.RWMutex                // Mutex for concurrency
}

func (si *StepInput) Lock() {
	si.mu.Lock()
}

func (si *StepInput) Unlock() {
	si.mu.Unlock()
}
