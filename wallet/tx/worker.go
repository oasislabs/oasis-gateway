package tx

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/conc"
	ethereum "github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/wallet/core"
)

const (
	maxConcurrentWallets = 128
)

type signRequest struct {
	Transaction *types.Transaction
}

type generateRequest struct {
	Context    context.Context
	URL        string
	PrivateKey *ecdsa.PrivateKey
}

// Worker implements a very simple transaction signing service/
type Worker struct {
	key    string
	wallet core.InternalWallet
}

// NewWorker creates a new instance of a worker
func NewWorker(key string) *Worker {
	w := &Worker{
		key:    key,
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
	return w.wallet.SignTransaction(req.Transaction)
}

func (w *Worker) generate(req generateRequest) errors.Err {
	dialer := ethereum.NewUniDialer(req.Context, req.URL)
	pooledClient := ethereum.NewPooledClient(ethereum.PooledClientProps{
		Pool:        dialer,
		RetryConfig: conc.RandomConfig,
	})
	logger := log.NewLogrus(log.LogrusLoggerProperties{})
	wallet := core.InternalWallet{
		PrivateKey: req.PrivateKey,
		Signer:     types.FrontierSigner{},
		Nonce:      0,
		Client:     pooledClient,
		Logger:     logger.ForClass("wallet", "InternalWallet"),
	}

	w.wallet = wallet

	return nil
}
