package steps

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

// @step name=webhook category=trigger description=Receives HTTP events and propagates them in the pipeline
type WebhookConfig struct {
	Path       string `step:"default=/webhook,desc=The URL path to listen on"`
	Method     string `step:"default=POST,desc=HTTP method to accept"`
	Continuous bool   `step:"default=false,desc=If true acts as entry point; if false waits for input before listening"`
}

// WebhookStep riceve eventi HTTP e li propaga nella pipeline
type WebhookStep struct {
	path       string
	method     string
	continuous bool // true se è un entry point, false se è mid-pipeline

	// Controllo attivazione per one-shot webhooks
	mu          sync.RWMutex
	active      bool
	handlerOnce sync.Once
}

func (s *WebhookStep) IsContinuous() bool {
	return s.continuous
}

// activate abilita l'handler a ricevere richieste
func (w *WebhookStep) activate() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.active = true
}

// deactivate disabilita l'handler
func (w *WebhookStep) deactivate() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.active = false
}

// isActive verifica se l'handler è attivo
func (w *WebhookStep) isActive() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.active
}

func (s *WebhookStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 10)
	errorChan := make(chan error, 1)

	// Se è mid-pipeline (one-shot), attende un input prima di attivare webhook
	if !s.continuous {
		go func() {
			defer close(outputChan)
			defer close(errorChan)

			for input := range inputs {
				// Canale per ricevere l'evento dal handler HTTP
				received := make(chan map[string]interface{}, 1)

				// Registra handler una sola volta (lazy initialization)
				s.handlerOnce.Do(func() {
					http.HandleFunc(s.path, func(w http.ResponseWriter, r *http.Request) {
						// Verifica se l'handler è attivo
						if !s.isActive() {
							w.WriteHeader(http.StatusServiceUnavailable)
							fmt.Fprintf(w, "Webhook handler not active")
							return
						}

						// Verifica metodo HTTP
						if !strings.EqualFold(r.Method, s.method) {
							w.WriteHeader(http.StatusMethodNotAllowed)
							fmt.Fprintf(w, "Method %s not allowed, expected %s", r.Method, s.method)
							return
						}

						// Prepara i dati dell'evento
						data := map[string]interface{}{
							"method": r.Method,
							"path":   r.URL.Path,
							"query":  r.URL.Query(),
						}

						// Send the event to the channel (non-blocking)
						select {
						case received <- data:
							w.WriteHeader(http.StatusOK)
							fmt.Fprintf(w, "Event received and queued")
						default:
							// Channel full or handler not listening
							w.WriteHeader(http.StatusServiceUnavailable)
							fmt.Fprintf(w, "Handler busy, try again later")
						}
					})
				})

				// Activate the handler
				s.activate()

				// Wait for first event or cancellation
				select {
				case data := <-received:
					// Deactivate immediately after receiving (one-shot)
					s.deactivate()

					outputChan <- models.StepOutput{
						Data:      models.CreateDefaultResultData(data),
						EventID:   input.EventID, // Maintain original EventID
						Timestamp: time.Now(),
					}

				case <-ctx.Done():
					s.deactivate()
					errorChan <- fmt.Errorf("webhook cancelled")
					return
				}
			}
		}()
	} else {
		// Webhook continuo: entry point che genera eventi indefinitamente
		go func() {
			defer close(outputChan)
			defer close(errorChan)

			events := make(chan map[string]interface{}, 10)
			http.HandleFunc(s.path, func(w http.ResponseWriter, r *http.Request) {
				if !strings.EqualFold(r.Method, s.method) {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				data := map[string]interface{}{
					"method": r.Method,
					"path":   r.URL.Path,
					"query":  r.URL.Query(),
				}
				events <- data
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "Event received")
			})

			// Propaga eventi fino a cancellazione
			for {
				select {
				case data := <-events:
					outputChan <- models.StepOutput{
						Data:      models.CreateDefaultResultData(data),
						EventID:   builder.GenerateEventID(), // Nuovo EventID per ogni webhook
						Timestamp: time.Now(),
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return outputChan, errorChan
}

func init() {
	builder.RegisterStepType("webhook", func(cfg map[string]any) (models.Step, error) {
		method, ok := cfg["method"].(string)
		if !ok {
			method = "POST" // Default to POST if not specified
		}

		path, ok := cfg["path"].(string)
		if !ok {
			path = "/webhook" // Default path
		}

		continuous, ok := cfg["continuous"].(bool)
		if !ok {
			continuous = false
		}

		return &WebhookStep{
			method:     method,
			path:       path,
			continuous: continuous,
		}, nil
	})
}
