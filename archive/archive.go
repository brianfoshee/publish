package archive

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
)

type Archive struct {
	Kind  string `json:"kind"`
	Year  int    `json:"year"`
	Month string `json:"month"`
}

// WriteFile writes a json-encoded file of JSONAPI-formatted Archive data
// This only needs to write /archives/posts.json or /archives/galleries.json
// TODO change func name
func WriteArchives(path string, archives []Archive) error {
	d := make(dataArchives, len(archives))

	for i, archive := range archives {
		da := dataArchive{
			Type:       "archives",
			ID:         fmt.Sprintf("%s/%d/%s", archive.Kind, archive.Year, archive.Month),
			Attributes: archive,
		}
		d[i] = da
	}

	sort.Sort(sort.Reverse(d))

	f, err := os.Create(path)
	if err != nil {
		log.Printf("could not open %s: %s", path, err)
		return err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(base{Data: d}); err != nil {
		log.Printf("error encoding archives %s: %s", path, err)
		return err
	}

	return nil
}

// to satisfy JSONAPI
type dataArchive struct {
	Type       string  `json:"type"`
	ID         string  `json:"id"`
	Attributes Archive `json:"attributes"`
}

// base is the base JSONAPI for either arrays or individual structs
type base struct {
	Data interface{} `json:"data"`
}

// dataArchvies is a type to use with sort.Sort etc
type dataArchives []dataArchive

func (d dataArchives) Len() int { return len(d) }

func (d dataArchives) Less(i, j int) bool {
	ia := d[i].Attributes
	ja := d[j].Attributes

	it := time.Date(ia.Year, monthMap[ia.Month], 1, 0, 0, 0, 0, time.UTC)
	jt := time.Date(ja.Year, monthMap[ja.Month], 1, 0, 0, 0, 0, time.UTC)

	return it.Before(jt)
}

func (d dataArchives) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

// Archive.Month's are downcased strings of the month name. dataArchives.Less
// needs to compare two time.Time objects and this map helps to create those
// using time.Date
var monthMap = map[string]time.Month{
	"january":   time.January,
	"february":  time.February,
	"march":     time.March,
	"april":     time.April,
	"may":       time.May,
	"june":      time.June,
	"july":      time.July,
	"august":    time.August,
	"september": time.September,
	"october":   time.October,
	"november":  time.November,
	"december":  time.December,
}
