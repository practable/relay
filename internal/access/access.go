package access

import (
	"flag"
	"fmt"
	"sync"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/security"
	"github.com/golang-jwt/jwt/v4"
	"github.com/practable/relay/internal/access/models"
	"github.com/practable/relay/internal/access/restapi"
	"github.com/practable/relay/internal/access/restapi/operations"
	"github.com/practable/relay/internal/deny"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/ttlcode"
	log "github.com/sirupsen/logrus"
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
// @ds - pointer to the Deny list
// allowNoBookingID - whether to accept tokens without bookingID (set to yes to be backwards compatible)
// @options - for future backwards compatibility (no options currently available)
// TODO put these options in a config struct
func API(closed <-chan struct{}, wg *sync.WaitGroup, port int, host, secret, target string, cs *ttlcode.CodeStore, ds *deny.Store, allowNoBookingID bool) {

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

			if claims.BookingID == "" && !allowNoBookingID { //if bookingID is empty, and this is not allowed
				c := "400"
				m := "empty bookingID field is not permitted"
				return operations.NewSessionBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
			}

			if ds.IsDenied(claims.BookingID) {
				c := "400"
				m := "bookingID has been deny-listed, probably because the session was cancelled"
				return operations.NewSessionBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
			} else {
				// track bookingIDs for which we have received connection requests
				ds.Allow(claims.BookingID, claims.ExpiresAt.Unix())
			}

			// TODO - have the scopes been checked already?

			pt := permission.NewToken(
				target,
				claims.ConnectionType,
				params.SessionID,
				claims.Scopes,
				claims.IssuedAt.Unix(),
				claims.NotBefore.Unix(),
				claims.ExpiresAt.Unix(),
			)

			pt.SetBookingID(claims.BookingID)

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

		if !token.Valid { //checks iat, nbf, exp
			log.Info("Token invalid")
			return nil, fmt.Errorf("token invalid")
		}

		if cc, ok := token.Claims.(*permission.Token); ok {

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
