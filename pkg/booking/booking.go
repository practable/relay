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
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/groups"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/pools"
	"github.com/timdrysdale/relay/pkg/limit"
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
func API(ctx context.Context, port int, host, secret string, ps *pool.PoolStore, l *limit.Limit) {

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
	api.LoginLoginHandler = login.LoginHandlerFunc(loginHandlerFunc(ps))
	api.GroupsGetGroupIDByNameHandler = groups.GetGroupIDByNameHandlerFunc(getGroupIDByNameHandlerFunc(ps))
	api.GroupsGetGroupDescriptionByIDHandler = groups.GetGroupDescriptionByIDHandlerFunc(getGroupDescriptionByIDHandlerFunc(ps))
	api.PoolsGetPoolsByGroupIDHandler = pools.GetPoolsByGroupIDHandlerFunc(getPoolsByGroupIDHandler(ps))
	api.PoolsGetPoolDescriptionByIDHandler = pools.GetPoolDescriptionByIDHandlerFunc(getPoolDescriptionByIDHandler(ps))
	api.PoolsGetPoolStatusByIDHandler = pools.GetPoolStatusByIDHandlerFunc(getPoolStatusByIDHandler(ps))
	api.PoolsRequestSessionByPoolIDHandler = pools.RequestSessionByPoolIDHandlerFunc(requestSessionByPoolIDHandler(ps, l))
	api.PoolsAddNewPoolHandler = pools.AddNewPoolHandlerFunc(addNewPoolHandler(ps))
	api.PoolsAddActivityByPoolIDHandler = pools.AddActivityByPoolIDHandlerFunc(addActivityByPoolIDHandler(ps))

	/*
	   PoolsUpdateActivityByIDHandler: pools.UpdateActivityByIDHandlerFunc(func(params pools.UpdateActivityByIDParams, principal interface{}) middleware.Responder {
	   	return middleware.NotImplemented("operation pools.UpdateActivityByID has not yet been implemented")
	   }),
	*/

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
