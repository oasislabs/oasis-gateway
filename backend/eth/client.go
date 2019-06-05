package eth

import (
	"context"
	"crypto/ecdsa"
	stderr "errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
)

const StatusOK = 1

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
	Nonce   uint64
	ID      uint64
	Address string
	Data    []byte
}

type executeTransactionResponse struct {
	ID      uint64
	Address string
	Output  string
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

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
}

type EthClientProperties struct {
	Wallet Wallet
	URL    string
}

type EthClient struct {
	ctx    context.Context
	wg     sync.WaitGroup
	inCh   chan ethRequest
	logger log.Logger
	wallet Wallet
	nonce  uint64
	signer types.Signer
	client eth.Client
	subman *eth.SubscriptionManager
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

func (c *EthClient) runTransaction(req ethRequest, fn func(uint64) (backend.Event, errors.Err)) {
	if req.GetAttempts() >= 10 {
		req.OutCh() <- ethResponse{
			Response: nil,
			Error: errors.New(
				errors.ErrMaxAttemptsReached,
				stderr.New("maximum number of attempts to execute the transaction reached")),
		}
		return
	}

	if req.GetAttempts() > 0 {
		// in case of previous failure make sure that the account nonce is correct
		if err := c.updateNonce(req.GetContext()); err != nil {
			req.OutCh() <- ethResponse{
				Response: nil,
				Error:    err,
			}
			return
		}
	}

	nonce := c.nonce
	c.nonce++

	go func() {
		event, err := fn(nonce)
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
		c.runTransaction(request, func(nonce uint64) (backend.Event, errors.Err) {
			return c.executeService(request.Context, nonce, request.ID, request.Request)
		})
	case *deployServiceRequest:
		c.runTransaction(request, func(nonce uint64) (backend.Event, errors.Err) {
			return c.deployService(request.Context, nonce, request.ID, request.Request)
		})
	default:
		panic("invalid request type received")
	}
}

func (c *EthClient) updateNonce(ctx context.Context) errors.Err {
	var (
		err   errors.Err
		nonce uint64
	)

	for attempts := 0; attempts < 10; attempts++ {
		nonce, err = c.Nonce(ctx, crypto.PubkeyToAddress(c.wallet.PrivateKey.PublicKey).Hex())
		if err != nil {
			continue
		}

		if c.nonce < nonce {
			c.nonce = nonce
		}

		return nil
	}

	return err
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
	contractAddress := req.Address

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "ExecuteTransactionAttempt",
		"id":        req.ID,
		"address":   req.Address,
	})

	gas, err := c.estimateGas(ctx, req.ID, req.Address, req.Data)
	if err != nil {
		err := errors.New(errors.ErrEstimateGas, err)
		c.logger.Debug(ctx, "failed to estimate gas", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return nil, err
	}

	var tx *types.Transaction
	if len(contractAddress) == 0 {
		tx = types.NewContractCreation(req.Nonce,
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	} else {
		tx = types.NewTransaction(req.Nonce, common.HexToAddress(req.Address),
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	}

	tx, err = types.SignTx(tx, c.signer, c.wallet.PrivateKey)
	if err != nil {
		err := errors.New(errors.ErrSignedTx, err)
		c.logger.Debug(ctx, "failure to sign transaction", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return nil, err
	}

	res, err := c.client.SendTransaction(ctx, tx)
	if err != nil {
		// depending on the error received it may be useful to return the error
		// and have an upper logic to decide whether to retry the request
		err := errors.New(errors.ErrSendTransaction, err)
		c.logger.Debug(ctx, "failure to send transaction", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return nil, err
	}

	if res.Status != StatusOK {
		p, e := hexutil.Decode(res.Output)
		if e != nil {
			c.logger.Debug(ctx, "failed to decode the output of the transaction as hex", log.MapFields{
				"call_type": "DecodeTransactionOutputFailure",
				"id":        req.ID,
				"address":   req.Address,
				"err":       e.Error(),
			})
		}

		output := string(p)
		msg := fmt.Sprintf("transaction receipt has status 0 which indicates a transaction execution failure with error %s", output)
		err := errors.New(errors.NewErrorCode(errors.InternalError, 1000, msg), stderr.New(msg))
		c.logger.Debug(ctx, "transaction execution failed", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return nil, err
	}

	if len(contractAddress) == 0 {
		receipt, err := c.client.TransactionReceipt(ctx, tx.Hash())
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

func (c *EthClient) deployService(ctx context.Context, nonce, id uint64, req backend.DeployServiceRequest) (backend.DeployServiceResponse, errors.Err) {
	data, err := c.decodeBytes(req.Data)
	if err != nil {
		return backend.DeployServiceResponse{}, err
	}

	res, err := c.executeTransaction(ctx, executeTransactionRequest{
		Nonce:   nonce,
		ID:      id,
		Address: "",
		Data:    data,
	})

	if err != nil {
		return backend.DeployServiceResponse{}, err
	}

	return backend.DeployServiceResponse{ID: id, Address: res.Address}, nil
}

func (c *EthClient) executeService(ctx context.Context, nonce, id uint64, req backend.ExecuteServiceRequest) (backend.ExecuteServiceResponse, errors.Err) {
	data, err := c.decodeBytes(req.Data)
	if err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	res, err := c.executeTransaction(ctx, executeTransactionRequest{
		Nonce:   nonce,
		ID:      id,
		Address: req.Address,
		Data:    data,
	})

	if err != nil {
		return backend.ExecuteServiceResponse{}, err
	}

	return backend.ExecuteServiceResponse{ID: id, Address: res.Address, Output: res.Output}, nil
}

func (c *EthClient) Nonce(ctx context.Context, address string) (uint64, errors.Err) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "NonceAttempt",
		"address":   address,
	})

	nonce, err := c.client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		err := errors.New(errors.ErrFetchPendingNonce, err)
		c.logger.Debug(ctx, "PendingNonceAt request failed", log.MapFields{
			"call_type": "NonceFailure",
			"address":   address,
		}, err)

		return 0, err
	}

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "NonceSuccess",
		"address":   address,
	})

	return nonce, nil
}

func (c *EthClient) estimateGas(ctx context.Context, id uint64, address string, data []byte) (uint64, error) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "EstimateGasAttempt",
		"id":        id,
		"address":   address,
	})

	var to *common.Address
	var hex common.Address
	if len(address) > 0 {
		hex = common.HexToAddress(address)
		to = &hex
	}

	gas, err := c.client.EstimateGas(ctx, ethereum.CallMsg{
		From:     crypto.PubkeyToAddress(c.wallet.PrivateKey.PublicKey),
		To:       to,
		Gas:      0,
		GasPrice: nil,
		Value:    nil,
		Data:     data,
	})

	if err != nil {
		c.logger.Debug(ctx, "", log.MapFields{
			"call_type": "EstimateGasFailure",
			"id":        id,
			"address":   address,
			"err":       err.Error(),
		})
		return 0, err
	}

	if gas == 2251799813685248 {
		err := stderr.New("gas estimation could not be completed because of execution failure")
		c.logger.Debug(ctx, "", log.MapFields{
			"call_type": "EstimateGasFailure",
			"id":        id,
			"address":   address,
			"err":       err.Error(),
		})
		return 0, err
	}

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "EstimateGasSuccess",
		"id":        id,
		"address":   address,
		"gas":       gas,
	})

	return gas, nil
}

func NewClient(ctx context.Context, logger log.Logger, wallet Wallet, client eth.Client) *EthClient {
	c := &EthClient{
		ctx:    ctx,
		wg:     sync.WaitGroup{},
		inCh:   make(chan ethRequest, 64),
		logger: logger.ForClass("eth", "EthClient"),
		nonce:  0,
		signer: types.FrontierSigner{},
		wallet: wallet,
		client: client,
		subman: eth.NewSubscriptionManager(eth.SubscriptionManagerProps{
			Context: ctx,
			Logger:  logger,
			Client:  client,
		}),
	}

	c.startLoop(ctx)
	return c
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

	return NewClient(ctx, logger, properties.Wallet, client), nil
}
