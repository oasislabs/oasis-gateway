package wallet

import (
	"context"
	"crypto/ecdsa"
	err "errors"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx/core"
)

const maxInactivityTimeout = time.Duration(10) * time.Minute

type Server struct {
	master *conc.Master
	pks    []*ecdsa.PrivateKey
	available chan int
	unavailable map[string]int
	client eth.Client
	logger log.Logger
}

func NewServer(ctx context.Context, logger log.Logger, pks []*ecdsa.PrivateKey, client *eth.Client) *Server {
	s := &Server{
		client: *client,
		pks:    pks,
		available: make(chan int, maxConcurrentWallets),
		logger: logger.ForClass("tx/wallet", "Server"),
	}

	s.master = conc.NewMaster(conc.MasterProps{
		MasterHandler:         conc.MasterHandlerFunc(s.handle),
		CreateWorkerOnRequest: true,
	})

	go func() {
		for i := 0; i < len(pks); i++ {
				s.available <- i
		}
	}()
	if err := s.master.Start(ctx); err != nil {
		panic("failed to start master")
	}

	return s
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
	worker := NewWorker(ev.Key, s.client)

	ev.Props.ErrC = nil
	ev.Props.WorkerHandler = conc.WorkerHandlerFunc(worker.handle)
	ev.Props.UserData = worker
	ev.Props.MaxInactivity = maxInactivityTimeout

	return nil
}

func (s *Server) destroy(ctx context.Context, ev conc.DestroyWorkerEvent) error {
	// nothing to do on a destroy to cleanup the worker
	return nil
}

// Sign signs the provided transaction.
func (s *Server) Sign(ctx context.Context, req core.SignRequest) (*types.Transaction, errors.Err) {
	tx, err := s.master.Request(ctx, req.Key, signRequest{Transaction: req.Transaction})
	if err != nil {
		return nil, errors.New(errors.ErrSignTransaction, err)
	}

	return tx.(*types.Transaction), nil
}

// Generates a new wallet, provisioning a key from the server.
func (s *Server) Generate(ctx context.Context, req core.GenerateRequest) errors.Err {
	var privateKey *ecdsa.PrivateKey
	select {
		case pkIndex, ok := <-s.available:
			if ok {
					privateKey = s.pks[pkIndex]
					s.unavailable[req.Key] = pkIndex
			} else {
					return errors.New(errors.ErrGenerateWallet, err.New("Internal service error."))
			}
		default:
			errors.New(errors.ErrGenerateWallet, err.New("You have hit your maximum concurrency limit."))
	}
	if _, err := s.master.Request(ctx, req.Key, generateRequest{Context: ctx, PrivateKey: privateKey}); err != nil {
		return errors.New(errors.ErrGenerateWallet, err)
	}

	return nil
}

// Remove the key's wallet and it's associated resources.
func (s *Server) Remove(ctx context.Context, req core.RemoveRequest) errors.Err {
	s.available <- s.unavailable[req.Key]
	delete(s.unavailable, req.Key)
	if err := s.master.Destroy(ctx, req.Key); err != nil {
		return errors.New(errors.ErrRemoveWallet, err)
	}

	return nil
}
