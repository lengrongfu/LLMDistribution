# LLM Distribution

A server for distributing Large Language Model (LLM) requests to different storage systems, with an API compatible with the Hugging Face Hub API.

## Architecture

The LLM Distribution system consists of the following components:

- **Distribution Interface**: Defines the interface for interacting with model storage.
- **Git Distribution**: Implements the Distribution interface for Git storage.
- **File Distribution**: Implements the Distribution interface for file storage.
- **Composite Distribution**: Combines multiple storage backends into a single Distribution.
- **Git Server**: A Git server for storing model code and configurations.
- **Git LFS**: Git Large File Storage for handling large model files.
- **FileStorage**: A file storage system for storing model files.

## Project Structure

```
LLMDistribution/
├── cmd/
│   └── llmdistribution/  # Main application entry point
│       └── main.go
├── pkg/
│   ├── api/              # Core API for the LLM Distribution system
│   │   ├── distribution.go  # Distribution interface and composite implementation
│   │   └── model/        # Model provider implementation
│   │       └── provider.go
│   ├── client/           # Client for interacting with the LLM Distribution server
│   │   └── client.go
│   ├── filestorage/      # File storage implementation
│   │   ├── distribution.go  # File distribution implementation
│   │   └── storage.go       # File storage backend
│   ├── git/              # Git storage implementation
│   │   ├── distribution.go  # Git distribution implementation
│   │   └── storage.go       # Git storage backend
│   └── server/           # HTTP server implementation
│       └── server.go
├── go.mod
├── go.sum
└── README.md
```

## Installation

```bash
go get github.com/lengrongfu/LLMDistribution
```

## Usage

### Run the server

```bash
# Run with default settings (host: 0.0.0.0, port: 8080)
go run cmd/llmdistribution/main.go

# Run with custom host and port
go run cmd/llmdistribution/main.go -host localhost -port 3000
```

### API Endpoints

The server implements the following API endpoints compatible with the Hugging Face Hub API:

#### Model Files

- `GET /api/models/{model_id}/resolve/{revision}/{filename}`: Resolve a model file
- `GET /api/models/{model_id}/{revision}/{filename}`: Get a model file
- `GET /api/models/{model_id}/info/{version}`: Get model index information
- `PUT /api/models/{model_id}?path={filename}`: Upload a model file

### Client Usage

You can interact with the LLM Distribution server using either HTTP requests or the provided Go client.

#### Using HTTP Requests

##### Upload a model file

```bash
curl -X PUT http://localhost:8080/api/models/gpt2?path=config.json \
  --data-binary @config.json
```

##### Download a model file

```bash
curl -X GET http://localhost:8080/api/models/gpt2/main/config.json \
  -o config.json
```

##### Get model index information

```bash
curl -X GET http://localhost:8080/api/models/Qwen/Qwen2-0.5B-Instruct/info/main
```

Example response:
```json
{
  "id": "Qwen/Qwen2-0.5B-Instruct",
  "modelId": "Qwen/Qwen2-0.5B-Instruct",
  "author": "Qwen",
  "sha": "c540970f9e29518b1d8f06ab8b24cba66ad77b6d",
  "lastModified": "2024-08-21T10:23:36.000Z",
  "disabled": false,
  "createdAt": "2024-06-03T09:06:06.000Z",
  "usedStorage": 7040278603,
  "siblings": [
    {
      "rfilename": ".gitattributes"
    },
    {
      "rfilename": "LICENSE"
    },
    {
      "rfilename": "README.md"
    },
    {
      "rfilename": "config.json"
    },
    {
      "rfilename": "generation_config.json"
    },
    {
      "rfilename": "merges.txt"
    },
    {
      "rfilename": "model.safetensors"
    },
    {
      "rfilename": "tokenizer.json"
    },
    {
      "rfilename": "tokenizer_config.json"
    },
    {
      "rfilename": "vocab.json"
    }
  ]
}
```

#### Using the Go Client

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lengrongfu/LLMDistribution/pkg/client"
)

func main() {
	// Create a new client
	c := client.NewClient("http://localhost:8080")

	// Upload a model file
	path, err := c.UploadModelFileFromPath("gpt2", "config.json", "config.json")
	if err != nil {
		log.Fatalf("Failed to upload model file: %v", err)
	}
	fmt.Printf("Uploaded to: %s\n", path)

	// Download a model file
	if err := c.DownloadModelFileToPath("gpt2", "main", "config.json", "downloaded_config.json"); err != nil {
		log.Fatalf("Failed to download model file: %v", err)
	}
	fmt.Println("Downloaded config.json")

	// Get model index information
	indexInfo, err := c.GetModelIndex(context.Background(), "Qwen/Qwen2-0.5B-Instruct", "main")
	if err != nil {
		log.Fatalf("Failed to get model index: %v", err)
	}
	fmt.Printf("Model ID: %s\n", indexInfo.ModelID)
	fmt.Printf("Author: %s\n", indexInfo.Author)
	fmt.Printf("SHA: %s\n", indexInfo.SHA)
	fmt.Printf("Last Modified: %s\n", indexInfo.LastModified)
	fmt.Printf("Created At: %s\n", indexInfo.CreatedAt)
	fmt.Printf("Used Storage: %d bytes\n", indexInfo.UsedStorage)

	// Print the sibling files
	fmt.Println("Sibling files:")
	for _, sibling := range indexInfo.Siblings {
		fmt.Printf("- %s\n", sibling.RFilename)
	}
}
```

## License

MIT
