package filestorage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lengrongfu/LLMDistribution/pkg/api/model"
)

// Distribution implements the api.Distribution interface for file storage
type Distribution struct {
	Storage *Storage // Exported for access in server
}

// NewDistribution creates a new file storage distribution
func NewDistribution(baseDir string) (*Distribution, error) {
	storage, err := NewStorage(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}

	return &Distribution{
		Storage: storage,
	}, nil
}

// StoreFile stores a file in file storage
func (d *Distribution) StoreFile(modelID, filename string, content io.Reader) (string, error) {
	return d.Storage.StoreFile(modelID, filename, content)
}

// GetFile retrieves a file from file storage
func (d *Distribution) GetFile(modelID, sha, filename string) (io.ReadSeeker, error) {
	return d.Storage.GetFile(modelID, sha, filename)
}

// FileExists checks if a file exists in file storage
func (d *Distribution) FileExists(modelID, sha, filename string) (os.FileInfo, bool) {
	return d.Storage.FileExists(modelID, sha, filename)
}

// ListFiles lists all files in file storage for a model
func (d *Distribution) ListFiles(modelID string) ([]string, error) {
	return d.Storage.ListFiles(modelID)
}

// GetStorageInfo gets storage information for a model in file storage
func (d *Distribution) GetStorageInfo(modelID string) (int64, error) {
	// Get the model directory path
	modelDir := filepath.Join(d.Storage.baseDir, modelID)

	// Check if the model directory exists
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return 0, fmt.Errorf("model not found: %s", modelID)
	}

	// Get the list of files
	files, err := d.ListFiles(modelID)
	if err != nil {
		return 0, err
	}

	// Calculate the total size
	var totalSize int64
	for _, file := range files {
		filePath := filepath.Join(modelDir, file)
		info, err := os.Stat(filePath)
		if err == nil {
			totalSize += info.Size()
		}
	}

	return totalSize, nil
}

func (d *Distribution) RepoInfo(modelID, version string) (model.ModelIndexInfo, error) {
	mode, err := d.Storage.RepoInfo(modelID, version)
	if err != nil {
		return model.ModelIndexInfo{}, err
	}
	siblings := make([]model.SiblingFile, len(mode.Siblings))
	for i, sibling := range mode.Siblings {
		siblings[i] = model.SiblingFile{
			RFilename: sibling.Rfilename,
		}
	}

	return model.ModelIndexInfo{
		ID:           mode.ID,
		ModelID:      mode.ModelID,
		Author:       mode.Author,
		SHA:          mode.SHA,
		LastModified: mode.LastModified,
		Disabled:     mode.Disabled,
		CreatedAt:    mode.CreatedAt,
		UsedStorage:  mode.UsedStorage,
		Siblings:     siblings,
	}, nil
}

func (d *Distribution) FileEtag(modelID, sha, filename string) string {
	return d.Storage.FileEtag(modelID, sha, filename)
}

func (d *Distribution) RepoSha(modelID, version string) string {
	if sha, err := d.Storage.getRepoSha(modelID, version); err != nil {
		return version
	} else {
		return sha
	}
}

// Model-related methods removed - not needed
