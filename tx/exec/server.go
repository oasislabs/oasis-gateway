package exec

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	callback "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx/core"
)

const maxInactivityTimeout = time.Duration(10) * time.Minute

type ServerServices struct {
	Logger    log.Logger
	Client    eth.Client
	Callbacks callback.Calls
}

type ServerProps struct {
	PrivateKeys []*ecdsa.PrivateKey
}

type Server struct {
	master *conc.Master
	client eth.Client
	logger log.Logger
}

func NewServer(ctx context.Context, services *ServerServices, props *ServerProps) (*Server, error) {
	s := &Server{
		client: services.Client,
		logger: services.Logger.ForClass("tx/wallet", "Server"),
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
		if err := s.master.Create(ctx, crypto.PubkeyToAddress(pk.PublicKey).Hex(), pk); err != nil {
			if err := s.master.Stop(); err != nil {
				return nil, err
			}
			return nil, err
		}
	}

	return s, nil
}

func (m *Server) handle(ctx context.Context, ev conc.MasterEvent) error {
	switch ev := ev.(type) {
	case conc.CreateWorkerEvent:
		return m.create(ctx, ev)
	case conc.DestroyWorkerEvent:
		return m.destroy(ctx, ev)
	default:
		panic("received unknown request")
	}
}

func (s *Server) create(ctx context.Context, ev conc.CreateWorkerEvent) error {
	executor := NewTransactionExecutor(
		ev.Value.(*ecdsa.PrivateKey),
		types.FrontierSigner{},
		0,
		s.client,
		s.logger.ForClass("wallet", "InternalWallet"),
	)

	ev.Props.ErrC = nil
	ev.Props.WorkerHandler = conc.WorkerHandlerFunc(executor.handle)
	ev.Props.UserData = executor
	ev.Props.MaxInactivity = maxInactivityTimeout
	return nil
}

func (s *Server) destroy(ctx context.Context, ev conc.DestroyWorkerEvent) error {
	// nothing to do on a destroy to cleanup the worker
	return nil
}

// Signs the desired transaction
func (s *Server) Sign(ctx context.Context, req core.SignRequest) (*types.Transaction, errors.Err) {
	tx, err := s.master.Execute(ctx, signRequest{Transaction: req.Transaction})
	if err != nil {
		if e, ok := err.(errors.Err); ok {
			return nil, e
		}

		return nil, errors.New(errors.ErrSignTransaction, err)
	}

	return tx.(*types.Transaction), nil
}

// Executes the desired transaction.
func (s *Server) Execute(ctx context.Context, req core.ExecuteRequest) (core.ExecuteResponse, errors.Err) {
	res, err := s.master.Execute(ctx, executeRequest{
		ID:      req.ID,
		Address: req.Address,
		Data:    req.Data,
	})
	if err != nil {
		if e, ok := err.(errors.Err); ok {
			return core.ExecuteResponse{}, e
		}

		return core.ExecuteResponse{}, errors.New(errors.ErrExecuteTransaction, err)
	}

	return res.(core.ExecuteResponse), nil
}

// Remove the key's wallet and it's associated resources.
func (s *Server) Remove(ctx context.Context, req core.RemoveRequest) errors.Err {
	if err := s.master.Destroy(ctx, req.Key); err != nil {
		return errors.New(errors.ErrRemoveWallet, err)
	}

	return nil
}
