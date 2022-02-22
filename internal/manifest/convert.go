package manifest

import (
	"github.com/timdrysdale/relay/pkg/bc/models"
)

// GetPool returns a pointer to the referenced pool
func (m *Manifest) GetPool(pref Ref) *models.Pool {

	return m.Pools[pref].ToModel()
}

// ToModel returns a pointer of type models.Pool type, representing the referenced pool
func (p *Pool) ToModel() *models.Pool {

	return &models.Pool{
		Description: p.Description.ToModel(),
		MinSession:  int64(p.MinSession),
		MaxSession:  int64(p.MaxSession),
	}

}

// GetActivitiesInPool rehydrates activities into a complete models.Activity type
// It needs data outside of the pool, hence method on Manifest, with pool as a ref
// just so it is not tempting to refactor to a method on a pool pointer like
// in pkg/pool (which won't work here due to refactoring the structure to make
// it easier to human edit - activities here are not childs of the pool in the data
// structure but siblings with a link-by-Ref scheme to let the parent-child r'ship
// be recreated in the models representation
func (m *Manifest) GetActivitiesInPool(pref Ref) []*models.Activity {

	mas := []*models.Activity{}

	p := m.Pools[pref]

	for _, aref := range p.Activities {

		mas = append(mas, m.GetActivityModel(aref))
	}

	return mas
}

// GetActivityModel returns a pointer of type models.Activity type representing the referenced activity
func (m *Manifest) GetActivityModel(aref Ref) *models.Activity {

	a := m.Activities[aref]

	d := m.Descriptions[a.Description]

	exp := float64(a.ExpiresAt)

	return &models.Activity{
		Config:      GetConfigModel(a.Config),
		Description: d.ToModel(),
		Exp:         &exp,
		Uis:         m.GetUISetModel(a.UISet),
		Streams:     GetStreamsModel(a.Streams),
	}
}

// GetConfigModel returns a pointer of type model.Config representing the configuration
func GetConfigModel(c Config) *models.Config {
	return &models.Config{URL: &c.URL}
}

// GetStreamsModel returns an array of pointers to models.Streams representing the stream map
func GetStreamsModel(streams map[string]*Stream) []*models.Stream {

	mss := []*models.Stream{}

	for _, s := range streams {
		mss = append(mss, s.ToModel())
	}
	return mss
}

// ToModel returns a pointer of type models.Stream representing the stream
func (s *Stream) ToModel() *models.Stream {
	if s == nil {
		return &models.Stream{}
	}
	return &models.Stream{
		For:  &s.For,
		URL:  &s.URL,
		Verb: &s.Verb,
		Permission: &models.Permission{
			Topic:          &s.Topic,
			ConnectionType: &s.ConnectionType,
			Audience:       &s.Audience,
			Scopes:         s.Scopes,
		},
	}
}

// GetUISetModel returns an array of pointers type models.UserInterface representing the referenced UISet
func (m *Manifest) GetUISetModel(usref Ref) []*models.UserInterface {

	uirefs := m.UISets[usref]

	uis := []*models.UserInterface{}

	for _, uiref := range *uirefs {
		uis = append(uis, m.GetUIModel(uiref))
	}

	return uis
}

// GetUIModel returns a pointer to the referenced UI as type models.UserInterface
func (m *Manifest) GetUIModel(uiref Ref) *models.UserInterface {

	ui := m.UIs[uiref]

	return &models.UserInterface{
		Description:     ui.Description.ToModel(),
		URL:             &ui.URL,
		StreamsRequired: ui.StreamsRequired,
	}

}

// ToModel returns a pointer of type models.Description representing the description
func (d *Description) ToModel() *models.Description {

	if d == nil {
		return &models.Description{}
	}
	return &models.Description{
		ID:      "",
		Further: d.Further,
		Image:   d.Image,
		Long:    d.Long,
		Name:    &d.Name,
		Short:   d.Short,
		Thumb:   d.Thumb,
		Type:    &d.Type,
	}

}
