package setup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadModel(t *testing.T) {
	// 1. Create dummy GGUF file
	tmpDir := t.TempDir()
	ggufPath := filepath.Join(tmpDir, "test-model.gguf")
	if err := os.WriteFile(ggufPath, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("Failed to create dummy GGUF: %v", err)
	}

	// 2. Mock Ollama Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL
		if r.URL.Path != "/api/create" {
			t.Errorf("Expected path /api/create, got %s", r.URL.Path)
			http.Error(w, "Not found", 404)
			return
		}

		// Verify Method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Verify Payload
		var payload struct {
			Name      string `json:"name"`
			Modelfile string `json:"modelfile"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
			http.Error(w, "Bad request", 400)
			return
		}

		if payload.Name != "test-model" {
			t.Errorf("Expected model name 'test-model', got '%s'", payload.Name)
		}

		// Modelfile should contain absolute path to GGUF
		absPath, _ := filepath.Abs(ggufPath)
		expectedFrom := "FROM " + absPath
		if !strings.Contains(payload.Modelfile, expectedFrom) {
			t.Errorf("Modelfile should contain '%s', got '%s'", expectedFrom, payload.Modelfile)
		}

		// Send success stream
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "parsing modelfile"})
		json.NewEncoder(w).Encode(map[string]string{"status": "creating model layer"})
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	// 3. Inject Mock URL
	originalURL := OllamaBaseURL
	OllamaBaseURL = server.URL
	defer func() { OllamaBaseURL = originalURL }()

	// 4. Run Test
	if err := LoadModel(ggufPath); err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}
}

func TestLoadModel_FileNotFound(t *testing.T) {
	err := LoadModel("non-existent-file.gguf")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
