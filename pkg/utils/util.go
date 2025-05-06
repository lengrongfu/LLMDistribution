package utils

import "strings"

// convertModelIDToHFPath converts a model ID like "Qwen/Qwen2-0.5B-Instruct" to the
// Hugging Face cache path format like "models--Qwen--Qwen2-0.5B-Instruct"
func ConvertModelIDToHFPath(modelID string) string {
	// Replace slashes with double dashes
	return "models--" + strings.ReplaceAll(modelID, "/", "--")
}
