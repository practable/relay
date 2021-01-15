// Package manifest provides an alternative data
// format for representing poolstores that is
// easier to edit by hand, because default options
// can be specified for activities within a given
// pool, where usually the details are similar
// except for the stream details
package manifest

import (
	"time"

	"github.com/go-openapi/runtime"
	apiclient "github.com/timdrysdale/relay/pkg/bc/client"
	"github.com/timdrysdale/relay/pkg/bc/client/admin"
	"github.com/timdrysdale/relay/pkg/bc/client/groups"
	"github.com/timdrysdale/relay/pkg/bc/client/pools"
	"github.com/timdrysdale/relay/pkg/bc/models"
)

func UploadManifest(bc *apiclient.Bc, auth runtime.ClientAuthInfoWriter, timeout time.Duration, m Manifest) (*models.StoreStatus, error) {

	pids := make(map[Ref]string)

	// Going pool by pool, add the pool, then add the activities to the pool
	// Track the pool ID vs pool Ref, for adding pools to groups later

	for pref, p := range m.Pools {

		// Add new pool

		resp, err := bc.Pools.AddNewPool(
			pools.NewAddNewPoolParams().
				WithTimeout(timeout).
				WithPool(p.ToModel()),
			auth)

		if err != nil {
			// Just use client debug mode?
			//log.WithFields(log.Fields{
			//	"resp":   resp,
			//	"source": "manifest",
			//	"event":  "uploadManifest:AddNewPool:error",
			//	"error":  err.Error(),
			//	"pref":   pref,
			//}).Error("error adding new pool")
			return nil, err
		}

		pid := *resp.GetPayload().ID

		pids[pref] = pid

		for _, a := range m.GetActivitiesInPool(pref) {

			_, err = bc.Pools.AddActivityByPoolID(
				pools.NewAddActivityByPoolIDParams().
					WithTimeout(timeout).
					WithPoolID(pid).
					WithActivity(a),
				auth)

			if err != nil {
				return nil, err
			}
		}
	}

	for name, g := range m.Groups {

		ty := "group"
		n := string(name)

		mg := &models.Group{
			Description: &models.Description{
				Name: &n,
				Type: &ty,
			},
		}

		gresp, err := bc.Groups.AddNewGroup(
			groups.NewAddNewGroupParams().
				WithTimeout(timeout).
				WithGroup(mg),
			auth)

		if err != nil {
			return nil, err
		}

		gid := *gresp.GetPayload().ID

		// add pools to group

		mids := models.IDList{}

		for _, pref := range g.Pools {
			mids = append(mids, pids[pref])
		}

		_, err = bc.Groups.AddPoolsByGroupID(
			groups.NewAddPoolsByGroupIDParams().
				WithTimeout(timeout).
				WithGroupID(gid).
				WithPools(mids),
			auth)

		if err != nil {
			return nil, err
		}

	}

	sresp, err := bc.Admin.GetStoreStatus(
		admin.NewGetStoreStatusParams().
			WithTimeout(timeout),
		auth)

	status := sresp.GetPayload()

	return status, err

}
