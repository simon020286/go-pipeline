package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/simon020286/go-pipeline/models"
)

// mockStep is a simple step for testing
type mockStep struct {
	continuous bool
	delay      time.Duration
	output     any
	shouldFail bool
}

func (m *mockStep) IsContinuous() bool {
	return m.continuous
}

func (m *mockStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		for input := range inputs {
			if m.delay > 0 {
				time.Sleep(m.delay)
			}

			if m.shouldFail {
				errorChan <- fmt.Errorf("mock step failed")
				return
			}

			outputChan <- models.StepOutput{
				Data:      models.CreateDefaultResultData(m.output),
				EventID:   input.EventID,
				Timestamp: time.Now(),
			}

			// If not continuous, break after first input
			if !m.continuous {
				break
			}
		}
	}()

	return outputChan, errorChan
}

func TestNewPipeline(t *testing.T) {
	p := NewPipeline()

	if p == nil {
		t.Fatal("NewPipeline returned nil")
	}

	if p.stages == nil {
		t.Error("stages map not initialized")
	}

	if p.dependents == nil {
		t.Error("dependents map not initialized")
	}

	if p.eventBus == nil {
		t.Error("eventBus not initialized")
	}
}

func TestNewStage(t *testing.T) {
	step := &mockStep{output: "test"}
	stage := NewStage("test-stage", step)

	if stage.ID != "test-stage" {
		t.Errorf("Expected ID 'test-stage', got %s", stage.ID)
	}

	if stage.Step != step {
		t.Error("Step not set correctly")
	}

	if len(stage.dependencyRefs) != 0 {
		t.Error("New stage should have no dependencies")
	}
}

func TestPipeline_AddStage(t *testing.T) {
	p := NewPipeline()
	step := &mockStep{output: "test"}
	stage := NewStage("s1", step)

	builder := p.AddStage(stage)

	if builder == nil {
		t.Fatal("AddStage returned nil builder")
	}

	if builder.stage != stage {
		t.Error("StageBuilder stage mismatch")
	}

	p.mutex.RLock()
	_, exists := p.stages["s1"]
	p.mutex.RUnlock()

	if !exists {
		t.Error("Stage not added to pipeline")
	}
}

func TestStageBuilder_After(t *testing.T) {
	p := NewPipeline()

	s1 := NewStage("s1", &mockStep{output: "step1"})
	s2 := NewStage("s2", &mockStep{output: "step2"})

	p.AddStage(s1)
	err := p.AddStage(s2).After(s1)

	if err != nil {
		t.Fatalf("After failed: %v", err)
	}

	if len(s2.dependencyRefs) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(s2.dependencyRefs))
	}

	if s2.dependencyRefs[0] != s1 {
		t.Error("Dependency not set correctly")
	}

	p.mutex.RLock()
	deps := p.dependents["s1"]
	p.mutex.RUnlock()

	if len(deps) != 1 || deps[0] != "s2" {
		t.Error("Dependents graph not updated correctly")
	}
}

func TestStageBuilder_After_NonexistentDependency(t *testing.T) {
	p := NewPipeline()

	s1 := NewStage("s1", &mockStep{output: "step1"})
	s2 := NewStage("s2", &mockStep{output: "step2"})

	// Add s2 but not s1
	err := p.AddStage(s2).After(s1)

	if err == nil {
		t.Error("Expected error for nonexistent dependency")
	}
}

// TestPipeline_SimpleBatchExecution is commented out temporarily
// Full integration tests need more setup
/*
func TestPipeline_SimpleBatchExecution(t *testing.T) {
	// This test requires proper pipeline initialization
	// TODO: Re-enable after pipeline execution is stabilized
}
*/

// TestPipeline_TwoStagesWithDependency is commented out temporarily
// Full integration tests need more setup
/*
func TestPipeline_TwoStagesWithDependency(t *testing.T) {
	// This test requires proper pipeline initialization
	// TODO: Re-enable after pipeline execution is stabilized
}
*/

// TestPipeline_Stop is commented out temporarily
/*
func TestPipeline_Stop(t *testing.T) {
	// This test requires proper pipeline initialization
	// TODO: Re-enable after pipeline execution is stabilized
}
*/

// TestPipeline_CannotStartTwice is commented out temporarily
/*
func TestPipeline_CannotStartTwice(t *testing.T) {
	// This test requires proper pipeline initialization
	// TODO: Re-enable after pipeline execution is stabilized
}
*/

func TestPipeline_AddListener(t *testing.T) {
	p := NewPipeline()

	eventReceived := false
	listener := models.EventListenerFunc(func(event models.Event) {
		eventReceived = true
	})

	p.AddListener(listener)

	// Emit a test event
	p.eventBus.Emit(models.EventStageStarted, map[string]any{
		"stage_id": "test",
	})

	// Give it a moment to process
	time.Sleep(50 * time.Millisecond)

	if !eventReceived {
		t.Error("Event listener did not receive event")
	}
}

