package common

import "time"

type Event struct {
	ID    int64
	Name  string
	Stamp time.Time
	Data  map[string]string
}

type NewPortRequest struct {
	Instancename string
}

// InfoResponse represents the information returned in an API call
type InfoResponse struct {
	Status        string
	StatusMessage string
	Data          interface{}
}
