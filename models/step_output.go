package models

import "time"

// StepOutput contiene i dati in uscita da uno step
type StepOutput struct {
	Data      map[string]*Data // Risultato dello step
	EventID   string           // Stesso EventID dell'input (per tracciamento)
	Timestamp time.Time        // Timestamp dell'output
}
