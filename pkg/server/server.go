package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/lengrongfu/LLMDistribution/pkg/api"
	"github.com/lengrongfu/LLMDistribution/pkg/filestorage"
	"github.com/lengrongfu/LLMDistribution/pkg/git"
)

// Server represents the LLM Distribution server
type Server struct {
	router       *mux.Router
	httpServer   *http.Server
	distribution api.Distribution
	baseDir      string
}

// Config represents the server configuration
type Config struct {
	Host        string
	Port        int
	StorageType api.StorageType
	GitBaseDir  string
	FileBaseDir string
}

// NewServer creates a new LLM Distribution server
func NewServer(config Config) (*Server, error) {
	// Create base directories
	if err := os.MkdirAll(config.GitBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create Git base directory: %w", err)
	}
	if err := os.MkdirAll(config.FileBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create File base directory: %w", err)
	}

	// Initialize the Git distribution
	gitDist, err := git.NewDistribution(config.GitBaseDir, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git distribution: %w", err)
	}

	// Initialize the File distribution
	fileDist, err := filestorage.NewDistribution(config.FileBaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create File distribution: %w", err)
	}

	// Create the router with StrictSlash option
	router := mux.NewRouter().StrictSlash(true)

	// Create the server
	server := &Server{
		router:  router,
		baseDir: filepath.Dir(config.GitBaseDir), // Use parent directory as base
	}
	switch config.StorageType {
	case api.GitStorage:
		server.distribution = gitDist
	case api.FileStorage:
		server.distribution = fileDist
	default:
		return nil, fmt.Errorf("invalid storage type: %d", config.StorageType)
	}

	// Set up routes
	server.setupRoutes()

	// Create the HTTP server
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server, nil
}

// setupRoutes sets up the server routes
func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Model routes - 顺序很重要，更具体的路由必须先定义
	// 使用正则表达式模式允许 model_id 包含斜杠
	api.HandleFunc("/models/{model_id:.+}/revision/{version}", s.handleGetModelIndex).Methods("GET")
	s.router.HandleFunc("/{model_id:.+}/resolve/{sha}/{filename:.+}", s.handleGetModelFile).Methods("GET", "HEAD")

	// Health check
	s.router.HandleFunc("/health", s.handleHealthCheck).Methods("GET")
}

// Start starts the server
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// handleHealthCheck handles health check requests
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetModelFile handles model file requests
func (s *Server) handleGetModelFile(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetModelFile called with URL: %s", r.URL.Path)
	vars := mux.Vars(r)
	modelID := vars["model_id"]
	shaOrVersion := vars["sha"]
	filename := vars["filename"]
	log.Printf("handleGetModelFile: modelID=%s, sha=%s, filename=%s", modelID, shaOrVersion, filename)

	sha := s.distribution.RepoSha(modelID, shaOrVersion)
	// 2. 检查文件是否存在
	fileInfo, exist := s.distribution.FileExists(modelID, sha, filename)
	if !exist {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	etga := s.distribution.FileEtag(modelID, sha, filename)
	log.Println("handleGetModelFile: etga=", etga)

	// 3. 设置 HTTP 头（关键优化点）
	w.Header().Set("X-Repo-Commit", sha)
	w.Header().Set("ETag", etga)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("inline; filename=\"%s\"", fileInfo.Name()))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	if r.Method == "HEAD" {
		return
	}

	// 4. 流式传输（核心代码）
	file, err := s.distribution.GetFile(modelID, sha, filename)
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), file)
}

// Dataset-related handlers removed

// Inference-related handlers removed

// handleUploadModelFile handles model file upload requests
func (s *Server) handleUploadModelFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["model_id"]

	// Get the filename from the query parameters
	filename := r.URL.Query().Get("path")
	if filename == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	// Store the file in the appropriate storage
	filePath, err := s.distribution.StoreFile(modelID, filename, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store file: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the file path
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"path": filePath})
}

// Dataset upload handler removed

// handleGetModelIndex handles model index information requests
func (s *Server) handleGetModelIndex(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetModelIndex called with URL: %s", r.URL.Path)
	vars := mux.Vars(r)
	log.Printf("handleGetModelIndex vars: %+v", vars)
	modelID := vars["model_id"]
	version := vars["version"]
	log.Printf("handleGetModelIndex: modelID=%s, version=%s", modelID, version)

	// Create the model index information
	indexInfo, err := s.distribution.RepoInfo(modelID, version)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get model index: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the model index information
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(indexInfo)
}
