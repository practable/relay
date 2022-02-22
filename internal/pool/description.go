package pool

import (
	"github.com/google/uuid"
	"github.com/practable/relay/internal/booking/models"
)

// NewDescriptionFromModel returns a pointer to a Description converted from the API's model
func NewDescriptionFromModel(md *models.Description) *Description {
	// assume empty names and types are ok
	// check pointer values so we don't get a panic
	var name, thisType string

	if md == nil {
		return &Description{}
	}

	if md.Name != nil {
		name = *md.Name
	}

	if md.Type != nil {
		thisType = *md.Type
	}

	d := &Description{
		Name: name,
		ID:   md.ID, // PUT methods may supply ID
		Type: thisType,
		DisplayInfo: DisplayInfo{
			Further: md.Further,
			Image:   md.Image,
			Long:    md.Long,
			Short:   md.Short,
			Thumb:   md.Thumb,
		},
	}

	return d
}

// NewConfigFromModel returns a pointer to a Config converted from the API's model
func NewConfigFromModel(mc *models.Config) *Config {

	var url string

	if mc == nil {
		return &Config{}
	}

	if mc.URL != nil {
		url = *mc.URL
	}

	c := &Config{
		URL: url,
	}

	return c
}

// ConvertToModel returns a pointer to the Description represented in the API's model
func (d *Description) ConvertToModel() *models.Description {

	// avoid pointers to the original description
	// in case we edit original after making this copy
	name := d.Name
	thisType := d.Type

	return &models.Description{
		Further: d.Further,
		ID:      d.ID,
		Image:   d.Image,
		Long:    d.Long,
		Name:    &name,
		Short:   d.Short,
		Thumb:   d.Thumb,
		Type:    &thisType,
	}

}

// NewDescription returns a pointer to a Description with given name and randomly generated UUID as ID
func NewDescription(name string) *Description {
	return &Description{
		Name: name,
		ID:   uuid.New().String(),
	}
}

// WithID sets the ID of the Description
func (d *Description) WithID(id string) *Description {
	d.ID = id
	return d
}

// SetID sets the ID of the Description
func (d *Description) SetID(item string) {
	d.ID = item
}

// SetType sets the Type field in the Description
func (d *Description) SetType(item string) {
	d.Type = item
}

// SetShort sets the short description field in the Descriptiom
func (d *Description) SetShort(item string) {
	d.Short = item
}

// SetLong sets the long description field in the Description
func (d *Description) SetLong(item string) {
	d.Long = item
}

// SetFurther sets the URL for further information (in string format)
func (d *Description) SetFurther(item string) {
	d.Further = item
}

// SetThumb sets the URL of the thumbnail image of the activity
func (d *Description) SetThumb(item string) {
	d.Thumb = item
}

// SetImage sets the URL of the large/main image of the activity
func (d *Description) SetImage(item string) {
	d.Image = item
}
