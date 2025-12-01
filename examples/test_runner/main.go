package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pipeline "github.com/simon020286/go-pipeline"
	"github.com/simon020286/go-pipeline/config"
	"github.com/simon020286/go-pipeline/models"
	_ "github.com/simon020286/go-pipeline/steps"
	"gopkg.in/yaml.v3"
)

// ConsoleLogger implementa EventListener per loggare eventi su console
type ConsoleLogger struct {
	verbose bool
}

func (cl *ConsoleLogger) OnEvent(event models.Event) {
	timestamp := event.Timestamp.Format("15:04:05.000")

	switch event.Type {
	case models.EventPipelineStarted:
		mode := event.Data["mode"].(string)
		log.Printf("[%s] 🚀 Pipeline started (mode: %s)", timestamp, mode)

	case models.EventPipelineCompleted:
		duration := event.Data["duration"].(time.Duration)
		log.Printf("[%s] ✅ Pipeline completed (duration: %v)", timestamp, duration)

	case models.EventPipelineError:
		err := event.Data["error"].(string)
		log.Printf("[%s] ❌ Pipeline error: %s", timestamp, err)

	case models.EventStageOutput:
		stageID := event.Data["stage_id"].(string)
		eventID := event.Data["event_id"].(string)
		if cl.verbose {
			output := event.Data["output"]
			outputJSON, _ := json.MarshalIndent(output, "", "  ")
			log.Printf("[%s] 📤 Stage '%s' output (event: %s):\n%s",
				timestamp, stageID, eventID, string(outputJSON))
		} else {
			log.Printf("[%s] 📤 Stage '%s' produced output (event: %s)",
				timestamp, stageID, eventID)
		}

	case models.EventStageError:
		stageID := event.Data["stage_id"].(string)
		eventID := event.Data["event_id"].(string)
		err := event.Data["error"].(string)
		log.Printf("[%s] ⚠️  Stage '%s' error (event: %s): %s",
			timestamp, stageID, eventID, err)
	}
}

func main() {
	// Parse command line flags
	verbose := flag.Bool("v", false, "Enable verbose output (show all stage outputs)")
	timeout := flag.Duration("t", 30*time.Second, "Timeout duration for the pipeline (0 for no timeout)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <pipeline.yaml>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Pipeline Test Runner - Execute and test YAML pipeline configurations\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s examples/foreach_pipeline.yaml\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -v examples/http_client_pipeline.yaml\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -t 60s examples/cron_pipeline.yaml\n", os.Args[0])
	}
	flag.Parse()

	// Check if file argument is provided
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	pipelineFile := flag.Arg(0)

	fmt.Printf("=== Pipeline Test Runner ===\n")
	fmt.Printf("Loading pipeline from: %s\n", pipelineFile)
	if *verbose {
		fmt.Printf("Verbose output: enabled\n")
	}
	if *timeout > 0 {
		fmt.Printf("Timeout: %v\n", *timeout)
	} else {
		fmt.Printf("Timeout: disabled (press Ctrl+C to stop)\n")
	}
	fmt.Println()

	// Load YAML file
	yamlData, err := os.ReadFile(pipelineFile)
	if err != nil {
		log.Fatalf("Failed to read pipeline file: %v", err)
	}

	// Parse YAML into config
	var cfg config.PipelineConfig
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	log.Printf("Pipeline name: %s", cfg.Name)
	log.Printf("Pipeline description: %s", cfg.Description)
	log.Printf("Stages: %d", len(cfg.Stages))
	fmt.Println()

	// Build pipeline from config
	p, err := pipeline.BuildFromConfig(&cfg)
	if err != nil {
		log.Fatalf("Failed to build pipeline: %v", err)
	}

	// Add console logger
	logger := &ConsoleLogger{verbose: *verbose}
	p.AddListener(logger)

	// Setup context with optional timeout
	var ctx context.Context
	var cancel context.CancelFunc

	if *timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), *timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n⚠️  Interrupt signal received, stopping pipeline...")
		cancel()
	}()

	// Start pipeline
	startTime := time.Now()
	if err := p.Start(ctx); err != nil {
		log.Fatalf("Failed to start pipeline: %v", err)
	}

	log.Println("Pipeline is running...")
	fmt.Println()

	// Wait for pipeline to complete or context cancellation
	p.Wait()

	elapsed := time.Since(startTime)
	fmt.Println()
	fmt.Printf("=== Pipeline Execution Complete ===\n")
	fmt.Printf("Total execution time: %v\n", elapsed)
}
