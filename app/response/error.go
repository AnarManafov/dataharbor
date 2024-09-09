package response

// BusErr represents a business error.
type BusErr struct {
	code    int
	message string
	err     error
}

// NewBusErr creates a new BusErr instance.
func NewBusErr(code int, err error, message string) BusErr {
	return BusErr{
		code:    code,
		message: message,
		err:     err,
	}
}

// Error returns the error message.
func (busErr *BusErr) Error() string {
	if busErr.err == nil {
		return busErr.message
	}
	return busErr.err.Error()
}

// Unwrap returns the underlying error.
func (busErr *BusErr) Unwrap() error {
	return busErr.err
}

// Append appends a message to the error message.
func (busErr *BusErr) Append(message string) *BusErr {
	busErr.message += ": " + message
	return busErr
}

// AppendErrMsg appends an error message to the error message.
func (busErr *BusErr) AppendErrMsg(err error) *BusErr {
	busErr.message = busErr.message + ": " + err.Error()
	return busErr
}

// SystemErr represents a system error.
var SystemErr = func(err error) *BusErr {
	return &BusErr{
		code:    400,
		message: "system error: " + err.Error(),
	}
}

// UnAuthenticateErr represents an unauthenticated error.
var UnAuthenticateErr = &BusErr{code: 401, message: "unauthenticated"}

// UnAuthorizationErr represents an unauthorized error.
var UnAuthorizationErr = &BusErr{code: 403, message: "unauthorized"}
