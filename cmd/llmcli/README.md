# LLM Distribution CLI Tool

A command-line tool for downloading models from Hugging Face and saving them locally for use with the LLM Distribution server.

## Installation

```bash
go build -o llmcli cmd/llmcli/main.go
```

## Usage

```bash
# Download a model with default settings
./llmcli Qwen/Qwen2-0.5B-Instruct

# Download a model with custom settings
./llmcli --base-dir /path/to/models --revision main Qwen/Qwen2-0.5B-Instruct
```

## Options

- `--base-dir`: Base directory for storing models (default: `/tmp/LLMDistribution`)
- `--revision`: Model revision/version to download (default: `main`)

## How It Works

1. The CLI tool sets the HF_HOME environment variable to the base directory.
2. It creates a directory structure that matches the Hugging Face cache format: `{base_dir}/hub/models--{owner}--{model_name}`.
3. It uses the Hugging Face client to download the model files, which automatically saves them to the correct location in the HF_HOME cache directory.
4. It requests model index information directly from the Hugging Face API (`https://huggingface.co/api/models/{model_id}/revision/{version}`).
5. If the API request fails, it creates a basic model index based on the downloaded files, checking both the model directory and the snapshots directory.
6. The model index information is saved to a `.modeindex` file in the model directory.

## Example

```bash
# Download the Qwen2-0.5B-Instruct model
./llmcli Qwen/Qwen2-0.5B-Instruct

# The model will be downloaded to /tmp/LLMDistribution/hub/models--Qwen--Qwen2-0.5B-Instruct
# The model index information will be saved to /tmp/LLMDistribution/hub/models--Qwen--Qwen2-0.5B-Instruct/.modeindex
```
