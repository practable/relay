package manifest

type Ref string //manifest reference

// Store represents all the pools and groups in the poolstore
type Manifest struct {

	// Groups represents all the groups in the poolstore
	Groups map[Ref]*Group `yaml:"groups"`

	// Pools represents all the pools in the poolstore
	Pools map[Ref]*Pool `yaml:"pools"`

	Activities map[Ref]*Activity `yaml:"activities"`

	UIs map[Ref]*UI

	UISets map[Ref]*UISet `yaml:"UISets"`

	Descriptions map[Ref]*Description
}

type Group struct {
	// Pools represents all the pools in the group
	Pools []Ref
}

type Pool struct {
	Description

	MinSession uint64

	MaxSession uint64

	Activities []Ref
}

type Activity struct {
	Description Ref `yaml:"description"`

	UISet Ref `yaml:"UISet"`

	ExpiresAt int64 `yaml:"exp"`

	Streams map[string]*Stream `yaml:"streams"`
}

type UISet []Ref
type StreamSet []Ref

type Stream struct {
	For            string   `yaml:"for"`
	URL            string   `yaml:"url"`
	Audience       string   `yaml:"audience"`
	ConnectionType string   `yaml:"ct"`
	Topic          string   `yaml:"topic"`
	Verb           string   `yaml:"verb"`
	Scopes         []string `yaml:"scopes"`
}

type Description struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Short   string `json:"short,omitempty"`
	Long    string `json:"long,omitempty"`
	Further string `json:"further,omitempty"`
	Thumb   string `json:"thumb,omitempty"`
	Image   string `json:"image,omitempty"`
}

type UI struct {
	// URL with moustache {{key}} templating for stream connections
	Description
	URL             string   `json:"url"`
	StreamsRequired []string `json:"streamsRequired"`
}
