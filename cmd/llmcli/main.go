package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lengrongfu/hf-hub/api"
)

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
	RawJSON      []byte        `json:"-"` // Raw JSON data from the API response, not serialized
}

// SiblingFile represents a file in the model repository
type SiblingFile struct {
	RFilename string `json:"rfilename"`
}

// convertModelIDToHFPath converts a model ID like "Qwen/Qwen2-0.5B-Instruct" to the
// Hugging Face cache path format like "models--Qwen--Qwen2-0.5B-Instruct"
func convertModelIDToHFPath(modelID string) string {
	// Replace slashes with double dashes
	return "models--" + strings.ReplaceAll(modelID, "/", "--")
}

func main() {
	// Parse command line flags
	baseDir := flag.String("base-dir", "/tmp/LLMDistribution", "Base directory for storing models")
	revision := flag.String("revision", "main", "Model revision/version to download")
	flag.Parse()

	// Get the model ID from the command line arguments
	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("Model ID is required")
	}
	modelID := args[0]

	// Set HF_HOME environment variable to baseDir
	os.Setenv("HF_HOME", *baseDir)
	log.Printf("Set HF_HOME environment variable to %s", *baseDir)

	// Convert model ID to Hugging Face cache path format
	hfModelPath := convertModelIDToHFPath(modelID)

	// Create the model directory path following Hugging Face structure
	modelDir := filepath.Join(*baseDir, "hub", hfModelPath)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		log.Fatalf("Failed to create model directory: %v", err)
	}

	// Download the model files from Hugging Face
	log.Printf("Downloading model %s to %s", modelID, modelDir)
	if err := downloadModelFiles(modelID, *revision); err != nil {
		log.Fatalf("Failed to download model files: %v", err)
	}

	// Get the model index information directly from the Hugging Face API
	log.Printf("Getting model index information for %s from Hugging Face API", modelID)
	indexInfo, err := getModelIndex(modelID, *revision)
	if err != nil {
		log.Printf("Warning: Failed to get model index from Hugging Face API: %v", err)
		// Create a basic model index if we couldn't get it from the API
		log.Printf("Creating basic model index based on downloaded files")
		indexInfo = createBasicModelIndex(modelID, modelDir)
	}

	// Save the model index information to a .modeindex file
	indexPath := filepath.Join(modelDir, ".modeindex")
	if err := saveModelIndex(indexInfo, indexPath); err != nil {
		log.Fatalf("Failed to save model index: %v", err)
	}

	log.Printf("Successfully downloaded model %s to %s", modelID, modelDir)
}

// downloadModelFiles downloads all files for a model from Hugging Face
func downloadModelFiles(modelID, revision string) error {
	// Create a new Hugging Face client
	client, err := api.NewApi()
	if err != nil {
		return fmt.Errorf("failed to create Hugging Face client: %w", err)
	}

	// Get the model
	model := client.Model(modelID)

	// Get model info to get the list of files
	info, err := model.Info()
	if err != nil {
		return fmt.Errorf("failed to get model info: %w", err)
	}

	// Download each file from the siblings list
	for _, sibling := range info.Siblings {
		filename := sibling.Rfilename

		// Skip directories or files we don't want to download
		if strings.HasSuffix(filename, "/") {
			continue
		}

		log.Printf("Downloading %s", filename)
		// Use the Get method which will download the file to the HF_HOME cache directory
		// and return the path to the cached file
		_, err := model.Get(filename)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", filename, err)
		}
	}

	// All files are now in the HF_HOME cache directory with the correct structure
	return nil
}

// getModelIndex gets model index information directly from the Hugging Face API
func getModelIndex(modelID, version string) (ModelIndexInfo, error) {
	// Create the URL to the Hugging Face API
	url := fmt.Sprintf("https://huggingface.co/api/models/%s/revision/%s", modelID, version)

	// Create the context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ModelIndexInfo{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ModelIndexInfo{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ModelIndexInfo{}, fmt.Errorf("failed to get model index from Hugging Face API: %s", string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModelIndexInfo{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Validate that the JSON is parseable
	if !json.Valid(body) {
		return ModelIndexInfo{}, fmt.Errorf("invalid JSON response")
	}

	// Parse into our ModelIndexInfo struct for returning
	var indexInfo ModelIndexInfo
	if err := json.Unmarshal(body, &indexInfo); err != nil {
		return ModelIndexInfo{}, fmt.Errorf("failed to parse response into ModelIndexInfo: %w", err)
	}

	// Store the raw JSON data for saving to file later
	indexInfo.RawJSON = body

	return indexInfo, nil
}

// createBasicModelIndex creates a basic model index based on the files in the model directory
func createBasicModelIndex(modelID, modelDir string) ModelIndexInfo {
	// Get the model author from the model ID
	author := modelID
	if strings.Contains(modelID, "/") {
		author = strings.Split(modelID, "/")[0]
	}

	// List all files in the model directory
	var siblings []SiblingFile

	// The snapshots directory contains the actual model files
	snapshotsDir := filepath.Join(modelDir, "snapshots")
	if _, err := os.Stat(snapshotsDir); err == nil {
		// If snapshots directory exists, use it
		err := filepath.Walk(snapshotsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				// Get the relative path
				relPath, err := filepath.Rel(snapshotsDir, path)
				if err != nil {
					return err
				}
				// Skip the .modeindex file
				if relPath == ".modeindex" {
					return nil
				}
				// Add the file to the siblings list
				siblings = append(siblings, SiblingFile{
					RFilename: relPath,
				})
			}
			return nil
		})
		if err != nil {
			log.Printf("Warning: Failed to walk snapshots directory: %v", err)
		}
	} else {
		// Otherwise, use the model directory
		err := filepath.Walk(modelDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				// Get the relative path
				relPath, err := filepath.Rel(modelDir, path)
				if err != nil {
					return err
				}
				// Skip the .modeindex file
				if relPath == ".modeindex" {
					return nil
				}
				// Add the file to the siblings list
				siblings = append(siblings, SiblingFile{
					RFilename: relPath,
				})
			}
			return nil
		})
		if err != nil {
			log.Printf("Warning: Failed to walk model directory: %v", err)
		}
	}

	// Calculate the total storage size
	var usedStorage int64
	for _, sibling := range siblings {
		var filePath string
		if _, err := os.Stat(snapshotsDir); err == nil {
			filePath = filepath.Join(snapshotsDir, sibling.RFilename)
		} else {
			filePath = filepath.Join(modelDir, sibling.RFilename)
		}

		info, err := os.Stat(filePath)
		if err == nil {
			usedStorage += info.Size()
		}
	}

	// Create the model index information
	return ModelIndexInfo{
		ID:           modelID,
		ModelID:      modelID,
		Author:       author,
		SHA:          "local",
		LastModified: time.Now(),
		Disabled:     false,
		CreatedAt:    time.Now(),
		UsedStorage:  usedStorage,
		Siblings:     siblings,
	}
}

// saveModelIndex saves model index information to a file
func saveModelIndex(indexInfo ModelIndexInfo, filePath string) error {
	// If we have raw JSON data from the API, use that directly
	if len(indexInfo.RawJSON) > 0 {
		// Write the raw JSON data to the file
		if err := os.WriteFile(filePath, indexInfo.RawJSON, 0644); err != nil {
			return fmt.Errorf("failed to write raw index file: %w", err)
		}
		return nil
	}

	// Otherwise, marshal the index information to JSON
	data, err := json.MarshalIndent(indexInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index information: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}
