package deny

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeny(t *testing.T) {

	ds := New()

	// mock time
	ds.SetNowFunc(func() int64 { return 1673952000 })

	// Test adding & getting IDs
	ds.Allow("id0", 1673952010)
	ds.Allow("id1", 1673952020)

	ds.Deny("id2", 1673952010)
	ds.Deny("id3", 1673952020)

	ae := []string{"id0", "id1"}
	aa := ds.GetAllowList()

	assert.Equal(t, ae, aa)

	de := []string{"id2", "id3"}
	da := ds.GetDenyList()

	assert.Equal(t, de, da)

	// Check status
	assert.Equal(t, false, ds.IsDenied("id0"))
	assert.Equal(t, false, ds.IsDenied("id1"))
	assert.Equal(t, true, ds.IsDenied("id2"))
	assert.Equal(t, true, ds.IsDenied("id3"))
	assert.Equal(t, false, ds.IsDenied("unknown"))

	// Test Pruning

	ds.SetNowFunc(func() int64 { return 1673952011 })

	ds.Prune()

	ae = []string{"id1"}
	aa = ds.GetAllowList()
	assert.Equal(t, ae, aa)

	de = []string{"id3"}
	da = ds.GetDenyList()
	assert.Equal(t, de, da)

	// Check status
	assert.Equal(t, false, ds.IsDenied("id0"))
	assert.Equal(t, false, ds.IsDenied("id1"))
	assert.Equal(t, false, ds.IsDenied("id2"))
	assert.Equal(t, true, ds.IsDenied("id3"))
	assert.Equal(t, false, ds.IsDenied("unknown"))
}
