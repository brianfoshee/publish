package jsonapi

// base is the base JSONAPI for either arrays or individual structs
type Base struct {
	Data     interface{}   `json:"data"`
	Links    *Links        `json:"links,omitempty"`
	Included []interface{} `json:"included,omitempty"`
}

type Links struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Next  string `json:"next,omitempty"`
	Prev  string `json:"prev,omitempty"`
}
