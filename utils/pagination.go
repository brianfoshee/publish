package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/brianfoshee/publish/jsonapi"
)

type Paginater interface {
	PerPage() int
	Len() int
	Data() []interface{}
	Kind() string
}

const domain = "https://www.brianfoshee.com"

var kindMap = map[string]string{
	"posts":     "blog",
	"galleries": "px",
}

func Paginate(p Paginater) error {
	kind := p.Kind()
	perPage := p.PerPage()
	total := p.Len()
	pages := math.Ceil(float64(total) / float64(perPage))
	data := p.Data()

	filePrefix := fmt.Sprintf("dist/%s/page", kind)
	urlPrefix := fmt.Sprintf("%s/%s/page", domain, kindMap[kind])

	for i := 1; i <= int(pages); i += 1 {
		fname := fmt.Sprintf("%s/%d.json", filePrefix, i)

		f, err := os.Create(fname)
		if err != nil {
			return fmt.Errorf("could not open %s: %s", fname, err)
		}

		low := perPage*i - perPage
		high := perPage * i
		if high > total {
			high = total
		}

		next, prev := "", ""
		if int(pages) > i {
			prev = fmt.Sprintf("%s/%d", urlPrefix, i+1)
		}
		if i > 1 {
			next = fmt.Sprintf("%s/%d", urlPrefix, i-1)
		}

		b := jsonapi.Base{
			Data: data[low:high],
			Links: &jsonapi.Links{
				First: fmt.Sprintf("%s/%s", urlPrefix, kindMap[kind]),
				Last:  fmt.Sprintf("%s/%d", urlPrefix, int(pages)),
				Next:  next,
				Prev:  prev,
			},
		}
		if err := json.NewEncoder(f).Encode(b); err != nil {
			return fmt.Errorf("error encoding %s page %d: %s", kind, i, err)
		}
		f.Close()
	}

	return nil
}
