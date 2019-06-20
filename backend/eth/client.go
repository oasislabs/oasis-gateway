package eth

import (
	"context"
	"crypto/ecdsa"
	stderr "errors"
	"fmt"
	"net/url"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	backend "github.com/oasislabs/developer-gateway/backend/core"
	callback "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/concurrent"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/stats"
	"github.com/oasislabs/developer-gateway/tx"
)

type methodName string

const (
	getPublicKey       methodName = "GetPublicKey"
	deployService      methodName = "DeployService"
	executeService     methodName = "ExecuteService"
	subscribeRequest   methodName = "SubscribeRequest"
	unsubscribeRequest methodName = "UnsubscribeRequest"
)

const StatusOK = 1

type executeTransactionRequest struct {
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
	PrivateKey *ecdsa.PrivateKey
	URL        string
}

type Client struct {
	ctx       context.Context
	logger    log.Logger
	client    eth.Client
	executor  *tx.Executor
	subman    *eth.SubscriptionManager
	latencies map[string]*stats.IntWindow
}

func (c *Client) Name() string {
	return "backend.eth.Client"
}

func (c *Client) Stats() stats.Metrics {
	s := make(stats.Metrics)

	for method, window := range c.latencies {
		latencyStats := window.Stats()
		methodStats := make(stats.Metrics)
		methodStats["latency"] = latencyStats
		s[method] = methodStats
	}

	return s
}

func (c *Client) GetPublicKey(
	ctx context.Context,
	req backend.GetPublicKeyRequest,
) (backend.GetPublicKeyResponse, errors.Err) {
	start := time.Now().UnixNano()
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetPublicKeyAttempt",
		"address":   req.Address,
	})

	if err := c.verifyAddress(req.Address); err != nil {
		return backend.GetPublicKeyResponse{}, err
	}

	pk, err := c.client.GetPublicKey(ctx, common.HexToAddress(req.Address))
	if err != nil {
		err := errors.New(errors.ErrInternalError, fmt.Errorf("failed to get public key %s", err.Error()))
		c.logger.Debug(ctx, "client call failed", log.MapFields{
			"call_type": "GetPublicKeyFailure",
			"address":   req.Address,
		}, err)
		return backend.GetPublicKeyResponse{}, err
	}

	latency := time.Now().UnixNano() - start
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetPublicKeySuccess",
		"address":   req.Address,
		"latency":   latency,
	})

	c.latencies[string(getPublicKey)].Add(latency)
	return backend.GetPublicKeyResponse{
		Address:   req.Address,
		Timestamp: pk.Timestamp,
		PublicKey: pk.PublicKey,
		Signature: pk.Signature,
	}, nil
}

func (c *Client) verifyAddress(addr string) errors.Err {
	if len(addr) != 42 {
		return errors.New(errors.ErrInvalidAddress, nil)
	}

	if _, err := hexutil.Decode(addr); err != nil {
		return errors.New(errors.ErrInvalidAddress, err)
	}

	return nil
}

func (c *Client) DeployService(
	ctx context.Context,
	id uint64,
	req backend.DeployServiceRequest,
) (backend.DeployServiceResponse, errors.Err) {
	start := time.Now().UnixNano()
	data, err := c.decodeBytes(req.Data)
	if err != nil {
		return backend.DeployServiceResponse{}, err
	}

	res, err := c.executeTransaction(ctx, executeTransactionRequest{
		ID:      id,
		Address: "",
		Data:    data,
	})
	if err != nil {
		return backend.DeployServiceResponse{}, err
	}

	latency := time.Now().UnixNano() - start
	c.latencies[string(deployService)].Add(latency)
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
	start := time.Now().UnixNano()
	if err := c.verifyAddress(req.Address); err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	data, err := c.decodeBytes(req.Data)
	if err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	res, err := c.executeTransaction(ctx, executeTransactionRequest{
		ID:      id,
		Address: req.Address,
		Data:    data,
	})
	if err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	latency := time.Now().UnixNano() - start
	c.latencies[string(executeService)].Add(latency)

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
	start := time.Now().UnixNano()

	if req.Topic != "logs" {
		return errors.New(errors.ErrTopicLogsSupported, nil)
	}

	if len(req.Address) == 0 {
		return errors.New(errors.ErrInvalidAddress, nil)
	}

	address := common.HexToAddress(req.Address)
	if err := c.subman.Create(ctx, req.SubID, &eth.LogSubscriber{
		FilterQuery: ethereum.FilterQuery{
			Addresses: []common.Address{address},
		},
	}, ch); err != nil {
		err := errors.New(errors.ErrInternalError, err)
		c.logger.Debug(ctx, "failed to create subscription", log.MapFields{
			"call_type": "SubscribeRequestFailure",
			"address":   req.Address,
		}, err)
		return err
	}

	latency := time.Now().UnixNano() - start
	c.latencies[string(subscribeRequest)].Add(latency)

	return nil
}

func (c *Client) UnsubscribeRequest(
	ctx context.Context,
	req backend.DestroySubscriptionRequest,
) errors.Err {
	start := time.Now().UnixNano()

	if err := c.subman.Destroy(ctx, req.SubID); err != nil {
		err := errors.New(errors.ErrInternalError, err)
		c.logger.Debug(ctx, "failed to destroy subscription", log.MapFields{
			"call_type": "UnsubscribeRequestFailure",
		}, err)
		return err
	}

	latency := time.Now().UnixNano() - start
	c.latencies[string(unsubscribeRequest)].Add(latency)

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
		"call_type":       "ExecuteTransactionSuccess",
		"id":              req.ID,
		"executeAddress":  req.Address,
		"contractAddress": res.Address,
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
		return nil, errors.New(errors.ErrStringNotHex, err)
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
	latencies := make(map[string]*stats.IntWindow)

	latencies[string(getPublicKey)] = stats.NewIntWindow(64)
	latencies[string(deployService)] = stats.NewIntWindow(64)
	latencies[string(executeService)] = stats.NewIntWindow(64)
	latencies[string(subscribeRequest)] = stats.NewIntWindow(64)
	latencies[string(unsubscribeRequest)] = stats.NewIntWindow(64)

	return &Client{
		ctx:       ctx,
		logger:    deps.Logger.ForClass("eth", "Client"),
		client:    deps.Client,
		executor:  deps.Executor,
		latencies: latencies,
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
		return nil, fmt.Errorf("Failed to parse url %s", err.Error())
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
	}, &tx.ExecutorProps{PrivateKeys: []*ecdsa.PrivateKey{props.PrivateKey}})
	if err != nil {
		return nil, err
	}

	return NewClientWithDeps(ctx, &ClientDeps{
		Logger:   services.Logger,
		Client:   client,
		Executor: executor,
	}), nil
}
