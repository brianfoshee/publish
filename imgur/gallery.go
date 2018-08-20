package imgur

import "time"

type Gallery struct {
	Title       string    `json:"title"`
	Slug        string    `json:"id"`
	Description string    `json:"description:` // from md
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published-at" yaml:"published-at"`
}
