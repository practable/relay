package manifest

// Ref represents a reference to a manifest
type Ref string //manifest reference

// Manifest represents a complete listing of the experiments available to book
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

// Group represnts a booking group (a list of pools)
type Group struct {
	// Pools represents all the pools in the group
	Pools []Ref
}

// Pool represents a booking Pool (a list of activities)
type Pool struct {
	Description

	MinSession uint64

	MaxSession uint64

	Activities []Ref
}

// Config represents a configuration for a user interface
type Config struct {
	URL string `yaml:"url"`
}

// Activity represents an activity
type Activity struct {
	Config Config `yaml:"config"`

	Description Ref `yaml:"description"`

	UISet Ref `yaml:"UISet"`

	ExpiresAt int64 `yaml:"exp"`

	Streams map[string]*Stream `yaml:"streams"`
}

// UISet is an array of references to UI to be used with an activity
type UISet []Ref

// StreamSet is an array of references to streams to be used with an activity
type StreamSet []Ref

// Stream represents a connection to a relay instance
type Stream struct {
	For            string   `yaml:"for"`
	URL            string   `yaml:"url"`
	Audience       string   `yaml:"audience"`
	ConnectionType string   `yaml:"ct"`
	Topic          string   `yaml:"topic"`
	Verb           string   `yaml:"verb"`
	Scopes         []string `yaml:"scopes"`
}

// Description represents a description of an activity, for displaying in the booking system / application
type Description struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Short   string `json:"short,omitempty"`
	Long    string `json:"long,omitempty"`
	Further string `json:"further,omitempty"`
	Thumb   string `json:"thumb,omitempty"`
	Image   string `json:"image,omitempty"`
}

// UI represents a UI (User Interface)
type UI struct {
	// URL with moustache {{key}} templating for stream connections
	Description
	URL             string   `json:"url"`
	StreamsRequired []string `json:"streamsRequired"`
}
