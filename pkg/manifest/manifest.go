// Package manifest provides an alternative data
// format for representing poolstores that is
// easier to edit by hand, because default options
// can be specified for activities within a given
// pool, where usually the details are similar
// except for the stream details
package manifest

import (
	"fmt"
	"time"

	"github.com/go-openapi/runtime"
	apiclient "github.com/timdrysdale/relay/pkg/bc/client"
	"github.com/timdrysdale/relay/pkg/bc/client/admin"
	"github.com/timdrysdale/relay/pkg/bc/client/groups"
	"github.com/timdrysdale/relay/pkg/bc/client/pools"
	"github.com/timdrysdale/relay/pkg/bc/models"
)

func UploadManifest(bc *apiclient.Bc, auth runtime.ClientAuthInfoWriter, timeout time.Duration, m Manifest) (*models.StoreStatus, error) {

	pnames := make(map[string]string)
	pcounts := make(map[string]int)
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
			fmt.Printf("Error adding pool: %s %s\n", p.Name, err.Error())
			return nil, err
		}

		pid := *resp.GetPayload().ID

		pids[pref] = pid

		activity_count := 0

		for _, a := range m.GetActivitiesInPool(pref) {

			_, err = bc.Pools.AddActivityByPoolID(
				pools.NewAddActivityByPoolIDParams().
					WithTimeout(timeout).
					WithPoolID(pid).
					WithActivity(a),
				auth)

			if err != nil {
				fmt.Printf("Error adding activity: %s %s %s\n", p.Name, *&a.Description.ID, err.Error())
				return nil, err
			}
			activity_count += 1
			// fmt.Printf("  - pool: %s\n", *a.Description.Name)
		}

		pnames[pid] = p.Name
		pcounts[pid] = activity_count
		fmt.Printf("Pool of %3d: %s\n", activity_count, p.Name)
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
			fmt.Printf("Error adding Group: %s\n", *mg.Description.Name)
			return nil, err
		}

		gid := *gresp.GetPayload().ID

		fmt.Printf("Group: %s \n", *mg.Description.Name)

		// add pools to group

		mids := models.IDList{}

		for _, pref := range g.Pools {
			mids = append(mids, pids[pref])
			fmt.Printf("  - %3dx %s\n", pcounts[pids[pref]], pnames[pids[pref]])
		}

		_, err = bc.Groups.AddPoolsByGroupID(
			groups.NewAddPoolsByGroupIDParams().
				WithTimeout(timeout).
				WithGroupID(gid).
				WithPools(mids),
			auth)

		if err != nil {
			fmt.Printf("Error adding Pool to Group %s %s\n", gid, err.Error())
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
