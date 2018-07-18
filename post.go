package cli

import (
	"time"

	yaml "gopkg.in/yaml.v2"
)

type Post struct {
	Title       string
	Slug        string
	Description string
	Draft       bool
	PublishedAt time.Time `yaml:"published-at"`
	Body        string    `yaml:"-"` // should this be byte?
}

func (p *Post) ParseYAML(b []byte) error {
	err := yaml.Unmarshal(b, p)
	if err != nil {
		return err
	}
	return nil
}
