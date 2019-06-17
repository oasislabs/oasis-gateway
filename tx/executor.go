package tx

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
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
	master    *conc.Master
	client    eth.Client
	logger    log.Logger
	callbacks Callbacks
}

func NewExecutor(ctx context.Context, services *ExecutorServices, props *ExecutorProps) (*Executor, error) {
	s := &Executor{
		client:    services.Client,
		callbacks: services.Callbacks,
		logger:    services.Logger.ForClass("tx/wallet", "Executor"),
	}

	s.master = conc.NewMaster(conc.MasterProps{
		MasterHandler:         conc.MasterHandlerFunc(s.handle),
		CreateWorkerOnRequest: true,
	})

	if err := s.master.Start(ctx); err != nil {
		return nil, err
	}

	// Create a worker for each provided private key
	for _, pk := range props.PrivateKeys {
		address := crypto.PubkeyToAddress(pk.PublicKey).Hex()
		req := createOwnerRequest{PrivateKey: pk}
		if err := s.master.Create(ctx, address, &req); err != nil {
			if err := s.master.Stop(); err != nil {
				return nil, err
			}
			return nil, err
		}
	}

	return s, nil
}

func (m *Executor) handle(ctx context.Context, ev conc.MasterEvent) error {
	switch ev := ev.(type) {
	case conc.CreateWorkerEvent:
		return m.create(ctx, ev)
	case conc.DestroyWorkerEvent:
		return m.destroy(ctx, ev)
	default:
		panic("received unknown request")
	}
}

func (s *Executor) create(ctx context.Context, ev conc.CreateWorkerEvent) error {
	req := ev.Value.(*createOwnerRequest)

	owner := NewWalletOwner(
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

	ev.Props.ErrC = nil
	ev.Props.WorkerHandler = conc.WorkerHandlerFunc(owner.handle)
	ev.Props.UserData = owner
	ev.Props.MaxInactivity = maxInactivityTimeout
	return nil
}

func (s *Executor) destroy(ctx context.Context, ev conc.DestroyWorkerEvent) error {
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
