package mem

import (
	"context"
	"sync"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/mqueue/core"
)

type insertWorkerRequest struct {
	Key     string
	Element core.Element
	Out     chan<- errors.Err
}

type retrieveWorkerRequest struct {
	Key    string
	Offset uint64
	Count  uint
	Out    chan<- retrieveResponse
}

type discardWorkerRequest struct {
	Key    string
	Offset uint64
	Out    chan<- errors.Err
}

type nextWorkerRequest struct {
	Key string
	Out chan<- nextResponse
}

type Server struct {
	ctx     context.Context
	wg      sync.WaitGroup
	logger  log.Logger
	doneCh  chan string
	inCh    chan interface{}
	workers map[string]*Worker
}

func NewServer(ctx context.Context, logger log.Logger) *Server {
	s := &Server{
		ctx:     ctx,
		wg:      sync.WaitGroup{},
		logger:  logger.ForClass("mqueue/mem", "Server"),
		doneCh:  make(chan string),
		inCh:    make(chan interface{}, 64),
		workers: make(map[string]*Worker),
	}

	s.startLoop()
	return s
}

func (s *Server) Stop() {
	close(s.inCh)
	s.wg.Wait()
}

func (s *Server) startLoop() {
	s.wg.Add(1)

	go func() {
		defer func() {
			s.wg.Done()
		}()

		for {
			select {
			case <-s.ctx.Done():
				return
			case key := <-s.doneCh:
				s.removeWorker(key)
			case arg, ok := <-s.inCh:
				if !ok {
					return
				}

				s.serveRequest(arg)
			}
		}
	}()
}

func (s *Server) removeWorker(key string) {
	w, ok := s.workers[key]
	if !ok {
		s.logger.Warn(s.ctx, "attempt remove worker that is not present", log.MapFields{
			"call_type": "RemoveWorkerFailure",
			"key":       "key",
		})
		return
	}

	w.Stop()
	delete(s.workers, key)
}

func (s *Server) serveRequest(req interface{}) {
	switch req := req.(type) {
	case insertWorkerRequest:
		s.insert(req)
	case retrieveWorkerRequest:
		s.retrieve(req)
	case discardWorkerRequest:
		s.discard(req)
	case nextWorkerRequest:
		s.next(req)
	default:
		panic("invalid request received for worker")
	}
}

func (s *Server) insert(req insertWorkerRequest) {
	worker, ok := s.workers[req.Key]
	if !ok {
		worker = NewWorker(s.ctx, req.Key, s.doneCh)
		s.workers[req.Key] = worker
	}

	worker.Insert(insertRequest{
		Element: req.Element,
		Out:     req.Out,
	})
}

func (s *Server) retrieve(req retrieveWorkerRequest) {
	worker, ok := s.workers[req.Key]
	if !ok {
		req.Out <- retrieveResponse{Elements: core.Elements{Offset: 0, Elements: nil},
			Error: nil,
		}
		return
	}

	worker.Retrieve(retrieveRequest{
		Offset: req.Offset,
		Count:  req.Count,
		Out:    req.Out,
	})
}

func (s *Server) discard(req discardWorkerRequest) {
	worker, ok := s.workers[req.Key]
	if !ok {
		req.Out <- errors.New(errors.ErrQueueDiscardNotExists, nil)
		return
	}

	worker.Discard(discardRequest{
		Offset: req.Offset,
		Out:    req.Out,
	})
}

func (s *Server) next(req nextWorkerRequest) {
	worker, ok := s.workers[req.Key]
	if !ok {
		worker = NewWorker(s.ctx, req.Key, s.doneCh)
		s.workers[req.Key] = worker
	}

	worker.Next(nextRequest{
		Out: req.Out,
	})
}

// Insert inserts the element to the provided offset.
func (s *Server) Insert(key string, element core.Element) errors.Err {
	out := make(chan errors.Err)
	s.inCh <- insertWorkerRequest{Key: key, Element: element, Out: out}
	return <-out
}

// Retrieve all available elements from the
// messaging queue after the provided offset
func (s *Server) Retrieve(key string, offset uint64, count uint) (core.Elements, errors.Err) {
	out := make(chan retrieveResponse)
	s.inCh <- retrieveWorkerRequest{Key: key, Offset: offset, Count: count, Out: out}
	res := <-out
	return res.Elements, res.Error
}

// Discard all elements that have a prior or equal
// offset to the provided offset
func (s *Server) Discard(key string, offset uint64) errors.Err {
	out := make(chan errors.Err)
	s.inCh <- discardWorkerRequest{Key: key, Offset: offset, Out: out}
	<-out
	return nil
}

// Next element offset that can be used for the queue.
func (s *Server) Next(key string) (uint64, errors.Err) {
	out := make(chan nextResponse)
	s.inCh <- nextWorkerRequest{Key: key, Out: out}
	res := <-out
	return res.Offset, res.Error
}
