package eth

import (
	"context"
	"crypto/ecdsa"
	stderr "errors"
	"fmt"
	"strings"
	"sync"

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

const gasPrice int64 = 1000000000

type ethRequest interface {
	RequestID() uint64
	IncAttempts()
	GetAttempts() uint
	GetContext() context.Context
	OutCh() chan<- ethResponse
}

type ethResponse struct {
	Response interface{}
	Error    errors.Err
}

type executeTransactionRequest struct {
	ID      uint64
	Address string
	Data    []byte
}

type executeTransactionResponse struct {
	ID      uint64
	Address string
	Output  []byte
}

type executeServiceRequest struct {
	Attempts uint
	Out      chan ethResponse
	Context  context.Context
	ID       uint64
	Request  backend.ExecuteServiceRequest
}

type deployServiceRequest struct {
	Attempts uint
	Out      chan ethResponse
	Context  context.Context
	ID       uint64
	Request  backend.DeployServiceRequest
}

func (r *executeServiceRequest) RequestID() uint64 {
	return r.ID
}

func (r *executeServiceRequest) IncAttempts() {
	r.Attempts++
}

func (r *executeServiceRequest) GetAttempts() uint {
	return r.Attempts
}

func (r *executeServiceRequest) GetContext() context.Context {
	return r.Context
}

func (r *executeServiceRequest) OutCh() chan<- ethResponse {
	return r.Out
}

func (r *deployServiceRequest) RequestID() uint64 {
	return r.ID
}

func (r *deployServiceRequest) IncAttempts() {
	r.Attempts++
}

func (r *deployServiceRequest) GetAttempts() uint {
	return r.Attempts
}

func (r *deployServiceRequest) GetContext() context.Context {
	return r.Context
}

func (r *deployServiceRequest) OutCh() chan<- ethResponse {
	return r.Out
}

type EthClientProperties struct {
	PrivateKeys []*ecdsa.PrivateKey
	URL         string
}

type EthClient struct {
	ctx     context.Context
	wg      sync.WaitGroup
	inCh    chan ethRequest
	logger  log.Logger
	client  eth.Client
	handler tx.TransactionHandler
	subman  *eth.SubscriptionManager
}

func (c *EthClient) startLoop(ctx context.Context) {
	c.wg.Add(1)

	go func() {
		defer func() {
			c.wg.Done()
		}()

		for {
			select {
			case <-c.ctx.Done():
				return
			case req, ok := <-c.inCh:
				if !ok {
					return
				}

				c.request(req)
			}
		}
	}()
}

func (c *EthClient) Stop() {
	close(c.inCh)
	c.wg.Wait()
}

func (c *EthClient) runTransaction(req ethRequest, fn func() (backend.Event, errors.Err)) {
	if req.GetAttempts() >= 10 {
		req.OutCh() <- ethResponse{
			Response: nil,
			Error: errors.New(
				errors.ErrMaxAttemptsReached,
				stderr.New("maximum number of attempts to execute the transaction reached")),
		}
		return
	}

	go func() {
		event, err := fn()
		if err != nil {
			// attempt a retry if there is a problem with the nonce.
			if strings.Contains(err.Error(), "nonce") {
				req.IncAttempts()
				c.inCh <- req
				return
			}

			req.OutCh() <- ethResponse{
				Response: nil,
				Error:    err,
			}
			return
		}

		req.OutCh() <- ethResponse{
			Response: event,
			Error:    nil,
		}
	}()
}

func (c *EthClient) request(req ethRequest) {
	switch request := req.(type) {
	case *executeServiceRequest:
		c.runTransaction(request, func() (backend.Event, errors.Err) {
			return c.executeService(request.Context, request.ID, request.Request)
		})
	case *deployServiceRequest:
		c.runTransaction(request, func() (backend.Event, errors.Err) {
			return c.deployService(request.Context, request.ID, request.Request)
		})
	default:
		panic("invalid request type received")
	}
}

func (c *EthClient) GetPublicKeyService(
	ctx context.Context,
	req backend.GetPublicKeyServiceRequest,
) (backend.GetPublicKeyServiceResponse, errors.Err) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetPublicKeyServiceAttempt",
		"address":   req.Address,
	})

	if err := c.verifyAddress(req.Address); err != nil {
		return backend.GetPublicKeyServiceResponse{}, err
	}

	pk, err := c.client.GetPublicKey(ctx, common.HexToAddress(req.Address))
	if err != nil {
		err := errors.New(errors.ErrInternalError, fmt.Errorf("failed to get public key %s", err.Error()))
		c.logger.Debug(ctx, "client call failed", log.MapFields{
			"call_type": "GetPublicKeyServiceFailure",
			"address":   req.Address,
		}, err)
		return backend.GetPublicKeyServiceResponse{}, err
	}

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "GetPublicKeyServiceSuccess",
		"address":   req.Address,
	})

	return backend.GetPublicKeyServiceResponse{
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
	out := make(chan ethResponse)
	c.inCh <- &deployServiceRequest{Attempts: 0, Out: out, Context: ctx, ID: id, Request: req}
	ethRes := <-out
	if ethRes.Error != nil {
		return backend.DeployServiceResponse{}, ethRes.Error
	}

	res := ethRes.Response.(backend.DeployServiceResponse)
	return res, nil
}

func (c *EthClient) ExecuteService(
	ctx context.Context,
	id uint64,
	req backend.ExecuteServiceRequest,
) (backend.ExecuteServiceResponse, errors.Err) {
	if err := c.verifyAddress(req.Address); err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	out := make(chan ethResponse)
	c.inCh <- &executeServiceRequest{Attempts: 0, Out: out, Context: ctx, ID: id, Request: req}
	ethRes := <-out
	if ethRes.Error != nil {
		return backend.ExecuteServiceResponse{}, ethRes.Error
	}

	res := ethRes.Response.(backend.ExecuteServiceResponse)
	return res, nil
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
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "ExecuteTransactionAttempt",
		"id":        req.ID,
		"address":   req.Address,
	})

	// TODO(ennsharma) return the output, not just receipt, as part of a transaction as well
	receipt, err := c.handler.Execute(ctx, tx.ExecuteRequest{
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

	if receipt.Status != 1 {
		err := errors.New(errors.ErrTransactionReceipt, stderr.New(
			"transaction receipt has status 0 which indicates a transaction execution failure"))
		c.logger.Debug(ctx, "transaction execution failed", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return nil, err
	}

	c.logger.Debug(ctx, "transaction sent successfully", log.MapFields{
		"call_type": "ExecuteTransactionSuccess",
		"id":        req.ID,
		"address":   req.Address,
	})

	address := req.Address
	if len(req.Address) == 0 {
		address = receipt.ContractAddress.Hex()
	}

	return &executeTransactionResponse{
		ID:      req.ID,
		Address: address,
		Output:  nil,
	}, nil
}

func (c *EthClient) decodeBytes(s string) ([]byte, errors.Err) {
	data, err := hexutil.Decode(s)
	if err != nil {
		return nil, errors.New(errors.ErrStringNotHex, err)
	}

	return data, nil
}

func (c *EthClient) deployService(ctx context.Context, id uint64, req backend.DeployServiceRequest) (backend.DeployServiceResponse, errors.Err) {
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

	return backend.DeployServiceResponse{ID: id, Address: res.Address}, nil
}

func (c *EthClient) executeService(ctx context.Context, id uint64, req backend.ExecuteServiceRequest) (backend.ExecuteServiceResponse, errors.Err) {
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

	// TODO(stan): handle response output once it's returned in  the transaction response
	return backend.ExecuteServiceResponse{ID: id, Address: res.Address, Output: ""}, nil
}

func NewClient(ctx context.Context, logger log.Logger, properties EthClientProperties) (*EthClient, error) {
	dialer := eth.NewUniDialer(ctx, properties.URL)
	handler, err := exec.NewServer(ctx, logger, properties.PrivateKeys, dialer)
	if err != nil {
		return nil, err
	}
	client := eth.NewPooledClient(eth.PooledClientProps{
		Pool:        dialer,
		RetryConfig: conc.RandomConfig,
	})

	c := &EthClient{
		ctx:     ctx,
		wg:      sync.WaitGroup{},
		inCh:    make(chan ethRequest, 64),
		logger:  logger.ForClass("eth", "EthClient"),
		client:  client,
		handler: handler,
		subman: eth.NewSubscriptionManager(eth.SubscriptionManagerProps{
			Context: ctx,
			Logger:  logger,
			Client:  client, // TODO(ennsharma): initialize correctly
		}),
	}

	c.startLoop(ctx)
	return c, nil
}

func DialContext(ctx context.Context, logger log.Logger, properties EthClientProperties) (*EthClient, error) {
	return NewClient(ctx, logger, properties)
}
