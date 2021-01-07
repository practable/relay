package pool

import (
	"github.com/google/uuid"
	"github.com/timdrysdale/relay/pkg/booking/models"
)

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

func NewDescription(name string) *Description {
	return &Description{
		Name: name,
		ID:   uuid.New().String(),
	}
}

func (d *Description) WithID(id string) *Description {
	d.ID = id
	return d
}

func (d *Description) SetID(item string) {
	d.ID = item
}

func (d *Description) SetType(item string) {
	d.Type = item
}

func (d *Description) SetShort(item string) {
	d.Short = item
}
func (d *Description) SetLong(item string) {
	d.Long = item
}
func (d *Description) SetFurther(item string) {
	d.Further = item
}

func (d *Description) SetThumb(item string) {
	d.Thumb = item
}

func (d *Description) SetImage(item string) {
	d.Image = item
}
