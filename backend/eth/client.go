package eth

import (
	"context"
	"crypto/ecdsa"
	stderr "errors"
	"fmt"
	"net/url"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	backend "github.com/oasislabs/developer-gateway/backend/core"
	callback "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx"
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
	ctx      context.Context
	logger   log.Logger
	client   eth.Client
	executor *tx.Executor
	subman   *eth.SubscriptionManager
}

func (c *Client) GetPublicKey(
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
		err := errors.New(errors.ErrInternalError, fmt.Errorf("failed to get public key %s", err.Error()))
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

	return nil
}

func (c *Client) UnsubscribeRequest(
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
	contractAddress := req.Address

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "ExecuteTransactionAttempt",
		"id":        req.ID,
		"address":   req.Address,
	})

	res, err := c.executor.Execute(ctx, tx.ExecuteRequest{
		ID:      req.ID,
		Address: req.Address,
		Data:    req.Data,
	})
	if err != nil {
		c.logger.Debug(ctx, "failure to retrieve transaction receipt", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return nil, err
	}

	if len(contractAddress) == 0 {
		receipt, err := c.client.TransactionReceipt(ctx, common.HexToHash(res.Hash))
		if err != nil {
			err := errors.New(errors.ErrTransactionReceipt, err)
			c.logger.Debug(ctx, "failure to retrieve transaction receipt", log.MapFields{
				"call_type": "ExecuteTransactionFailure",
				"id":        req.ID,
				"address":   req.Address,
			}, err)

			return nil, err
		}

		contractAddress = receipt.ContractAddress.Hex()
	}

	c.logger.Debug(ctx, "transaction sent successfully", log.MapFields{
		"call_type": "ExecuteTransactionSuccess",
		"id":        req.ID,
		"address":   req.Address,
	})

	return &executeTransactionResponse{
		ID:      req.ID,
		Address: contractAddress,
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
	return &Client{
		ctx:      ctx,
		logger:   deps.Logger.ForClass("eth", "Client"),
		client:   deps.Client,
		executor: deps.Executor,
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
		RetryConfig: conc.RandomConfig,
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
