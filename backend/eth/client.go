package eth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"
	"sync"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/oasislabs/developer-gateway/api/v0/service"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

const gasPrice int64 = 1000000000

type executeServiceRequest struct {
	Attempts uint
	Out      chan backend.Event
	Context  context.Context
	ID       uint64
	Request  backend.ExecuteServiceRequest
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
	inCh   chan interface{}
	logger log.Logger
	wallet Wallet
	nonce  uint64
	signer types.Signer
	client *ethclient.Client
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

func (c *EthClient) request(req interface{}) {
	switch req := req.(type) {
	case executeServiceRequest:
		if req.Attempts >= 10 {
			req.Out <- service.ErrorEvent{
				ID:    req.ID,
				Cause: rpc.Error{Description: "failed to execute service", ErrorCode: -1},
			}
			return
		}

		if req.Attempts > 0 {
			// in case of previous failure make sure that the account nonce is correct
			if err := c.updateNonce(req.Context); err != nil {
				req.Out <- service.ErrorEvent{
					ID:    req.ID,
					Cause: rpc.Error{Description: "failed to update nonce", ErrorCode: -1},
				}
				return
			}
		}

		nonce := c.nonce
		c.nonce++

		go func() {
			event, err := c.executeService(req.Context, nonce, req.ID, req.Request)
			if err != nil {
				// attempt a retry if there is a problem with the nonce.
				if strings.Contains(err.Error(), "nonce") {
					req.Attempts++
					c.inCh <- req
					return
				}

				event = backend.ErrorEvent{
					ID:    req.ID,
					Cause: rpc.Error{Description: err.Error(), ErrorCode: -1},
				}
			}

			req.Out <- event
		}()
	default:
		panic("invalid request type received")
	}
}

func (c *EthClient) updateNonce(ctx context.Context) error {
	for attempts := 0; attempts < 10; attempts++ {
		nonce, err := c.Nonce(ctx, crypto.PubkeyToAddress(c.wallet.PrivateKey.PublicKey).Hex())
		if err != nil {
			continue
		}

		if c.nonce < nonce {
			c.nonce = nonce
		}

		return nil
	}

	return errors.New("exceeded attempts to update nonce")
}

func (c *EthClient) ExecuteService(ctx context.Context, id uint64, req backend.ExecuteServiceRequest) backend.Event {
	out := make(chan backend.Event)
	c.inCh <- executeServiceRequest{Attempts: 0, Out: out, Context: ctx, ID: id, Request: req}
	return <-out
}

func (c *EthClient) executeService(ctx context.Context, nonce, id uint64, req backend.ExecuteServiceRequest) (backend.Event, error) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "ExecuteServiceAttempt",
		"id":        id,
		"address":   req.Address,
	})

	gas, err := c.estimateGas(ctx, id, req.Address, []byte(req.Data))
	if err != nil {
		c.logger.Debug(ctx, "failed to estimate gas", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"id":        id,
			"address":   req.Address,
			"err":       err.Error(),
		})

		return backend.ErrorEvent{
			ID:    id,
			Cause: rpc.Error{Description: err.Error(), ErrorCode: -1},
		}, nil
	}

	address := common.HexToAddress(req.Address)
	tx := types.NewTransaction(nonce, address, big.NewInt(0), gas, big.NewInt(gasPrice), []byte(req.Data))
	tx, err = types.SignTx(tx, c.signer, c.wallet.PrivateKey)
	if err != nil {
		c.logger.Debug(ctx, "failure to sign transaction", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"id":        id,
			"address":   req.Address,
			"err":       err.Error(),
		})

		return backend.ErrorEvent{
			ID:    id,
			Cause: rpc.Error{Description: err.Error(), ErrorCode: -1},
		}, nil
	}

	if err := c.client.SendTransaction(ctx, tx); err != nil {
		// depending on the error received it may be useful to return the error
		// and have an upper logic to decide whether to retry the request
		c.logger.Debug(ctx, "failure to send transaction", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"id":        id,
			"address":   req.Address,
			"err":       err.Error(),
		})

		return nil, err
	}

	c.logger.Debug(ctx, "transaction sent successfully", log.MapFields{
		"call_type": "ExecuteServiceSuccess",
		"id":        id,
		"address":   req.Address,
	})
	return backend.ExecuteServiceEvent{
		ID:      id,
		Address: address.Hex(),
	}, nil
}

func (c *EthClient) Nonce(ctx context.Context, address string) (uint64, error) {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "NonceAttempt",
		"address":   address,
	})

	nonce, err := c.client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		c.logger.Debug(ctx, "PendingNonceAt request failed", log.MapFields{
			"call_type": "NonceFailure",
			"address":   address,
			"err":       err.Error(),
		})

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

	to := common.HexToAddress(address)
	gas, err := c.client.EstimateGas(ctx, ethereum.CallMsg{
		From:     crypto.PubkeyToAddress(c.wallet.PrivateKey.PublicKey),
		To:       &to,
		Gas:      0,
		GasPrice: big.NewInt(gasPrice),
		Value:    big.NewInt(0),
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

	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "EstimateGasSuccess",
		"id":        id,
		"address":   address,
	})

	return gas, nil
}

func Dial(ctx context.Context, logger log.Logger, properties EthClientProperties) (*EthClient, error) {
	if len(properties.URL) == 0 {
		return nil, errors.New("no url provided for eth client")
	}

	client, err := ethclient.Dial(properties.URL)
	if err != nil {
		return nil, err
	}

	c := &EthClient{
		ctx:    ctx,
		wg:     sync.WaitGroup{},
		inCh:   make(chan interface{}, 64),
		logger: logger.ForClass("eth", "EthClient"),
		nonce:  0,
		signer: types.FrontierSigner{},
		wallet: properties.Wallet,
		client: client,
	}

	c.startLoop(ctx)
	return c, nil
}
