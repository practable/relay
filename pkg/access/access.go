package access

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/security"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/access/restapi"
	"github.com/timdrysdale/relay/pkg/access/restapi/operations"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

// API starts the API
// Inputs
// @closed - channel will be closed when server shutsdown
// @wg - waitgroup, we must wg.Done() when we are shutdown
// @port - where to listen locally
// @host - external FQDN of the host (for checking against tokens) e.g. https://relay-access.practable.io
// @target - FQDN of the relay instance e.g. wss://relay.practable.io
// @secret- HMAC shared secret which incoming tokens will be signed with
// @cs - pointer to the CodeStore this API shares with the crossbar websocket relay
// @options - for future backwards compatibility (no options currently available)
func API(closed <-chan struct{}, wg *sync.WaitGroup, port int, host, secret, target string, cs *ttlcode.CodeStore) {

	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	//create new service API
	api := operations.NewAccessAPI(swaggerSpec)
	server := restapi.NewServer(api)

	//parse flags
	flag.Parse()

	// set the port this service will run on
	server.Port = port

	// set the Authorizer
	api.BearerAuth = validateHeader(secret, host)

	// set the Handler
	api.SessionHandler = operations.SessionHandlerFunc(
		func(params operations.SessionParams, principal interface{}) middleware.Responder {

			token, ok := principal.(*jwt.Token)
			if !ok {
				return operations.NewSessionUnauthorized().WithPayload("Token Not JWT")
			}

			// save checking for key existence individually by checking all at once
			claims, ok := token.Claims.(*permission.Token)

			if !ok {
				return operations.NewSessionUnauthorized().WithPayload("Token Claims Incorrect Type")
			}

			if !permission.HasRequiredClaims(*claims) {
				return operations.NewSessionUnauthorized().WithPayload("Token Missing Required Claims")
			}

			if params.SessionID == "" {
				return operations.NewSessionUnauthorized().WithPayload("Path Missing SessionID")
			}

			if claims.Topic != params.SessionID {
				log.WithFields(log.Fields{"topic": claims.Topic, "session_id": params.SessionID}).Debug("topic does not match sessionID")
				return operations.NewSessionUnauthorized().WithPayload("Token Wrong Topic")
			}

			// TODO - have the scopes been checked already?

			pt := permission.NewToken(
				target,
				claims.ConnectionType,
				params.SessionID,
				claims.Scopes,
				claims.IssuedAt,
				claims.NotBefore,
				claims.ExpiresAt,
			)

			code := cs.SubmitToken(pt)

			log.Trace(fmt.Sprintf("submitting token of type %T", pt))

			uri := target + "/" + claims.ConnectionType + "/" + claims.Topic + "?code=" + code

			return operations.NewSessionOK().WithPayload(
				&operations.SessionOKBody{
					URI: uri,
				})
		})

	go func() {
		<-closed
		err := server.Shutdown()
		if err != nil {
			log.Errorf("Server shudown error %s", err.Error())
		}
	}()

	//serve API
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}

	wg.Done()

}

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
			log.WithFields(log.Fields{"error": err}).Info(err.Error())
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
