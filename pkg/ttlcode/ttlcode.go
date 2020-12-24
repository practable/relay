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
	store map[string]ExpToken

	// TTL is lifetime in seconds of a code
	ttl int64

	//
	closed chan struct{}
}

// GetTime gets the current Unix time in seconds.
func GetTime() int64 {
	return time.Now().Unix()
}

func (c *CodeStore) GetTime() int64 {
	return GetTime()
}

// NewDefaultCodeSTore returns a codestore with code lifetime of 30seconds.
func NewDefaultCodeStore() *CodeStore {
	c := &CodeStore{
		store:  make(map[string]ExpToken),
		ttl:    30,
		closed: make(chan struct{}),
	}
	go c.keepClean()
	return c
}

// WithTTL sets the code lifetime of the new CodeStore (in seconds).
func (c *CodeStore) WithTTL(ttl int64) *CodeStore {
	c.ttl = ttl
	return c
}

func (c *CodeStore) Close() {
	c.Lock()
	defer c.Unlock()
	close(c.closed)
}

// keepClean periodically removes stale codes/tokens
func (c *CodeStore) keepClean() {
	for {
		select {
		case <-c.closed:
			return
		case <-time.After(time.Duration(2*c.ttl) * time.Second):
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
	c.store[code] = NewExpToken(token, c.ttl)
	return code
}

// ExchangeCode swaps a (valid) code for the associated token.
func (c *CodeStore) ExchangeCode(code string) (jwt.Token, error) {
	c.Lock()
	defer c.Unlock()
	token, ok := c.store[code]
	if !ok {
		return jwt.Token{}, errors.New("No code")
	}
	// can only get code once.
	delete(c.store, code)
	return token.Token, nil

}

// CleanExpired removes stale codes & tokens from the CodeStore
func (c *CodeStore) CleanExpired() {
	c.Lock()
	defer c.Unlock()
	for code, token := range c.store {
		if token.Expired() {
			delete(c.store, code)
		}
	}
}

func (c *CodeStore) GetTTL() int64 {
	return c.ttl
}

func (c *CodeStore) GetCodeCount() int {
	return len(c.store)
}
