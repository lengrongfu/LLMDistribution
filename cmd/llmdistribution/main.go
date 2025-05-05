package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lengrongfu/LLMDistribution/pkg/api"
	"github.com/lengrongfu/LLMDistribution/pkg/server"
)

func main() {
	// Create base directories for storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	// Parse command line flags
	host := flag.String("host", "0.0.0.0", "Server host")
	port := flag.Int("port", 8081, "Server port")
	gitBaseDir := flag.String("git-base-dir", filepath.Join(homeDir, ".llm-distribution", "git"), "Git base directory")
	fileBaseDir := flag.String("file-base-dir", "/Users/lengrongfu/.cache/huggingface/", "File base directory")
	storageType := flag.Int("storage-type", 1, "Storage type (0: Git, 1: File)")
	flag.Usage = func() {
		log.Println("Usage: llmdistribution [options]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Create the server configuration
	config := server.Config{
		Host:        *host,
		Port:        *port,
		StorageType: api.StorageType(*storageType),
		GitBaseDir:  *gitBaseDir,
		FileBaseDir: *fileBaseDir,
	}

	// Create the server
	srv, err := server.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shut down the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
