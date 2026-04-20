package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const ollamaBase = "http://localhost:11434"

type OllamaModel struct {
	Name string `json:"name"`
}

type OllamaPSResponse struct {
	Models []OllamaModel `json:"models"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

func OllamaRunningModels() ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ollamaBase + "/api/ps")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var ps OllamaPSResponse
	if err := json.Unmarshal(body, &ps); err != nil {
		return nil, err
	}
	var names []string
	for _, m := range ps.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

// OllamaAvailableModels fetches all locally pulled models from /api/tags
func OllamaAvailableModels() ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ollamaBase + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tags OllamaTagsResponse
	if err := json.Unmarshal(body, &tags); err != nil {
		return nil, err
	}
	var names []string
	for _, m := range tags.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

// LoadModel warms up a model by sending a generate request with keep_alive=-1
func LoadModel(name string) error {
	payload := map[string]interface{}{
		"model":      name,
		"prompt":     "",
		"keep_alive": -1,
	}
	data, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(ollamaBase+"/api/generate", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

func UnloadModel(name string) error {
	payload := map[string]interface{}{
		"model":      name,
		"keep_alive": 0,
	}
	data, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(ollamaBase+"/api/generate", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

func UnloadAllModels() ([]string, error) {
	models, err := OllamaRunningModels()
	if err != nil {
		return nil, err
	}
	var unloaded []string
	for _, m := range models {
		if err := UnloadModel(m); err == nil {
			unloaded = append(unloaded, m)
		}
	}
	return unloaded, nil
}
