package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBusErr(t *testing.T) {
	err := errors.New("underlying error")
	busErr := NewTransferProtocolError(500, err, "Transfer protocol error")

	assert.Equal(t, 500, busErr.code)
	assert.Equal(t, "Transfer protocol error", busErr.message)
	assert.Equal(t, err, busErr.err)
}

func TestBusErr_Error(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		err := errors.New("underlying error")
		busErr := NewTransferProtocolError(500, err, "Transfer protocol error")

		assert.Equal(t, "underlying error", busErr.Error())
	})

	t.Run("without underlying error", func(t *testing.T) {
		busErr := NewTransferProtocolError(500, nil, "Transfer protocol error")

		assert.Equal(t, "Transfer protocol error", busErr.Error())
	})
}

func TestBusErr_Unwrap(t *testing.T) {
	err := errors.New("underlying error")
	busErr := NewTransferProtocolError(500, err, "Transfer protocol error")

	assert.Equal(t, err, busErr.Unwrap())
}

func TestSystemErr(t *testing.T) {
	err := errors.New("system error")
	systemErr := SystemErr(err)

	assert.Equal(t, 400, systemErr.code)
	assert.Equal(t, "system error", systemErr.message)
	assert.Equal(t, err, systemErr.err)
}

func TestUnAuthenticateErr(t *testing.T) {
	assert.Equal(t, http.StatusUnauthorized, UnAuthenticateErr.code)
	assert.Equal(t, http.StatusText(http.StatusUnauthorized), UnAuthenticateErr.message)
}

func TestUnAuthorizationErr(t *testing.T) {
	assert.Equal(t, http.StatusForbidden, UnAuthorizationErr.code)
	assert.Equal(t, http.StatusText(http.StatusForbidden), UnAuthorizationErr.message)
}
