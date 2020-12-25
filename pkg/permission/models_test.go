package permission

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/util"
)

func TestNewTokenConvertToJWTValidate(t *testing.T) {

	debug := false

	host := "some.host.io"
	topic := "someid"
	scopes := []string{"read", "write"}
	nbf := time.Now().Unix()
	iat := nbf
	exp := nbf + 10
	ct := "session"

	token := NewToken(host, topic, ct, scopes, iat, nbf, exp)

	jwtToken := ConvertToJWT(token)

	mc := jwtToken.Claims

	assert.Equal(t, token, mc)

	assert.True(t, ValidPermissionToken(jwtToken))

	p, err := GetPermissionToken(jwtToken)

	assert.NoError(t, err)

	assert.Equal(t, token, p)

	if debug {
		fmt.Println(util.Pretty(p))
	}

}
