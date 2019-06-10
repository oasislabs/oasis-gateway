package exec

import (
	"context"
	"crypto/ecdsa"
	stderr "errors"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx/core"
)

// StatusOK defined by ethereum is the value of status
// for a transaction that succeeds
const StatusOK = 1

const gasPrice int64 = 1000000000

type signRequest struct {
	Transaction *types.Transaction
}

type executeRequest struct {
	ID      uint64
	Address string
	Data    []byte
}

type TransactionExecutor struct {
	wallet *InternalWallet
	nonce  uint64
	client eth.Client
	logger log.Logger
}

func NewTransactionExecutor(
	privateKey *ecdsa.PrivateKey,
	signer types.Signer,
	nonce uint64,
	client eth.Client,
	logger log.Logger,
) *TransactionExecutor {
	wallet := NewWallet(privateKey, signer)
	executor := &TransactionExecutor{
		wallet: wallet,
		nonce:  nonce,
		client: client,
		logger: logger,
	}

	return executor
}

func (e *TransactionExecutor) handle(ctx context.Context, ev conc.WorkerEvent) (interface{}, error) {
	switch ev := ev.(type) {
	case conc.RequestWorkerEvent:
		v, err := e.handleRequestEvent(ctx, ev)
		return v, err
	case conc.ErrorWorkerEvent:
		return e.handleErrorEvent(ctx, ev)
	default:
		panic("received unexpected event type")
	}
}

func (e *TransactionExecutor) handleRequestEvent(ctx context.Context, ev conc.RequestWorkerEvent) (interface{}, error) {
	switch req := ev.Value.(type) {
	case signRequest:
		return e.signTransaction(req.Transaction)
	case executeRequest:
		return e.executeTransaction(ctx, req)
	default:
		panic("invalid request received for worker")
	}
}

func (e *TransactionExecutor) handleErrorEvent(ctx context.Context, ev conc.ErrorWorkerEvent) (interface{}, error) {
	// a worker should not be passing errors to the conc.Worker so
	// in that case the error is returned and the execution of the
	// worker should halt
	return nil, ev.Error
}

func (e *TransactionExecutor) transactionNonce() uint64 {
	nonce := e.nonce
	e.nonce++
	return nonce
}

func (e *TransactionExecutor) updateNonce(ctx context.Context) errors.Err {
	var err error
	for attempts := 0; attempts < 10; attempts++ {

		address := e.wallet.Address().Hex()
		nonce, err := e.client.NonceAt(ctx, common.HexToAddress(address))
		if err != nil {
			e.logger.Debug(ctx, "NonceAt request failed", log.MapFields{
				"call_type": "NonceFailure",
				"address":   address,
			}, errors.New(errors.ErrFetchNonce, err))
			continue
		}

		e.nonce = nonce
		e.logger.Debug(ctx, "", log.MapFields{
			"call_type": "NonceSuccess",
			"address":   address,
		})

		return nil
	}

	e.logger.Debug(ctx, "Exceeded NonceAt request limit", log.MapFields{
		"call_type": "NonceFailure",
	}, errors.New(errors.ErrFetchNonce, err))

	return errors.New(errors.ErrFetchNonce, err)
}

func (e *TransactionExecutor) signTransaction(tx *types.Transaction) (*types.Transaction, errors.Err) {
	return e.wallet.SignTransaction(tx)
}

func (e *TransactionExecutor) estimateGas(ctx context.Context, id uint64, address string, data []byte) (uint64, errors.Err) {
	e.logger.Debug(ctx, "", log.MapFields{
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

	gas, err := e.client.EstimateGas(ctx, ethereum.CallMsg{
		From:     e.wallet.Address(),
		To:       to,
		Gas:      0,
		GasPrice: nil,
		Value:    nil,
		Data:     data,
	})

	if err != nil {
		e.logger.Debug(ctx, "", log.MapFields{
			"call_type": "EstimateGasFailure",
			"id":        id,
			"address":   address,
			"err":       err.Error(),
		})
		return 0, errors.New(errors.ErrEstimateGas, err)
	}

	if gas == 2251799813685248 {
		err := stderr.New("gas estimation could not be completed because of execution failure")
		e.logger.Debug(ctx, "", log.MapFields{
			"call_type": "EstimateGasFailure",
			"id":        id,
			"address":   address,
			"err":       err.Error(),
		})
		return 0, errors.New(errors.ErrEstimateGas, err)
	}

	e.logger.Debug(ctx, "", log.MapFields{
		"call_type": "EstimateGasSuccess",
		"id":        id,
		"address":   address,
		"gas":       gas,
	})

	return gas, nil
}

func (e *TransactionExecutor) generateAndSignTransaction(ctx context.Context, req executeRequest, gas uint64) (*types.Transaction, errors.Err) {
	nonce := e.transactionNonce()

	var tx *types.Transaction
	if len(req.Address) == 0 {
		tx = types.NewContractCreation(nonce,
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	} else {
		tx = types.NewTransaction(nonce, common.HexToAddress(req.Address),
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	}

	tx, err := e.signTransaction(tx)
	if err != nil {
		err := errors.New(errors.ErrSignedTx, err)
		e.logger.Debug(ctx, "failure to sign transaction", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return nil, err
	}
	return tx, nil
}

func (e *TransactionExecutor) executeTransaction(ctx context.Context, req executeRequest) (core.ExecuteResponse, errors.Err) {
	contractAddress := req.Address
	gas, err := e.estimateGas(ctx, req.ID, req.Address, req.Data)
	if err != nil {
		e.logger.Debug(ctx, "failed to estimate gas", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return core.ExecuteResponse{}, err
	}

	tx, err := e.generateAndSignTransaction(ctx, req, gas)
	if err != nil {
		return core.ExecuteResponse{}, err
	}

	res, derr := e.client.SendTransaction(ctx, tx)
	if derr != nil {
		// If the nonce is incorrect, update the nonce and try again
		if strings.Contains(derr.Error(), "Invalid transaction nonce") {
			err := e.updateNonce(ctx)
			if err != nil {
				e.logger.Debug(ctx, "failed to retrieve nonce", log.MapFields{
					"call_type": "ExecuteTransactionFailure",
					"id":        req.ID,
					"address":   req.Address,
				}, err)

				return core.ExecuteResponse{}, err
			}

			tx, err = e.generateAndSignTransaction(ctx, req, gas)
			if err != nil {
				return core.ExecuteResponse{}, err
			}

			res, derr = e.client.SendTransaction(ctx, tx)
			if derr != nil {
				// depending on the error received it may be useful to return the error
				// and have an upper logic to decide whether to retry the request
				err := errors.New(errors.ErrSendTransaction, derr)
				e.logger.Debug(ctx, "failure to send transaction", log.MapFields{
					"call_type": "ExecuteTransactionFailure",
					"id":        req.ID,
					"address":   req.Address,
				}, err)
				return core.ExecuteResponse{}, err
			}
		}
	}

	if res.Status != StatusOK {
		p, derr := hexutil.Decode(res.Output)
		if derr != nil {
			e.logger.Debug(ctx, "failed to decode the output of the transaction as hex", log.MapFields{
				"call_type": "DecodeTransactionOutputFailure",
				"id":        req.ID,
				"address":   req.Address,
				"err":       derr.Error(),
			})
		}

		output := string(p)
		msg := fmt.Sprintf("transaction receipt has status 0 which indicates a transaction execution failure with error %s", output)
		err := errors.New(errors.NewErrorCode(errors.InternalError, 1000, msg), stderr.New(msg))
		e.logger.Debug(ctx, "transaction execution failed", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return core.ExecuteResponse{}, err
	}

	if len(contractAddress) == 0 {
		receipt, err := e.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			err := errors.New(errors.ErrTransactionReceipt, err)
			e.logger.Debug(ctx, "failure to retrieve transaction receipt", log.MapFields{
				"call_type": "ExecuteTransactionFailure",
				"id":        req.ID,
				"address":   req.Address,
			}, err)

			return core.ExecuteResponse{}, err
		}

		contractAddress = receipt.ContractAddress.Hex()
	}

	return core.ExecuteResponse{
		Address: contractAddress,
		Output:  res.Output,
		Hash:    res.Hash,
	}, nil
}
