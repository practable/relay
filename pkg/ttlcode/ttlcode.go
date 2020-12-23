package ttlcode

import (
	"errors"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

// Token represents a token and its expiry time.
// Tokens are assumed valid from time of submission.
type ExpToken struct {

	// Token represents a token of arbitrary type.
	Token jwt.Token

	// Exp represents the expiry Unix time in seconds.
	Exp int64
}

// Expired returns true if the token has expired.
func (t *ExpToken) Expired() bool {
	return GetTime() > t.Exp
}

// NewExpToken creates a new token expiring in ttl seconds from now
func NewExpToken(token jwt.Token, ttl int64) ExpToken {

	return ExpToken{
		Token: token,
		Exp:   GetTime() + ttl,
	}

}

// CodeStore represents the codes, and their associated expiring tokens.
type CodeStore struct {
	// Prevent multiple clients getting the same token by mutexing.
	sync.Mutex

	// Store is a map of codes to Tokens.
	Store map[string]ExpToken

	// TTL is lifetime in seconds of a code
	TTL int64

	//
	closed chan struct{}
}

// GetTime gets the current Unix time in seconds.
func GetTime() int64 {
	return time.Now().Unix()
}

// NewDefaultCodeSTore returns a codestore with code lifetime of 30seconds.
func NewDefaultCodeStore() *CodeStore {
	c := &CodeStore{
		Store:  make(map[string]ExpToken),
		TTL:    30,
		closed: make(chan struct{}),
	}
	go c.keepClean()
	return c
}

// WithTTL sets the code lifetime of the new CodeStore (in seconds).
func (c *CodeStore) WithTTL(ttl int64) *CodeStore {
	c.TTL = ttl
	return c
}

func (c *CodeStore) Close() {
	c.Lock()
	defer c.Unlock()
	close(c.closed)
}

func (c *CodeStore) keepClean() {

	for {
		select {
		case <-c.closed:
			return
		case <-time.After(time.Duration(c.TTL) * time.Second):
			c.CleanExpired()
		}

	}

}

// GenerateCode returns a unique string to be used as a code
func GenerateCode() string {
	// no practical need to check uniqueness with uuid
	return uuid.New().String()
}

// SubmitToken returns a code that can be swapped for the token, until the code/token becomes stale.
func (c *CodeStore) SubmitToken(token jwt.Token) string {
	c.Lock()
	defer c.Unlock()
	code := GenerateCode()
	c.Store[code] = NewExpToken(token, c.TTL)
	return code
}

// ExchangeCode swaps a (valid) code for the associated token.
func (c *CodeStore) ExchangeCode(code string) (jwt.Token, error) {
	c.Lock()
	defer c.Unlock()
	token, ok := c.Store[code]
	if !ok {
		return jwt.Token{}, errors.New("No code")
	}
	// can only get code once.
	delete(c.Store, code)
	return token.Token, nil

}

// CleanExpired removes stale codes & tokens from the CodeStore
func (c *CodeStore) CleanExpired() {
	c.Lock()
	defer c.Unlock()
	for code, token := range c.Store {
		if token.Expired() {
			delete(c.Store, code)
		}
	}
}
