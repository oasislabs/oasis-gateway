package eth

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/url"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	stderr "github.com/pkg/errors"

	backend "github.com/oasislabs/oasis-gateway/backend/core"
	callback "github.com/oasislabs/oasis-gateway/callback/client"
	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/oasislabs/oasis-gateway/errors"
	"github.com/oasislabs/oasis-gateway/eth"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/stats"
	"github.com/oasislabs/oasis-gateway/tx"
)

const (
	getCode            string = "GetCode"
	getExpiry          string = "GetExpiry"
	getPublicKey       string = "GetPublicKey"
	deployService      string = "DeployService"
	executeService     string = "ExecuteService"
	subscribeRequest   string = "SubscribeRequest"
	unsubscribeRequest string = "UnsubscribeRequest"
)

const StatusOK = 1

type executeTransactionRequest struct {
	AAD     string
	ID      uint64
	Address string
	Data    []byte
}

type executeTransactionResponse struct {
	ID      uint64
	Address string
	Output  string
}

type ClientProps struct {
	PrivateKeys []*ecdsa.PrivateKey
	URL         string
}

type Client struct {
	ctx      context.Context
	logger   log.Logger
	client   eth.Client
	executor *tx.Executor
	subman   *eth.SubscriptionManager
	tracker  *stats.MethodTracker
}

func (c *Client) Name() string {
	return "backend.eth.Client"
}

func (c *Client) Stats() stats.Metrics {
	methodStats := c.tracker.Stats()
	walletStats := c.executor.Stats()
	return stats.Metrics{
		"methods": methodStats,
		"wallets": walletStats,
	}
}

func (c *Client) getCode(
	ctx context.Context,
	req backend.GetCodeRequest,
) (backend.GetCodeResponse, errors.Err) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetCodeAttempt",
		"address":   req.Address,
	})

	if err := c.verifyAddress(req.Address); err != nil {
		return backend.GetCodeResponse{}, err
	}

	code, err := c.client.GetCode(ctx, common.HexToAddress(req.Address))
	if err != nil {
		err := errors.New(errors.ErrInternalError, stderr.Wrapf(err, "failed to get code for address %s", req.Address))
		c.logger.Debug(ctx, "client call failed", log.MapFields{
			"call_type": "GetCodeFailure",
			"address":   req.Address,
		}, err)
		return backend.GetCodeResponse{}, err
	}

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetCodeSuccess",
		"address":   req.Address,
	})

	return backend.GetCodeResponse{
		Address: req.Address,
		Code:    code,
	}, nil
}

func (c *Client) GetCode(
	ctx context.Context,
	req backend.GetCodeRequest,
) (backend.GetCodeResponse, errors.Err) {
	v, err := c.tracker.Instrument(getCode, func() (interface{}, error) {
		return c.getCode(ctx, req)
	})

	if err != nil {
		return backend.GetCodeResponse{}, err.(errors.Err)
	}

	return v.(backend.GetCodeResponse), nil
}

func (c *Client) getExpiry(
	ctx context.Context,
	req backend.GetExpiryRequest,
) (backend.GetExpiryResponse, errors.Err) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetExpiryAttempt",
		"address":   req.Address,
	})

	if err := c.verifyAddress(req.Address); err != nil {
		return backend.GetExpiryResponse{}, err
	}

	expiry, err := c.client.GetExpiry(ctx, common.HexToAddress(req.Address))
	if err != nil {
		err := errors.New(errors.ErrInternalError, stderr.Wrapf(err, "failed to get expiry for address %s", req.Address))
		c.logger.Debug(ctx, "client call failed", log.MapFields{
			"call_type": "GetExpiryFailure",
			"address":   req.Address,
		}, err)
		return backend.GetExpiryResponse{}, err
	}

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetExpirySuccess",
		"address":   req.Address,
	})

	return backend.GetExpiryResponse{
		Address: req.Address,
		Expiry:  expiry,
	}, nil
}

func (c *Client) GetExpiry(
	ctx context.Context,
	req backend.GetExpiryRequest,
) (backend.GetExpiryResponse, errors.Err) {
	v, err := c.tracker.Instrument(getExpiry, func() (interface{}, error) {
		return c.getExpiry(ctx, req)
	})

	if err != nil {
		return backend.GetExpiryResponse{}, err.(errors.Err)
	}

	return v.(backend.GetExpiryResponse), nil
}

func (c *Client) getPublicKey(
	ctx context.Context,
	req backend.GetPublicKeyRequest,
) (backend.GetPublicKeyResponse, errors.Err) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetPublicKeyAttempt",
		"address":   req.Address,
	})

	if err := c.verifyAddress(req.Address); err != nil {
		return backend.GetPublicKeyResponse{}, err
	}

	pk, err := c.client.GetPublicKey(ctx, common.HexToAddress(req.Address))
	if err != nil {
		err := errors.New(errors.ErrInternalError, stderr.Wrapf(err, "failed to get public key for address %s", req.Address))
		c.logger.Debug(ctx, "client call failed", log.MapFields{
			"call_type": "GetPublicKeyFailure",
			"address":   req.Address,
		}, err)
		return backend.GetPublicKeyResponse{}, err
	}

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetPublicKeySuccess",
		"address":   req.Address,
	})

	return backend.GetPublicKeyResponse{
		Address:   req.Address,
		Timestamp: pk.Timestamp,
		PublicKey: pk.PublicKey,
		Signature: pk.Signature,
	}, nil
}

func (c *Client) GetPublicKey(
	ctx context.Context,
	req backend.GetPublicKeyRequest,
) (backend.GetPublicKeyResponse, errors.Err) {
	v, err := c.tracker.Instrument(getPublicKey, func() (interface{}, error) {
		return c.getPublicKey(ctx, req)
	})

	if err != nil {
		return backend.GetPublicKeyResponse{}, err.(errors.Err)
	}

	return v.(backend.GetPublicKeyResponse), nil
}

func (c *Client) verifyAddress(addr string) errors.Err {
	if len(addr) != 42 {
		return errors.New(errors.ErrInvalidAddress, stderr.New(fmt.Sprintf("Address hex should be 42 bytes long; got %s", addr)))
	}

	if _, err := hexutil.Decode(addr); err != nil {
		return errors.New(errors.ErrInvalidAddress, stderr.Wrapf(err, "failed to decode address %s", addr))
	}

	return nil
}

func (c *Client) DeployService(
	ctx context.Context,
	id uint64,
	req backend.DeployServiceRequest,
) (backend.DeployServiceResponse, errors.Err) {
	v, err := c.tracker.Instrument(deployService, func() (interface{}, error) {
		return c.deployService(ctx, id, req)
	})
	if err != nil {
		return backend.DeployServiceResponse{}, err.(errors.Err)
	}

	return v.(backend.DeployServiceResponse), nil
}

func (c *Client) deployService(
	ctx context.Context,
	id uint64,
	req backend.DeployServiceRequest,
) (backend.DeployServiceResponse, errors.Err) {
	data, err := c.decodeBytes(req.Data)
	if err != nil {
		return backend.DeployServiceResponse{}, err
	}

	res, err := c.executeTransaction(ctx, executeTransactionRequest{
		AAD:     req.AAD,
		ID:      id,
		Address: "",
		Data:    data,
	})
	if err != nil {
		return backend.DeployServiceResponse{}, err
	}

	return backend.DeployServiceResponse{
		ID:      res.ID,
		Address: res.Address,
	}, nil
}

func (c *Client) ExecuteService(
	ctx context.Context,
	id uint64,
	req backend.ExecuteServiceRequest,
) (backend.ExecuteServiceResponse, errors.Err) {
	v, err := c.tracker.Instrument(executeService, func() (interface{}, error) {
		return c.executeService(ctx, id, req)
	})
	if err != nil {
		return backend.ExecuteServiceResponse{}, err.(errors.Err)
	}

	return v.(backend.ExecuteServiceResponse), nil
}

func (c *Client) executeService(
	ctx context.Context,
	id uint64,
	req backend.ExecuteServiceRequest,
) (backend.ExecuteServiceResponse, errors.Err) {
	if err := c.verifyAddress(req.Address); err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	data, err := c.decodeBytes(req.Data)
	if err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	res, err := c.executeTransaction(ctx, executeTransactionRequest{
		AAD:     req.AAD,
		ID:      id,
		Address: req.Address,
		Data:    data,
	})
	if err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	return backend.ExecuteServiceResponse{
		ID:      res.ID,
		Address: res.Address,
		Output:  res.Output,
	}, nil
}

func (c *Client) SubscribeRequest(
	ctx context.Context,
	req backend.CreateSubscriptionRequest,
	ch chan<- interface{},
) errors.Err {
	_, err := c.tracker.Instrument(subscribeRequest, func() (interface{}, error) {
		return nil, c.subscribeRequest(ctx, req, ch)
	})
	if err != nil {
		return err.(errors.Err)
	}

	return nil
}

func (c *Client) subscribeRequest(
	ctx context.Context,
	req backend.CreateSubscriptionRequest,
	ch chan<- interface{},
) errors.Err {
	if req.Event != "logs" {
		return errors.New(errors.ErrTopicLogsSupported, nil)
	}

	var topics [][]common.Hash
	for _, topic := range req.Topics {
		topics = append(topics, []common.Hash{common.HexToHash(topic)})
	}

	var addresses = []common.Address{}
	if req.Address != "" {
		addresses = []common.Address{common.HexToAddress(req.Address)}
	}

	if err := c.subman.Create(ctx, req.SubID, &eth.LogSubscriber{
		FilterQuery: ethereum.FilterQuery{
			Addresses: addresses,
			Topics:    topics,
		},
	}, ch); err != nil {
		err := errors.New(errors.ErrInternalError, err)
		c.logger.Debug(ctx, "failed to create subscription", log.MapFields{
			"call_type": "SubscribeRequestFailure",
			"address":   req.Address,
		}, err)
		return err
	}

	return nil
}

func (c *Client) UnsubscribeRequest(
	ctx context.Context,
	req backend.DestroySubscriptionRequest,
) errors.Err {
	_, err := c.tracker.Instrument(unsubscribeRequest, func() (interface{}, error) {
		return nil, c.unsubscribeRequest(ctx, req)
	})
	if err != nil {
		return err.(errors.Err)
	}

	return nil
}

func (c *Client) unsubscribeRequest(
	ctx context.Context,
	req backend.DestroySubscriptionRequest,
) errors.Err {
	if err := c.subman.Destroy(ctx, req.SubID); err != nil {
		err := errors.New(errors.ErrInternalError, err)
		c.logger.Debug(ctx, "failed to destroy subscription", log.MapFields{
			"call_type": "UnsubscribeRequestFailure",
		}, err)
		return err
	}

	return nil
}

func (c *Client) executeTransaction(
	ctx context.Context,
	req executeTransactionRequest,
) (*executeTransactionResponse, errors.Err) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type":      "ExecuteTransactionAttempt",
		"id":             req.ID,
		"executeAddress": req.Address,
	})

	res, err := c.executor.Execute(ctx, tx.ExecuteRequest{
		AAD:     req.AAD,
		ID:      req.ID,
		Address: req.Address,
		Data:    req.Data,
	})
	if err != nil {
		c.logger.Debug(ctx, "failure to retrieve transaction receipt", log.MapFields{
			"call_type":      "ExecuteTransactionFailure",
			"id":             req.ID,
			"executeAddress": req.Address,
		}, err)

		return nil, err
	}

	c.logger.Debug(ctx, "transaction sent successfully", log.MapFields{
		"call_type":      "ExecuteTransactionSuccess",
		"id":             req.ID,
		"executeAddress": req.Address,
		"serviceAddress": res.Address,
	})

	return &executeTransactionResponse{
		ID:      req.ID,
		Address: res.Address,
		Output:  res.Output,
	}, nil
}

func (c *Client) decodeBytes(s string) ([]byte, errors.Err) {
	data, err := hexutil.Decode(s)
	if err != nil {
		return nil, errors.New(errors.ErrStringNotHex, stderr.WithStack(err))
	}

	return data, nil
}

type ClientDeps struct {
	Logger   log.Logger
	Client   eth.Client
	Executor *tx.Executor
}

type ClientServices struct {
	Logger    log.Logger
	Callbacks callback.Calls
}

func NewClientWithDeps(ctx context.Context, deps *ClientDeps) *Client {

	return &Client{
		ctx:      ctx,
		logger:   deps.Logger.ForClass("eth", "Client"),
		client:   deps.Client,
		executor: deps.Executor,
		tracker: stats.NewMethodTracker(getPublicKey,
			deployService,
			executeService,
			subscribeRequest,
			unsubscribeRequest),
		subman: eth.NewSubscriptionManager(eth.SubscriptionManagerProps{
			Context: ctx,
			Logger:  deps.Logger,
			Client:  deps.Client,
		}),
	}
}

func DialContext(ctx context.Context, services *ClientServices, props *ClientProps) (*Client, error) {
	if len(props.URL) == 0 {
		return nil, stderr.New("no url provided for eth client")
	}

	url, err := url.Parse(props.URL)
	if err != nil {
		return nil, stderr.New(fmt.Sprintf("Failed to parse url %s", err.Error()))
	}

	if url.Scheme != "wss" && url.Scheme != "ws" {
		return nil, stderr.New("Only schemes supported are ws and wss")
	}

	dialer := eth.NewUniDialer(ctx, props.URL)
	client := eth.NewPooledClient(eth.PooledClientProps{
		Pool:        dialer,
		RetryConfig: concurrent.RandomConfig,
	})

	executor, err := tx.NewExecutor(ctx, &tx.ExecutorServices{
		Logger:    services.Logger,
		Client:    client,
		Callbacks: services.Callbacks,
	}, &tx.ExecutorProps{PrivateKeys: props.PrivateKeys})
	if err != nil {
		return nil, err
	}

	return NewClientWithDeps(ctx, &ClientDeps{
		Logger:   services.Logger,
		Client:   client,
		Executor: executor,
	}), nil
}
