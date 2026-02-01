package model

import "time"

type ModelInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Modified   time.Time `json:"modified_at"`
	Digest     string    `json:"digest,omitempty"`
	Family     string    `json:"family,omitempty"`
	Parameters string    `json:"parameters,omitempty"`
	Format     string    `json:"format,omitempty"`
}

type PullProgress struct {
	Status    string  `json:"status"`
	Digest    string  `json:"digest,omitempty"`
	Total     int64   `json:"total,omitempty"`
	Completed int64   `json:"completed,omitempty"`
	Percent   float64 `json:"-"`
}
