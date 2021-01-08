package booking

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/runtime/middleware"
	"github.com/google/uuid"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/pool"
)

func loginHandler(ps *pool.PoolStore) func(login.LoginParams, interface{}) middleware.Responder {
	return func(params login.LoginParams, principal interface{}) middleware.Responder {

		token, ok := principal.(*jwt.Token)
		if !ok {
			return login.NewLoginUnauthorized().WithPayload("Token Not JWT")
		}

		// save checking for key existence individually by checking all at once
		claims, ok := token.Claims.(*lit.Token)

		if !ok {
			return login.NewLoginUnauthorized().WithPayload("Token Claims Incorrect Type")
		}

		if !lit.HasRequiredClaims(*claims) {
			return login.NewLoginUnauthorized().WithPayload("Token Missing Required Claims")
		}

		hasLoginUserScope := false
		hasLoginAdminScope := false

		scopes := []string{}

		for _, scope := range claims.Scopes {
			if scope == "login:user" {
				hasLoginUserScope = true
			} else if scope == "login:admin" {
				hasLoginAdminScope = true
			} else {
				scopes = append(scopes, scope)
			}
		}

		if !(hasLoginUserScope || hasLoginAdminScope) {
			return login.NewLoginUnauthorized().WithPayload("Missing login:user or login:admin scope")
		}

		if hasLoginAdminScope {
			scopes = append(scopes, "booking:admin")
		}
		if hasLoginUserScope {
			scopes = append(scopes, "booking:user")
		}

		// make a new uuid for the user so we can manage their booked sessions
		subject := uuid.New().String()

		// keep uuid from previous booking token if we received it in the body of the request
		// code in the login pages needs to look for this in cache and add to body if found
		if params.Expired.Token != "" {

			// decode token
			ebt, err := jwt.ParseWithClaims(params.Expired.Token,
				&lit.Token{},
				func(token *jwt.Token) (interface{}, error) {
					return []byte(ps.Secret), nil
				})
			if err == nil {
				claims, ok = ebt.Claims.(*lit.Token)
				if ok {
					if claims.Subject != "" {
						subject = claims.Subject //if subject is usable, use it
					}
				}
			}
		}

		bookingClaims := claims
		//keep groups and any other fields added
		bookingClaims.Scopes = scopes //update scopes

		bookingClaims.IssuedAt = ps.GetTime() - 1
		bookingClaims.NotBefore = ps.GetTime() - 1
		bookingClaims.ExpiresAt = bookingClaims.NotBefore + ps.BookingTokenDuration
		bookingClaims.Subject = subject

		// Get a list of all the pool_ids that can be booked

		pids := bookingClaims.Pools

		for _, group_name := range bookingClaims.Groups {

			gps, err := ps.GetGroupsByName(group_name)
			if err != nil {
				// don't throw error in case other groups are valid
				continue
			}
			for _, g := range gps {
				pls := g.GetPools()
				for _, p := range pls {
					pids = append(pids, p.ID)
				}
			}

		}

		bookingClaims.Pools = pids

		// sign user token
		// Create a new token object, specifying signing method and the claims
		// you would like it to contain.

		bookingToken := jwt.NewWithClaims(jwt.SigningMethodHS256, bookingClaims)

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := bookingToken.SignedString(ps.Secret)

		if err != nil {
			return login.NewLoginInternalServerError().WithPayload("Could Not Generate Booking Token")
		}

		exp := float64(bookingClaims.ExpiresAt)
		iat := float64(bookingClaims.ExpiresAt)
		nbf := float64(bookingClaims.ExpiresAt)

		return login.NewLoginOK().WithPayload(
			&models.Bookingtoken{
				Aud:    &bookingClaims.Audience,
				Exp:    &exp,
				Groups: bookingClaims.Groups,
				Iat:    iat,
				Nbf:    &nbf,
				Scopes: bookingClaims.Scopes,
				Sub:    &bookingClaims.Subject,
				Token:  &tokenString,
				Pools:  bookingClaims.Pools,
			})
	}
}
