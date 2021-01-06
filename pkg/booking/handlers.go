package booking

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/runtime/middleware"
	"github.com/google/uuid"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/groups"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/pools"
	"github.com/timdrysdale/relay/pkg/limit"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/pool"
)

func addActivityByPoolIDHandler(ps *pool.PoolStore) func(params pools.AddActivityByPoolIDParams, principal interface{}) middleware.Responder {
	return func(params pools.AddActivityByPoolIDParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return pools.NewAddActivityByPoolIDUnauthorized().WithPayload(err.Error())
		}

		aid := "not implemented yet"
		mid := &models.ID{
			ID: &aid,
		}
		return pools.NewAddActivityByPoolIDOK().WithPayload(mid)
	}
}

func addNewPoolHandler(ps *pool.PoolStore) func(params pools.AddNewPoolParams, principal interface{}) middleware.Responder {
	return func(params pools.AddNewPoolParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload(err.Error())
		}

		pd := params.Pool.Description
		name := *pd.Name

		if name == "" {
			return pools.NewAddNewPoolNotFound().WithPayload("Pool Missing Name")
		}

		if pd.ID != "" {
			return pools.NewAddNewPoolNotFound().WithPayload("Do Not Specify ID - Will Be Assigned")
		}

		d := pool.NewDescription(name)
		d.SetFurther(pd.Further)
		d.SetID(uuid.New().String())
		d.SetImage(pd.Image)
		d.SetLong(pd.Long)
		d.SetShort(pd.Short)
		d.SetThumb(pd.Thumb)
		d.SetType(*pd.Type)

		p := pool.NewPool(name).WithDescription(*d)

		if params.Pool.MinSession != 0 {
			p.SetMinSesssion(uint64(params.Pool.MinSession))
		}

		if params.Pool.MinSession != 0 {
			p.SetMaxSesssion(uint64(params.Pool.MaxSession))
		}

		ps.AddPool(p)

		id := p.GetID()
		mid := &models.ID{
			ID: &id,
		}

		return pools.NewAddNewPoolOK().WithPayload(mid)
	}
}

func requestSessionByPoolIDHandler(ps *pool.PoolStore, l *limit.Limit) func(params pools.RequestSessionByPoolIDParams, principal interface{}) middleware.Responder {

	return func(params pools.RequestSessionByPoolIDParams, principal interface{}) middleware.Responder {

		claims, err := isBookingUser(principal)

		if err != nil {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload(err.Error())
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
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload("Pool Not In Authorized Groups")
		}

		p, err := ps.GetPoolByID(params.PoolID)
		if err != nil {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload("Pool Does Not Exist")
		}

		duration := uint64(params.Duration)

		if duration < p.GetMinSession() {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload("Requested duration too short")
		}

		if duration > p.GetMaxSession() {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload("Requested duration too long")
		}

		// check for user name
		if claims.Subject == "" {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload("No subject in booking token")
		}

		exp := ps.Now() + int64(duration)
		// Check we have concurrent booking quota left over, by making a provisional booking
		confirm, err := l.ProvisionalRequest(claims.Subject, exp)

		if err != nil {
			return pools.NewRequestSessionByPoolIDPaymentRequired().WithPayload("Maximum conconcurrent sessions already reached. Try again later.")
		}

		aid, err := p.ActivityRequestAny(duration)

		if err != nil {
			return pools.NewRequestSessionByPoolIDNotFound().WithPayload(err.Error())
		}

		a, err := p.GetActivityByID(aid)

		if err != nil {
			return pools.NewRequestSessionByPoolIDInternalServerError().WithPayload(err.Error())
		}

		// Make session tokens for each stream
		// If permission token information is wrong
		// let that be determined by the recipient
		// we cannot really know

		streams := []*models.Stream{}

		iat := ps.Now() - 1
		nbf := ps.Now() - 1

		for _, s := range a.Streams {

			// copy token
			claims := s.Permission
			// update timing
			claims.IssuedAt = iat
			claims.NotBefore = nbf
			claims.ExpiresAt = exp

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			bearer, err := token.SignedString(ps.Secret)
			if err != nil {
				return pools.NewRequestSessionByPoolIDInternalServerError().WithPayload(err.Error())
			}

			// populate API stream type
			stream := models.Stream{}
			stream.Exp = float64(claims.ExpiresAt)
			stream.For = &s.For
			stream.Iat = float64(claims.IssuedAt)
			stream.Nbf = float64(claims.NotBefore)
			stream.Token = &bearer
			stream.URL = &s.URL
			stream.Verb = &s.Verb

			streams = append(streams, &stream)
		}

		uis := []*models.UserInterface{}

		for _, u := range a.UI {
			ui := &models.UserInterface{}
			ui.URL = &u.URL
			ui.StreamsRequired = u.StreamsRequired
			ui.Description = &models.Description{}
			ui.Description.Further = u.Description.Further
			ui.Description.Image = u.Description.Image
			ui.Description.Long = u.Description.Long
			ui.Description.Name = &u.Description.Name
			ui.Description.Short = u.Description.Short
			ui.Description.Thumb = u.Description.Thumb
			ui.Description.Type = &u.Description.Type
			ui.Description.ID = u.Description.ID

			uis = append(uis, ui)
		}

		fnbf := float64(nbf)
		fexp := float64(exp)

		ma := models.Activity{}
		ma.Iat = float64(iat)
		ma.Nbf = &fnbf
		ma.Exp = &fexp
		ma.Streams = streams
		ma.Uis = uis
		ma.Description = &models.Description{}
		ma.Description.Further = a.Description.Further
		ma.Description.Image = a.Description.Image
		ma.Description.Long = a.Description.Long
		ma.Description.Name = &a.Description.Name
		ma.Description.Short = a.Description.Short
		ma.Description.Thumb = a.Description.Thumb
		ma.Description.Type = &a.Description.Type
		ma.Description.ID = a.Description.ID

		confirm() // confirm booking with Limit checker
		return pools.NewRequestSessionByPoolIDOK().WithPayload(&ma)
	}
}

func getPoolStatusByIDHandler(ps *pool.PoolStore) func(params pools.GetPoolStatusByIDParams, principal interface{}) middleware.Responder {
	return func(params pools.GetPoolStatusByIDParams, principal interface{}) middleware.Responder {

		isAdmin, claims, err := isBookingAdminOrUser(principal)

		if err != nil {
			return pools.NewGetPoolStatusByIDUnauthorized().WithPayload(err.Error())
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

		if !hasPool && !isAdmin {
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

		isAdmin, claims, err := isBookingAdminOrUser(principal)

		if err != nil {
			return pools.NewGetPoolDescriptionByIDUnauthorized().WithPayload(err.Error())
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

		if !hasPool && !isAdmin {
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

		isAdmin, claims, err := isBookingAdminOrUser(principal)

		if err != nil {
			return pools.NewGetPoolsByGroupIDUnauthorized().WithPayload(err.Error())
		}

		gp, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return pools.NewGetPoolsByGroupIDNotFound().WithPayload(err.Error())
		}

		isAllowedGroup := false

		for _, name := range claims.Groups {
			if name != gp.Name {
				continue
			}
			isAllowedGroup = true
			break
		}

		if !isAllowedGroup && !isAdmin {
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

		isAdmin, claims, err := isBookingAdminOrUser(principal)

		if err != nil {
			return groups.NewGetGroupDescriptionByIDUnauthorized().WithPayload(err.Error())
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

		if !isAllowedGroup && !isAdmin {
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

		isAdmin, claims, err := isBookingAdminOrUser(principal)

		if err != nil {
			return groups.NewGetGroupIDByNameUnauthorized().WithPayload(err.Error())
		}

		isAllowedGroup := false

		for _, gp := range claims.Groups {
			if gp != params.Name {
				continue
			}
			isAllowedGroup = true
			break
		}

		if !isAllowedGroup && !isAdmin {
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
