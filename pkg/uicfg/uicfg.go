// Package uicfg holds models of the json format config files for ui
package uicfg

type Image struct {
	For        string `json:"for"`
	Src        string `json:"src"`
	Alt        string `json:"alt"`
	FigCaption string `json:"figcaption"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type KV struct {
	K string `json:"k"`
	V string `json:"v"`
}

type Parameters struct {
	For string `json:"for"`
	Are []KV   `json:"are"`
}

type Config struct {
	Name       string       `json:"name"`
	Version    string       `json:"version"`
	Date       int64        `json:"date"`
	Aud        string       `json:"aud"`
	Images     []Image      `json:"images"`
	Parameters []Parameters `json:"parameters"`
}
