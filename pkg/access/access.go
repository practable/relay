package access

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/security"
	"github.com/go-openapi/swag"
	"github.com/timdrysdale/crossbar/pkg/access/restapi"
	"github.com/timdrysdale/crossbar/pkg/access/restapi/operations"
)

// if adding omit_empty or other decorators, then improve reflection code as per
// https://stackoverflow.com/questions/40864840/how-to-get-the-json-field-names-of-a-struct-in-golang

// NewPermission creates a new Permission object.
func NewPermission() *Permission {
	return &Permission{}
}

// CheckClaims makes sure all required claims are present
func checkClaims(claims jwt.MapClaims) (*Permission, error) {

	p := NewPermission()

	v := reflect.ValueOf(*p)
	ty := v.Type()

	for i := 0; i < v.NumField(); i++ {

		k := ty.Field(i).Tag.Get("json")

		if v, ok := claims[k]; ok {
			fmt.Println(k, v)
		} else {
			return nil, fmt.Errorf("missing claim %s", k)
		}
	}

	return p, nil
}

// ValidateHeader checks the bearer token.
// wrap the secret so we can get it at runtime without using global
func validateHeaderSecret(secret string) security.TokenAuthentication {

	return func(bearerHeader string) (interface{}, error) {
		// For apiKey security syntax see https://swagger.io/docs/specification/2-0/authentication/
		bearerToken := strings.Split(bearerHeader, " ")[1]
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(bearerToken, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("error decoding token")
			}
			return []byte(secret), nil
		})
		if err != nil {
			// TODO - send correct error code, 401 / 403 rather than 500
			return nil, err
		}
		if !token.Valid {
			return nil, errors.New("invalid token")
		}
		return checkClaims(claims)
	}
}

type Options struct {
	DisableAuth bool
}

func DefaultOptions() *Options {
	return &Options{}
}

func API(closed <-chan struct{}, wg *sync.WaitGroup, port int, host, secret string, options Options) {

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
	api.BearerAuth = validateHeaderSecret(secret)

	// set the Handler
	//
	api.SessionHandler = operations.SessionHandlerFunc(
		func(params operations.SessionParams, principal interface{}) middleware.Responder {
			fmt.Println(pretty(params))
			name := swag.StringValue(&params.SessionID)
			if name == "" {
				name = "World"
			}

			greeting := fmt.Sprintf("Hello, %s!", name)
			return operations.NewSessionOK().WithPayload(greeting + pretty(principal))
		})

	go func() {
		<-closed
		server.Shutdown()
	}()

	//serve API
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}

	wg.Done()

}

func pretty(t interface{}) string {

	json, err := json.MarshalIndent(t, "", "\t")
	if err != nil {
		return ""
	}

	return string(json)
}

// checkAuth checks the claims are ok
//
// the route must match the audience
// the lifetime is the number of seconds for which the token is valid

func checkAuth(data []byte, secret string, route string, now int64) (int64, error) {

	tokenError := errors.New("Token invalid for this resource")
	var lifetime int64 = 0

	token, err := jwt.Parse(strings.TrimSpace(string(data)), func(token *jwt.Token) (interface{}, error) {
		// verify alg is expected
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// global config option
		return []byte(secret), nil
	})

	if err != nil {
		msg := fmt.Sprintf("Error reading token %s", err.Error())
		log.WithFields(log.Fields{"error": err}).Warn(msg)
		tokenError = errors.New(msg)

	} else {

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

			if claims["aud"] == route {
				tokenError = nil
				log.WithFields(log.Fields{"route": route}).Info("Authorised - client can communicate")
			} else {
				msg := fmt.Sprintf("Denied - not permitted to access %s with token for %s", route, claims["aud"])
				log.WithFields(log.Fields{"wanted": claims["aud"], "actual": route}).Warn(msg)
				tokenError = errors.New(msg)
			}

			if val, ok := claims["exp"]; ok {

				if exp, ok := val.(float64); ok {
					lifetime = int64(exp) - time.Now().Unix()
					log.WithFields(log.Fields{"lifetime": lifetime, "exp": claims["exp"]}).Info("Lifetime")
				} else {
					msg := "Couldn't calculate lifetime, leaving as zero lifetime connection"
					log.WithFields(log.Fields{"exp": claims["exp"]}).Info(msg)
					tokenError = errors.New(msg)
				}
			}

		} else {
			msg := "Error checking claims in JWT"
			log.WithFields(log.Fields{"err": err}).Error(msg)
			tokenError = errors.New(msg)
		}

	}

	return lifetime, tokenError
}
