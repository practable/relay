package booking

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/runtime/middleware"
	"github.com/google/uuid"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/groups"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/pools"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/pool"
)

func getPoolStatusByIDHandler(ps *pool.PoolStore) func(params pools.GetPoolStatusByIDParams, principal interface{}) middleware.Responder {
	return func(params pools.GetPoolStatusByIDParams, principal interface{}) middleware.Responder {
		token, ok := principal.(*jwt.Token)
		if !ok {
			return pools.NewGetPoolStatusByIDUnauthorized().WithPayload("Token Not JWT")
		}

		// save checking for key existence individually by checking all at once
		claims, ok := token.Claims.(*lit.Token)

		if !ok {
			return pools.NewGetPoolStatusByIDUnauthorized().WithPayload("Token Claims Incorrect Type")
		}

		if !lit.HasRequiredClaims(*claims) {
			return pools.NewGetPoolStatusByIDUnauthorized().WithPayload("Token Missing Required Claims")
		}

		hasBookingScope := false

		for _, scope := range claims.Scopes {
			if scope == "booking" {
				hasBookingScope = true
			}
		}

		if !hasBookingScope {
			return pools.NewGetPoolStatusByIDUnauthorized().WithPayload("Missing booking Scope")
		}

		// is this user allowed to access this pool? i.e. is this pool in our of our authorised groups?
		hasPool := false

		for _, pool := range claims.Pools {
			if pool != params.PoolID {
				continue
			}
			hasPool = true
			break
		}

		if !hasPool {
			return pools.NewGetPoolStatusByIDUnauthorized().WithPayload("Pool Not In Authorized Groups")
		}

		p, err := ps.GetPoolByID(params.PoolID)
		if err != nil {
			return pools.NewGetPoolStatusByIDUnauthorized().WithPayload("Pool Does Not Exist")
		}

		s := models.Status{}

		duration := uint64(300)
		if params.Duration != nil {
			duration = uint64(*params.Duration)
		}

		wait, err := p.ActivityWaitDuration(duration)
		s.Later = (err == nil) //err means no kit avail later
		s.Wait = int64(wait)
		avail := int64(p.CountAvailable())
		s.Available = &avail
		s.Used = int64(p.CountInUse())

		return pools.NewGetPoolStatusByIDOK().WithPayload(&s)
	}
}

func getPoolDescriptionByIDHandler(ps *pool.PoolStore) func(params pools.GetPoolDescriptionByIDParams, principal interface{}) middleware.Responder {
	return func(params pools.GetPoolDescriptionByIDParams, principal interface{}) middleware.Responder {
		token, ok := principal.(*jwt.Token)
		if !ok {
			return pools.NewGetPoolDescriptionByIDUnauthorized().WithPayload("Token Not JWT")
		}

		// save checking for key existence individually by checking all at once
		claims, ok := token.Claims.(*lit.Token)

		if !ok {
			return pools.NewGetPoolDescriptionByIDUnauthorized().WithPayload("Token Claims Incorrect Type")
		}

		if !lit.HasRequiredClaims(*claims) {
			return pools.NewGetPoolDescriptionByIDUnauthorized().WithPayload("Token Missing Required Claims")
		}

		hasBookingScope := false

		for _, scope := range claims.Scopes {
			if scope == "booking" {
				hasBookingScope = true
			}
		}

		if !hasBookingScope {
			return pools.NewGetPoolDescriptionByIDUnauthorized().WithPayload("Missing booking Scope")
		}

		// is this user allowed to access this pool? i.e. is this pool in our of our authorised groups?
		hasPool := false

		for _, pool := range claims.Pools {
			if pool != params.PoolID {
				continue
			}
			hasPool = true
			break
		}

		if !hasPool {
			return pools.NewGetPoolDescriptionByIDUnauthorized().WithPayload("Pool Not In Authorized Groups")
		}

		p, err := ps.GetPoolByID(params.PoolID)

		if err != nil {
			return pools.NewGetPoolDescriptionByIDUnauthorized().WithPayload("Pool Does Not Exist")
		}

		d := models.Description{}
		pd := p.Description

		d.Further = pd.Further
		d.Image = pd.Image
		d.Long = pd.Long
		d.Name = &pd.Name
		d.Short = pd.Short
		d.Thumb = pd.Thumb
		d.Type = &pd.Type
		d.ID = pd.ID

		return pools.NewGetPoolDescriptionByIDOK().WithPayload(&d)

	}
}

func getPoolsByGroupIDHandler(ps *pool.PoolStore) func(params pools.GetPoolsByGroupIDParams, principal interface{}) middleware.Responder {
	return func(params pools.GetPoolsByGroupIDParams, principal interface{}) middleware.Responder {

		token, ok := principal.(*jwt.Token)
		if !ok {
			return pools.NewGetPoolsByGroupIDUnauthorized().WithPayload("Token Not JWT")
		}

		// save checking for key existence individually by checking all at once
		claims, ok := token.Claims.(*lit.Token)

		if !ok {
			return pools.NewGetPoolsByGroupIDUnauthorized().WithPayload("Token Claims Incorrect Type")
		}

		if !lit.HasRequiredClaims(*claims) {
			return pools.NewGetPoolsByGroupIDUnauthorized().WithPayload("Token Missing Required Claims")
		}

		hasBookingScope := false

		for _, scope := range claims.Scopes {
			if scope == "booking" {
				hasBookingScope = true
			}
		}

		if !hasBookingScope {
			return pools.NewGetPoolsByGroupIDUnauthorized().WithPayload("Missing booking Scope")
		}

		gp, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return pools.NewGetPoolsByGroupIDUnauthorized().WithPayload(err.Error())
		}

		isAllowedGroup := false

		for _, name := range claims.Groups {
			if name != gp.Name {
				continue
			}
			isAllowedGroup = true
			break
		}

		if !isAllowedGroup {
			return pools.NewGetPoolsByGroupIDUnauthorized().WithPayload("Missing Group Name in Groups Claim")
		}

		ids := []string{}

		for _, p := range gp.GetPools() {
			ids = append(ids, p.ID)
		}

		return pools.NewGetPoolsByGroupIDOK().WithPayload(ids)
	}
}

func getGroupDescriptionByIDHandlerFunc(ps *pool.PoolStore) func(groups.GetGroupDescriptionByIDParams, interface{}) middleware.Responder {
	return func(params groups.GetGroupDescriptionByIDParams, principal interface{}) middleware.Responder {

		token, ok := principal.(*jwt.Token)
		if !ok {
			return groups.NewGetGroupDescriptionByIDUnauthorized().WithPayload("Token Not JWT")
		}

		// save checking for key existence individually by checking all at once
		claims, ok := token.Claims.(*lit.Token)

		if !ok {
			return groups.NewGetGroupDescriptionByIDUnauthorized().WithPayload("Token Claims Incorrect Type")
		}

		if !lit.HasRequiredClaims(*claims) {
			return groups.NewGetGroupDescriptionByIDUnauthorized().WithPayload("Token Missing Required Claims")
		}

		hasBookingScope := false

		for _, scope := range claims.Scopes {
			if scope == "booking" {
				hasBookingScope = true
			}
		}

		if !hasBookingScope {
			return groups.NewGetGroupDescriptionByIDUnauthorized().WithPayload("Missing booking Scope")
		}

		gp, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return groups.NewGetGroupIDByNameInternalServerError().WithPayload(err.Error())
		}

		isAllowedGroup := false

		for _, name := range claims.Groups {
			if name != gp.Name {
				continue
			}
			isAllowedGroup = true
			break
		}

		if !isAllowedGroup {
			return groups.NewGetGroupDescriptionByIDUnauthorized().WithPayload("Missing Group Name in Groups Claim")
		}

		g := gp.Description
		d := models.Description{}

		d.Further = g.Further
		d.Image = g.Image
		d.Long = g.Long
		d.Name = &g.Name
		d.Short = g.Short
		d.Thumb = g.Thumb
		d.Type = &g.Type
		d.ID = g.ID

		return groups.NewGetGroupDescriptionByIDOK().WithPayload(&d)
	}
}

func getGroupIDByNameHandlerFunc(ps *pool.PoolStore) func(groups.GetGroupIDByNameParams, interface{}) middleware.Responder {
	return func(params groups.GetGroupIDByNameParams, principal interface{}) middleware.Responder {

		// check group name is in the token
		token, ok := principal.(*jwt.Token)
		if !ok {
			return groups.NewGetGroupIDByNameUnauthorized().WithPayload("Token Not JWT")
		}

		// save checking for key existence individually by checking all at once
		claims, ok := token.Claims.(*lit.Token)

		if !ok {
			return groups.NewGetGroupIDByNameUnauthorized().WithPayload("Token Claims Incorrect Type")
		}

		if !lit.HasRequiredClaims(*claims) {
			return groups.NewGetGroupIDByNameUnauthorized().WithPayload("Token Missing Required Claims")
		}

		hasBookingScope := false

		for _, scope := range claims.Scopes {
			if scope == "booking" {
				hasBookingScope = true
			}
		}

		if !hasBookingScope {
			return groups.NewGetGroupIDByNameUnauthorized().WithPayload("Missing booking Scope")
		}

		isAllowedGroup := false

		for _, gp := range claims.Groups {
			if gp != params.Name {
				continue
			}
			isAllowedGroup = true
			break
		}

		if !isAllowedGroup {
			return groups.NewGetGroupIDByNameUnauthorized().WithPayload("Missing Group in Groups Claim")
		}
		gps, err := ps.GetGroupsByName(params.Name)

		if err != nil {
			return groups.NewGetGroupIDByNameInternalServerError().WithPayload(err.Error())
		}

		ids := []string{}

		for _, gp := range gps {
			ids = append(ids, gp.ID)
		}

		return groups.NewGetGroupIDByNameOK().WithPayload(ids)
	}
}

func loginHandlerFunc(ps *pool.PoolStore) func(login.LoginParams, interface{}) middleware.Responder {
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

		hasLoginScope := false

		scopes := []string{"booking"}

		for _, scope := range claims.Scopes {
			if scope == "login" {
				hasLoginScope = true
			} else {
				scopes = append(scopes, scope)
			}
		}

		if !hasLoginScope {
			return login.NewLoginUnauthorized().WithPayload("Missing login Scope")
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
