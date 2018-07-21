package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	blackfriday "gopkg.in/russross/blackfriday.v2"
	yaml "gopkg.in/yaml.v2"
)

type Post struct {
	Title       string    `json:"title"`
	Slug        string    `json:"id"`
	Description string    `json:"description"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published-at" yaml:"published-at"`
	Body        string    `json:"body", yaml:"-"`
}

// TODO validate
func (p *Post) parseYAML(b []byte) error {
	return yaml.Unmarshal(b, p)
}

func (p *Post) parseMarkdown(b []byte) error {
	output := blackfriday.Run(b)
	p.Body = string(output)
	return nil
}

func (p *Post) processFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	// separate frontmatter (yaml) & markdown
	arr := strings.Split(string(b), "---")
	if len(arr) != 2 {
		return fmt.Errorf("splitting file did not give two parts")
	}
	fm := arr[0]
	md := arr[1]

	// validate frontmatter contains all required fields
	if err := p.parseYAML([]byte(fm)); err != nil {
		return err
	}
	// convert markdown into html
	if err := p.parseMarkdown([]byte(md)); err != nil {
		return err
	}

	return nil
}
