package response

type BusErr struct {
	Code    int
	Message string
	Err     error
}

func NewBusErr(code int, err error, message string) BusErr {
	return BusErr{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func (busErr BusErr) Error() string {
	if busErr.Err == nil {
		return busErr.Message
	}
	return busErr.Err.Error()
}

func (busErr BusErr) Unwrap() error {
	return busErr.Err
}

func (busErr BusErr) Append(message string) BusErr {
	busErr.Message += ": " + message
	return busErr
}

func (busErr BusErr) AppendErrMsg(err error) BusErr {
	busErr.Message += ": " + err.Error()
	return busErr
}

var (
	SystemErr          = BusErr{Code: 400, Message: "system error"}
	UnAuthenticateErr  = BusErr{Code: 401, Message: "unauthenticated "}
	UnAuthorizationErr = BusErr{Code: 403, Message: "unauthorized"}
)
