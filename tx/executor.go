package tx

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/oasislabs/oasis-gateway/errors"
	"github.com/oasislabs/oasis-gateway/eth"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/stats"
)

const maxInactivityTimeout = time.Duration(10) * time.Minute

type ExecutorServices struct {
	Logger    log.Logger
	Client    eth.Client
	Callbacks Callbacks
}

type ExecutorProps struct {
	PrivateKeys []*ecdsa.PrivateKey
}

type Executor struct {
	WalletAddresses []common.Address
	master          *concurrent.Master
	client          eth.Client
	logger          log.Logger
	callbacks       Callbacks
}

func NewExecutor(ctx context.Context, services *ExecutorServices, props *ExecutorProps) (*Executor, error) {
	s := &Executor{
		WalletAddresses: make([]common.Address, 0, len(props.PrivateKeys)),
		client:          services.Client,
		callbacks:       services.Callbacks,
		logger:          services.Logger.ForClass("tx/wallet", "Executor"),
	}

	s.master = concurrent.NewMaster(concurrent.MasterProps{
		MasterHandler:         concurrent.MasterHandlerFunc(s.handle),
		CreateWorkerOnRequest: true,
	})

	if err := s.master.Start(ctx); err != nil {
		return nil, err
	}

	// Create a worker for each provided private key
	for _, pk := range props.PrivateKeys {
		address := crypto.PubkeyToAddress(pk.PublicKey)
		s.WalletAddresses = append(s.WalletAddresses, address)
		req := createOwnerRequest{PrivateKey: pk}
		if err := s.master.Create(ctx, address.Hex(), &req); err != nil {
			if err := s.master.Stop(); err != nil {
				return nil, err
			}
			return nil, err
		}
	}

	return s, nil
}

func (m *Executor) Name() string {
	return "tx.Executor"
}

func (m *Executor) Stats() stats.Metrics {
	metrics := make(stats.Metrics)

	ctx := context.Background()
	responses, err := m.master.Broadcast(ctx, statsRequest{})
	if err != nil {
		m.logger.Warn(ctx, "failed to fetch stats from wallet owners", log.MapFields{
			"call_type": "StatsCollectionFailure",
			"err":       err.Error(),
		})
		return metrics
	}

	for _, res := range responses {
		if res.Error != nil {
			metrics[res.Key] = map[string]interface{}{
				"error": res.Error.Error(),
			}
		} else {
			metrics[res.Key] = res.Value
		}
	}

	return metrics
}

func (m *Executor) handle(ctx context.Context, ev concurrent.MasterEvent) error {
	switch ev := ev.(type) {
	case concurrent.CreateWorkerEvent:
		return m.create(ctx, ev)
	case concurrent.DestroyWorkerEvent:
		return m.destroy(ctx, ev)
	default:
		panic("received unknown request")
	}
}

func (s *Executor) create(ctx context.Context, ev concurrent.CreateWorkerEvent) error {
	req := ev.Value.(*createOwnerRequest)

	owner, err := NewWalletOwner(
		ctx,
		&WalletOwnerServices{
			Client:    s.client,
			Callbacks: s.callbacks,
			Logger:    s.logger,
		},
		&WalletOwnerProps{
			PrivateKey: req.PrivateKey,
			Signer:     types.FrontierSigner{},
			Nonce:      0,
		})
	if err != nil {
		return err
	}

	ev.Props.ErrC = nil
	ev.Props.WorkerHandler = concurrent.WorkerHandlerFunc(owner.handle)
	ev.Props.UserData = owner
	ev.Props.MaxInactivity = maxInactivityTimeout
	return nil
}

func (s *Executor) destroy(ctx context.Context, ev concurrent.DestroyWorkerEvent) error {
	// nothing to do on a destroy to cleanup the worker
	return nil
}

// Executes the desired transaction.
func (s *Executor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, errors.Err) {
	res, err := s.master.Execute(ctx, req)
	if err != nil {
		if e, ok := err.(errors.Err); ok {
			return ExecuteResponse{}, e
		}

		return ExecuteResponse{}, errors.New(errors.ErrExecuteTransaction, err)
	}

	return res.(ExecuteResponse), nil
}
