package mem

import (
	"context"
	"time"

	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/oasislabs/oasis-gateway/stats"
)

const maxInactivityTimeout = time.Duration(10) * time.Minute

type Server struct {
	master *concurrent.Master
	logger log.Logger
}

type Services struct {
	Logger log.Logger
}

func NewServer(ctx context.Context, services Services) *Server {
	s := &Server{
		logger: services.Logger.ForClass("mqueue/mem", "Server"),
	}

	s.master = concurrent.NewMaster(concurrent.MasterProps{
		MasterHandler:         concurrent.MasterHandlerFunc(s.handle),
		CreateWorkerOnRequest: true,
	})

	if err := s.master.Start(ctx); err != nil {
		panic("failed to start master")
	}

	return s
}

func (m *Server) handle(ctx context.Context, ev concurrent.MasterEvent) error {
	switch ev := ev.(type) {
	case concurrent.CreateWorkerEvent:
		return m.create(ctx, ev)
	case concurrent.DestroyWorkerEvent:
		return m.destroy(ctx, ev)
	default:
		panic("received unknown request")
	}
}

func (s *Server) create(ctx context.Context, ev concurrent.CreateWorkerEvent) error {
	worker := NewMessageHandler(ev.Key)

	ev.Props.ErrC = nil
	ev.Props.WorkerHandler = concurrent.WorkerHandlerFunc(worker.handle)
	ev.Props.UserData = worker
	ev.Props.MaxInactivity = maxInactivityTimeout

	return nil
}

func (s *Server) destroy(ctx context.Context, ev concurrent.DestroyWorkerEvent) error {
	// nothing to do on a destroy to cleanup the worker
	return nil
}

// Insert inserts the element to the provided offset.
func (s *Server) Insert(ctx context.Context, req core.InsertRequest) error {
	_, err := s.master.Request(ctx, req.Key, insertRequest{Element: req.Element})
	return err
}

// Retrieve all available elements from the
// messaging queue after the provided offset
func (s *Server) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.Elements, error) {
	v, err := s.master.Request(ctx, req.Key, retrieveRequest{Offset: req.Offset, Count: req.Count})
	if err != nil {
		return core.Elements{}, err
	}

	return v.(core.Elements), nil
}

// Discard all elements that have a prior or equal
// offset to the provided offset
func (s *Server) Discard(ctx context.Context, req core.DiscardRequest) error {
	_, err := s.master.Request(ctx, req.Key, discardRequest{
		KeepPrevious: req.KeepPrevious,
		Count:        req.Count,
		Offset:       req.Offset,
	})
	return err
}

// Next element offset that can be used for the queue.
func (s *Server) Next(ctx context.Context, req core.NextRequest) (uint64, error) {
	v, err := s.master.Request(ctx, req.Key, nextRequest{})
	if err != nil {
		return 0, err
	}

	return v.(uint64), nil
}

// Remove the key's queue and it's associated resources
func (s *Server) Remove(ctx context.Context, req core.RemoveRequest) error {
	return s.master.Destroy(ctx, req.Key)
}

// Exists returns true if there is a queue allocated with the
// provided key
func (s *Server) Exists(ctx context.Context, req core.ExistsRequest) (bool, error) {
	return s.master.Exists(ctx, req.Key)
}

func (s *Server) Name() string {
	return "mqueue.mem.Server"
}

func (s *Server) Stats() stats.Metrics {
	return nil
}
