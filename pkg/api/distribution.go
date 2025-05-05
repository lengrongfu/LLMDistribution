package api

import (
	"io"
	"os"

	"github.com/lengrongfu/LLMDistribution/pkg/api/model"
)

// StorageType represents the type of storage
type StorageType int

const (
	// GitStorage represents Git storage (Git Server + Git LFS)
	GitStorage StorageType = iota
	// FileStorage represents file storage
	FileStorage
)

// StorageBackend represents a storage backend
type StorageBackend interface {
	// StoreFile stores a file and returns the path to the stored file
	StoreFile(modelID, filename string, content io.Reader) (string, error)
	// GetFile retrieves a file and returns the path to the file
	GetFile(modelID, filename string) (string, error)
	// FileExists checks if a file exists
	FileExists(modelID, filename string) bool
	// ListFiles lists all files for a model
	ListFiles(modelID string) ([]string, error)
}

// Distribution is an interface that defines the methods for interacting with model storage
type Distribution interface {
	// StoreFile stores a file and returns the path to the stored file
	StoreFile(modelID, filename string, content io.Reader) (string, error)
	// ListFiles lists all files for a model
	ListFiles(modelID string) ([]string, error)
	// GetStorageInfo gets storage information for a model
	GetStorageInfo(modelID string) (int64, error)
	// FileEtag gets the ETag for a file
	FileEtag(modelID, sha, filename string) string
	// FileExists checks if a file exists
	FileExists(modelID, sha, filename string) (os.FileInfo, bool)
	// GetFile retrieves a file and returns the path to the file
	GetFile(modelID, sha, filename string) (io.ReadSeeker, error)
	// RepoInfo gets repository information for a model
	RepoInfo(modeID, version string) (model.ModelIndexInfo, error)
	// RepoSha gets the SHA for a repository
	RepoSha(modelID, version string) string
}
