package steps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/simon020286/go-pipeline/config"
	"github.com/simon020286/go-pipeline/models"
)

func TestHTTPClientStep_IsContinuous(t *testing.T) {
	step := &HTTPClientStep{}
	if step.IsContinuous() {
		t.Error("HTTPClientStep should not be continuous")
	}
}

func TestHTTPClientStep_SimpleGET(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "success"})
	}))
	defer server.Close()

	step := &HTTPClientStep{
		urlSpec:      config.StaticValue{Value: server.URL},
		methodSpec:   config.StaticValue{Value: "GET"},
		headers:      make(map[string]string),
		responseType: "json",
	}

	inputChan := make(chan *models.StepInput, 1)
	inputChan <- &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}
	close(inputChan)

	ctx := context.Background()
	outputChan, errorChan := step.Run(ctx, inputChan)

	select {
	case output := <-outputChan:
		if output.EventID != "test-event" {
			t.Errorf("Expected event ID 'test-event', got %s", output.EventID)
		}

		// Check the response data
		resultData := output.Data["default"]
		resp, ok := resultData.Value.(*HTTPClientResponse)
		if !ok {
			t.Fatal("Expected HTTPClientResponse type")
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}

		bodyMap, ok := resp.Body.(map[string]any)
		if !ok {
			t.Fatal("Expected body to be map[string]any")
		}

		if bodyMap["message"] != "success" {
			t.Errorf("Expected message 'success', got %v", bodyMap["message"])
		}

	case err := <-errorChan:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for output")
	}
}

func TestHTTPClientStep_POSTWithBody(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "test" {
			t.Errorf("Expected name 'test', got %v", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}))
	defer server.Close()

	step := &HTTPClientStep{
		urlSpec:    config.StaticValue{Value: server.URL},
		methodSpec: config.StaticValue{Value: "POST"},
		headers:    make(map[string]string),
		bodySpec: config.StaticValue{Value: map[string]any{
			"name": "test",
		}},
		responseType: "json",
	}

	inputChan := make(chan *models.StepInput, 1)
	inputChan <- &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}
	close(inputChan)

	ctx := context.Background()
	outputChan, errorChan := step.Run(ctx, inputChan)

	select {
	case output := <-outputChan:
		resultData := output.Data["default"]
		resp, ok := resultData.Value.(*HTTPClientResponse)
		if !ok {
			t.Fatal("Expected HTTPClientResponse type")
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}

	case err := <-errorChan:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for output")
	}
}

func TestHTTPClientStep_WithDynamicURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer server.Close()

	// Use dynamic value to construct URL
	step := &HTTPClientStep{
		urlSpec: config.DynamicValue{
			Language:   "js",
			Expression: "'" + server.URL + "' + '/api/v1'",
		},
		methodSpec:   config.StaticValue{Value: "GET"},
		headers:      make(map[string]string),
		responseType: "json",
	}

	inputChan := make(chan *models.StepInput, 1)
	inputChan <- &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}
	close(inputChan)

	ctx := context.Background()
	outputChan, errorChan := step.Run(ctx, inputChan)

	select {
	case <-outputChan:
		// Success
	case err := <-errorChan:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for output")
	}
}

func TestHTTPClientStep_ErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	step := &HTTPClientStep{
		urlSpec:      config.StaticValue{Value: server.URL},
		methodSpec:   config.StaticValue{Value: "GET"},
		headers:      make(map[string]string),
		responseType: "json",
	}

	inputChan := make(chan *models.StepInput, 1)
	inputChan <- &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}
	close(inputChan)

	ctx := context.Background()
	outputChan, errorChan := step.Run(ctx, inputChan)

	select {
	case <-outputChan:
		t.Fatal("Expected error, got output")
	case err := <-errorChan:
		if err == nil {
			t.Fatal("Expected error for 404 response")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for error")
	}
}

func TestHTTPClientStep_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("Expected Authorization header 'Bearer token123', got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer server.Close()

	step := &HTTPClientStep{
		urlSpec:    config.StaticValue{Value: server.URL},
		methodSpec: config.StaticValue{Value: "GET"},
		headers: map[string]string{
			"Authorization": "Bearer token123",
		},
		responseType: "json",
	}

	inputChan := make(chan *models.StepInput, 1)
	inputChan <- &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}
	close(inputChan)

	ctx := context.Background()
	outputChan, errorChan := step.Run(ctx, inputChan)

	select {
	case <-outputChan:
		// Success
	case err := <-errorChan:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for output")
	}
}

func TestHTTPClientStep_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello World"))
	}))
	defer server.Close()

	step := &HTTPClientStep{
		urlSpec:      config.StaticValue{Value: server.URL},
		methodSpec:   config.StaticValue{Value: "GET"},
		headers:      make(map[string]string),
		responseType: "text",
	}

	inputChan := make(chan *models.StepInput, 1)
	inputChan <- &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}
	close(inputChan)

	ctx := context.Background()
	outputChan, errorChan := step.Run(ctx, inputChan)

	select {
	case output := <-outputChan:
		resultData := output.Data["default"]
		resp, ok := resultData.Value.(*HTTPClientResponse)
		if !ok {
			t.Fatal("Expected HTTPClientResponse type")
		}

		if resp.Body != "Hello World" {
			t.Errorf("Expected body 'Hello World', got %v", resp.Body)
		}

	case err := <-errorChan:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for output")
	}
}
