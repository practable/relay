// Package uicfg holds models of the json format config files for ui
package uicfg

// Image represents an image to be used in the UI
type Image struct {
	For        string `json:"for"`
	Src        string `json:"src"`
	Alt        string `json:"alt"`
	FigCaption string `json:"figcaption"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

// KV represents a Key-Value parameter
type KV struct {
	K string `json:"k"`
	V string `json:"v"`
}

// Parameters represents an array of Key-Value parameters
type Parameters struct {
	For string `json:"for"`
	Are []KV   `json:"are"`
}

// Config represents the custom configuration elements of a UI
type Config struct {
	Name       string       `json:"name"`
	Version    string       `json:"version"`
	Date       int64        `json:"date"`
	Aud        string       `json:"aud"`
	Images     []Image      `json:"images"`
	Parameters []Parameters `json:"parameters"`
}
