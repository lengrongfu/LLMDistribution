package git

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lengrongfu/LLMDistribution/pkg/api/model"
)

// Distribution implements the api.Distribution interface for Git storage
type Distribution struct {
	Storage *Storage // Exported for access in server
}

// NewDistribution creates a new Git distribution
func NewDistribution(baseDir string, useLFS bool) (*Distribution, error) {
	storage, err := NewStorage(baseDir, useLFS)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git storage: %w", err)
	}

	return &Distribution{
		Storage: storage,
	}, nil
}

// StoreFile stores a file in Git storage
func (d *Distribution) StoreFile(modelID, filename string, content io.Reader) (string, error) {
	return d.Storage.StoreFile(modelID, filename, content)
}

// GetFile retrieves a file from Git storage
func (d *Distribution) GetFile(modelID, sha, filename string) (io.ReadSeeker, error) {
	return d.Storage.GetFile(modelID, filename)
}

// FileExists checks if a file exists in Git storage
func (d *Distribution) FileExists(modelID, sha, filename string) (fs.FileInfo, bool) {
	return d.Storage.FileExists(modelID, filename)
}

// ListFiles lists all files in Git storage for a model
func (d *Distribution) ListFiles(modelID string) ([]string, error) {
	return d.Storage.ListFiles(modelID)
}

// GetStorageInfo gets storage information for a model in Git storage
func (d *Distribution) GetStorageInfo(modelID string) (int64, error) {
	// Get the repository path
	repoPath := filepath.Join(d.Storage.baseDir, modelID)

	// Check if the repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("repository not found: %s", modelID)
	}

	// Get the list of files
	files, err := d.ListFiles(modelID)
	if err != nil {
		return 0, err
	}

	// Calculate the total size
	var totalSize int64
	for _, file := range files {
		filePath := filepath.Join(repoPath, file)
		info, err := os.Stat(filePath)
		if err == nil {
			totalSize += info.Size()
		}
	}

	return totalSize, nil
}

func (d *Distribution) RepoInfo(modelID, version string) (model.ModelIndexInfo, error) {
	return model.ModelIndexInfo{}, nil
}

func (d *Distribution) FileEtag(modelID, sha, filename string) string {
	return ""
}

func (d *Distribution) RepoSha(modelID, version string) string {
	return ""
}

// Model-related methods removed - not needed
