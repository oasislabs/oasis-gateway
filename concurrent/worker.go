package concurrent

import (
	"context"
	"sync/atomic"
	"time"
)

// Worker handles requests issued by the master in a separate
// goroutine and gives back results. Its lifetime is managed
// by the Master
type Worker struct {
	// lastEventTimestamp is the timestamp at which the worker handled
	// the latest event
	lastEventTimestamp int64

	// maxInactivity is the maximum time the worker is allowed to exist
	// without serving any request. When this time expires the worker
	// should destroy itself
	maxInactivity time.Duration

	// key is the string that uniquely identifies a worker
	key string

	// handler is the user defined handler for events that
	// a worker needs to handle
	handler WorkerHandler

	// SharedC is the shared channel between the master and the worker
	// for requests met by the worker who is available
	SharedC <-chan executeRequest

	// C is the channel the worker only reads from
	C chan workerRequest

	// ShutdownC is a channel used by the worker to signal that it
	// has been completely shutdown and removed
	ShutdownC chan error

	// ErrC is an error channel the worker can listen to and report errors
	// though events in case they happen
	ErrC <-chan error

	// doneC is a write once channel the worker uses to notify the master
	// that the worker has exited
	doneC chan<- workerDestroyed

	// UserData is data that the user can attach to the worker in case any
	// external context is required
	UserData interface{}
}

// WorkerEvent is the interface defined for events that the worker emits
type WorkerEvent interface {
	GetWorker() *Worker
}

// ErrorWorkerEvent is emitted by the worker when an event on the
// error channel is received
type ErrorWorkerEvent struct {
	Worker *Worker
	Error  error
}

// GetWorker implementation of WorkerEvent for ErrorWorkervent
func (e ErrorWorkerEvent) GetWorker() *Worker {
	return e.Worker
}

// RequestWorkerEvent is emitted by the worker when a request
// is received by the worker
type RequestWorkerEvent struct {
	Worker *Worker
	Value  interface{}
}

// GetWorker implementation of WorkerEvent for ErrorWorkerEvent
func (e RequestWorkerEvent) GetWorker() *Worker {
	return e.Worker
}

// WorkerHandler is the user defined handler to handle events
// targeting a worker
type WorkerHandler interface {
	Handle(ctx context.Context, req WorkerEvent) (interface{}, error)
}

// WorkerHandlerFunc is the implementation of MasterHandler for functions
type WorkerHandlerFunc func(ctx context.Context, ev WorkerEvent) (interface{}, error)

// Handle implementation of WorkerHandler for WorkerHandlerFunc
func (f WorkerHandlerFunc) Handle(ctx context.Context, ev WorkerEvent) (interface{}, error) {
	return f(ctx, ev)
}

// CreateWorkerProps is the place where a user defined MasterHandler can put
// the defined properties for a Worker on a CreateWorkerEvent
type CreateWorkerProps struct {
	// WorkerHandler is the handler used by the worker to handle
	// incoming requests
	WorkerHandler WorkerHandler

	// ErrC is an error channel the worker can listen to and report errors
	// though events in case they happen
	ErrC <-chan error

	// UserData is data that the user can attach to the worker in case any
	// external context is required
	UserData interface{}

	// MaxInactivity is the maximum time the worker is allowed to exist
	// without serving any request. When this time expires the worker
	// should destroy itself
	MaxInactivity time.Duration
}

// CreateWorkerEvent is triggered by a master when a new worker
// is created and available to be sent events to
type CreateWorkerEvent struct {
	Key   string
	Value interface{}
	Props *CreateWorkerProps
}

// WorkerKey implementation of MasterEvent for CreateWorkerEvent
func (e CreateWorkerEvent) WorkerKey() string {
	return e.Key
}

// DestroyWorkerEvent is triggered by a master when an existing worker
// is destroyed
type DestroyWorkerEvent struct {
	Worker *Worker
	Key    string
}

// WorkerKey implementation of MasterEvent for DestroyWorkerEvent
func (e DestroyWorkerEvent) WorkerKey() string {
	return e.Key
}

// WorkerProps are the properties used to construct a worker instance
type WorkerProps struct {
	// Key that uniquely identifies the worker
	Key string

	// DoneC is a write once channel the worker uses to notify the master
	// that the worker has exited
	DoneC chan<- workerDestroyed

	// WorkerHandler is the handler used by the worker to handle
	// incoming requests
	WorkerHandler WorkerHandler

	// C is the channel the worker gets requests from
	C chan workerRequest

	// ErrC is an error channel the worker can listen to and report errors
	// though events in case they happen
	ErrC <-chan error

	// SharedC is the shared channel between the master and the worker
	// for requests met by the worker who is available
	SharedC <-chan executeRequest

	// UserData is data that the user can attach to the worker in case any
	// external context is required
	UserData interface{}

	// MaxInactivity is the maximum time the worker is allowed to exist
	// without serving any request. When this time expires the worker
	// should destroy itself
	MaxInactivity time.Duration
}

// workerDestroyed is the event sent by a worker to the
// master to signal the end of the worker. If the worker
// was shutdown because of an error Cause may be set
type workerDestroyed struct {
	Context context.Context

	// Key uniquely identifies a worker
	Key string

	// Cause may be set by the worker if the condititions in
	// which it terminated were abnormal
	Cause error
}

type Response struct {
	Value interface{}
	Key   string
	Error error
}

type workerRequest struct {
	Context context.Context
	Key     string
	Value   interface{}
	Out     chan Response
	Count   *int32
}

type broadcastRequest struct {
	Context context.Context
	Value   interface{}
	Out     chan Response
}

type executeRequest struct {
	Context context.Context
	Value   interface{}
	Out     chan Response
}

func (r workerRequest) GetContext() context.Context {
	return r.Context
}

func (r broadcastRequest) GetContext() context.Context {
	return r.Context
}

func (r executeRequest) GetContext() context.Context {
	return r.Context
}

// NewWorker creates a new worker instance
func NewWorker(ctx context.Context, props WorkerProps) *Worker {
	if props.MaxInactivity == 0 {
		props.MaxInactivity = time.Duration(1) * time.Hour
	}

	w := &Worker{
		lastEventTimestamp: time.Now().Unix(),
		maxInactivity:      props.MaxInactivity,
		key:                props.Key,
		handler:            props.WorkerHandler,
		SharedC:            props.SharedC,
		C:                  props.C,

		// ShutdownC may be closed with an error if there are no listeners
		// for it. In that case we should not block
		ShutdownC: make(chan error, 2),
		ErrC:      props.ErrC,
		doneC:     props.DoneC,
		UserData:  props.UserData,
	}

	go w.startLoop(ctx)
	return w
}

func (w *Worker) startLoop(ctx context.Context) {
	timer := time.NewTimer(w.maxInactivity)
	var err error

	defer func() {
		timer.Stop()

		if r := recover(); r != nil {
			err = errorFromPanic(r)
		}

		w.doneC <- workerDestroyed{Context: ctx, Key: w.key, Cause: err}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			current := time.Now().Unix()
			if time.Duration(current-w.lastEventTimestamp) > w.maxInactivity {
				return

			} else {
				if ok := timer.Reset(w.maxInactivity); ok {
					panic("resetting timer when it was already running")
				}
			}
		case err, ok := <-w.ErrC:
			if !ok {
				return
			}

			err = w.handleError(err)
			if err != nil {
				return
			}

		case req, ok := <-w.SharedC:
			if !ok {
				return
			}

			w.handleExecute(req)
		case req, ok := <-w.C:
			if !ok {
				return
			}

			w.handleRequest(req)
		}
	}
}

func (w *Worker) handleError(req error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errorFromPanic(r)
		}
	}()

	// using the err defined in the context so that if the worker returns
	// that error will be reported when the worker is destroyed
	_, err = w.handler.Handle(context.Background(), ErrorWorkerEvent{
		Worker: w,
		Error:  req,
	})

	return err
}

func (w *Worker) handleExecute(req executeRequest) {
	count := int32(1)
	w.handleRequest(workerRequest{
		Context: req.Context,
		Key:     w.key,
		Value:   req.Value,
		Out:     req.Out,
		Count:   &count,
	})
}

func (w *Worker) processRequest(req workerRequest) Response {
	if req.Key != w.key {
		panic("received request intended for another worker")
	}

	v, err := w.handler.Handle(req.Context, RequestWorkerEvent{
		Worker: w,
		Value:  req.Value,
	})

	return Response{Value: v, Key: w.key, Error: err}
}

func (w *Worker) handleRequest(req workerRequest) {
	defer func() {
		var err error
		if r := recover(); r != nil {
			err = errorFromPanic(r)
			req.Out <- Response{Value: nil, Key: w.key, Error: err}
			if value := atomic.AddInt32(req.Count, -1); value == 0 {
				close(req.Out)
			}
		}
	}()

	req.Out <- w.processRequest(req)
	if value := atomic.AddInt32(req.Count, -1); value == 0 {
		close(req.Out)
	}
}
