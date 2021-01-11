package pool

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/permission"
)

// NewActivityFromModel converts from pkg/booking type to
// pkg/booking type, making no assumption about presence of ID
// Use .WithNewRandomID() if this is a new activity that
// should have a new random ID associated with it

func CheckActivity(a *Activity) error {

	if a == nil {
		return errors.New("nil pointer to activity")
	}

	if a.RWMutex == nil {
		return errors.New("nil pointer to activity mutex")
	}

	a.RLock()
	defer a.RUnlock()

	if a.ExpiresAt < time.Now().Unix() {
		return fmt.Errorf("activity already expired at %d (time now is %d)", a.ExpiresAt, time.Now().Unix())
	}

	if (Description{}) == a.Description {
		return errors.New("empty description")
	}

	// we can live without pretty much any part of the description in the most
	// basic use cases (won't be pretty on the booking page)
	// but no ID is a problem as we cannot track this activity without

	if a.ID == "" {
		return errors.New("no id")
	}

	for _, s := range a.Streams {

		// empty s.For may be ok in some use cases
		// e.g. streams all of same type
		// so don't check it here as we
		// can't warn, we can only throw an error
		// and that is not be desirable for that case

		p := s.Permission
		if p.Audience == "" {
			return fmt.Errorf("empty audience")
		}

		_, err := url.ParseRequestURI(p.Audience)
		if err != nil {
			return fmt.Errorf("audience not an url because %s", err.Error())
		}

		ct := p.ConnectionType

		if ct != "session" && ct != "shell" {
			return fmt.Errorf("connection_type %s is not session or shell", ct)
		}

		if p.Topic == "" {
			return fmt.Errorf("empty topic")
		}

	}

	for _, u := range a.UI {
		if u.URL == "" {
			return fmt.Errorf("user interface %s missing url", u.Name)
		}
	}

	return nil
}

func NewActivityFromModel(ma *models.Activity) *Activity {
	if ma == nil {
		return &Activity{
			RWMutex: &sync.RWMutex{},
		}
	}
	exp := int64(0)

	if ma.Exp != nil {
		exp = int64(*ma.Exp)
	}

	return &Activity{
		&sync.RWMutex{},
		*NewDescriptionFromModel(ma.Description),
		exp,
		NewStreamsFromModel(ma.Streams),
		NewUIsFromModel(ma.Uis),
	}
}

func NewUIsFromModel(modelUIs []*models.UserInterface) []*UI {

	poolStoreUIs := []*UI{}

	for _, modelUI := range modelUIs {
		poolStoreUIs = append(poolStoreUIs, NewSingleUIFromModel(modelUI))
	}
	return poolStoreUIs
}

func NewSingleUIFromModel(mui *models.UserInterface) *UI {

	if mui == nil {
		return &UI{}
	}

	URL := ""
	if mui.URL != nil {
		URL = *mui.URL
	}

	return &UI{
		Description:     *NewDescriptionFromModel(mui.Description),
		URL:             URL,
		StreamsRequired: mui.StreamsRequired,
	}

}

func NewStreamsFromModel(modelStreams []*models.Stream) map[string]*Stream {

	poolStoreStreams := make(map[string]*Stream)

	for _, modelStream := range modelStreams {
		stream := NewSingleStreamFromModel(modelStream)

		key := uuid.New().String() //backup option
		// avoid issues of mapping to null key by providing
		// a unique key so that
		// if stream is not assigned to a particular
		// purpose - this might occur in some single-stream
		// use cases if it seems superfluous to map properly
		// So this should also help in fault finding because
		// this won't be simple like "video" or "data"
		if stream.For != "" {
			// preferred case!
			key = stream.For
		}
		poolStoreStreams[key] = stream
	}
	return poolStoreStreams
}

func NewSingleStreamFromModel(ms *models.Stream) *Stream {

	if ms == nil {
		return &Stream{}
	}

	var For, URL, Verb string

	if ms.For != nil {
		For = *ms.For
	}

	if ms.URL != nil {
		URL = *ms.URL
	}
	if ms.Verb != nil {
		Verb = *ms.Verb
	}

	empty := ""

	if ms.Permission == nil {
		ms.Permission = &models.Permission{
			Audience:       &empty,
			ConnectionType: &empty,
			Scopes:         []string{},
			Topic:          &empty,
		}
	}

	if ms.Permission.Audience == nil {
		ms.Permission.Audience = &empty
	}
	if ms.Permission.ConnectionType == nil {
		ms.Permission.ConnectionType = &empty
	}
	if ms.Permission.Topic == nil {
		ms.Permission.Topic = &empty
	}

	return &Stream{
		RWMutex: &sync.RWMutex{},
		For:     For,
		URL:     URL,
		Token:   ms.Token,
		Verb:    Verb,
		Permission: permission.Token{
			Topic:          *ms.Permission.Topic,
			ConnectionType: *ms.Permission.ConnectionType,
			Scopes:         ms.Permission.Scopes,
			StandardClaims: jwt.StandardClaims{
				Audience: *ms.Permission.Audience,
			},
		},
	}
}

func (a *Activity) WithNewRandomID() *Activity {
	a.Lock()
	defer a.Unlock()
	a.ID = uuid.New().String()
	return a
}

func MakeClaims(mp *models.Permission) permission.Token {

	if mp == nil {
		return permission.Token{}
	}

	return permission.Token{
		StandardClaims: jwt.StandardClaims{
			Audience: *mp.Audience,
		},
		ConnectionType: *mp.ConnectionType,
		Scopes:         mp.Scopes,
		Topic:          *mp.Topic,
	}

}

func (a *Activity) ConvertToModel() *models.Activity {

	exp := float64(a.ExpiresAt)

	return &models.Activity{
		Description: a.Description.ConvertToModel(),
		Exp:         &exp,
		Streams:     StreamsToModel(a.Streams),
		Uis:         UIsToModel(a.UI),
	}

}

func SingleStreamToModel(s *Stream) *models.Stream {
	if s == nil {
		return &models.Stream{}
	}
	return &models.Stream{
		For:   &s.For,
		URL:   &s.URL,
		Token: s.Token,
		Verb:  &s.Verb,
		Permission: &models.Permission{
			Topic:          &s.Permission.Topic,
			ConnectionType: &s.Permission.ConnectionType,
			Scopes:         s.Permission.Scopes,
			Audience:       &s.Permission.Audience,
		},
	}

}

func StreamsToModel(streams map[string]*Stream) []*models.Stream {

	ms := []*models.Stream{}

	for _, stream := range streams {
		ms = append(ms, SingleStreamToModel(stream))
	}

	return ms
}

func SingleUIToModel(u *UI) *models.UserInterface {
	if u == nil {
		return &models.UserInterface{}
	}

	URL := u.URL

	return &models.UserInterface{
		Description:     u.Description.ConvertToModel(),
		StreamsRequired: u.StreamsRequired,
		URL:             &URL,
	}
}

func UIsToModel(uis []*UI) []*models.UserInterface {
	muis := []*models.UserInterface{}

	for _, ui := range uis {
		muis = append(muis, SingleUIToModel(ui))
	}

	return muis

}

func NewActivity(name string, expires int64) *Activity {
	return &Activity{
		&sync.RWMutex{},
		*NewDescription(name),
		expires,
		make(map[string]*Stream),
		[]*UI{},
	}
}

func (a *Activity) WithID(id string) *Activity {
	a.Lock()
	defer a.Unlock()
	a.ID = id
	return a
}

func (a *Activity) SetID(id string) {
	a.Lock()
	defer a.Unlock()
	a.ID = id
}

func (a *Activity) GetID() string {
	a.Lock()
	defer a.Unlock()
	return a.ID
}

func (a *Activity) AddID() string {
	a.Lock()
	defer a.Unlock()
	id := uuid.New().String()
	a.ID = id
	return id
}

func (a *Activity) AddStream(key string, stream *Stream) {
	a.Lock()
	defer a.Unlock()
	stream.For = key
	s := a.Streams
	s[key] = stream
	a.Streams = s
}

func (a *Activity) AddUI(ui *UI) {

	a.UI = append(a.UI, ui)
}

func NewStream(url string) *Stream {
	s := &Stream{
		&sync.RWMutex{},
		"",
		url,
		"",
		"",
		permission.Token{},
	}
	return s
}

func (s *Stream) WithPermission(p permission.Token) *Stream {
	s.Lock()
	defer s.Unlock()
	s.Permission = p
	return s
}

func (s *Stream) GetPermission() permission.Token {
	s.Lock()
	defer s.Unlock()
	return s.Permission
}

func (s *Stream) SetPermission(p permission.Token) {
	s.Lock()
	defer s.Unlock()
	s.Permission = p
}

func NewUI(url string) *UI {
	return &UI{
		URL: url,
	}
}

func (u *UI) WithStreamsRequired(names []string) *UI {
	u.StreamsRequired = names
	return u
}

func (u *UI) WithDescription(d Description) *UI {
	u.Description = d
	return u
}
