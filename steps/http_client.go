package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/simon020286/go-pipeline/config"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

// @step name=http_client category=network description=HTTP client for making API requests
type HTTPClientConfig struct {
	URL         string            `step:"required,desc=The URL to call"`
	Method      string            `step:"default=GET,desc=HTTP method (GET POST PUT DELETE etc)"`
	Headers     map[string]string `step:"desc=HTTP headers to send with the request"`
	Body        any               `step:"desc=Request body for POST PUT etc"`
	ContentType string            `step:"name=content_type,default=application/json,desc=Content-Type header for the request body"`
	Response    string            `step:"default=json,desc=Expected response type (json or text)"`
}

type HTTPClientStep struct {
	urlSpec      config.ValueSpec
	methodSpec   config.ValueSpec
	headers      map[string]config.ValueSpec
	bodySpec     config.ValueSpec
	contentType  string
	responseType string
}

type HTTPClientResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       any               `json:"body"`
}

func (s *HTTPClientStep) IsContinuous() bool {
	return false // Step batch, esegue e termina
}

func (s *HTTPClientStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Processa TUTTI gli input in arrivo
		for input := range inputs {
			// Risolvi URL usando ValueSpec
			urlResolved, err := s.urlSpec.Resolve(input)
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve URL: %w", err)
				return
			}
			url := fmt.Sprintf("%v", urlResolved)

			// Risolvi metodo HTTP usando ValueSpec
			methodResolved, err := s.methodSpec.Resolve(input)
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve method: %w", err)
				return
			}
			method := fmt.Sprintf("%v", methodResolved)

			// Risolvi body se presente
			var bodyReader io.Reader = nil
			if s.bodySpec != nil {
				bodyData, err := s.bodySpec.Resolve(input)
				if err != nil {
					errorChan <- fmt.Errorf("failed to resolve body: %w", err)
					return
				}

				// Serializza il body in base al content-type
				bodyBytes, err := serializeBody(bodyData, s.contentType)
				if err != nil {
					errorChan <- fmt.Errorf("failed to serialize body: %w", err)
					return
				}
				bodyReader = bytes.NewReader(bodyBytes)
			}

			// Crea la richiesta HTTP
			req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
			if err != nil {
				errorChan <- fmt.Errorf("failed to create HTTP request: %w", err)
				return
			}

			// Risolvi e aggiungi headers
			for key, valueSpec := range s.headers {
				headerValue, err := valueSpec.Resolve(input)
				if err != nil {
					errorChan <- fmt.Errorf("failed to resolve header '%s': %w", key, err)
					return
				}
				req.Header.Set(key, fmt.Sprintf("%v", headerValue))
			}

			// Se c'è un body, imposta Content-Type dal campo contentType
			if bodyReader != nil && s.contentType != "" {
				req.Header.Set("Content-Type", s.contentType)
			}

			// Esegui la richiesta
			client := &http.Client{
				Timeout: 30 * time.Second,
			}

			resp, err := client.Do(req)
			if err != nil {
				errorChan <- fmt.Errorf("HTTP request failed: %w", err)
				return
			}
			defer resp.Body.Close()

			// Verifica status code
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				bodyBytes, _ := io.ReadAll(resp.Body)
				errorChan <- fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
				return
			}

			// Prepara la risposta
			responseData := &HTTPClientResponse{
				StatusCode: resp.StatusCode,
				Headers:    make(map[string]string),
			}

			// Copia headers (prendi il primo valore)
			for key, values := range resp.Header {
				if len(values) > 0 {
					responseData.Headers[key] = values[0]
				}
			}

			// Decodifica il body in base al tipo di risposta
			switch s.responseType {
			case "json":
				var bodyData any
				if err := json.NewDecoder(resp.Body).Decode(&bodyData); err != nil {
					errorChan <- fmt.Errorf("failed to decode JSON response: %w", err)
					return
				}
				responseData.Body = bodyData
			case "text":
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					errorChan <- fmt.Errorf("failed to read text response: %w", err)
					return
				}
				responseData.Body = string(bodyBytes)
			default:
				// Default to raw bytes
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					errorChan <- fmt.Errorf("failed to read response: %w", err)
					return
				}
				responseData.Body = string(bodyBytes)
			}

			// Send the result
			select {
			case outputChan <- models.StepOutput{
				Data:      models.CreateDefaultResultData(responseData),
				EventID:   input.EventID,
				Timestamp: time.Now(),
			}:
			case <-ctx.Done():
				errorChan <- errors.New("step cancelled")
				return
			}
		}
	}()

	return outputChan, errorChan
}

// serializeBody serializza il body in base al content-type
func serializeBody(body any, contentType string) ([]byte, error) {
	switch contentType {
	case "application/json", "":
		// Default to JSON
		return json.Marshal(body)
	case "application/x-www-form-urlencoded":
		// TODO: implement form-urlencoded serialization
		// For now, fall back to JSON
		return json.Marshal(body)
	case "text/plain":
		// Convert to string
		return []byte(fmt.Sprintf("%v", body)), nil
	default:
		// For unknown content types, try JSON
		return json.Marshal(body)
	}
}

func init() {
	builder.RegisterStepType("http_client", func(cfg map[string]any) (models.Step, error) {
		urlRaw, ok := cfg["url"]
		if !ok {
			return nil, errors.New("missing 'url' in http_client step")
		}

		methodRaw, ok := cfg["method"]
		if !ok {
			methodRaw = "GET" // Default to GET if not specified
		}

		headers, _ := cfg["headers"].(map[string]config.ValueSpec)
		headersMap := make(map[string]config.ValueSpec)
		for k, v := range headers {
			headersMap[k] = v
		}

		responseType, ok := cfg["response"].(string)
		if !ok {
			responseType = "json" // Default to JSON
		}

		contentType, ok := cfg["content_type"].(string)
		if !ok {
			contentType = "application/json" // Default to JSON
		}

		bodyRaw := cfg["body"] // Body can be optional (nil)

		// Converti i valori in ValueSpec
		var urlSpec config.ValueSpec
		if vs, ok := urlRaw.(config.ValueSpec); ok {
			urlSpec = vs
		} else {
			urlSpec = builder.ParseConfigValue(urlRaw)
		}

		var methodSpec config.ValueSpec
		if vs, ok := methodRaw.(config.ValueSpec); ok {
			methodSpec = vs
		} else {
			methodSpec = config.NewStaticValue(methodRaw)
		}

		var bodySpec config.ValueSpec
		if bodyRaw != nil {
			if vs, ok := bodyRaw.(config.ValueSpec); ok {
				bodySpec = vs
			} else {
				bodySpec = config.NewStaticValue(bodyRaw)
			}
		}

		return &HTTPClientStep{
			urlSpec:      urlSpec,
			methodSpec:   methodSpec,
			headers:      headersMap,
			bodySpec:     bodySpec,
			contentType:  contentType,
			responseType: responseType,
		}, nil
	})
}
