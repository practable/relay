// Package booking provides an API for booking experiments
package booking

import (
	"context"
	"flag"

	"github.com/go-openapi/loads"
	log "github.com/sirupsen/logrus"
	"github.com/practable/relay/internal/booking/restapi"
	"github.com/practable/relay/internal/booking/restapi/operations"
	"github.com/practable/relay/internal/booking/restapi/operations/admin"
	"github.com/practable/relay/internal/booking/restapi/operations/groups"
	"github.com/practable/relay/internal/booking/restapi/operations/login"
	"github.com/practable/relay/internal/booking/restapi/operations/pools"
	"github.com/practable/relay/internal/bookingstore"
	"github.com/practable/relay/internal/pool"
)

// API starts the API
// Inputs
// @closed - channel will be closed when server shutsdown
// @wg - waitgroup, we must wg.Done() when we are shutdown
// @port - where to listen locally
// @host - external FQDN of the host (for checking against tokens) e.g. https://relay-access.practable.io
// @target - FQDN of the relay instance e.g. wss://relay.practable.io
// @secret- HMAC shared secret which incoming tokens will be signed with
// @cs - pointer to the CodeStore this API shares with the shellbar websocket relay
// @options - for future backwards compatibility (no options currently available)
func API(ctx context.Context, port int, host, secret string, ps *pool.Store, l *bookingstore.Limit) {

	// TODO

	// endpoints for checking how many bookings you have, retrieving those bookings,
	// and for finding out whether you can make more bookings (would help to disable
	// booking in the booking page if no bookings can be made)

	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	//create new service API
	api := operations.NewBookingAPI(swaggerSpec)
	server := restapi.NewServer(api)

	//parse flags
	flag.Parse()

	// set the port this service will run on
	server.Port = port

	// set the Authorizer
	api.BearerAuth = validateHeader(secret, host)

	// set the Handlers

	// *** ADMIN *** //
	api.AdminDeletePoolStoreHandler = admin.DeletePoolStoreHandlerFunc(deletePoolStore(ps, l))
	api.AdminExportPoolStoreHandler = admin.ExportPoolStoreHandlerFunc(exportPoolStore(ps, l))
	api.AdminGetStoreStatusHandler = admin.GetStoreStatusHandlerFunc(getStoreStatus(ps, l))
	api.AdminImportPoolStoreHandler = admin.ImportPoolStoreHandlerFunc(importPoolStore(ps, l))
	api.AdminSetLockHandler = admin.SetLockHandlerFunc(setLock(ps, l))

	// *** GROUPS *** //
	api.GroupsAddNewGroupHandler = groups.AddNewGroupHandlerFunc(addNewGroup(ps))
	api.GroupsAddPoolsByGroupIDHandler = groups.AddPoolsByGroupIDHandlerFunc(addPoolsByGroupID(ps))
	api.GroupsDeleteGroupHandler = groups.DeleteGroupHandlerFunc(deleteGroup(ps))
	api.GroupsDeletePoolsByGroupIDHandler = groups.DeletePoolsByGroupIDHandlerFunc(deletePoolsByGroupID(ps))
	api.GroupsGetGroupIDByNameHandler = groups.GetGroupIDByNameHandlerFunc(getGroupIDByName(ps))
	api.GroupsGetGroupDescriptionByIDHandler = groups.GetGroupDescriptionByIDHandlerFunc(getGroupDescriptionByID(ps))
	api.GroupsGetPoolsByGroupIDHandler = groups.GetPoolsByGroupIDHandlerFunc(getPoolsByGroupID(ps)) //TODO implementation issues?
	api.GroupsReplacePoolsByGroupIDHandler = groups.ReplacePoolsByGroupIDHandlerFunc(replacePoolsByGroupID(ps))
	api.PoolsUpdateActivityByIDHandler = pools.UpdateActivityByIDHandlerFunc(updateActivityByID(ps))

	// *** POOLS *** //
	api.PoolsAddActivityByPoolIDHandler = pools.AddActivityByPoolIDHandlerFunc(addActivityByPoolID(ps))
	api.PoolsAddNewPoolHandler = pools.AddNewPoolHandlerFunc(addNewPool(ps))
	api.PoolsDeleteActivityByIDHandler = pools.DeleteActivityByIDHandlerFunc(deleteActivityByID(ps))
	api.PoolsDeletePoolHandler = pools.DeletePoolHandlerFunc(deletePool(ps))
	api.PoolsGetActivityByIDHandler = pools.GetActivityByIDHandlerFunc(getActivityByID(ps))
	api.PoolsGetAllPoolsHandler = pools.GetAllPoolsHandlerFunc(getAllPools(ps))
	api.PoolsGetPoolDescriptionByIDHandler = pools.GetPoolDescriptionByIDHandlerFunc(getPoolDescriptionByID(ps))
	api.PoolsGetPoolStatusByIDHandler = pools.GetPoolStatusByIDHandlerFunc(getPoolStatusByID(ps))
	api.PoolsRequestSessionByPoolIDHandler = pools.RequestSessionByPoolIDHandlerFunc(requestSessionByPoolID(ps, l))
	api.PoolsUpdateActivityByIDHandler = pools.UpdateActivityByIDHandlerFunc(updateActivityByID(ps))

	// *** USERS *** //
	api.LoginLoginHandler = login.LoginHandlerFunc(loginHandler(ps, host))
	api.LoginGetCurrentBookingsHandler = login.GetCurrentBookingsHandlerFunc(getCurrentBookings(ps, l))

	// *** NOT IN API YET ***
	// Get current bookings
	// Get max bookings allowed
	// admin lock/unlock bookings

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(); err != nil {
			log.Fatalln(err)
		}

	}()

	//serve API
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}

}
