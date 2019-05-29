package tx

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/wallet/core"
)

const maxInactivityTimeout = time.Duration(10) * time.Minute

type Server struct {
	master *conc.Master
	logger log.Logger
}

func NewServer(ctx context.Context, logger log.Logger) *Server {
	s := &Server{
		logger: logger.ForClass("wallet/tx", "Server"),
	}

	s.master = conc.NewMaster(conc.MasterProps{
		MasterHandler:         conc.MasterHandlerFunc(s.handle),
		CreateWorkerOnRequest: true,
	})

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
	worker := NewWorker(ev.Key)

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

func (s *Server) Generate(ctx context.Context, req core.GenerateRequest) errors.Err {
	if _, err := s.master.Request(ctx, req.Key, generateRequest{Context: ctx, URL: req.URL, PrivateKey: req.PrivateKey}); err != nil {
		return errors.New(errors.ErrGenerateWallet, err)
	}

	return nil
}

// Remove the key's wallet and it's associated resources
func (s *Server) Remove(ctx context.Context, req core.RemoveRequest) errors.Err {
	if err := s.master.Destroy(ctx, req.Key); err != nil {
		return errors.New(errors.ErrRemoveWallet, err)
	}

	return nil
}
