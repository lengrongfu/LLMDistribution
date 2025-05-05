package git

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Storage represents a Git storage system
type Storage struct {
	// Base directory for Git repositories
	baseDir string
	// Whether to use Git LFS
	useLFS bool
}

// NewStorage creates a new Git storage
func NewStorage(baseDir string, useLFS bool) (*Storage, error) {
	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Storage{
		baseDir: baseDir,
		useLFS:  useLFS,
	}, nil
}

// StoreFile stores a file in the Git repository
func (s *Storage) StoreFile(modelID, filename string, content io.Reader) (string, error) {
	// Initialize the repository if it doesn't exist
	repoPath, err := s.initRepository(modelID)
	if err != nil {
		return "", fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Create the file path
	filePath := filepath.Join(repoPath, filename)

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

	// Track the file with Git LFS if enabled and it's a binary file
	if s.useLFS {
		ext := filepath.Ext(filename)
		if isBinaryExtension(ext) {
			cmd := exec.Command("git", "lfs", "track", filename)
			cmd.Dir = repoPath
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("failed to track file with Git LFS: %w", err)
			}
		}
	}

	// Add the file to Git
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to add file to Git: %w", err)
	}

	// Commit the changes
	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Add %s", filename))
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to commit changes: %w", err)
	}

	return filePath, nil
}

// GetFile retrieves a file from the Git repository
func (s *Storage) GetFile(modelID, filename string) (io.ReadSeeker, error) {
	// Get the repository path
	repoPath := filepath.Join(s.baseDir, modelID)

	// Create the file path
	filePath := filepath.Join(repoPath, filename)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// FileExists checks if a file exists in the Git repository
func (s *Storage) FileExists(modelID, filename string) (fs.FileInfo, bool) {
	// Get the repository path
	repoPath := filepath.Join(s.baseDir, modelID)

	// Create the file path
	filePath := filepath.Join(repoPath, filename)

	// Check if the file exists
	info, err := os.Stat(filePath)
	return info, err == nil
}

// ListFiles lists all files in the Git repository for a model
func (s *Storage) ListFiles(modelID string) ([]string, error) {
	// Get the repository path
	repoPath := filepath.Join(s.baseDir, modelID)

	// Check if the repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository not found: %s", modelID)
	}

	// Use git ls-files to list all files in the repository
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// If git ls-files fails, fall back to listing files in the directory
		return listFilesInDirectory(repoPath)
	}

	// Parse the output
	files := strings.Split(string(output), "\n")

	// Filter out empty strings
	var result []string
	for _, file := range files {
		if file != "" {
			result = append(result, file)
		}
	}

	return result, nil
}

// initRepository initializes a Git repository
func (s *Storage) initRepository(repoName string) (string, error) {
	repoPath := filepath.Join(s.baseDir, repoName)

	// Check if the repository already exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		return repoPath, nil
	}

	// Create the repository directory if it doesn't exist
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Initialize the Git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to initialize Git repository: %w", err)
	}

	// Initialize Git LFS if enabled
	if s.useLFS {
		cmd = exec.Command("git", "lfs", "install")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to initialize Git LFS: %w", err)
		}
	}

	// Configure Git user
	cmd = exec.Command("git", "config", "user.name", "LLM Distribution")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to configure Git user name: %w", err)
	}

	cmd = exec.Command("git", "config", "user.email", "llm-distribution@example.com")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to configure Git user email: %w", err)
	}

	return repoPath, nil
}

// isBinaryExtension checks if a file extension is typically used for binary files
func isBinaryExtension(ext string) bool {
	binaryExtensions := map[string]bool{
		".bin":         true,
		".pt":          true,
		".pth":         true,
		".ckpt":        true,
		".safetensors": true,
		".onnx":        true,
		".h5":          true,
		".pb":          true,
	}

	return binaryExtensions[ext]
}

// listFilesInDirectory recursively lists all files in a directory
func listFilesInDirectory(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the relative path
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Add the file to the list
		files = append(files, relPath)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
