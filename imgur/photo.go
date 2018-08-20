package imgur

import "time"

type Photo struct {
	Taken       time.Time `json:"taken"`
	Description string    `json:"description"`
	Gallery     string    `json:"gallery"`
}
