package access

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
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

type Config struct {
	AllowNoBookingID bool
	CodeStore        *ttlcode.CodeStore
	DenyChannel      chan string
	DenyStore        *deny.Store
	Host             string
	Port             int
	Secret           string
	Target           string
}

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
func API(closed <-chan struct{}, wg *sync.WaitGroup, config Config) {

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
	server.Port = config.Port

	// set the Authorizer
	api.BearerAuth = validateHeader(config.Secret, config.Host)

	// set the Handler
	api.SessionHandler = operations.SessionHandlerFunc(sessionHandler(config))
	api.AllowHandler = operations.AllowHandlerFunc(allowHandler(config))
	api.DenyHandler = operations.DenyHandlerFunc(denyHandler(config))

	api.ListDeniedHandler = operations.ListDeniedHandlerFunc(listDeniedHandler(config))
	api.ListAllowedHandler = operations.ListAllowedHandlerFunc(listAllowedHandler(config))

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
			msg := "error parsing token " + err.Error()
			log.Info(msg)
			return nil, errors.New(msg)
		}

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

func sessionHandler(config Config) func(operations.SessionParams, interface{}) middleware.Responder {
	return func(params operations.SessionParams, principal interface{}) middleware.Responder {

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

		if claims.BookingID == "" && !config.AllowNoBookingID { //if bookingID is empty, and this is not allowed
			c := "400"
			m := "empty bookingID field is not permitted"
			return operations.NewSessionBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		if config.DenyStore.IsDenied(claims.BookingID) {
			c := "400"
			m := "bookingID has been deny-listed, probably because the session was cancelled"
			return operations.NewSessionBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
		} else {
			// track bookingIDs for which we have received connection requests
			config.DenyStore.Allow(claims.BookingID, claims.ExpiresAt.Unix())
		}

		// TODO - have the scopes been checked already?

		pt := permission.NewToken(
			config.Target,
			claims.ConnectionType,
			params.SessionID,
			claims.Scopes,
			claims.IssuedAt.Unix(),
			claims.NotBefore.Unix(),
			claims.ExpiresAt.Unix(),
		)

		pt.SetBookingID(claims.BookingID)

		code := config.CodeStore.SubmitToken(pt)

		log.Trace(fmt.Sprintf("submitting token of type %T", pt))

		uri := config.Target + "/" + claims.ConnectionType + "/" + claims.Topic + "?code=" + code

		return operations.NewSessionOK().WithPayload(
			&operations.SessionOKBody{
				URI: uri,
			})
	}
}

func denyHandler(config Config) func(operations.DenyParams, interface{}) middleware.Responder {
	return func(params operations.DenyParams, principal interface{}) middleware.Responder {

		// check token for whether admin or not (see booking server code)
		// the deny listing has to be done by admin, typically a booking system
		// else anyone could spam deny requests

		_, err := isRelayAdmin(principal)

		if err != nil {
			c := "401"
			m := "is token missing relay:admin scope? " + err.Error()
			return operations.NewDenyUnauthorized().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		if params.Bid == "" {
			c := "400"
			m := "bid (booking id) missing"
			return operations.NewDenyBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		if params.Exp < config.DenyStore.Now() {
			c := "400"
			m := "booking expiry time (exp) of [" + strconv.Itoa(int(params.Exp)) + "] missing or in the past"
			return operations.NewDenyBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		config.DenyStore.Deny(params.Bid, params.Exp)

		config.DenyChannel <- params.Bid // alert crossbar we need to cancel some connections

		return operations.NewDenyNoContent()
	}
}

// allowHandler undo a previous deny operation (don't fail if there was no denial operation as it does
// not affect security to add a booking id to the allow list - e.g. a subsequent deny operation will reverse it,
// and it does not prevent a token from having to be valid)
func allowHandler(config Config) func(operations.AllowParams, interface{}) middleware.Responder {
	return func(params operations.AllowParams, principal interface{}) middleware.Responder {

		// check token for whether admin or not (see booking server code)
		// the allow listing has to be done by admin, typically a booking system
		// else anyone could spam allow requests

		_, err := isRelayAdmin(principal)

		if err != nil {
			c := "401"
			m := "token missing relay:admin scope"
			return operations.NewAllowUnauthorized().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		if params.Bid == "" {
			c := "400"
			m := "bid (booking id) missing"
			return operations.NewAllowBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		if params.Exp < config.DenyStore.Now() {
			c := "400"
			m := "exp (booking expiry time) missing or in the past"
			return operations.NewAllowBadRequest().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		config.DenyStore.Allow(params.Bid, params.Exp)

		return operations.NewAllowNoContent()
	}
}

func listDeniedHandler(config Config) func(operations.ListDeniedParams, interface{}) middleware.Responder {
	return func(params operations.ListDeniedParams, principal interface{}) middleware.Responder {

		// check token for whether admin or not (see booking server code)
		// the deny listing has to be done by admin, typically a booking system
		// else anyone could spam deny requests

		_, err := isRelayAdmin(principal)

		if err != nil {
			c := "401"
			m := "token missing relay:admin scope"
			return operations.NewDenyUnauthorized().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		d := config.DenyStore.GetDenyList()

		return operations.NewListDeniedOK().WithPayload(&models.BookingIDs{BookingIds: d})
	}
}

func listAllowedHandler(config Config) func(operations.ListAllowedParams, interface{}) middleware.Responder {
	return func(params operations.ListAllowedParams, principal interface{}) middleware.Responder {

		// check token for whether admin or not (see booking server code)
		// the deny listing has to be done by admin, typically a booking system
		// else anyone could spam deny requests

		_, err := isRelayAdmin(principal)

		if err != nil {
			c := "401"
			m := "token missing relay:admin scope"
			return operations.NewDenyUnauthorized().WithPayload(&models.Error{Code: &c, Message: &m})
		}

		d := config.DenyStore.GetAllowList()

		return operations.NewListAllowedOK().WithPayload(&models.BookingIDs{BookingIds: d})
	}
}

// Function isBookingAdmin does in-handler validation for booking:admin tasks
func isRelayAdmin(principal interface{}) (*permission.Token, error) {

	claims, err := claimsCheck(principal)

	if err != nil {
		return nil, err
	}

	hasAdminScope := false

	for _, scope := range claims.Scopes {
		if scope == "relay:admin" {
			hasAdminScope = true
		}
	}

	if !hasAdminScope {
		return nil, errors.New("Missing relay:admin Scope")
	}

	return claims, nil
}

func claimsCheck(principal interface{}) (*permission.Token, error) {

	token, ok := principal.(*jwt.Token)
	if !ok {
		return nil, errors.New("Token Not JWT")
	}

	// save checking for key existence individually by checking all at once
	claims, ok := token.Claims.(*permission.Token)

	if !ok {
		return nil, errors.New("Token Claims Incorrect Type")
	}

	if len(claims.Scopes) == 0 ||
		len(claims.RegisteredClaims.Audience) == 0 ||
		(*claims.RegisteredClaims.ExpiresAt).IsZero() {
		return nil, errors.New("Token Missing Required Claims")
	}

	return claims, nil
}
