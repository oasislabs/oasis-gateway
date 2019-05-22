package mem

import (
	"context"
	stderr "errors"
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

type removeWorkerRequest struct {
	Key string
	Out chan<- removeWorkerResponse
}

type removeWorkerResponse struct {
	Error errors.Err
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
			case key, ok := <-s.doneCh:
				if !ok {
					s.logger.Debug(s.ctx, "done channel closed so server is exiting", log.MapFields{
						"call_type": "ServerLoopEndSuccess",
					})
					return
				}

				s.removeWorker(key)
			case arg, ok := <-s.inCh:
				if !ok {
					s.logger.Debug(s.ctx, "input channel closed so server is exiting", log.MapFields{
						"call_type": "ServerLoopSuccess",
					})
					return
				}

				s.serveRequest(arg)
			}
		}
	}()
}

func (s *Server) removeWorker(key string) {
	_, ok := s.workers[key]
	if !ok {
		s.logger.Warn(s.ctx, "attempt remove worker that is not present", log.MapFields{
			"call_type": "RemoveWorkerFailure",
			"key":       "key",
		})
		return
	}

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
	case removeWorkerRequest:
		s.remove(req)
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
		req.Out <- retrieveResponse{Elements: core.Elements{Offset: req.Offset, Elements: nil},
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

func (s *Server) remove(req removeWorkerRequest) {
	worker, ok := s.workers[req.Key]
	if !ok {
		err := errors.New(errors.ErrQueueNotFound, stderr.New("cannot remove worker that does not exist"))
		req.Out <- removeWorkerResponse{Error: err}
		return
	}

	worker.Stop()
	req.Out <- removeWorkerResponse{Error: nil}
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

// Remove the key's queue and it's associated resources
func (s *Server) Remove(key string) errors.Err {
	out := make(chan removeWorkerResponse)
	s.inCh <- removeWorkerRequest{Key: key, Out: out}
	res := <-out
	return res.Error
}
