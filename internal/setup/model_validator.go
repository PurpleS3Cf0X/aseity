package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CommonModelMappings maps common incorrect model names to correct ones
var CommonModelMappings = map[string]string{
	// Remove "ollama/" prefix
	"ollama/qwen2.5:32b":       "qwen2.5:32b",
	"ollama/qwen2.5:72b":       "qwen2.5:72b",
	"ollama/qwen2.5:14b":       "qwen2.5:14b",
	"ollama/qwen2.5:7b":        "qwen2.5:7b",
	"ollama/llama3.1:70b":      "llama3.1:70b",
	"ollama/llama3.1:8b":       "llama3.1:8b",
	"ollama/deepseek-r1":       "deepseek-r1",
	"ollama/qwen2.5-coder:32b": "qwen2.5-coder:32b",

	// Common typos
	"qwen2.5-32b":     "qwen2.5:32b",
	"qwen2.5-72b":     "qwen2.5:72b",
	"qwen2.5-14b":     "qwen2.5:14b",
	"qwen-2.5:32b":    "qwen2.5:32b",
	"llama-3.1:70b":   "llama3.1:70b",
	"llama-3.1:8b":    "llama3.1:8b",
	"deepseek-r1:1.5": "deepseek-r1",
}

// PopularModels lists popular models with their sizes for suggestions
var PopularModels = []struct {
	Name string
	Size string
	Desc string
}{
	{"qwen2.5:7b", "4.7GB", "Best small model for general use"},
	{"qwen2.5:14b", "9.0GB", "Good balance of quality and speed"},
	{"qwen2.5:32b", "20GB", "High quality, needs 24GB+ VRAM"},
	{"qwen2.5:72b", "47GB", "Best quality, needs 48GB+ VRAM"},
	{"qwen2.5-coder:7b", "4.7GB", "Best for coding tasks (small)"},
	{"qwen2.5-coder:32b", "20GB", "Best for coding tasks (large)"},
	{"llama3.1:8b", "4.7GB", "Fast and efficient"},
	{"llama3.1:70b", "40GB", "High quality, needs 48GB+ VRAM"},
	{"deepseek-r1", "70GB", "Reasoning model, needs 80GB+ VRAM"},
}

// ValidateModelName checks if a model name is valid and suggests corrections
func ValidateModelName(model string) (corrected string, suggestion string, err error) {
	// Check if it's a common incorrect name
	if correct, ok := CommonModelMappings[model]; ok {
		return correct, fmt.Sprintf("Did you mean '%s'? (removing 'ollama/' prefix)", correct), nil
	}

	// Check if model exists in Ollama registry
	exists, err := checkModelExistsInRegistry(model)
	if err != nil {
		// Network error - can't validate, proceed anyway
		return model, "", nil
	}

	if exists {
		return model, "", nil
	}

	// Model doesn't exist - suggest similar models
	suggestions := findSimilarModels(model)
	if len(suggestions) > 0 {
		suggestionText := fmt.Sprintf("Model '%s' not found. Did you mean one of these?\n", model)
		for _, s := range suggestions {
			suggestionText += fmt.Sprintf("  - %s\n", s)
		}
		return "", suggestionText, fmt.Errorf("model not found")
	}

	// No similar models found - show popular models
	suggestionText := fmt.Sprintf("Model '%s' not found. Popular models:\n", model)
	for _, m := range PopularModels[:5] { // Show top 5
		suggestionText += fmt.Sprintf("  - %s (%s) - %s\n", m.Name, m.Size, m.Desc)
	}
	suggestionText += "\nView all models: https://ollama.com/library"

	return "", suggestionText, fmt.Errorf("model not found")
}

// checkModelExistsInRegistry checks if a model exists in Ollama's public registry
func checkModelExistsInRegistry(model string) (bool, error) {
	// Extract base model name (before :)
	baseName := model
	if idx := strings.Index(model, ":"); idx > 0 {
		baseName = model[:idx]
	}

	// Check Ollama library API
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://ollama.com/api/tags/%s", baseName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return false, nil
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("registry returned %d", resp.StatusCode)
	}

	// Parse response to check if specific tag exists
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If we can't parse, assume it exists (base model is valid)
		return true, nil
	}

	// Check if the specific tag exists
	for _, m := range result.Models {
		if m.Name == model || strings.HasPrefix(m.Name, model+"-") {
			return true, nil
		}
	}

	// Base model exists but tag might not - still return true
	return true, nil
}

// findSimilarModels finds models with similar names
func findSimilarModels(model string) []string {
	var similar []string
	modelLower := strings.ToLower(model)

	// Check popular models for similarity
	for _, m := range PopularModels {
		if strings.Contains(strings.ToLower(m.Name), modelLower) ||
			strings.Contains(modelLower, strings.ToLower(strings.Split(m.Name, ":")[0])) {
			similar = append(similar, fmt.Sprintf("%s (%s)", m.Name, m.Size))
		}
	}

	return similar
}
