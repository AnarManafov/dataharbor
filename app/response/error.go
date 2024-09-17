package response

import "net/http"

// Error codes:
// https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
//

// TransferProtocolError represents a transfer protocol error.
type TransferProtocolError struct {
	code    int
	message string
	err     error
}

// NewTransferProtocolError creates a new TransferProtocolError.
func NewTransferProtocolError(code int, err error, message string) TransferProtocolError {
	return TransferProtocolError{
		code:    code,
		message: message,
		err:     err,
	}
}

// Error returns the error message.
func (busErr *TransferProtocolError) Error() string {
	if busErr.err == nil {
		return busErr.message
	}
	return busErr.err.Error()
}

// Unwrap returns the underlying error.
func (busErr *TransferProtocolError) Unwrap() error {
	return busErr.err
}

// SystemErr represents a system error.
var SystemErr = func(err error) *TransferProtocolError {
	return &TransferProtocolError{
		code:    http.StatusBadRequest,
		message: err.Error(),
		err:     err,
	}
}

// UnAuthenticateErr represents an unauthenticated error.
var UnAuthenticateErr = &TransferProtocolError{code: http.StatusUnauthorized, message: http.StatusText(http.StatusUnauthorized)}

// UnAuthorizationErr represents an unauthorized error.
var UnAuthorizationErr = &TransferProtocolError{code: http.StatusForbidden, message: http.StatusText(http.StatusForbidden)}
