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
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	tx "github.com/oasislabs/developer-gateway/tx/core"
	"github.com/oasislabs/developer-gateway/tx/exec"
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

type EthClientProperties struct {
	PrivateKeys []*ecdsa.PrivateKey
	URL         string
}

type EthClient struct {
	ctx     context.Context
	logger  log.Logger
	client  eth.Client
	handler tx.TransactionHandler
	subman  *eth.SubscriptionManager
}

func (c *EthClient) GetPublicKey(
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

func (c *EthClient) verifyAddress(addr string) errors.Err {
	if len(addr) != 42 {
		return errors.New(errors.ErrInvalidAddress, nil)
	}

	if _, err := hexutil.Decode(addr); err != nil {
		return errors.New(errors.ErrInvalidAddress, err)
	}

	return nil
}

func (c *EthClient) DeployService(
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

func (c *EthClient) ExecuteService(
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

func (c *EthClient) SubscribeRequest(
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

func (c *EthClient) UnsubscribeRequest(
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

func (c *EthClient) executeTransaction(
	ctx context.Context,
	req executeTransactionRequest,
) (*executeTransactionResponse, errors.Err) {
	contractAddress := req.Address

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "ExecuteTransactionAttempt",
		"id":        req.ID,
		"address":   req.Address,
	})

	res, err := c.handler.Execute(ctx, tx.ExecuteRequest{
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

func (c *EthClient) decodeBytes(s string) ([]byte, errors.Err) {
	data, err := hexutil.Decode(s)
	if err != nil {
		return nil, errors.New(errors.ErrStringNotHex, err)
	}

	return data, nil
}

func NewClient(ctx context.Context, logger log.Logger, privateKeys []*ecdsa.PrivateKey, client eth.Client) (*EthClient, error) {
	handler, err := exec.NewServer(ctx, logger, privateKeys, client)
	if err != nil {
		return nil, err
	}

	c := &EthClient{
		ctx:     ctx,
		logger:  logger.ForClass("eth", "EthClient"),
		client:  client,
		handler: handler,
		subman: eth.NewSubscriptionManager(eth.SubscriptionManagerProps{
			Context: ctx,
			Logger:  logger,
			Client:  client,
		}),
	}

	return c, nil
}

func DialContext(ctx context.Context, logger log.Logger, properties EthClientProperties) (*EthClient, error) {
	if len(properties.URL) == 0 {
		return nil, stderr.New("no url provided for eth client")
	}

	url, err := url.Parse(properties.URL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url %s", err.Error())
	}

	if url.Scheme != "wss" && url.Scheme != "ws" {
		return nil, stderr.New("Only schemes supported are ws and wss")
	}

	dialer := eth.NewUniDialer(ctx, properties.URL)
	client := eth.NewPooledClient(eth.PooledClientProps{
		Pool:        dialer,
		RetryConfig: conc.RandomConfig,
	})

	return NewClient(ctx, logger, properties.PrivateKeys, client)
}
