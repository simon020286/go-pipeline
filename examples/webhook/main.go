package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	pipeline "github.com/simon020286/go-pipeline"
	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

func main() {
	fmt.Println("=== Webhook Pipeline Example (Streaming Mode) ===")

	// Crea la pipeline
	pipe := pipeline.NewPipeline()

	pipe.AddListener(models.EventListenerFunc(func(event models.Event) {
		var message strings.Builder
		message.WriteString(fmt.Sprintf("Event: %s at %s", event.Type, event.Timestamp.Format("15:04:05")))
		if event.Type == models.EventStageError {
			if errMsg, ok := event.Data["error"].(string); ok {
				message.WriteString(fmt.Sprintf(" | Error: %s", errMsg))
			}
		}
		if event.Type == models.EventStageOutput {
			if output, ok := event.Data["output"]; ok {
				outputJSON, _ := json.MarshalIndent(output, "", "  ")
				message.WriteString(fmt.Sprintf(" | Output from %s: %s", event.Data["stage_id"], string(outputJSON)))
			}
		}
		fmt.Println(message.String())
	}))

	webhookFactory, _ := builder.GetStepFactory("webhook")

	// Step 1: Webhook trigger (continuous - entry point)
	webhookStep, err := webhookFactory(map[string]any{
		"method":     "GET",
		"path":       "/webhook",
		"continuous": true,
	})

	if err != nil {
		fmt.Printf("Error creating webhook step: %v\n", err)
		return
	}

	// Step 2: Process webhook data
	delayFactory, _ := builder.GetStepFactory("delay")
	delayStep1, err := delayFactory(map[string]any{
		"ms": 500,
	})
	if err != nil {
		fmt.Printf("Error creating delay step: %v\n", err)
		return
	}

	webhookStep2, err := webhookFactory(map[string]any{
		"method":     "GET",
		"path":       "/webhook2",
		"continuous": false,
	})

	if err != nil {
		fmt.Printf("Error creating webhook step 2: %v\n", err)
		return
	}

	delayStep2, err := delayFactory(map[string]any{
		"ms": 500,
	})
	if err != nil {
		fmt.Printf("Error creating delay step: %v\n", err)
		return
	}

	// Crea stage
	webhookStage := pipeline.NewStage("webhook1", webhookStep)
	webhookStage2 := pipeline.NewStage("webhook2", webhookStep2)
	delayStage1 := pipeline.NewStage("delay-step1", delayStep1)
	delayStage2 := pipeline.NewStage("delay-step2", delayStep2)

	// Build pipeline: webhook -> process -> save
	pipe.AddStage(webhookStage) // Entry point (continuous)
	pipe.AddStage(delayStage1).After(webhookStage)
	pipe.AddStage(webhookStage2).After(delayStage1) // Second webhook
	pipe.AddStage(delayStage2).After(webhookStage2)

	// Valida
	if err := pipe.Validate(); err != nil {
		fmt.Printf("Pipeline validation error: %v\n", err)
		return
	}
	fmt.Println("✓ Pipeline validated successfully")

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := &http.Server{Addr: ":8080", Handler: http.DefaultServeMux}

	go srv.ListenAndServe()
	defer srv.Shutdown(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start pipeline (streaming mode)
	fmt.Println("Starting webhook pipeline in streaming mode...")
	fmt.Println("Send HTTP requests to: http://localhost:8080/webhook")
	fmt.Println("Press Ctrl+C to stop")

	if err := pipe.Start(ctx); err != nil {
		fmt.Printf("❌ Pipeline start failed: %v\n", err)
		return
	}

	// Wait for signal
	<-sigChan
	fmt.Println("\n\nShutdown signal received. Stopping pipeline...")

	// Stop pipeline gracefully
	if err := pipe.Stop(); err != nil {
		fmt.Printf("❌ Pipeline stop failed: %v\n", err)
		return
	}

	pipe.Wait()
	fmt.Println("✓ Pipeline stopped gracefully")
}
