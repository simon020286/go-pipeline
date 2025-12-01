package models

import (
	"time"
)

// EventType rappresenta il tipo di evento emesso dalla pipeline
type EventType string

const (
	// Eventi della pipeline
	EventPipelineStarted   EventType = "pipeline.started"
	EventPipelineCompleted EventType = "pipeline.completed"
	EventPipelineError     EventType = "pipeline.error"

	// Eventi degli stage
	EventStageStarted   EventType = "stage.started"
	EventStageCompleted EventType = "stage.completed"
	EventStageError     EventType = "stage.error"
	EventStageOutput    EventType = "stage.output"

	// Eventi degli step
	EventStepStarted   EventType = "step.started"
	EventStepCompleted EventType = "step.completed"
	EventStepError     EventType = "step.error"
)

// Event rappresenta un evento generico della pipeline
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// PipelineStartedEvent evento emesso all'avvio della pipeline
type PipelineStartedEvent struct {
	Mode string `json:"mode"` // "batch" o "streaming"
}

// PipelineCompletedEvent evento emesso al completamento della pipeline
type PipelineCompletedEvent struct {
	Duration time.Duration `json:"duration"`
}

// PipelineErrorEvent evento emesso in caso di errore della pipeline
type PipelineErrorEvent struct {
	Error string `json:"error"`
}

// StageStartedEvent evento emesso all'avvio di uno stage
type StageStartedEvent struct {
	StageID string `json:"stage_id"`
	StepID  string `json:"step_id"`
	EventID string `json:"event_id"`
}

// StageCompletedEvent evento emesso al completamento di uno stage
type StageCompletedEvent struct {
	StageID  string        `json:"stage_id"`
	StepID   string        `json:"step_id"`
	EventID  string        `json:"event_id"`
	Duration time.Duration `json:"duration"`
}

// StageErrorEvent evento emesso in caso di errore di uno stage
type StageErrorEvent struct {
	StageID string `json:"stage_id"`
	StepID  string `json:"step_id"`
	EventID string `json:"event_id"`
	Error   string `json:"error"`
}

// StageOutputEvent event emitted when a stage produces output
type StageOutputEvent struct {
	StageID   string                 `json:"stage_id"`
	StepID    string                 `json:"step_id"`
	EventID   string                 `json:"event_id"`
	Output    map[string]*Data       `json:"output"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventListener è l'interfaccia che deve essere implementata per ricevere eventi dalla pipeline
type EventListener interface {
	OnEvent(event Event)
}

// EventListenerFunc è un adapter per usare funzioni come EventListener
type EventListenerFunc func(event Event)

func (f EventListenerFunc) OnEvent(event Event) {
	f(event)
}
