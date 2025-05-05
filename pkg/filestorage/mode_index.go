package filestorage

import "time"

// Storage represents a file storage system
type Model struct {
	ObjectID         string           `json:"_id"`
	ID               string           `json:"id"`
	Private          bool             `json:"private"`
	PipelineTag      string           `json:"pipeline_tag"`
	LibraryName      string           `json:"library_name"`
	Tags             []string         `json:"tags"`
	Downloads        int              `json:"downloads"`
	Likes            int              `json:"likes"`
	ModelID          string           `json:"modelId"`
	Author           string           `json:"author"`
	SHA              string           `json:"sha"`
	LastModified     time.Time        `json:"lastModified"`
	Gated            bool             `json:"gated"`
	Disabled         bool             `json:"disabled"`
	WidgetData       []WidgetData     `json:"widgetData"`
	ModelIndex       interface{}      `json:"model-index"` // 根据实际情况可能需要定义具体类型
	Config           Config           `json:"config"`
	CardData         CardData         `json:"cardData"`
	TransformersInfo TransformersInfo `json:"transformersInfo"`
	Siblings         []Sibling        `json:"siblings"`
	Spaces           []string         `json:"spaces"`
	CreatedAt        time.Time        `json:"createdAt"`
	Safetensors      Safetensors      `json:"safetensors"`
	UsedStorage      int64            `json:"usedStorage"`
}

type WidgetData struct {
	Text string `json:"text"`
}

type Config struct {
	Architectures   []string        `json:"architectures"`
	ModelType       string          `json:"model_type"`
	TokenizerConfig TokenizerConfig `json:"tokenizer_config"`
}

type TokenizerConfig struct {
	BosToken     *string `json:"bos_token,omitempty"`
	ChatTemplate string  `json:"chat_template"`
	EosToken     string  `json:"eos_token"`
	PadToken     string  `json:"pad_token"`
	UnkToken     *string `json:"unk_token,omitempty"`
}

type CardData struct {
	License     string   `json:"license"`
	Language    []string `json:"language"`
	PipelineTag string   `json:"pipeline_tag"`
	Tags        []string `json:"tags"`
	BaseModel   string   `json:"base_model"`
}

type TransformersInfo struct {
	AutoModel   string `json:"auto_model"`
	PipelineTag string `json:"pipeline_tag"`
	Processor   string `json:"processor"`
}

type Sibling struct {
	Rfilename string `json:"rfilename"`
}

type Safetensors struct {
	Parameters Parameters `json:"parameters"`
	Total      int64      `json:"total"`
}

type Parameters struct {
	BF16 int64 `json:"BF16"`
}
