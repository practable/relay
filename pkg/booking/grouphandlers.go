package booking

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/groups"
	"github.com/timdrysdale/relay/pkg/pool"
)

func getPoolsByGroupID(ps *pool.PoolStore) func(params groups.GetPoolsByGroupIDParams, principal interface{}) middleware.Responder {
	return func(params groups.GetPoolsByGroupIDParams, principal interface{}) middleware.Responder {

		isAdmin, claims, err := isBookingAdminOrUser(principal)

		if err != nil {
			return groups.NewGetPoolsByGroupIDUnauthorized().WithPayload(err.Error())
		}

		gp, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return groups.NewGetPoolsByGroupIDNotFound().WithPayload(err.Error())
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
			return groups.NewGetPoolsByGroupIDUnauthorized().WithPayload("Missing Group Name in Groups Claim")
		}

		ids := []string{}

		for _, p := range gp.GetPools() {
			ids = append(ids, p.ID)
		}

		return groups.NewGetPoolsByGroupIDOK().WithPayload(ids)
	}
}

func getGroupDescriptionByID(ps *pool.PoolStore) func(groups.GetGroupDescriptionByIDParams, interface{}) middleware.Responder {
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

		d := gp.Description.ConvertToModel()

		return groups.NewGetGroupDescriptionByIDOK().WithPayload(d)
	}
}

func getGroupIDByName(ps *pool.PoolStore) func(groups.GetGroupIDByNameParams, interface{}) middleware.Responder {
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
