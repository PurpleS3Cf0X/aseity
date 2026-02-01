package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Status struct {
	Provider  string
	BaseURL   string
	Reachable bool
	Models    []string
	Error     string
	Latency   time.Duration
}

// Check verifies that a provider endpoint is reachable and responding.
// For OpenAI-compatible endpoints (Ollama, vLLM, OpenAI, HF), it hits /models.
// For Anthropic/Google, it does a lightweight connectivity test.
func Check(ctx context.Context, providerType, baseURL, apiKey string) Status {
	s := Status{BaseURL: baseURL}
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch providerType {
	case "openai":
		s = checkOpenAICompat(ctx, baseURL, apiKey)
	case "anthropic":
		s = checkAnthropic(ctx, apiKey)
	case "google":
		s = checkGoogle(ctx, apiKey)
	default:
		s.Error = fmt.Sprintf("unknown provider type: %s", providerType)
	}

	s.Latency = time.Since(start)
	s.BaseURL = baseURL
	return s
}

func checkOpenAICompat(ctx context.Context, baseURL, apiKey string) Status {
	s := Status{}
	url := strings.TrimRight(baseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.Error = err.Error()
		return s
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.Error = fmt.Sprintf("cannot reach %s: %s", baseURL, friendlyError(err))
		return s
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		s.Error = "authentication failed — check your API key"
		return s
	}
	if resp.StatusCode != 200 {
		s.Error = fmt.Sprintf("endpoint returned HTTP %d", resp.StatusCode)
		return s
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Some endpoints return non-standard JSON but are still reachable
		s.Reachable = true
		return s
	}

	s.Reachable = true
	for _, m := range result.Data {
		s.Models = append(s.Models, m.ID)
	}
	return s
}

func checkAnthropic(ctx context.Context, apiKey string) Status {
	s := Status{BaseURL: "https://api.anthropic.com"}
	if apiKey == "" {
		s.Error = "no API key configured (set ANTHROPIC_API_KEY)"
		return s
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		s.Error = err.Error()
		return s
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.Error = fmt.Sprintf("cannot reach Anthropic API: %s", friendlyError(err))
		return s
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		s.Error = "invalid API key"
		return s
	}
	s.Reachable = true
	return s
}

func checkGoogle(ctx context.Context, apiKey string) Status {
	s := Status{BaseURL: "https://generativelanguage.googleapis.com"}
	if apiKey == "" {
		s.Error = "no API key configured (set GEMINI_API_KEY)"
		return s
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s&pageSize=1", apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.Error = err.Error()
		return s
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.Error = fmt.Sprintf("cannot reach Google API: %s", friendlyError(err))
		return s
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		s.Error = "invalid API key"
		return s
	}
	s.Reachable = true
	return s
}

// CheckModel verifies that a specific model is available on the provider.
func CheckModel(ctx context.Context, providerType, baseURL, apiKey, modelName string) error {
	if providerType != "openai" {
		return nil // can't easily verify for Anthropic/Google without making a real request
	}
	status := checkOpenAICompat(ctx, baseURL, apiKey)
	if !status.Reachable {
		return fmt.Errorf("provider not reachable: %s", status.Error)
	}
	if len(status.Models) == 0 {
		return nil // endpoint doesn't list models, skip check
	}
	for _, m := range status.Models {
		if m == modelName {
			return nil
		}
	}
	return fmt.Errorf("model %q not found — available: %s", modelName, strings.Join(status.Models, ", "))
}

func friendlyError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "connection refused") {
		return "connection refused (is the service running?)"
	}
	if strings.Contains(msg, "no such host") {
		return "host not found (check the URL)"
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
		return "connection timed out (service may be starting up)"
	}
	return msg
}
