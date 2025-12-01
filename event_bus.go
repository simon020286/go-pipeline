package pipeline

import (
	"sync"
	"time"

	"github.com/simon020286/go-pipeline/models"
)

// eventBus manages event distribution to registered listeners (private)
type eventBus struct {
	listeners   []models.EventListener
	mutex       sync.RWMutex
	pendingWg   sync.WaitGroup // Tracks events being processed
}

// newEventBus creates a new eventBus instance (private)
func newEventBus() *eventBus {
	return &eventBus{
		listeners: make([]models.EventListener, 0),
	}
}

// addListener registers a new listener
func (eb *eventBus) addListener(listener models.EventListener) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	eb.listeners = append(eb.listeners, listener)
}

// RemoveAllListeners removes all listeners
func (eb *eventBus) RemoveAllListeners() {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	eb.listeners = make([]models.EventListener, 0)
}

// Emit sends an event to all registered listeners
func (eb *eventBus) Emit(eventType models.EventType, data map[string]interface{}) {
	eb.mutex.RLock()
	listeners := make([]models.EventListener, len(eb.listeners))
	copy(listeners, eb.listeners)
	eb.mutex.RUnlock()

	event := models.Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Notify all listeners asynchronously to avoid blocking execution
	for _, listener := range listeners {
		eb.pendingWg.Add(1)
		go func(l models.EventListener) {
			defer eb.pendingWg.Done()
			l.OnEvent(event)
		}(listener)
	}
}

// Wait waits for all pending events to be processed
func (eb *eventBus) Wait() {
	eb.pendingWg.Wait()
}

// EmitPipelineStarted emits a pipeline start event
func (eb *eventBus) EmitPipelineStarted(mode string) {
	eb.Emit(models.EventPipelineStarted, map[string]interface{}{
		"mode": mode,
	})
}

// EmitPipelineCompleted emits a pipeline completion event
func (eb *eventBus) EmitPipelineCompleted(duration time.Duration) {
	eb.Emit(models.EventPipelineCompleted, map[string]interface{}{
		"duration": duration,
	})
}

// EmitPipelineError emits a pipeline error event
func (eb *eventBus) EmitPipelineError(err error) {
	eb.Emit(models.EventPipelineError, map[string]interface{}{
		"error": err.Error(),
	})
}

// EmitStageStarted emits a stage start event
func (eb *eventBus) EmitStageStarted(stageID, stepID, eventID string) {
	eb.Emit(models.EventStageStarted, map[string]interface{}{
		"stage_id": stageID,
		"step_id":  stepID,
		"event_id": eventID,
	})
}

// EmitStageCompleted emits a stage completion event
func (eb *eventBus) EmitStageCompleted(stageID, stepID, eventID string, duration time.Duration) {
	eb.Emit(models.EventStageCompleted, map[string]interface{}{
		"stage_id": stageID,
		"step_id":  stepID,
		"event_id": eventID,
		"duration": duration,
	})
}

// EmitStageError emits a stage error event
func (eb *eventBus) EmitStageError(stageID, stepID, eventID string, err error) {
	eb.Emit(models.EventStageError, map[string]interface{}{
		"stage_id": stageID,
		"step_id":  stepID,
		"event_id": eventID,
		"error":    err.Error(),
	})
}

// EmitStageOutput emits a stage output event
func (eb *eventBus) EmitStageOutput(stageID, stepID, eventID string, output map[string]*models.Data, metadata map[string]interface{}) {
	eb.Emit(models.EventStageOutput, map[string]interface{}{
		"stage_id": stageID,
		"step_id":  stepID,
		"event_id": eventID,
		"output":   output,
		"metadata": metadata,
	})
}
