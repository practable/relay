package login

import (
	"github.com/dgrijalva/jwt-go"
)

// Permission represents claims required in the apiKey JWT
type Token struct {

	// Groups represent the sets of pools that a user can access
	Groups []string `json:"groups"`

	// Scopes controlling access to relay;
	// either ["login"],["user"], or ["login","admin"]
	Scopes []string `json:"scopes"`

	jwt.StandardClaims
}

func NewToken(audience string, groups, scopes []string, iat, nbf, exp int64) Token {
	return Token{
		Groups: groups,
		Scopes: scopes,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  iat,
			NotBefore: nbf,
			ExpiresAt: exp,
			Audience:  audience,
		},
	}
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
