package core

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticateOK(t *testing.T) {
	auth := &NilAuth{}
	multi := &MultiAuth{}
	multi.Add(auth)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.Nil(t, err)

	res, err := multi.Authenticate(req)
	assert.Nil(t, err)

	v := res.Context().Value(multi)
	assert.Equal(t, auth, v)
}
