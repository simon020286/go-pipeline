//go:generate go run ../codegen/cmd/stepgen .

package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

// @step name=cron category=trigger description=Triggers pipeline execution on a schedule
type CronConfig struct {
	Schedule string `step:"required,desc=Cron expression or duration (e.g. @every 5m or 1h30m)"`
}

type CronStep struct {
	schedule string
}

func (s *CronStep) IsContinuous() bool {
	return true
}

func (s *CronStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 10)
	errorChan := make(chan error, 1)

	// Parse the schedule and start the ticker
	duration, err := parseCronExpression(s.schedule)
	if err != nil {
		errorChan <- fmt.Errorf("failed to parse cron expression: %w", err)
		return nil, errorChan
	}

	// Cron continuo: emette eventi indefinitamente
	go func() {
		defer close(outputChan)
		defer close(errorChan)

		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		for {
			select {
			case t := <-ticker.C:
				outputChan <- models.StepOutput{
					Data:      models.CreateDefaultResultData(nil),
					EventID:   builder.GenerateEventID(), // Nuovo EventID per ogni tick
					Timestamp: t,
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return outputChan, errorChan
}

// parseCronExpression converts a cron-like expression to a time.Duration
// Supported formats:
// - "@every <duration>" - e.g., "@every 5m", "@every 1h30m"
// - Simple interval formats: "5m", "1h", "30s"
// - Full cron expressions: "* * * * *" (minute hour day month weekday)
func parseCronExpression(expr string) (time.Duration, error) {
	// Handle @every prefix
	if len(expr) > 7 && expr[:7] == "@every " {
		durationStr := expr[7:]
		return time.ParseDuration(durationStr)
	}

	// Try to parse as simple duration
	if d, err := time.ParseDuration(expr); err == nil {
		return d, nil
	}

	// TODO: Implement full cron expression parsing (e.g., "0 */2 * * *")
	// For now, return an error for unsupported formats
	return 0, fmt.Errorf("unsupported cron expression format: %s (use @every <duration> or simple duration like 5m, 1h)", expr)
}

func init() {
	builder.RegisterStepType("cron", func(cfg map[string]any) (models.Step, error) {
		schedule, ok := cfg["schedule"].(string)
		if !ok {
			return nil, fmt.Errorf("cron trigger requires 'schedule' configuration")
		}

		// Validate the schedule expression
		_, err := parseCronExpression(schedule)
		if err != nil {
			return nil, fmt.Errorf("invalid schedule expression: %w", err)
		}

		return &CronStep{
			schedule: schedule,
		}, nil
	})
}
