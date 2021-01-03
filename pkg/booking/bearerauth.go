package booking

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/runtime/security"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/permission"
)

// ValidateHeader checks the bearer token.
// wrap the secret so we can get it at runtime without using global
func validateHeader(secret, host string) security.TokenAuthentication {

	return func(bearerToken string) (interface{}, error) {
		// For apiKey security syntax see https://swagger.io/docs/specification/2-0/authentication/
		claims := &permission.Token{}

		token, err := jwt.ParseWithClaims(bearerToken, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method was %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil {
			log.WithFields(log.Fields{"error": err, "token": bearerToken}).Info(err.Error())
			return nil, fmt.Errorf("error reading token was %s", err.Error())
		}

		if !token.Valid { //checks iat, nbf, exp
			log.Info("Token invalid")
			return nil, fmt.Errorf("token invalid")
		}

		if claims.Audience != host {

			log.WithFields(log.Fields{"aud": claims.Audience, "host": host}).Info("aud does not match this host")
			return nil, fmt.Errorf("aud %s does not match this host %s", claims.Audience, host)
		}

		// already checked but belt and braces ....
		if claims.ExpiresAt <= time.Now().Unix() {
			log.Info(fmt.Sprintf("Expired at %d", claims.ExpiresAt))
			return nil, fmt.Errorf("expired at %d", claims.ExpiresAt)
		}

		return token, nil
	}
}
