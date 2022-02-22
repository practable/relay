package booking

import (
	"fmt"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/pools"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	"github.com/timdrysdale/relay/pkg/pool"
)

func addActivityByPoolID(ps *pool.Store) func(params pools.AddActivityByPoolIDParams, principal interface{}) middleware.Responder {
	return func(params pools.AddActivityByPoolIDParams, principal interface{}) middleware.Responder {

		log.Trace("started")

		_, err := isBookingAdmin(principal)

		if err != nil {
			return pools.NewAddActivityByPoolIDUnauthorized().WithPayload(err.Error())
		}

		a := pool.NewActivityFromModel(params.Activity).WithNewRandomID()

		err = pool.CheckActivity(a)

		if err != nil {
			return pools.NewAddActivityByPoolIDInternalServerError().WithPayload(err.Error())
		}

		p, err := ps.GetPoolByID(params.PoolID)
		if err != nil {
			return pools.NewRequestSessionByPoolIDNotFound().WithPayload("Pool Does Not Exist")
		}

		err = p.AddActivity(a)
		if err != nil {
			return pools.NewAddActivityByPoolIDInternalServerError().WithPayload(err.Error())
		}

		aid := a.ID

		mid := &models.ID{
			ID: &aid,
		}

		fields := log.Fields{"pool_id": params.PoolID,
			"activity_id": a.ID,
			"name":        a.Name,
			"type":        "activity",
			"action":      "new"}

		log.WithFields(fields).Debugf("New activity named %s with ID %s added to pool %s", a.Name, a.ID, params.PoolID)

		return pools.NewAddActivityByPoolIDOK().WithPayload(mid)
	}
}

func addNewPool(ps *pool.Store) func(params pools.AddNewPoolParams, principal interface{}) middleware.Responder {
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

		d := pool.NewDescriptionFromModel(pd).WithID(uuid.New().String())

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

func deleteActivityByID(ps *pool.Store) func(pools.DeleteActivityByIDParams, interface{}) middleware.Responder {
	return func(params pools.DeleteActivityByIDParams, principal interface{}) middleware.Responder {
		_, err := isBookingAdmin(principal)

		if err != nil {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload(err.Error())
		}

		if params.ActivityID == "" || params.PoolID == "" {
			return pools.NewDeleteActivityByIDNotFound()
		}

		p, err := ps.GetPoolByID(params.PoolID)
		if err != nil {
			return pools.NewDeleteActivityByIDNotFound()
		}

		a, err := p.GetActivityByID(params.ActivityID)

		if err != nil {
			return pools.NewDeleteActivityByIDNotFound()
		}

		p.DeleteActivity(a)

		return pools.NewDeleteActivityByIDNotFound()

	}
}

func deletePool(ps *pool.Store) func(pools.DeletePoolParams, interface{}) middleware.Responder {
	return func(params pools.DeletePoolParams, principal interface{}) middleware.Responder {
		_, err := isBookingAdmin(principal)

		if err != nil {
			return pools.NewRequestSessionByPoolIDUnauthorized().WithPayload(err.Error())
		}

		p, err := ps.GetPoolByID(params.PoolID)
		if err != nil {
			return pools.NewDeleteActivityByIDNotFound()
		}

		ps.DeletePool(p)

		return pools.NewDeletePoolNotFound()
	}
}

// Function getAllPools returns all pools, all pools starting with the name string, or all pools exactly matching the name string
// This is an admin-only task.
func getAllPools(ps *pool.Store) func(params pools.GetAllPoolsParams, principal interface{}) middleware.Responder {

	return func(params pools.GetAllPoolsParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return pools.NewGetAllPoolsUnauthorized().WithPayload(err.Error())
		}

		name := ""
		if params.Name != nil {
			name = *params.Name
		}

		exact := false
		if params.Exact != nil {
			exact = *params.Exact
		}

		var pl []*pool.Pool

		if name == "" {

			pl = ps.GetAllPools() //let err=nil carry through from above
			log.Tracef("all pools requested, found  %d", len(pl))
		} else {
			if exact {
				pl, err = ps.GetPoolsByName(name)
				if err != nil {
					return pools.NewGetAllPoolsNotFound()
				}
				log.Tracef("pools named exactly %s requested, found  %d", name, len(pl))
			} else {
				pl, err = ps.GetPoolsByNamePrefix(name)
				if err != nil {
					return pools.NewGetAllPoolsNotFound()
				}
				log.Tracef("pools prefixed with %s requested, found  %d", name, len(pl))
			}
		}

		idl := models.IDList{}
		for _, p := range pl {
			idl = append(idl, p.ID)
		}

		return pools.NewGetAllPoolsOK().WithPayload(idl)
	}
}

func getActivityByID(ps *pool.Store) func(params pools.GetActivityByIDParams, principal interface{}) middleware.Responder {
	return func(params pools.GetActivityByIDParams, principal interface{}) middleware.Responder {

		isAdmin, claims, err := isBookingAdminOrUser(principal)

		if err != nil {
			return pools.NewGetActivityByIDUnauthorized().WithPayload(err.Error())
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
			return pools.NewGetActivityByIDUnauthorized().WithPayload("Pool Not In Authorized Groups")
		}

		p, err := ps.GetPoolByID(params.PoolID)
		if err != nil {
			return pools.NewGetActivityByIDNotFound().WithPayload("Pool Does Not Exist")
		}

		a, err := p.GetActivityByID(params.ActivityID)
		if err != nil {
			return pools.NewGetActivityByIDNotFound().WithPayload("Activity Does Not Exist")
		}

		d := a.Description.ConvertToModel()

		return pools.NewGetActivityByIDOK().WithPayload(d)

	}
}

func getPoolDescriptionByID(ps *pool.Store) func(params pools.GetPoolDescriptionByIDParams, principal interface{}) middleware.Responder {
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

		d := p.Description.ConvertToModel()

		return pools.NewGetPoolDescriptionByIDOK().WithPayload(d)

	}
}

func getPoolStatusByID(ps *pool.Store) func(params pools.GetPoolStatusByIDParams, principal interface{}) middleware.Responder {
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
		avail := int64(p.CountAvailable() - p.CountInUse())
		s.Available = &avail
		s.Used = int64(p.CountInUse())

		return pools.NewGetPoolStatusByIDOK().WithPayload(&s)
	}
}

func requestSessionByPoolID(ps *pool.Store, l *bookingstore.Limit) func(params pools.RequestSessionByPoolIDParams, principal interface{}) middleware.Responder {

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
			return pools.NewRequestSessionByPoolIDNotFound().WithPayload("Pool Does Not Exist")
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
		cancelBooking, confirmBooking, sID, err := l.ProvisionalRequest(claims.Subject, exp)

		if err != nil {

			event := "requestSession:activityRequest:overLimit"
			msg := "Maximum concurrent sessions already reached. Try again later."

			if err.Error() == "denied: no new sessions allowed" {
				event = "requestSession:activityRequest:denied:noNewSessionsAllowed"
				msg = "No new sessions allowed. Try again later."
			}

			lf := log.Fields{
				"source":    "booking",
				"event":     event,
				"poolID":    params.PoolID,
				"userID":    claims.Subject,
				"duration":  duration,
				"expiresAt": exp,
				"error":     err.Error(),
			}
			log.WithFields(lf).Info("booking:requestSession:activityRequest:OverLimit")
			return pools.NewRequestSessionByPoolIDPaymentRequired().WithPayload(msg)
		}

		aID, err := p.ActivityRequestAny(duration)

		if err != nil {
			lf := log.Fields{
				"source":    "booking",
				"event":     "requestSession:activityRequest:notFound",
				"sessionID": sID,
				"poolID":    params.PoolID,
				"userID":    claims.Subject,
				"duration":  duration,
				"expiresAt": exp,
				"error":     err.Error(),
			}
			log.WithFields(lf).Error("booking:requestSession:activityRequest:notFound")
			cancelBooking()
			return pools.NewRequestSessionByPoolIDNotFound().WithPayload(err.Error())
		}

		a, err := p.GetActivityByID(aID)

		if err != nil {

			lf := log.Fields{
				"source":     "booking",
				"event":      "requestSession:getActivity:internalError",
				"activityID": aID,
				"sessionID":  sID,
				"poolID":     params.PoolID,
				"userID":     claims.Subject,
				"duration":   duration,
				"expiresAt":  exp,
				"error":      err.Error(),
			}
			log.WithFields(lf).Error("booking:requestSession:getActivity:internalError")
			cancelBooking()
			return pools.NewRequestSessionByPoolIDInternalServerError().WithPayload(err.Error())
		}

		// convert our activity from pkg/pool type to booking/models type
		ma := a.ConvertToModel()
		flexp := float64(exp)
		ma.Exp = &flexp

		// Iterate through sessions, making tokens

		iat := ps.Now() - 1
		nbf := ps.Now() - 1

		bearers := make([]string, len(ma.Streams))

		for idx, s := range ma.Streams {

			claims := pool.MakeClaims(s.Permission)

			claims.IssuedAt = jwt.NewNumericDate(time.Unix(iat, 0))
			claims.NotBefore = jwt.NewNumericDate(time.Unix(nbf, 0))
			claims.ExpiresAt = jwt.NewNumericDate(time.Unix(exp, 0))

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			bearer, err := token.SignedString(ps.Secret)
			if err != nil {

				lf := log.Fields{
					"source":     "booking",
					"event":      "requestSession:makeTokens:internalError",
					"activityID": aID,
					"sessionID":  sID,
					"poolID":     params.PoolID,
					"userID":     claims.Subject,
					"duration":   duration,
					"issuedAt":   iat,
					"expiresAt":  exp,
					"error":      err.Error(),
				}
				log.WithFields(lf).Error("booking:requestSession:makeTokens:internalError")
				cancelBooking()
				return pools.NewRequestSessionByPoolIDInternalServerError().WithPayload(err.Error())
			}

			bearers[idx] = bearer

		}

		for idx, bearer := range bearers {
			ma.Streams[idx].Token = bearer
		}

		lf := log.Fields{
			"source":     "booking:requestSession",
			"event":      "granted",
			"activityID": aID,
			"sessionID":  sID,
			"poolID":     params.PoolID,
			"userID":     claims.Subject,
			"duration":   duration,
			"issuedAt":   iat,
			"expiresAt":  exp,
		}
		log.WithFields(lf).Info("booking:requestSession:granted")

		confirmBooking(ma) // confirm booking with Limit checker
		return pools.NewRequestSessionByPoolIDOK().WithPayload(ma)
	}
}

func updateActivityByID(ps *pool.Store) func(params pools.UpdateActivityByIDParams, principal interface{}) middleware.Responder {
	return func(params pools.UpdateActivityByIDParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)
		if err != nil {
			return pools.NewUpdateActivityByIDUnauthorized().WithPayload(err.Error())
		}

		a := pool.NewActivityFromModel(params.Activity) // leave ID unchanged
		err = pool.CheckActivity(a)
		if err != nil {
			return pools.NewUpdateActivityByIDInternalServerError().WithPayload(err.Error())
		}

		// check activity ID in body matches that in path
		if a.ID != params.ActivityID {
			return pools.NewUpdateActivityByIDNotFound().WithPayload(fmt.Sprintf("ID in path (%s) does not match that in body (%s)", params.ActivityID, a.ID))
		}

		p, err := ps.GetPoolByID(params.PoolID)
		if err != nil {
			return pools.NewUpdateActivityByIDNotFound().WithPayload("Pool Does Not Exist")
		}

		err = p.AddActivity(a)
		if err != nil {
			return pools.NewUpdateActivityByIDInternalServerError().WithPayload(err.Error())
		}

		aid := a.ID

		mid := &models.ID{
			ID: &aid,
		}

		fields := log.Fields{"pool_id": params.PoolID,
			"activity_id": a.ID,
			"name":        a.Name,
			"type":        "activity",
			"action":      "update"}

		log.WithFields(fields).Debugf("Update activity named %s with ID %s in pool %s", a.Name, a.ID, params.PoolID)

		return pools.NewUpdateActivityByIDOK().WithPayload(mid)
	}
}
