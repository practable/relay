package booking

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/groups"
	"github.com/timdrysdale/relay/pkg/pool"
)

func addNewGroup(ps *pool.PoolStore) func(groups.AddNewGroupParams, interface{}) middleware.Responder {
	return func(params groups.AddNewGroupParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return groups.NewAddNewGroupUnauthorized().WithPayload(err.Error())
		}

		if params.Group == nil {
			groups.NewAddNewGroupInternalServerError().WithPayload("no group description in body")
		}

		pools := []*pool.Pool{}

		for _, pool := range params.Group.Pools {
			p, err := ps.GetPoolByID(pool)
			if err != nil {
				return groups.NewAddNewGroupInternalServerError().WithPayload(pool + " " + err.Error())
			}
			pools = append(pools, p)
		}

		g, err := pool.NewGroupFromModel(params.Group)

		if err != nil {
			return groups.NewAddNewGroupInternalServerError().WithPayload(err.Error())
		}

		g.SetNewRandomID()
		g.SetPools(pools)

		mid := &models.ID{
			ID: &g.ID,
		}
		return groups.NewAddNewGroupOK().WithPayload(mid)

	}
}

func addPoolsByGroupID(ps *pool.PoolStore) func(groups.AddPoolsByGroupIDParams, interface{}) middleware.Responder {
	return func(params groups.AddPoolsByGroupIDParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return groups.NewAddPoolsByGroupIDUnauthorized().WithPayload(err.Error())
		}

		g, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return groups.NewAddPoolsByGroupIDNotFound()
		}

		pools := []*pool.Pool{}

		for _, pool := range params.Pools {
			p, err := ps.GetPoolByID(pool)
			if err != nil {
				return groups.NewAddPoolsByGroupIDInternalServerError().WithPayload(pool + " " + err.Error())
			}
			pools = append(pools, p)
		}

		g.AddPools(pools)

		pids := g.GetPoolIDs()

		ids := models.IDList(pids)

		return groups.NewAddPoolsByGroupIDOK().WithPayload(ids)
	}
}

func deleteGroup(ps *pool.PoolStore) func(groups.DeleteGroupParams, interface{}) middleware.Responder {
	return func(params groups.DeleteGroupParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return groups.NewDeleteGroupUnauthorized().WithPayload(err.Error())
		}

		g, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return groups.NewDeleteGroupNotFound()
		}

		ps.DeleteGroup(g)

		return groups.NewDeleteGroupNotFound()

	}
}

func deletePoolsByGroupID(ps *pool.PoolStore) func(groups.DeletePoolsByGroupIDParams, interface{}) middleware.Responder {
	return func(params groups.DeletePoolsByGroupIDParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return groups.NewDeletePoolsByGroupIDUnauthorized().WithPayload(err.Error())
		}

		g, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return groups.NewDeletePoolsByGroupIDNotFound()
		}

		for _, pool := range params.Pools {
			p, err := ps.GetPoolByID(pool)
			if err != nil {
				continue //not an error if already deleted
			}
			g.DeletePool(p)
		}

		pids := g.GetPoolIDs()

		ids := models.IDList(pids)

		return groups.NewDeletePoolsByGroupIDOK().WithPayload(ids)

	}
}

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

func replacePoolsByGroupID(ps *pool.PoolStore) func(groups.ReplacePoolsByGroupIDParams, interface{}) middleware.Responder {

	return func(params groups.ReplacePoolsByGroupIDParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return groups.NewReplacePoolsByGroupIDUnauthorized().WithPayload(err.Error())
		}

		g, err := ps.GetGroupByID(params.GroupID)

		if err != nil {
			return groups.NewReplacePoolsByGroupIDNotFound()
		}

		pools := []*pool.Pool{}

		for _, pool := range params.Pools {
			p, err := ps.GetPoolByID(pool)
			if err != nil {
				return groups.NewReplacePoolsByGroupIDInternalServerError().WithPayload(pool + " " + err.Error())
			}
			pools = append(pools, p)
		}

		g.SetPools(pools)

		pids := g.GetPoolIDs()

		ids := models.IDList(pids)

		return groups.NewReplacePoolsByGroupIDOK().WithPayload(ids)

	}
}
