package exec

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
	"github.com/oasislabs/developer-gateway/tx/core"
)

const maxInactivityTimeout = time.Duration(10) * time.Minute

type Server struct {
	master *conc.Master
	dialer *eth.UniDialer
	logger log.Logger
}

func NewServer(ctx context.Context, logger log.Logger, pks []*ecdsa.PrivateKey, dialer *eth.UniDialer) (*Server, error) {
	s := &Server{
		dialer: dialer,
		logger: logger.ForClass("tx/wallet", "Server"),
	}

	s.master = conc.NewMaster(conc.MasterProps{
		MasterHandler:         conc.MasterHandlerFunc(s.handle),
		CreateWorkerOnRequest: true,
	})

	if err := s.master.Start(ctx); err != nil {
		return nil, err
	}

	// Create a worker for each provided private key
	for _, pk := range pks {
		if err := s.master.Create(ctx, crypto.PubkeyToAddress(pk.PublicKey).Hex(), pk); err != nil {
			if e := s.master.Stop(); e != nil {
				return nil, e
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
	logger := log.NewLogrus(log.LogrusLoggerProperties{})

	client := eth.NewPooledClient(eth.PooledClientProps{
		Pool:        s.dialer,
		RetryConfig: conc.RandomConfig,
	})
	executor := NewTransactionExecutor(
		ev.Value.(*ecdsa.PrivateKey),
		types.FrontierSigner{},
		0,
		client,
		logger.ForClass("wallet", "InternalWallet"),
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

// Executes the desired transaction.
func (s *Server) Execute(ctx context.Context, req core.ExecuteRequest) (*types.Receipt, errors.Err) {
	var (
		receipt interface{}
		err     error
	)

	if req.Key == "" {
		receipt, err = s.master.Execute(ctx, executeRequest{
			ID:      req.ID,
			Address: req.Address,
			Data:    req.Data,
		})
	} else {
		receipt, err = s.master.Request(ctx, req.Key, executeRequest{
			ID:      req.ID,
			Address: req.Address,
			Data:    req.Data,
		})
	}
	if err != nil {
		return nil, errors.New(errors.ErrExecuteTransaction, err)
	}

	return receipt.(*types.Receipt), nil
}

// Retrieves the public key for the desired address
func (s *Server) PublicKey(ctx context.Context, req core.PublicKeyRequest) (eth.PublicKey, errors.Err) {
	publicKey, err := s.master.Execute(ctx, publicKeyRequest{})
	if err != nil {
		return eth.PublicKey{}, errors.New(errors.ErrGetPublicKey, err)
	}

	return publicKey.(eth.PublicKey), nil
}

// Remove the key's wallet and it's associated resources.
func (s *Server) Remove(ctx context.Context, req core.RemoveRequest) errors.Err {
	if err := s.master.Destroy(ctx, req.Key); err != nil {
		return errors.New(errors.ErrRemoveWallet, err)
	}

	return nil
}
