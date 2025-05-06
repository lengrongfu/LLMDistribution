package filestorage

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lengrongfu/LLMDistribution/pkg/utils"
)

// Storage represents a file storage system
type Storage struct {
	// Base directory for file storage
	baseDir string
}

// NewStorage creates a new file storage
func NewStorage(baseDir string) (*Storage, error) {
	baseDir = filepath.Join(baseDir, "hub")
	// Create the base directory if it doesn't exist
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create base directory: %w", err)
		}
	}
	return &Storage{
		baseDir: baseDir,
	}, nil
}

// StoreFile stores a file in the file storage
func (s *Storage) StoreFile(modelID, filename string, content io.Reader) (string, error) {
	// Create the model directory if it doesn't exist
	modelDir := filepath.Join(s.baseDir, modelID)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create model directory: %w", err)
	}

	// Create the file path
	filePath := filepath.Join(modelDir, filename)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write the content to the file
	if _, err := io.Copy(file, content); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

// GetFile retrieves a file from the file storage
func (s *Storage) GetFile(modelID, sha, filename string) (io.ReadSeeker, error) {
	modelPath := utils.ConvertModelIDToHFPath(modelID)
	// Create the file path
	filePath := filepath.Join(s.baseDir, modelPath, "snapshots", sha, filename)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s/%s", modelID, filename)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// FileExists checks if a file exists in the file storage
func (s *Storage) FileExists(modelID, sha, filename string) (os.FileInfo, bool) {
	modelPath := utils.ConvertModelIDToHFPath(modelID)
	// Create the file path
	filePath := filepath.Join(s.baseDir, modelPath, "snapshots", sha, filename)

	// Check if the file exists
	info, err := os.Stat(filePath)
	return info, err == nil
}

// DeleteFile deletes a file from the file storage
func (s *Storage) DeleteFile(modelID, filename string) error {
	// Create the file path
	filePath := filepath.Join(s.baseDir, modelID, filename)

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// ListFiles lists all files for a model in the file storage
func (s *Storage) ListFiles(modelID string) ([]string, error) {
	modePath := utils.ConvertModelIDToHFPath(modelID)
	// Create the model directory path
	modelDir := filepath.Join(s.baseDir, modePath)

	// Check if the model directory exists
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	// Read the directory
	entries, err := os.ReadDir(modelDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Extract file names
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

func (s *Storage) RepoInfo(modelID, version string) (*Model, error) {
	modePath := utils.ConvertModelIDToHFPath(modelID)
	modelIndexPath := filepath.Join(s.baseDir, modePath, ".modeindex")

	if _, err := os.Stat(modelIndexPath); err != nil {
		if os.IsNotExist(err) {
			log.Println("Warning: .modelindex file not found, building model index from scratch")
			// don't .modelindex file, return customer data
			return s.buildModelIndex(modelID, version)
		}
		return nil, fmt.Errorf("modelindex file not found: %s", modelID)
	}

	data, err := os.ReadFile(modelIndexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read modelindex file: %w", err)
	}

	var model Model
	err = json.Unmarshal(data, &model)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal modelindex file: %w", err)
	}

	return &model, nil
}

func (s *Storage) buildModelIndex(modelID, version string) (*Model, error) {
	author := strings.Split(modelID, "/")[0]
	modePath := utils.ConvertModelIDToHFPath(modelID)

	sha, err := s.getRepoSha(modelID, version)
	if err != nil {
		return nil, err
	}

	modelDir := filepath.Join(s.baseDir, modePath, "snapshots", sha)
	log.Println("buildModelIndex", modelDir)
	var (
		totalSize int64
		fileList  []Sibling = make([]Sibling, 0)
	)
	err = filepath.WalkDir(modelDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fileList = append(fileList, Sibling{Rfilename: d.Name()})
		if info.Mode()&os.ModeSymlink == 0 {
			totalSize += info.Size()
			return nil
		}
		target, err := os.Readlink(filepath.Join(modelDir, d.Name()))
		if err != nil {
			return err
		}
		// log.Println("buildModelIndex", target)
		_, etag := filepath.Split(target)
		absPath := filepath.Join(s.baseDir, modePath, "blobs", etag)
		targetInfo, err := os.Stat(absPath)
		if err != nil {
			return err
		}
		totalSize += targetInfo.Size()
		return nil
	})
	if err != nil {
		log.Printf("failed to walk model directory: %v", err)
		return nil, err
	}

	return &Model{
		ID:           modelID,
		ModelID:      modelID,
		Author:       author,
		SHA:          sha,
		LastModified: time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
		// TODO, this field is not file total size, is this model is need gpu memory.
		UsedStorage: totalSize,
		Siblings:    fileList,
	}, nil
}

func (s *Storage) getRepoSha(modelID, version string) (string, error) {
	modePath := utils.ConvertModelIDToHFPath(modelID)
	versionFilePath := filepath.Join(s.baseDir, modePath, "refs", version)
	if _, err := os.Stat(versionFilePath); err != nil {
		return "", fmt.Errorf("version file not found: %s", versionFilePath)
	}
	data, err := os.ReadFile(versionFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read version file: %w", err)
	}
	return string(data), nil
}

func (s *Storage) FileEtag(modelID, sha, filename string) string {
	modelPath := utils.ConvertModelIDToHFPath(modelID)
	filePath := filepath.Join(s.baseDir, modelPath, "snapshots", sha, filename)
	targetPath, err := os.Readlink(filePath)
	if err != nil {
		return ""
	}
	absPath, _ := filepath.Abs(targetPath)
	_, etag := filepath.Split(absPath)
	return etag
}
