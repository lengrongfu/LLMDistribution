package model

import (
	"time"
)

// SiblingFile represents a file in the model repository
type SiblingFile struct {
	RFilename string `json:"rfilename"`
}

// ModelIndexInfo represents model index information
type ModelIndexInfo struct {
	ID           string        `json:"id"`
	ModelID      string        `json:"modelId"`
	Author       string        `json:"author"`
	SHA          string        `json:"sha"`
	LastModified time.Time     `json:"lastModified"`
	Disabled     bool          `json:"disabled"`
	CreatedAt    time.Time     `json:"createdAt"`
	UsedStorage  int64         `json:"usedStorage"`
	Siblings     []SiblingFile `json:"siblings"`
}
