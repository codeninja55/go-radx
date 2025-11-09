package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// Attachment represents a FHIR Attachment.
type Attachment struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Mime type of the content, with charset etc.
	ContentType *string `json:"contentType,omitempty"`
	// Human language of the content (BCP-47)
	Language *string `json:"language,omitempty"`
	// Data inline, base64ed
	Data *string `json:"data,omitempty"`
	// Uri where the data can be found
	URL *string `json:"url,omitempty"`
	// Number of bytes of content (if url provided)
	Size *int64 `json:"size,omitempty"`
	// Hash of the data (sha-1, base64ed)
	Hash *string `json:"hash,omitempty"`
	// Label to display in place of the data
	Title *string `json:"title,omitempty"`
	// Date attachment was first created
	Creation *primitives.DateTime `json:"creation,omitempty"`
	// Height of the image in pixels (photo/video)
	Height *int `json:"height,omitempty"`
	// Width of the image in pixels (photo/video)
	Width *int `json:"width,omitempty"`
	// Number of frames if > 1 (photo)
	Frames *int `json:"frames,omitempty"`
	// Length in seconds (audio / video)
	Duration *float64 `json:"duration,omitempty"`
	// Number of printed pages
	Pages *int `json:"pages,omitempty"`
}
