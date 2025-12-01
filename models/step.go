package models

import "context"

// Step represents the interface for executable logic (streaming-first)
// Each implementation receives an input channel and returns output and error channels
type Step interface {
	// Run executes the step processing input from the channel and producing output
	// The output channel is closed when the step terminates
	Run(ctx context.Context, inputs <-chan *StepInput) (<-chan StepOutput, <-chan error)
	// IsContinuous indicates if the step emits events continuously (webhook, cron)
	// false = executes once and closes the channel (batch step)
	// true = remains active until ctx.Done() (trigger/stream step)
	IsContinuous() bool
}
