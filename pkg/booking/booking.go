// Package shellaccess provides an API for accessing shellbar.
// Shellaccess is aware of the roles of hosts and clients in
// establishing ssh connections, and will allocate unique
// connections to new clients. It requires shellbar to trigger
// the SSH host to connect AFTER the client has connected,
// because SSH is a server-speaks-first protocol. Hence shellaccess
// does not need to transmit the URI of the unique connection to the host
// because shellbar will do this when the client makes its
// websocket connection. There is no guarantee a host is connected
// at any given time, and if it drops its management channel
// which is connected to the base session_id, then it cannot be
// reached. As shellbar puts a websocket wrapper around the
// already-encrypted TCP/IP, the communication remains encrypted
// end-to-end. For more details on SSH security properties, see
// https://docstore.mik.ua/orelly/networking_2ndEd/ssh/ch03_01.htm
package booking

import (
	"context"
	"flag"

	"github.com/go-openapi/loads"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/booking/restapi"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/admin"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/groups"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/pools"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	"github.com/timdrysdale/relay/pkg/pool"
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
func API(ctx context.Context, port int, host, secret string, ps *pool.PoolStore, l *bookingstore.Limit) {

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
	api.LoginLoginHandler = login.LoginHandlerFunc(loginHandler(ps))
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
