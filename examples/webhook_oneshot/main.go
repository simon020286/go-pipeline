package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	pipeline "github.com/simon020286/go-pipeline"
	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

func main() {
	fmt.Println("=== Webhook One-Shot Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates webhook handler activation/deactivation")
	fmt.Println()

	// Avvia il server HTTP
	go func() {
		fmt.Println("Starting HTTP server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Aspetta che il server sia pronto
	time.Sleep(100 * time.Millisecond)

	// Crea la pipeline
	p := pipeline.NewPipeline()

	// Aggiungi listener per vedere gli eventi
	p.AddListener(models.EventListenerFunc(func(event models.Event) {
		if event.Type == models.EventStageOutput {
			stageID := event.Data["stage_id"].(string)
			output := event.Data["output"]
			fmt.Printf("✅ Stage '%s' completed with output: %v\n", stageID, output)
		}
	}))

	// Crea uno step trigger (simula input iniziale)
	cronStep, _ := builder.CreateStep("cron", map[string]any{
		"schedule": "@every 5s",
	})

	// Crea webhook one-shot
	webhookStep, _ := builder.CreateStep("webhook", map[string]any{
		"path":       "/process",
		"method":     "POST",
		"continuous": false, // One-shot mode
	})

	// Delay step to see the result
	delayStep, _ := builder.CreateStep("delay", map[string]any{
		"ms": 100,
	})

	// Configure pipeline: cron -> webhook (one-shot) -> delay
	cronStage := pipeline.NewStage("cron-trigger", cronStep)
	webhookStage := pipeline.NewStage("webhook-processor", webhookStep)
	delayStage := pipeline.NewStage("finalizer", delayStep)

	p.AddStage(cronStage)
	p.AddStage(webhookStage).After(cronStage)
	p.AddStage(delayStage).After(webhookStage)

	// Avvia la pipeline
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := p.Start(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Pipeline running! Testing webhook activation...")
	fmt.Println()

	// Test della sequenza di attivazione
	testWebhookActivation()

	// Lascia girare per vedere più cicli
	time.Sleep(20 * time.Second)

	fmt.Println()
	fmt.Println("Stopping pipeline...")
	p.Stop()
	fmt.Println("Done!")
}

func testWebhookActivation() {
	client := &http.Client{Timeout: 2 * time.Second}

	tests := []struct {
		name     string
		delay    time.Duration
		expectOK bool
		reason   string
	}{
		{
			name:     "Request before activation",
			delay:    500 * time.Millisecond,
			expectOK: false,
			reason:   "Handler should return 503 (not active yet)",
		},
		{
			name:     "Request during activation window",
			delay:    5500 * time.Millisecond, // Dopo il primo cron tick
			expectOK: true,
			reason:   "Handler should be active and accept request",
		},
		{
			name:     "Request after deactivation",
			delay:    500 * time.Millisecond,
			expectOK: false,
			reason:   "Handler should return 503 again (deactivated)",
		},
		{
			name:     "Request during second activation",
			delay:    5000 * time.Millisecond, // Secondo tick
			expectOK: true,
			reason:   "Handler should be active again",
		},
	}

	for _, tt := range tests {
		time.Sleep(tt.delay)

		fmt.Printf("📡 Test: %s\n", tt.name)
		fmt.Printf("   Expected: %v (%s)\n", tt.expectOK, tt.reason)

		resp, err := client.Post("http://localhost:8080/process", "application/json", nil)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		success := resp.StatusCode == http.StatusOK
		fmt.Printf("   Actual: %v (status: %d)\n", success, resp.StatusCode)

		if success == tt.expectOK {
			fmt.Printf("   ✅ PASS\n")
		} else {
			fmt.Printf("   ⚠️  FAIL (expected %v, got %v)\n", tt.expectOK, success)
		}
		fmt.Println()
	}
}
