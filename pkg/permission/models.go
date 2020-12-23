package permission

import (
	"errors"
	"reflect"

	"github.com/dgrijalva/jwt-go"
)

// Permission represents claims required in the apiKey JWT
type Token struct {
	// Host must match the incoming request's intended host
	Host string `json:"host"`

	// Topic represents the communication channel;
	// either /session/{session_id} or /shell/{session_id}.
	Topic string `json:"topic"`

	// Scopes controlling access to relay;
	// either ["read"],["write"], or ["read","write"] for session, or ["host"]/["client"] for shell
	Scopes []string `json:"scopes"`

	jwt.StandardClaims
}

func NewToken(host, topic string, scopes []string, iat, exp, nbf int64) *Token {
	return &Token{
		Host:   host,
		Topic:  topic,
		Scopes: scopes,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  iat,
			NotBefore: nbf,
			ExpiresAt: exp,
		},
	}
}

func ConvertToJWT(token *Token) *jwt.Token {
	/*
		mc := jwt.MapClaims{}

		v := reflect.ValueOf(token)

		for i := 0; i < v.Type().NumField(); i++ {

			mc[v.Type().Field(i).Tag.Get("json")] = v.Field(i).Interface()
		}
	*/
	return jwt.NewWithClaims(jwt.SigningMethodHS256, token)
}

func ValidPermissionToken(token *jwt.Token) bool {

	actualType := reflect.ValueOf(token.Claims).Type()
	expectedType := reflect.ValueOf(&Token{}).Type()

	if actualType != expectedType {
		return false
	}

	return true

}

func GetPermissionToken(token *jwt.Token) (*Token, error) {
	permission, ok := token.Claims.(*Token)
	if !ok {
		return nil, errors.New("Not a permission token")
	}
	return permission, nil
}
