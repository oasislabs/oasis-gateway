package wallet

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
)

const (
	maxConcurrentWallets = 128
)

type signRequest struct {
	Transaction *types.Transaction
}

type generateRequest struct {
	Context    context.Context
	PrivateKey *ecdsa.PrivateKey
}

// Worker implements a very simple transaction signing service
type Worker struct {
	key      string
	client   eth.Client
	executor *TransactionExecutor
}

// NewWorker creates a new instance of a worker
func NewWorker(key string, client eth.Client) *Worker {
	w := &Worker{
		key:    key,
		client: client,
	}

	return w
}

func (w *Worker) handle(ctx context.Context, ev conc.WorkerEvent) (interface{}, error) {
	switch ev := ev.(type) {
	case conc.RequestWorkerEvent:
		return w.handleRequestEvent(ctx, ev)
	case conc.ErrorWorkerEvent:
		return w.handleErrorEvent(ctx, ev)
	default:
		panic("receive unexpected event type")
	}
}

func (w *Worker) handleRequestEvent(ctx context.Context, ev conc.RequestWorkerEvent) (interface{}, error) {
	switch req := ev.Value.(type) {
	case signRequest:
		return w.sign(req)
	case generateRequest:
		err := w.generate(req)
		return nil, err
	default:
		panic("invalid request received for worker")
	}
}

func (w *Worker) handleErrorEvent(ctx context.Context, ev conc.ErrorWorkerEvent) (interface{}, error) {
	// a worker should not be passing errors to the conc.Worker so
	// in that case the error is returned and the execution of the
	// worker should halt
	return nil, ev.Error
}

func (w *Worker) sign(req signRequest) (*types.Transaction, errors.Err) {
	return w.executor.SignTransaction(req.Transaction)
}

func (w *Worker) generate(req generateRequest) errors.Err {
	logger := log.NewLogrus(log.LogrusLoggerProperties{})
	w.executor = NewTransactionExecutor(
		req.PrivateKey,
		types.FrontierSigner{},
		0,
		w.client,
		logger.ForClass("wallet", "InternalWallet"),
	)

	return nil
}
