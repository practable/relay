package permission

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
)

// Permission represents claims required in the apiKey JWT
type Token struct {

	// Topic represents the communication channel;
	// either /session/{session_id} or /shell/{session_id}.
	Topic string `json:"topic"`

	ConnectionType string `json:"prefix"`

	// Scopes controlling access to relay;
	// either ["read"],["write"], or ["read","write"] for session, or ["host"]/["client"] for shell
	Scopes []string `json:"scopes"`

	jwt.StandardClaims
}

func NewToken(audience, connectionType, topic string, scopes []string, iat, nbf, exp int64) Token {
	return Token{
		Topic:          topic,
		Scopes:         scopes,
		ConnectionType: connectionType,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  iat,
			NotBefore: nbf,
			ExpiresAt: exp,
			Audience:  audience,
		},
	}
}

func ConvertToJWT(token Token) *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, token)
}

func ValidPermissionToken(token *jwt.Token) bool {

	actualType := reflect.ValueOf(token.Claims).Type()
	expectedType := reflect.ValueOf(Token{}).Type()

	log.Trace(fmt.Sprintf("actualType: %v\nexptecType: %v\n", actualType, expectedType))

	if actualType != expectedType {
		return false
	}

	return true

}

func GetPermissionToken(token *jwt.Token) (Token, error) {
	if !ValidPermissionToken(token) {
		return Token{}, errors.New("Not a permission token")
	}
	return token.Claims.(Token), nil
}
