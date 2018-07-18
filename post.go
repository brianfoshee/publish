package cli

import (
	"time"

	blackfriday "gopkg.in/russross/blackfriday.v2"
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
	return yaml.Unmarshal(b, p)
}

func (p *Post) ParseMarkdown(b []byte) error {
	output := blackfriday.Run(b)
	p.Body = string(output)
	return nil
}
