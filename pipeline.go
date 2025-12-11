package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
	_ "github.com/simon020286/go-pipeline/steps"
)

// ExecutionMode indicates the pipeline execution mode
type ExecutionMode int

const (
	// ExecutionModeBatch executes once and terminates
	ExecutionModeBatch ExecutionMode = iota
	// ExecutionModeStreaming remains listening until cancellation
	ExecutionModeStreaming
)

// Stage represents a pipeline node
// Contains the Step to execute and dependencies (previous stages)
type Stage struct {
	ID             string      // Unique identifier of the stage
	Step           models.Step // The step to execute
	dependencyRefs []*Stage    // References to dependency stages (instance-based, private)
}

// NewStage creates a new stage without dependencies
// Dependencies are added via pipeline.AddStage(stage).After(deps...)
func NewStage(id string, step models.Step) *Stage {
	return &Stage{
		ID:             id,
		Step:           step,
		dependencyRefs: []*Stage{},
	}
}

// Pipeline orchestrates stage execution
type Pipeline struct {
	stages     map[string]*Stage   // Map ID -> Stage for fast access
	dependents map[string][]string // Map ID -> stages that depend on this (inverse graph)
	mutex      sync.RWMutex

	// Lifecycle management
	ctx     context.Context
	cancel  context.CancelFunc
	running atomic.Bool
	done    chan struct{} // Signals when the pipeline has terminated

	// Execution mode
	mode ExecutionMode

	// Event handling (private)
	eventBus *eventBus

	// Global configuration
	globalVariables map[string]any // Global variables accessible to all stages
	globalSecrets   map[string]any // Global secrets accessible to all stages
}

// StageBuilder allows configuring a stage with fluent API
type StageBuilder struct {
	pipeline *Pipeline
	stage    *Stage
}

// After defines the stage dependencies
// Returns error if a dependency doesn't exist in the pipeline
func (sb *StageBuilder) After(dependencies ...*Stage) error {
	sb.pipeline.mutex.Lock()
	defer sb.pipeline.mutex.Unlock()

	for _, dep := range dependencies {
		// Verify the dependency exists in the pipeline
		if _, exists := sb.pipeline.stages[dep.ID]; !exists {
			return fmt.Errorf("dependency stage '%s' not found in pipeline", dep.ID)
		}

		// Add to dependents graph
		sb.pipeline.dependents[dep.ID] = append(sb.pipeline.dependents[dep.ID], sb.stage.ID)

		// Add to stage references
		sb.stage.dependencyRefs = append(sb.stage.dependencyRefs, dep)
	}

	return nil
}

// NewPipeline creates a new pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{
		stages:     make(map[string]*Stage),
		dependents: make(map[string][]string),
		done:       make(chan struct{}),
		eventBus:   newEventBus(),
	}
}

// AddListener adds a listener to receive events from the pipeline
func (p *Pipeline) AddListener(listener models.EventListener) {
	p.eventBus.addListener(listener)
}

// SetGlobalVariables sets the global variables accessible to all stages
func (p *Pipeline) SetGlobalVariables(variables map[string]any) {
	p.globalVariables = variables
}

// SetGlobalSecrets sets the global secrets accessible to all stages
func (p *Pipeline) SetGlobalSecrets(secrets map[string]any) {
	p.globalSecrets = secrets
}

// Start avvia la pipeline in background (non bloccante)
func (p *Pipeline) Start(parentCtx context.Context) error {
	if !p.running.CompareAndSwap(false, true) {
		return fmt.Errorf("pipeline already running")
	}

	// Valida prima di avviare
	if err := p.Validate(); err != nil {
		p.running.Store(false)
		return fmt.Errorf("pipeline validation failed: %w", err)
	}

	// Determina execution mode
	p.mode = p.detectExecutionMode()

	// Crea context cancellabile
	p.ctx, p.cancel = context.WithCancel(parentCtx)

	// Ricrea done channel
	p.done = make(chan struct{})

	// Emetti evento di avvio
	modeStr := "batch"
	if p.mode == ExecutionModeStreaming {
		modeStr = "streaming"
	}
	p.eventBus.EmitPipelineStarted(modeStr)

	// Avvia esecuzione in background
	startTime := time.Now()
	go func() {
		defer func() {
			p.running.Store(false)
			duration := time.Since(startTime)
			p.eventBus.EmitPipelineCompleted(duration)

			// Aspetta che tutti gli eventi siano stati processati
			p.eventBus.Wait()

			close(p.done)
		}()

		p.execute(p.ctx)
	}()

	return nil
}

// Stop ferma la pipeline gracefully
func (p *Pipeline) Stop() error {
	if !p.running.Load() {
		return fmt.Errorf("pipeline not running")
	}

	// Triggera cancellazione
	if p.cancel != nil {
		p.cancel()
	}

	// Aspetta terminazione con timeout
	timeout := time.After(30 * time.Second)
	select {
	case <-p.done:
		return nil
	case <-timeout:
		return fmt.Errorf("pipeline stop timeout")
	}
}

// Wait aspetta che la pipeline termini
func (p *Pipeline) Wait() {
	if p.done != nil {
		<-p.done
	}
}

// IsRunning indica se la pipeline è attualmente in esecuzione
func (p *Pipeline) IsRunning() bool {
	return p.running.Load()
}

// Execute esegue la pipeline in modo bloccante (per compatibilità)
func (p *Pipeline) Execute(ctx context.Context) error {
	if err := p.Start(ctx); err != nil {
		return err
	}
	p.Wait()
	return nil
}

// detectExecutionMode determina se la pipeline è batch o streaming
// in base agli entry points (stage senza dipendenze)
func (p *Pipeline) detectExecutionMode() ExecutionMode {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, stage := range p.stages {
		if len(stage.dependencyRefs) == 0 {
			if stage.Step.IsContinuous() {
				return ExecutionModeStreaming
			}
		}
	}

	return ExecutionModeBatch
}

// AddStage aggiunge uno stage alla pipeline e restituisce un builder per configurare le dipendenze
func (p *Pipeline) AddStage(stage *Stage) *StageBuilder {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if stage == nil {
		panic("stage cannot be nil")
	}
	if stage.ID == "" {
		panic("stage ID cannot be empty")
	}
	if stage.Step == nil {
		panic("stage step cannot be nil")
	}

	p.stages[stage.ID] = stage

	return &StageBuilder{
		pipeline: p,
		stage:    stage,
	}
}

// GetStage restituisce uno stage dato il suo ID
func (p *Pipeline) GetStage(id string) (*Stage, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	stage, exists := p.stages[id]
	return stage, exists
}

// Validate verifica che la pipeline sia valida
// - Controlla che tutte le dipendenze esistano
// - Verifica che non ci siano cicli nelle dipendenze
func (p *Pipeline) Validate() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Verifica che tutte le dipendenze esistano
	for id, stage := range p.stages {
		for _, dep := range stage.dependencyRefs {
			if _, exists := p.stages[dep.ID]; !exists {
				return fmt.Errorf("stage '%s' depends on non-existent stage '%s'", id, dep.ID)
			}
		}
	}

	// Verifica cicli (DFS)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(id string) bool {
		visited[id] = true
		recStack[id] = true

		stage := p.stages[id]
		for _, dep := range stage.dependencyRefs {
			if !visited[dep.ID] {
				if hasCycle(dep.ID) {
					return true
				}
			} else if recStack[dep.ID] {
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for id := range p.stages {
		if !visited[id] {
			if hasCycle(id) {
				return fmt.Errorf("circular dependency detected in pipeline")
			}
		}
	}

	return nil
}

// execute è la logica interna di esecuzione
func (p *Pipeline) execute(ctx context.Context) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Per ogni stage, creo un channel dedicato per ogni consumer
	// Key: "producerID->consumerID"
	stageConnections := make(map[string]chan models.StepOutput)

	// Prepara le connessioni
	for consumerID, consumerStage := range p.stages {
		for _, dep := range consumerStage.dependencyRefs {
			key := fmt.Sprintf("%s->%s", dep.ID, consumerID)
			stageConnections[key] = make(chan models.StepOutput, 10)
		}
	}

	var wg sync.WaitGroup

	// Avvia tutti gli stage
	for id, stage := range p.stages {
		wg.Add(1)

		go func(stageID string, stg *Stage) {
			defer wg.Done()

			// Chiudi i channel di output verso i consumer al termine
			defer func() {
				for consumerID := range p.stages {
					key := fmt.Sprintf("%s->%s", stageID, consumerID)
					if ch, exists := stageConnections[key]; exists {
						close(ch)
					}
				}
			}()

			// Crea channel di input da dipendenze
			inputChan := p.createInputChannelV2(ctx, stageID, stageConnections)

			// Ottieni il nome dello step type (se disponibile)
			stepID := fmt.Sprintf("%T", stg.Step)

			// Esegui step
			outputChan, errorChan := stg.Step.Run(ctx, inputChan)

			// Forward outputs a TUTTI i consumer
			var forwardWg sync.WaitGroup
			forwardWg.Add(2)

			// Forward outputs (broadcast a tutti i consumer)
			go func() {
				defer forwardWg.Done()
				for out := range outputChan {
					// Emetti evento di output
					p.eventBus.EmitStageOutput(stageID, stepID, out.EventID, out.Data, nil)

					// Trova tutti i consumer di questo stage
					for consumerID := range p.stages {
						key := fmt.Sprintf("%s->%s", stageID, consumerID)
						if ch, exists := stageConnections[key]; exists {
							select {
							case ch <- out:
							case <-ctx.Done():
								return
							}
						}
					}
				}
			}()

			// Forward errors
			go func() {
				defer forwardWg.Done()
				for err := range errorChan {
					// Emetti evento di errore
					p.eventBus.EmitStageError(stageID, stepID, "", err)
				}
			}()

			// Aspetta che entrambi i forward finiscano
			forwardWg.Wait()

		}(id, stage)
	}

	// Aspetta completamento
	wg.Wait()
}

// createInputChannelV2 crea il channel di input usando le connessioni dedicate
func (p *Pipeline) createInputChannelV2(ctx context.Context, stageID string, connections map[string]chan models.StepOutput) <-chan *models.StepInput {
	inputChan := make(chan *models.StepInput, 10)
	stage := p.stages[stageID]

	go func() {
		defer close(inputChan)

		// Se non ha dipendenze, emetti un input iniziale
		if len(stage.dependencyRefs) == 0 {
			select {
			case inputChan <- &models.StepInput{
				Data:            make(map[string]map[string]*models.Data),
				EventID:         builder.GenerateEventID(),
				Timestamp:       time.Now(),
				GlobalVariables: p.globalVariables,
				GlobalSecrets:   p.globalSecrets,
			}:
			case <-ctx.Done():
			}

			// Se lo step è continuous, aspetta cancellazione
			if stage.Step.IsContinuous() {
				<-ctx.Done()
			}
			return
		}

		// Altrimenti, raccogli output dalle dipendenze
		for {
			data := make(map[string]map[string]*models.Data)
			eventID := ""

			// Leggi da TUTTE le dipendenze
			allClosed := true
			for _, dep := range stage.dependencyRefs {
				key := fmt.Sprintf("%s->%s", dep.ID, stageID)
				ch := connections[key]

				select {
				case out, ok := <-ch:
					if !ok {
						continue
					}
					allClosed = false
					data[dep.ID] = out.Data
					if eventID == "" {
						eventID = out.EventID
					}
				case <-ctx.Done():
					return
				}
			}

			// Se tutti chiusi, termina
			if allClosed {
				return
			}

			// Emetti input se abbiamo dati
			if len(data) > 0 {
				select {
				case inputChan <- &models.StepInput{
					Data:            data,
					EventID:         eventID,
					Timestamp:       time.Now(),
					GlobalVariables: p.globalVariables,
					GlobalSecrets:   p.globalSecrets,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return inputChan
}

// GetStages restituisce tutti gli stage della pipeline
func (p *Pipeline) GetStages() map[string]*Stage {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Crea una copia per evitare modifiche esterne
	stages := make(map[string]*Stage, len(p.stages))
	for id, stage := range p.stages {
		stages[id] = stage
	}
	return stages
}
