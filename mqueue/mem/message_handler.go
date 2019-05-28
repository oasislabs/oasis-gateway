package mem

import (
	"context"

	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/mqueue/core"
)

const (
	maxElementsPerQueue = 1024
)

type insertRequest struct {
	Element core.Element
}

type retrieveRequest struct {
	Offset uint64
	Count  uint
}

type discardRequest struct {
	Offset uint64
}

type nextRequest struct{}

// RequestHandler implements a very simple messaging queue-like
// functionality serving requests for a single queue.
type RequestHandler struct {
	key    string
	window SlidingWindow
}

// NewRequestHandler creates a new instance of a worker
func NewRequestHandler(key string) *RequestHandler {
	w := &RequestHandler{
		key:    key,
		window: NewSlidingWindow(SlidingWindowProps{MaxSize: maxElementsPerQueue}),
	}

	return w
}

func (w *RequestHandler) handle(ctx context.Context, ev conc.WorkerEvent) (interface{}, error) {
	switch ev := ev.(type) {
	case conc.RequestWorkerEvent:
		return w.handleRequestEvent(ctx, ev)
	case conc.ErrorWorkerEvent:
		return w.handleErrorEvent(ctx, ev)
	default:
		panic("receive unexpected event type")
	}
}

func (w *RequestHandler) handleRequestEvent(ctx context.Context, ev conc.RequestWorkerEvent) (interface{}, error) {
	switch req := ev.Value.(type) {
	case insertRequest:
		err := w.insert(req)
		return nil, err
	case retrieveRequest:
		return w.retrieve(req)
	case discardRequest:
		err := w.discard(req)
		return nil, err
	case nextRequest:
		return w.next(req)
	default:
		panic("invalid request received for worker")
	}
}

func (w *RequestHandler) handleErrorEvent(ctx context.Context, ev conc.ErrorWorkerEvent) (interface{}, error) {
	// a worker should not be passing errors to the conc.Worker so
	// in that case the error is returned and the execution of the
	// worker should halt
	return nil, ev.Error
}

func (w *RequestHandler) insert(req insertRequest) error {
	return w.window.Set(req.Element.Offset, req.Element.Value)
}

func (w *RequestHandler) retrieve(req retrieveRequest) (core.Elements, error) {
	return w.window.Get(req.Offset, req.Count)
}

func (w *RequestHandler) discard(req discardRequest) error {
	_, err := w.window.Slide(req.Offset)
	return err
}

func (w *RequestHandler) next(req nextRequest) (uint64, error) {
	return w.window.ReserveNext()
}
