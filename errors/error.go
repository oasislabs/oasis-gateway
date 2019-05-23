package errors

import (
	"fmt"

	"github.com/oasislabs/developer-gateway/log"
)

type Err interface {
	Error() string
	Cause() error
	ErrorCode() ErrorCode
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

	ErrMaxAttemptsReached = ErrorCode{
		category: InternalError,
		code:     1008,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenDial = ErrorCode{
		category: InternalError,
		code:     1009,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenParseCertificate = ErrorCode{
		category: InternalError,
		code:     1009,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenGetCommittee = ErrorCode{
		category: InternalError,
		code:     1011,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenCommitteeKindUndefined = ErrorCode{
		category: InternalError,
		code:     1012,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenCommitteeNotFound = ErrorCode{
		category: InternalError,
		code:     1013,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenCommitteeLeaderNotFound = ErrorCode{
		category: InternalError,
		code:     1014,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenGetNode = ErrorCode{
		category: InternalError,
		code:     1015,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenParseNodeCertificate = ErrorCode{
		category: InternalError,
		code:     1016,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenNodeNoAddress = ErrorCode{
		category: InternalError,
		code:     1017,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenPortInvalid = ErrorCode{
		category: InternalError,
		code:     1019,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenAddressInvalid = ErrorCode{
		category: InternalError,
		code:     1019,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenAddressTransportUnsupported = ErrorCode{
		category: InternalError,
		code:     1020,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenEncodeTx = ErrorCode{
		category: InternalError,
		code:     1021,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenSubmitTx = ErrorCode{
		category: InternalError,
		code:     1022,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenEncodeRLPTx = ErrorCode{
		category: InternalError,
		code:     1023,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenSignTx = ErrorCode{
		category: InternalError,
		code:     1024,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenDecodeResponse = ErrorCode{
		category: InternalError,
		code:     1025,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenEmptyResponse = ErrorCode{
		category: InternalError,
		code:     1026,
		desc:     "Internal Error. Please check the status of the service.",
	}

	ErrEkidenGetPublicKey = ErrorCode{
		category: InternalError,
		code:     1027,
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
		code:     2005,
		desc:     "Failed to deserialize body as JSON.",
	}

	ErrInvalidAddress = ErrorCode{
		category: InputError,
		code:     2006,
		desc:     "Provided invalid address.",
	}

	ErrEmptyInput = ErrorCode{
		category: InputError,
		code:     2007,
		desc:     "Input cannot be empty.",
	}

	ErrUnknownSubscriptionType = ErrorCode{
		category: InputError,
		code:     2008,
		desc:     "Unknown subscription type.",
	}

	ErrParseQueryParams = ErrorCode{
		category: InputError,
		code:     2009,
		desc:     "Failed to parse query parameters.",
	}

	ErrSubscribeFilterAddress = ErrorCode{
		category: InputError,
		code:     2010,
		desc:     "Only address is available at this time for filtering.",
	}

	ErrInvalidKey = ErrorCode{
		category: InputError,
		code:     2011,
		desc:     "Provided invalid key.",
	}

	ErrTopicLogsSupported = ErrorCode{
		category: InputError,
		code:     2012,
		desc:     "Only logs topic supported for subscriptions.",
	}

	ErrStringNotHex = ErrorCode{
		category: InputError,
		code:     2013,
		desc:     "Provided string is not a valid hex encoding.",
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

	ErrSubscriptionAlreadyExists = ErrorCode{
		category: StateConflict,
		code:     4002,
		desc:     "Attempt to create a subscription that already exists.",
	}

	ErrAPINotImplemented = ErrorCode{
		category: NotImplemented,
		code:     5001,
		desc:     "API not Implemented.",
	}

	ErrQueueNotFound = ErrorCode{
		category: NotFound,
		code:     6001,
		desc:     "Queue not found.",
	}

	ErrSubscriptionNotFound = ErrorCode{
		category: NotFound,
		code:     6002,
		desc:     "Subscription not found.",
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

	// NotFoundErrors refers to errors in which an action is attempted
	// to be executed on a specific instance of a resource which does
	// not exist
	NotFound Category = "NotFound"

	// NotImplemented refers to errors in which the client attempts to
	// execute an action that has not yet been implemented by the server
	NotImplemented Category = "Not Implemented"
)

// Error is the implementation of an error for this package. It contains
// an instance of an ErrorCode which provides information about the error
// and a cause which might be nil if there's no underlying cause for
// the error
type Error struct {
	cause     error
	errorCode ErrorCode
}

// Error is the implementation of error for Error
func (e Error) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("[%d] error code %s with desc %s",
			e.errorCode.Code(), e.errorCode.Category(), e.errorCode.Desc())
	} else {
		return fmt.Sprintf("[%d] error code %s with desc %s with cause %s",
			e.errorCode.Code(), e.errorCode.Category(), e.errorCode.Desc(), e.cause)
	}
}

// Log implementation of log.Loggable
func (e Error) Log(fields log.Fields) {
	fields.Add("err", e.errorCode.Desc())
	fields.Add("errorCode", e.errorCode.Code())

	if e.cause != nil {
		fields.Add("cause", e.Error())
	}
}

// Cause implementation offset Err
func (e Error) Cause() error {
	return e.cause
}

// ErrorCode implementation of Err
func (e Error) ErrorCode() ErrorCode {
	return e.errorCode
}

// New creates a new instance of an error
func New(errorCode ErrorCode, cause error) Error {
	return Error{cause: cause, errorCode: errorCode}
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
