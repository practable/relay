package booking

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/runtime/middleware"
	"github.com/google/uuid"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/pool"
)

//func loginHandlerFunc(ps *pool.PoolStore) func(login.LoginParams, interface{}) middleware.Responder {
//	return func(params login.LoginParams, principal interface{}) middleware.Responder {
//
//		return middleware.NotImplemented("operation login.Login has not yet been implemented")
//
//	}
//}

func loginHandlerFunc(ps *pool.PoolStore) func(login.LoginParams, interface{}) middleware.Responder {
	return func(params login.LoginParams, principal interface{}) middleware.Responder {

		token, ok := principal.(*jwt.Token)
		if !ok {
			return operations.NewLoginnauthorized().WithPayload("Token Not JWT")
		}

		// save checking for key existence individually by checking all at once
		claims, ok := token.Claims.(*login.Token)

		if !ok {
			return operations.NewLoginUnauthorized().WithPayload("Token Claims Incorrect Type")
		}

		if !permission.HasRequiredClaims(*claims) {
			return operations.NewLoginUnauthorized().WithPayload("Token Missing Required Claims")
		}

		if params.ShellID == "" {
			return operations.NewShellUnauthorized().WithPayload("Path Missing ShellID")
		}

		if claims.Topic != params.ShellID {
			return operations.NewShellUnauthorized().WithPayload("ShellID Does Not Match Token")
		}

		// Now we check the scopes ....
		// If "host" is present, then we connect to the base session
		// If "client" is present, then we connect to a unique sub-session
		//  Scopes are modified to be read, write
		// If both scopes are offered, then the behaviour depends on the routing
		// default to treating as a host
		// unless a ConnectionID present in query e.g.
		// &connection_id=134234234324
		// in which case, distinguishing between host and client is irrelevant

		hasClientScope := false
		hasHostScope := false
		hasStatsScope := false

		for _, scope := range claims.Scopes {
			if scope == "host" {
				hasHostScope = true
			}
			if scope == "client" {
				hasClientScope = true
			}
			if scope == "stats" {
				hasStatsScope = true
			}
		}

		if hasStatsScope && params.ShellID != "stats" {
			return operations.NewShellUnauthorized().WithPayload("Path Not Valid for Stats Scope")
		}

		if !hasStatsScope && params.ShellID == "stats" {
			return operations.NewShellUnauthorized().WithPayload("Path Not Valid Without Stats Scope")
		}

		if !(hasClientScope || hasHostScope || hasStatsScope) {
			return operations.NewShellUnauthorized().WithPayload("Missing client, host or stats Scope")
		}

		if hasClientScope && hasHostScope {
			return operations.NewShellUnauthorized().WithPayload("Can only have Client Or Host Scope, Not Both")
		}

		topic := claims.Topic
		topicSalt := ""
		alertHost := false

		if hasClientScope { //need a new unique connection
			topic = topic + "/" + uuid.New().String()
			topicSalt = uuid.New().String()
			alertHost = true
		}

		// Shellbar will take care of alerting the admin channel of
		// the new connection for protocol timing reasons
		// Because ssh is "server speaks first", we want to bridge
		// to the server only when client already in place and
		// listening. There are no further hits on the access endpoint
		// though - the rest is done via websockets
		// hence no handler is needed for https://{access-host}/shell/{shell_id}/{connection_id}

		pt := permission.NewToken(
			target,
			claims.ConnectionType,
			topic,
			[]string{"read", "write"}, // sanitise out of abundance of caution - all use cases are read+write only
			claims.IssuedAt,
			claims.NotBefore,
			claims.ExpiresAt,
		)

		permission.SetTopicSalt(&pt, topicSalt)
		permission.SetAlertHost(&pt, alertHost)

		code := cs.SubmitToken(pt)

		uri := target + "/" + claims.ConnectionType + "/" + topic + "?code=" + code

		return operations.NewLoginOK().WithPayload(
			&operations.LoginOKBody{
				Token: userToken,
			})
	}
}
