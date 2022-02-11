package login

import (
	"encoding/json"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

// Token represents a token used for login or booking
type Token struct {

	// Groups represent the group names for the sets of pools
	// which a user can access
	Groups []string `json:"groups"`

	// Scopes controlling access to relay;
	// either ["login"],["user"], or ["login","admin"]
	Scopes []string `json:"scopes"`

	// Pools is a list of pool_id for the pools in the groups
	Pools []string `json:"pools"`

	jwt.StandardClaims
}

// TokenInBody represents a token marshalled into a string
type TokenInBody struct {
	Token string `json:"token"`
}

// String converts a token into a string, returning the string
func (t *Token) String() string {

	pretty, err := json.MarshalIndent(*t, "", "\t")

	if err != nil {
		return fmt.Sprintf("%+v", *t)
	}

	return string(pretty)
}

// NewToken creates a new token (but does not sign it)
func NewToken(audience string, groups, pools []string, scopes []string, iat, nbf, exp int64) Token {
	return Token{
		Groups: groups,
		Scopes: scopes,
		Pools:  pools,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  iat,
			NotBefore: nbf,
			ExpiresAt: exp,
			Audience:  audience,
		},
	}
}

// Signed signs a token and returns the signed token as a string
func Signed(token Token, secret string) (string, error) {

	return jwt.NewWithClaims(jwt.SigningMethodHS256, token).SignedString([]byte(secret))
}

// SetSubject sets the subject of the token
func SetSubject(token *Token, subject string) {
	token.Subject = subject
}

// HasRequiredClaims checks that there is at least one group, and at least one scope, returning true if there is at least one group and one scope
func HasRequiredClaims(token Token) bool {

	if len(token.Groups) == 0 || len(token.Scopes) == 0 {
		return false
	}
	return true
}
