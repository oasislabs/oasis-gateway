package errors

import (
	"fmt"

	"github.com/oasislabs/developer-gateway/log"
)

type Err interface {
	Error() string
	log.Loggable
}

var (
	ErrInternalError = ErrorCode{
		category: InternalError,
		code:     1000,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrInvalidStateChangeError = ErrorCode{
		category: InternalError,
		code:     1001,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEstimateGas = ErrorCode{
		category: InternalError,
		code:     1002,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrSignedTx = ErrorCode{
		category: InternalError,
		code:     1003,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrSendTransaction = ErrorCode{
		category: InternalError,
		code:     1004,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrTransactionReceipt = ErrorCode{
		category: InternalError,
		code:     1005,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrTransactionReceiptStatus = ErrorCode{
		category: InternalError,
		code:     1006,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrFetchPendingNonce = ErrorCode{
		category: InternalError,
		code:     1007,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrOutOfRange = ErrorCode{
		category: InputError,
		code:     2001,
		desc:     "Out of range. Provided value is out of the allowed range.",
	}

	ErrHttpContentLengthMissing = ErrorCode{
		category: InputError,
		code:     2002,
		desc:     "Content-length header missing from request.",
	}

	ErrHttpContentLengthLimit = ErrorCode{
		category: InputError,
		code:     2003,
		desc:     "Content-length exceeds request limit.",
	}

	ErrHttpContentTypeApplicationJson = ErrorCode{
		category: InputError,
		code:     2004,
		desc:     "Content-type should be application/json.",
	}

	ErrDeserializeJSON = ErrorCode{
		category: InputError,
		code:     2004,
		desc:     "Failed to deserialize body as JSON.",
	}

	ErrInvalidAddress = ErrorCode{
		category: InputError,
		code:     2005,
		desc:     "Provided invalid address.",
	}

	ErrQueueLimitReached = ErrorCode{
		category: ResourceLimitReached,
		code:     3001,
		desc: "The number of unconfirmed requests has reached its limit. " +
			"No further requests can be processed until requests are confirmed.",
	}

	ErrQueueDiscardNotExists = ErrorCode{
		category: StateConflict,
		code:     4001,
		desc:     "Attempt to discard elements from a queue that does not exist.",
	}

	ErrAPINotImplemented = ErrorCode{
		category: NotImplemented,
		code:     5001,
		desc:     "API not Implemented.",
	}
)

// Category defines error categories that logically group them. This classification
// may be useful when mapping error categories together to a specific error type
// as it could be done by mapping errors to Http Status codes
type Category string

const (
	// InternalError refers to errors related to
	// programming errors or other unexpected errors in the normal
	// execution of an action, such as failing to reach another component
	// on the network. The only action a user can take out of an
	// InternalError is reach out to the operator
	InternalError Category = "InternalError"

	// InputError refers to errors that are returned because the input
	// provided to execute an action is incorrect, malformed or could
	// not be parsed
	InputError Category = "InputError"

	// StateConflict refers to errors that occur because of an attempt
	// to modify the state of an object breaking the defined rules
	StateConflict Category = "StateConflict"

	// ResourceLimitReached refers to errors in which the client has
	// reached the limit of a particular resource and may need to
	// take some action to clear up unused resources
	ResourceLimitReached Category = "ResourceLimitReached"

	// NotImplemented refers to errors in which the client attempts to
	// execute an action that has not yet been implemented by the server
	NotImplemented Category = "Not Implemented"
)

// Error is the implementation of an error for this package. It contains
// an instance of an ErrorCode which provides information about the error
// and a cause which might be nil if there's no underlying cause for
// the error
type Error struct {
	Cause     error
	ErrorCode ErrorCode
}

// Error is the implementation of error for Error
func (e Error) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("[%d] error code %s with desc %s",
			e.ErrorCode.Code(), e.ErrorCode.Category(), e.ErrorCode.Desc())
	} else {
		return fmt.Sprintf("[%d] error code %s with desc with cause %s",
			e.ErrorCode.Code(), e.ErrorCode.Category(), e.Cause)
	}
}

// Log implementation of log.Loggable
func (e Error) Log(fields log.Fields) {
	fields.Add("err", e.ErrorCode.Desc())
	fields.Add("errorCode", e.ErrorCode.Code())

	if e.Cause != nil {
		fields.Add("cause", e.Cause)
	}
}

// New creates a new instance of an error
func New(errorCode ErrorCode, cause error) Error {
	return Error{Cause: cause, ErrorCode: errorCode}
}

// ErrorCode holds the necessary information to uniquely identify an error
// and make sure that a valuable response is returned to the user
// in case of encountering an error
type ErrorCode struct {
	// category is the type of the error
	category Category

	// code is a unique identifier for the error that can be used to identify
	// the particular type of error encountered
	code int

	// desc is a human readable description of the error that occurred
	// to aid the client in debugging
	desc string
}

// Category getter for category
func (e ErrorCode) Category() Category {
	return e.category
}

// Code getter for code
func (e ErrorCode) Code() int {
	return e.code
}

// Desc getter for desc
func (e ErrorCode) Desc() string {
	return e.desc
}
