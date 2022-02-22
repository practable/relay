package booking

import (
	"errors"
	"fmt"

	"github.com/go-openapi/runtime/security"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
	lit "github.com/timdrysdale/relay/pkg/login"
)

func claimsCheck(principal interface{}) (*lit.Token, error) {

	token, ok := principal.(*jwt.Token)
	if !ok {
		return nil, errors.New("Token Not JWT")
	}

	// save checking for key existence individually by checking all at once
	claims, ok := token.Claims.(*lit.Token)

	if !ok {
		return nil, errors.New("Token Claims Incorrect Type")
	}

	if !lit.HasRequiredClaims(*claims) {
		return nil, errors.New("Token Missing Required Claims")
	}

	return claims, nil
}

// Function isBookingAdmin does in-handler validation for booking:admin tasks
func isBookingAdmin(principal interface{}) (*lit.Token, error) {

	claims, err := claimsCheck(principal)

	if err != nil {
		return nil, err
	}

	hasAdminScope := false

	for _, scope := range claims.Scopes {
		if scope == "booking:admin" {
			hasAdminScope = true
		}
	}

	if !hasAdminScope {
		return nil, errors.New("Missing booking:admin Scope")
	}

	return claims, nil
}

func isBookingUser(principal interface{}) (*lit.Token, error) {

	claims, err := claimsCheck(principal)

	if err != nil {
		return nil, err
	}

	hasUserScope := false

	for _, scope := range claims.Scopes {
		if scope == "booking:user" {
			hasUserScope = true
		}
	}

	if !hasUserScope {
		return nil, errors.New("Missing booking:user Scope")
	}

	return claims, nil
}

func isBookingAdminOrUser(principal interface{}) (bool, *lit.Token, error) {

	claims, err := claimsCheck(principal)

	if err != nil {
		return false, nil, err
	}

	hasAdminScope := false
	hasUserScope := false

	for _, scope := range claims.Scopes {
		if scope == "booking:admin" {
			hasAdminScope = true
		}
		if scope == "booking:user" {
			hasUserScope = true
		}
	}

	if !hasAdminScope && !hasUserScope {
		return false, nil, errors.New("Missing booking:admin or booking:user Scope")
	}

	return hasAdminScope, claims, nil
}

// ValidateHeader checks the bearer token.
// wrap the secret so we can get it at runtime without using global
func validateHeader(secret, host string) security.TokenAuthentication {

	return func(bearerToken string) (interface{}, error) {
		// For apiKey security syntax see https://swagger.io/docs/specification/2-0/authentication/

		claims := &lit.Token{}

		token, err := jwt.ParseWithClaims(bearerToken, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method was %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if !token.Valid { //checks iat, nbf, exp
			log.Info("Token invalid")
			return nil, fmt.Errorf("token invalid")
		}

		if cc, ok := token.Claims.(*lit.Token); ok {

			if !cc.RegisteredClaims.VerifyAudience(host, true) {
				log.WithFields(log.Fields{"aud": cc.RegisteredClaims.Audience, "host": host}).Info("aud does not match this host")
				return nil, fmt.Errorf("aud %s does not match this host %s", cc.RegisteredClaims.Audience, host)
			}

		} else {
			log.WithFields(log.Fields{"token": bearerToken, "host": host}).Info("Error parsing token")
			return nil, err
		}

		return token, nil
	}
}
