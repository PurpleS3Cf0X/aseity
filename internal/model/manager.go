package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Manager struct {
	ollamaURL string
	hfToken   string
	client    *http.Client
}

func NewManager(ollamaURL, hfToken string) *Manager {
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	return &Manager{
		ollamaURL: strings.TrimRight(ollamaURL, "/"),
		hfToken:   hfToken,
		client:    &http.Client{},
	}
}

// List returns all locally available models from Ollama.
func (m *Manager) List(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.ollamaURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Ollama at %s: %w", m.ollamaURL, err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			Size       int64  `json:"size"`
			ModifiedAt string `json:"modified_at"`
			Digest     string `json:"digest"`
			Details    struct {
				Family          string `json:"family"`
				ParameterSize   string `json:"parameter_size"`
				QuantizationLvl string `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, len(result.Models))
	for i, rm := range result.Models {
		models[i] = ModelInfo{
			Name:       rm.Name,
			Size:       rm.Size,
			Digest:     rm.Digest,
			Family:     rm.Details.Family,
			Parameters: rm.Details.ParameterSize,
			Format:     rm.Details.QuantizationLvl,
		}
	}
	return models, nil
}

// Pull downloads a model. Supports:
//   - "llama3.2" -> Ollama registry
//   - "hf.co/user/repo" -> HuggingFace via Ollama
func (m *Manager) Pull(ctx context.Context, ref string, progress func(PullProgress)) error {
	body := map[string]any{"name": ref, "stream": true}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", m.ollamaURL+"/api/pull", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed (%d): %s", resp.StatusCode, string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var p PullProgress
		if err := json.Unmarshal(scanner.Bytes(), &p); err != nil {
			continue
		}
		if p.Total > 0 {
			p.Percent = float64(p.Completed) / float64(p.Total) * 100
		}
		if progress != nil {
			progress(p)
		}
	}
	return scanner.Err()
}

// Remove deletes a model from Ollama.
func (m *Manager) Remove(ctx context.Context, name string) error {
	body := map[string]string{"name": name}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "DELETE", m.ollamaURL+"/api/delete", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(b))
	}
	return nil
}

// SearchHuggingFace searches for models on HuggingFace Hub.
func (m *Manager) SearchHuggingFace(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	if limit == 0 {
		limit = 20
	}
	url := fmt.Sprintf("https://huggingface.co/api/models?search=%s&limit=%d&sort=downloads&direction=-1&filter=gguf", query, limit)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if m.hfToken != "" {
		req.Header.Set("Authorization", "Bearer "+m.hfToken)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []struct {
		ID        string `json:"id"`
		Downloads int    `json:"downloads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, len(results))
	for i, r := range results {
		models[i] = ModelInfo{Name: "hf.co/" + r.ID}
	}
	return models, nil
}
