package info

import (
	"context"

	ethereum "github.com/ethereum/go-ethereum/common"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/rpc"
)

// Client interface for the underlying operations needed for the API
// implementation
type Client interface {
	Senders() []ethereum.Address
}

type Services struct {
	Logger log.Logger
	Client Client
}

// InfoHandler is the handler to satisfy information requests.
type InfoHandler struct {
	logger log.Logger
	client Client
}

// NewHandler creates a new instance of a version handler
func NewInfoHandler(services Services) InfoHandler {
	if services.Client == nil {
		panic("Request must be provided as a service")
	}
	if services.Logger == nil {
		panic("Logger must be provided as a service")
	}

	return InfoHandler{
		logger: services.Logger.ForClass("info", "handler"),
		client: services.Client,
	}
}

// GetVersion returns the version of the component
func (h InfoHandler) GetVersion(ctx context.Context, v interface{}) (interface{}, error) {
	return &GetVersionResponse{
		Version: 0,
	}, nil
}

// GetSenders returns the addresses of the accounts the gateway uses
// to sign transactions.
func (h InfoHandler) GetSenders(ctx context.Context, v interface{}) (interface{}, error) {
	addresses := h.client.Senders()
	hexAddresses := make([]string, 0, len(addresses))
	for _, address := range addresses {
		hexAddresses = append(hexAddresses, address.Hex())
	}
	return &GetSendersResponse{
		Addresses: hexAddresses,
	}, nil
}

// BindHandler binds the version handler to the handler binder
func BindHandler(services Services, binder rpc.HandlerBinder) {
	handler := NewInfoHandler(services)

	binder.Bind("GET", "/v0/api/version", rpc.HandlerFunc(handler.GetVersion),
		rpc.EntityFactoryFunc(func() interface{} { return nil }))

	binder.Bind("GET", "/v0/api/getSenders", rpc.HandlerFunc(handler.GetSenders),
		rpc.EntityFactoryFunc(func() interface{} { return nil }))
}
