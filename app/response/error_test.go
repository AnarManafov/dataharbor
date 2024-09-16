package response

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBusErr(t *testing.T) {
	err := errors.New("underlying error")
	busErr := NewBusErr(500, err, "business error")

	assert.Equal(t, 500, busErr.code)
	assert.Equal(t, "business error", busErr.message)
	assert.Equal(t, err, busErr.err)
}

func TestBusErr_Error(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		err := errors.New("underlying error")
		busErr := NewBusErr(500, err, "business error")

		assert.Equal(t, "underlying error", busErr.Error())
	})

	t.Run("without underlying error", func(t *testing.T) {
		busErr := NewBusErr(500, nil, "business error")

		assert.Equal(t, "business error", busErr.Error())
	})
}

func TestBusErr_Unwrap(t *testing.T) {
	err := errors.New("underlying error")
	busErr := NewBusErr(500, err, "business error")

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
	assert.Equal(t, 401, UnAuthenticateErr.code)
	assert.Equal(t, "unauthenticated", UnAuthenticateErr.message)
}

func TestUnAuthorizationErr(t *testing.T) {
	assert.Equal(t, 403, UnAuthorizationErr.code)
	assert.Equal(t, "unauthorized", UnAuthorizationErr.message)
}
