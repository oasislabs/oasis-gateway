package mem

import (
	"context"
	"errors"
	"sync"

	"github.com/oasislabs/developer-gateway/mqueue/core"
)

type insertWorkerRequest struct {
	Key     string
	Element core.Element
	Out     chan<- error
}

type retrieveWorkerRequest struct {
	Key    string
	Offset uint64
	Count  uint
	Out    chan<- []*core.Element
}

type discardWorkerRequest struct {
	Key    string
	Offset uint64
	Out    chan<- error
}

type nextWorkerRequest struct {
	Key string
	Out chan<- uint64
}

type Server struct {
	ctx     context.Context
	wg      sync.WaitGroup
	doneCh  chan string
	inCh    chan interface{}
	workers map[string]*Worker
}

func NewServer(ctx context.Context) *Server {
	s := &Server{
		ctx:     ctx,
		wg:      sync.WaitGroup{},
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
		req.Out <- nil
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
		req.Out <- errors.New("attempt to discard queue that does not exist")
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
func (s *Server) Insert(key string, element core.Element) error {
	out := make(chan error)
	s.inCh <- insertWorkerRequest{Key: key, Element: element, Out: out}
	<-out
	return nil
}

// Retrieve all available elements from the
// messaging queue after the provided offset
func (s *Server) Retrieve(key string, offset uint64, count uint) ([]*core.Element, error) {
	out := make(chan []*core.Element)
	s.inCh <- retrieveWorkerRequest{Key: key, Offset: offset, Count: count, Out: out}
	return <-out, nil
}

// Discard all elements that have a prior or equal
// offset to the provided offset
func (s *Server) Discard(key string, offset uint64) error {
	out := make(chan error)
	s.inCh <- discardWorkerRequest{Key: key, Offset: offset, Out: out}
	<-out
	return nil
}

// Next element offset that can be used for the queue.
func (s *Server) Next(key string) (uint64, error) {
	out := make(chan uint64)
	s.inCh <- nextWorkerRequest{Key: key, Out: out}
	return <-out, nil
}
