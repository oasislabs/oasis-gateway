package tx

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	stderr "github.com/pkg/errors"

	callback "github.com/oasislabs/oasis-gateway/callback/client"
	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/oasislabs/oasis-gateway/errors"
	"github.com/oasislabs/oasis-gateway/eth"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/stats"
)

// Callbacks implemented by the WalletOwner
type Callbacks interface {
	// TransactionCommitted is called when the owner has successfully committed a
	// transaction
	TransactionCommitted(ctx context.Context, body callback.TransactionCommittedBody)

	// WalletOutOfFunds is called when the wallet owned by the
	// WalletOwner does not have enough funds for a transaction
	WalletOutOfFunds(ctx context.Context, body callback.WalletOutOfFundsBody)

	// WalletReachedFundsThreshold is called when the wallet owned by the
	// WalletOwner realizes it's wallet balance has go down a certain
	// threshold
	WalletReachedFundsThreshold(ctx context.Context, body callback.WalletReachedFundsThresholdBody)
}

// StatusOK defined by ethereum is the value of status
// for a transaction that succeeds
const StatusOK = 1

const gasPrice int64 = 1000000000

var retryConfig = concurrent.RetryConfig{
	Random:            false,
	UnlimitedAttempts: false,
	Attempts:          10,
	BaseExp:           2,
	BaseTimeout:       time.Second,
	MaxRetryTimeout:   5 * time.Second,
}

type signRequest struct {
	Transaction *types.Transaction
}

type createOwnerRequest struct {
	PrivateKey *ecdsa.PrivateKey
}

type statsRequest struct{}

// WalletOwner is the only instance that should interact
// with a wallet. Its main goal is to send transactions
// and keep the funding and nonce of the wallet up to
// date
type WalletOwner struct {
	wallet          Wallet
	nonce           uint64
	currentBalance  *big.Int
	startBalance    *big.Int
	consumedBalance *big.Int
	client          eth.Client
	callbacks       Callbacks
	logger          log.Logger
}

type WalletOwnerServices struct {
	Client    eth.Client
	Callbacks Callbacks
	Logger    log.Logger
}

type WalletOwnerProps struct {
	PrivateKey *ecdsa.PrivateKey
	Signer     types.Signer
	Nonce      uint64
}

// NewWalletOwner creates a new instance of a wallet
// owner. The wallet is derived from the private key
// provided
func NewWalletOwner(
	ctx context.Context,
	services *WalletOwnerServices,
	props *WalletOwnerProps,
) (*WalletOwner, error) {
	wallet := NewWallet(props.PrivateKey, props.Signer)
	owner := &WalletOwner{
		wallet:    wallet,
		nonce:     props.Nonce,
		client:    services.Client,
		callbacks: services.Callbacks,
		logger:    services.Logger.ForClass("tx", "WalletOwner"),
	}

	if err := owner.updateBalance(ctx); err != nil {
		return nil, err
	}

	if err := owner.updateNonce(ctx); err != nil {
		return nil, err
	}

	owner.startBalance = owner.currentBalance
	owner.consumedBalance = big.NewInt(0)

	return owner, nil
}

func (e *WalletOwner) updateBalance(ctx context.Context) errors.Err {
	balanceBefore := e.currentBalance

	balance, err := e.client.BalanceAt(ctx, e.wallet.Address(), nil)
	if err != nil {
		err := errors.New(errors.ErrGetBalance, err)
		e.logger.Debug(ctx, "BalanceAt request failed", log.MapFields{
			"call_type": "BalanceFailure",
			"address":   e.wallet.Address(),
		}, err)
		return err
	}

	e.currentBalance = balance

	e.callbacks.WalletReachedFundsThreshold(ctx, callback.WalletReachedFundsThresholdBody{
		Address: e.wallet.Address().Hex(),
		Before:  balanceBefore,
		After:   new(big.Int).Set(e.currentBalance),
	})

	return nil
}

func (e *WalletOwner) handle(ctx context.Context, ev concurrent.WorkerEvent) (interface{}, error) {
	switch ev := ev.(type) {
	case concurrent.RequestWorkerEvent:
		v, err := e.handleRequestEvent(ctx, ev)
		return v, err
	case concurrent.ErrorWorkerEvent:
		return e.handleErrorEvent(ctx, ev)
	default:
		panic("received unexpected event type")
	}
}

func (e *WalletOwner) handleRequestEvent(ctx context.Context, ev concurrent.RequestWorkerEvent) (interface{}, error) {
	switch req := ev.Value.(type) {
	case signRequest:
		return e.signTransaction(req.Transaction)
	case statsRequest:
		return e.getStats(ctx), nil
	case ExecuteRequest:
		return e.executeTransaction(ctx, req)
	default:
		panic("invalid request received for worker")
	}
}

func (e *WalletOwner) getStats(ctx context.Context) stats.Metrics {
	metrics := make(stats.Metrics)
	metrics["startingBalance"] = fmt.Sprintf("0x%x", e.startBalance)
	metrics["consumedBalance"] = fmt.Sprintf("0x%x", e.consumedBalance)
	metrics["currentBalance"] = fmt.Sprintf("0x%x", e.currentBalance)
	return metrics
}

func (e *WalletOwner) handleErrorEvent(ctx context.Context, ev concurrent.ErrorWorkerEvent) (interface{}, error) {
	// a worker should not be passing errors to the concurrent.Worker so
	// in that case the error is returned and the execution of the
	// worker should halt
	return nil, ev.Error
}

func (e *WalletOwner) transactionNonce() uint64 {
	nonce := e.nonce
	e.nonce++
	return nonce
}

func (e *WalletOwner) updateNonce(ctx context.Context) errors.Err {
	address := e.wallet.Address().Hex()
	nonce, err := e.client.NonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		err := errors.New(errors.ErrFetchNonce, err)
		e.logger.Debug(ctx, "NonceAt request failed", log.MapFields{
			"call_type": "NonceFailure",
			"address":   address,
		}, err)
		return err
	}

	e.nonce = nonce
	e.logger.Debug(ctx, "", log.MapFields{
		"call_type": "NonceSuccess",
		"address":   address,
		"nonce":     nonce,
	})

	return nil
}

func (e *WalletOwner) signTransaction(tx *types.Transaction) (*types.Transaction, errors.Err) {
	return e.wallet.SignTransaction(tx)
}

func (e *WalletOwner) estimateGas(ctx context.Context, id uint64, address string, data []byte) (uint64, errors.Err) {
	if len(address) == 0 {
		return e.estimateGasNonConfidential(ctx, id, address, data)
	}

	// TODO(stan): parse the data to identify whether the service is confidential.
	// estimateGas does not work for confidential services so in that case we provide a reasonable
	// amount of gas that may work
	return 60710088, nil
}

func (e *WalletOwner) estimateGasNonConfidential(ctx context.Context, id uint64, address string, data []byte) (uint64, errors.Err) {
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

	// when the gateway fails to estimate the gas of a transaction
	// returns this number which far exceeds the limit of gas in
	// a block. In this case, we should just return an error
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

func (e *WalletOwner) generateAndSignTransaction(ctx context.Context, req sendTransactionRequest, gas uint64) (*types.Transaction, error) {
	nonce := e.transactionNonce()

	var tx *types.Transaction
	if len(req.Address) == 0 {
		tx = types.NewContractCreation(nonce,
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	} else {
		tx = types.NewTransaction(nonce, common.HexToAddress(req.Address),
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	}

	return e.wallet.SignTransaction(tx)
}

type sendTransactionRequest struct {
	AAD     string
	ID      uint64
	Address string
	Gas     uint64
	Data    []byte
}

func (e *WalletOwner) sendTransaction(
	ctx context.Context,
	req sendTransactionRequest,
) (eth.SendTransactionResponse, errors.Err) {
	v, err := concurrent.RetryWithConfig(ctx, concurrent.SupplierFunc(func() (interface{}, error) {
		tx, err := e.generateAndSignTransaction(ctx, req, req.Gas)
		if err != nil {
			return ExecuteResponse{}, errors.New(errors.ErrSignedTx, err)
		}

		res, err := e.client.SendTransaction(ctx, tx)
		if err != nil {
			switch {
			case stderr.Is(err, eth.ErrExceedsBalance):
				e.callbacks.WalletOutOfFunds(ctx, callback.WalletOutOfFundsBody{
					Address: e.wallet.Address().Hex(),
				})

				return eth.SendTransactionResponse{},
					concurrent.ErrCannotRecover{Cause: errors.New(errors.ErrSendTransaction, err)}

			case stderr.Is(err, eth.ErrExceedsBlockLimit):
				return eth.SendTransactionResponse{},
					concurrent.ErrCannotRecover{Cause: errors.New(errors.ErrSendTransaction, err)}
			case stderr.Is(err, eth.ErrInvalidNonce):
				if err := e.updateNonce(ctx); err != nil {
					// if we fail to update the nonce we cannot proceed
					return eth.SendTransactionResponse{},
						concurrent.ErrCannotRecover{Cause: err}
				}

				return eth.SendTransactionResponse{}, err
			default:
				return eth.SendTransactionResponse{},
					concurrent.ErrCannotRecover{
						Cause: errors.New(errors.ErrSendTransaction, err),
					}
			}
		}

		return res, nil
	}), retryConfig)

	if err != nil {
		if err, ok := err.(errors.Err); ok {
			return eth.SendTransactionResponse{}, err
		}

		return eth.SendTransactionResponse{}, errors.New(errors.ErrSendTransaction, err)
	}

	res := v.(eth.SendTransactionResponse)
	e.callbacks.TransactionCommitted(ctx, callback.TransactionCommittedBody{
		AAD:     req.AAD,
		Address: e.wallet.Address().Hex(),
		Hash:    res.Hash,
	})

	return res, nil
}

func (e *WalletOwner) executeTransaction(ctx context.Context, req ExecuteRequest) (ExecuteResponse, errors.Err) {
	serviceAddress := req.Address
	gas, err := e.estimateGas(ctx, req.ID, req.Address, req.Data)
	if err != nil {
		e.logger.Debug(ctx, "failed to estimate gas", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return ExecuteResponse{}, err
	}

	res, err := e.sendTransaction(ctx, sendTransactionRequest{
		AAD:     req.AAD,
		ID:      req.ID,
		Address: req.Address,
		Data:    req.Data,
		Gas:     gas,
	})
	if err != nil {
		return ExecuteResponse{}, err
	}

	// failing to update the balance should not fail the execution of
	// the transaction
	_ = e.updateBalance(ctx)

	if res.Status != StatusOK {
		msg := fmt.Sprintf("transaction receipt has status %d which indicates a transaction execution failure with error %s", res.Status, res.Output)
		err := errors.New(errors.NewErrorCode(errors.InternalError, 1000, msg), stderr.New(msg))
		e.logger.Debug(ctx, "transaction execution failed", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return ExecuteResponse{}, err
	}

	receipt, err := e.transactionReceipt(ctx, res.Hash)
	if err != nil {
		e.logger.Debug(ctx, "failure to retrieve transaction receipt", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return ExecuteResponse{}, err
	}

	if len(serviceAddress) == 0 {
		// retrieve the code for the service to make sure that it has been deployed
		// successfully
		code, err := e.getCode(ctx, receipt.ContractAddress)
		if err != nil {
			e.logger.Debug(ctx, "failure to retrieve service code", log.MapFields{
				"call_type": "ExecuteTransactionFailure",
				"id":        req.ID,
				"address":   req.Address,
			}, err)

			return ExecuteResponse{}, err
		}

		// if the service's code is "0x" it means that the service failed to
		// deploy which should be returned as an error
		if len(code) <= 2 {
			err := errors.New(errors.ErrServiceCodeNotDeployed, stderr.New("service code is 0x"))
			e.logger.Debug(ctx, "failure to deploy service code", log.MapFields{
				"call_type": "ExecuteTransactionFailure",
				"id":        req.ID,
				"address":   req.Address,
			}, err)
			return ExecuteResponse{}, err
		}

		serviceAddress = receipt.ContractAddress.Hex()
	}

	// update the consumed gas
	var gasUsed big.Int
	gasUsed.SetUint64(receipt.GasUsed)
	e.consumedBalance = e.consumedBalance.Add(e.consumedBalance, &gasUsed)

	return ExecuteResponse{
		Address: serviceAddress,
		Output:  res.Output,
		Hash:    res.Hash,
	}, nil
}

func (e *WalletOwner) getCode(ctx context.Context, addr common.Address) (string, errors.Err) {
	code, err := e.client.GetCode(ctx, addr)
	if err != nil {
		return "", errors.New(errors.ErrGetServiceCode, err)
	}

	return code, nil
}

func (e *WalletOwner) transactionReceipt(ctx context.Context, hash string) (*types.Receipt, errors.Err) {
	receipt, err := e.client.TransactionReceipt(ctx, common.HexToHash(hash))
	if err != nil {
		return nil, errors.New(errors.ErrTransactionReceipt, err)
	}

	return receipt, nil
}
