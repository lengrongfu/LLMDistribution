package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client represents a client for the LLM Distribution system
type Client struct {
	// Base URL of the LLM Distribution server
	baseURL string
	// HTTP client
	httpClient *http.Client
}

// NewClient creates a new client for the LLM Distribution system
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// UploadModelFile uploads a model file to the LLM Distribution server
func (c *Client) UploadModelFile(modelID, filename string, content io.Reader) (string, error) {
	// Create the URL
	url := fmt.Sprintf("%s/api/models/%s?path=%s", c.baseURL, modelID, filename)

	// Create the request
	req, err := http.NewRequest("PUT", url, content)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload file: %s", string(body))
	}

	// Parse the response
	var response struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Path, nil
}

// UploadModelFileFromPath uploads a model file from a local path to the LLM Distribution server
func (c *Client) UploadModelFileFromPath(modelID, filename, filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload the file
	return c.UploadModelFile(modelID, filename, file)
}

// DownloadModelFile downloads a model file from the LLM Distribution server
func (c *Client) DownloadModelFile(modelID, revision, filename string) ([]byte, error) {
	// Create the URL
	url := fmt.Sprintf("%s/api/models/%s/%s/%s", c.baseURL, modelID, revision, filename)

	// Send the request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to download file: %s", string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// DownloadModelFileToPath downloads a model file from the LLM Distribution server to a local path
func (c *Client) DownloadModelFileToPath(modelID, revision, filename, filePath string) error {
	// Download the file
	content, err := c.DownloadModelFile(modelID, revision, filename)
	if err != nil {
		return err
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Inference-related methods removed

// SiblingFile represents a file in the model repository
type SiblingFile struct {
	RFilename string `json:"rfilename"`
}

// ModelIndexInfo represents model index information
type ModelIndexInfo struct {
	ID           string        `json:"id"`
	ModelID      string        `json:"modelId"`
	Author       string        `json:"author"`
	SHA          string        `json:"sha"`
	LastModified time.Time     `json:"lastModified"`
	Disabled     bool          `json:"disabled"`
	CreatedAt    time.Time     `json:"createdAt"`
	UsedStorage  int64         `json:"usedStorage"`
	Siblings     []SiblingFile `json:"siblings"`
}

// GetModelIndex gets model index information from the LLM Distribution server
func (c *Client) GetModelIndex(ctx context.Context, modelID, version string) (*ModelIndexInfo, error) {
	// Create the URL
	url := fmt.Sprintf("%s/api/models/%s/info/%s", c.baseURL, modelID, version)

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get model index: %s", string(body))
	}

	// Parse the response
	var indexInfo ModelIndexInfo
	if err := json.NewDecoder(resp.Body).Decode(&indexInfo); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &indexInfo, nil
}
