package login

import (
	"encoding/json"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

// Permission represents claims required in the apiKey JWT
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

type TokenInBody struct {
	Token string `json:"token"`
}

func (t *Token) String() string {

	pretty, err := json.MarshalIndent(*t, "", "\t")

	if err != nil {
		return fmt.Sprintf("%+v", *t)
	}

	return string(pretty)
}

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

func Signed(token Token, secret string) (string, error) {

	return jwt.NewWithClaims(jwt.SigningMethodHS256, token).SignedString([]byte(secret))
}

func SetSubject(token *Token, subject string) {
	token.Subject = subject
}

func HasRequiredClaims(token Token) bool {

	if len(token.Groups) == 0 || len(token.Scopes) == 0 {
		return false
	}
	return true
}
