package conc

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

const (
	stopped  = 0
	started  = 1
	stopping = 2
)

func errorFromPanic(r interface{}) error {
	stacktrace := debug.Stack()

	switch x := r.(type) {
	case string:
		return fmt.Errorf("panic error %s at %s", x, string(stacktrace))
	case error:
		return fmt.Errorf("panic error %s at %s", x.Error(), string(stacktrace))
	default:
		return fmt.Errorf("unknown panic %+v at %s", r, string(stacktrace))
	}
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
}

// MasterEvent is the interface implemented by all events triggered
// by the master and handled for a MasterHandler
type MasterEvent interface {
	WorkerKey() string
}

// CreateWorkerEvent is triggered by a master when a new worker
// is created and available to be sent events to
type CreateWorkerEvent struct {
	Context context.Context
	Key     string
	Value   interface{}
	Props   *CreateWorkerProps
}

// WorkerKey implementation of MasterEvent for CreateWorkerEvent
func (e CreateWorkerEvent) WorkerKey() string {
	return e.Key
}

// DestroyWorkerEvent is triggered by a master when an existing worker
// is destroyed
type DestroyWorkerEvent struct {
	Context context.Context
	Worker  *Worker
	Key     string
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

	// UserData is data that the user can attach to the worker in case any
	// external context is required
	UserData interface{}
}

// Worker handles requests issued by the master in a separate
// goroutine and gives back results. Its lifetime is managed
// by the Master
type Worker struct {
	// key is the string that uniquely identifies a worker
	key string

	// handler is the user defined handler for events that
	// a worker needs to handle
	handler WorkerHandler

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

// NewWorker creates a new worker instance
func NewWorker(ctx context.Context, props WorkerProps) *Worker {
	w := &Worker{
		key:     props.Key,
		handler: props.WorkerHandler,
		C:       props.C,

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
	var err error

	defer func() {
		if r := recover(); r != nil {
			err = errorFromPanic(r)
		}

		w.doneC <- workerDestroyed{Context: ctx, Key: w.key, Cause: err}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-w.ErrC:
			if !ok {
				return
			}

			err = w.handleError(err)
			if err != nil {
				return
			}

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
		Error:  err,
	})

	return err
}

func (w *Worker) handleRequest(req workerRequest) {
	defer func() {
		var err error
		if r := recover(); r != nil {
			err = errorFromPanic(r)
			req.Out <- response{Value: nil, Error: err}
		}
	}()

	if req.Key != w.key {
		panic("received request intended for another worker")
	}

	v, err := w.handler.Handle(req.Context, RequestWorkerEvent{
		Worker: w,
		Value:  req.Value,
	})

	req.Out <- response{Value: v, Error: err}
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

type response struct {
	Value interface{}
	Error error
}

type workerRequest struct {
	Context context.Context
	Key     string
	Value   interface{}
	Out     chan response
}

func (r workerRequest) GetContext() context.Context {
	return r.Context
}

func (r workerRequest) WorkerKey() string {
	return r.Key
}

type createRequest struct {
	Context context.Context
	Key     string
	Out     chan error
	Value   interface{}
}

func (r createRequest) GetContext() context.Context {
	return r.Context
}

func (r createRequest) WorkerKey() string {
	return r.Key
}

type destroyRequest struct {
	Context context.Context
	Key     string
	Out     chan response
}

func (r destroyRequest) GetContext() context.Context {
	return r.Context
}

func (r destroyRequest) WorkerKey() string {
	return r.Key
}

type existsRequest struct {
	Context context.Context
	Key     string
	Out     chan bool
}

func (r existsRequest) GetContext() context.Context {
	return r.Context
}

func (r existsRequest) WorkerKey() string {
	return r.Key
}

type request interface {
	GetContext() context.Context
	WorkerKey() string
}

// Master manages a set of workers and distributes workers
// amongst them. It also keeps track of the workers lifetimes
type Master struct {
	// shutdownCh is the channel used by the Master to signal
	// a shutdown to itself
	shutdownCh chan interface{}

	// doneCh is the channel used by workers to notify to the
	// Master that their lifetime has ended
	doneCh chan workerDestroyed

	// workerCount to keep track of the number of workers and
	// ensure a graceful shutdown of the master and its workers
	workerCount sync.WaitGroup

	// communication channels that the master has with the
	// workers. The master uses the channels as write only and
	// the workers use the channels as read only. The workers
	// communicate back with the master with a channel created
	// on a per request basis
	workers map[string]*Worker

	// shutdownWorkers are the workers that are shutting down
	// and we are waiting for a doneCh event
	shutdownWorkers map[string]*Worker

	// state keeps track of whether the master is running. It
	// needs to be accessed in a thread safe manner.
	state uint32

	// inCh is the channel used by the master to pass on requests
	// from external goroutines to the event loop
	inCh chan request

	// handler is the user defined handler for events that
	// need to be handled by the master
	handler MasterHandler

	// ctx is the context that the master uses for the duration
	// of its Start-Stop span
	ctx context.Context

	// Error is set in case of exiting with an error
	Error error
}

// MasterHandler is the user defined handler to handle events
// for the master
type MasterHandler interface {
	Handle(ctx context.Context, ev MasterEvent) error
}

// WorkerHandler is the user defined handler to handle events
// targeting a worker
type WorkerHandler interface {
	Handle(ctx context.Context, req WorkerEvent) (interface{}, error)
}

// MasterHandlerFunc is the implementation of MasterHandler for functions
type MasterHandlerFunc func(ctx context.Context, ev MasterEvent) error

// Handle implementation of MasterHandler for MasterHandlerFunc
func (f MasterHandlerFunc) Handle(ctx context.Context, ev MasterEvent) error {
	return f(ctx, ev)
}

// WorkerHandlerFunc is the implementation of MasterHandler for functions
type WorkerHandlerFunc func(ctx context.Context, ev WorkerEvent) (interface{}, error)

// Handle implementation of WorkerHandler for WorkerHandlerFunc
func (f WorkerHandlerFunc) Handle(ctx context.Context, ev WorkerEvent) (interface{}, error) {
	return f(ctx, ev)
}

// MasterProps are the properties used by the master to define
// its behaviour and that of its workers
type MasterProps struct {
	// MasterHandler is the handler the master will use to provide access
	// to the master events
	MasterHandler MasterHandler
}

// NewMaster creates a new master
func NewMaster(props MasterProps) *Master {
	return &Master{
		handler:         props.MasterHandler,
		workers:         make(map[string]*Worker),
		shutdownWorkers: make(map[string]*Worker),
		state:           stopped,
	}
}

// IsStopped returns true if the master is not running
func (m *Master) IsStopped() bool {
	return atomic.LoadUint32(&m.state) == stopped
}

// Start the master
func (m *Master) Start(ctx context.Context) error {
	ok := atomic.CompareAndSwapUint32(&m.state, 0, 1)
	if !ok {
		return errors.New("master is not stopped")
	}

	m.doneCh = make(chan workerDestroyed, 64)
	m.shutdownCh = make(chan interface{})
	m.inCh = make(chan request)

	go m.startLoop(ctx)
	return nil
}

// Stop the master and shutdown all the workers that are still running.
// This method blocks until all the workers have exited
func (m *Master) Stop() error {
	ok := atomic.CompareAndSwapUint32(&m.state, started, stopping)
	if !ok {
		return errors.New("master is not started")
	}

	close(m.shutdownCh)
	m.workerCount.Wait()

	close(m.inCh)
	close(m.doneCh)
	if len(m.workers) > 0 {
		panic("failed to shutdown all workers gracefully")
	}

	ok = atomic.CompareAndSwapUint32(&m.state, stopping, stopped)
	if !ok {
		panic("concurrency error in transition to stopped")
	}

	return nil
}

// Create a new worker
func (m *Master) Create(ctx context.Context, key string, value interface{}) error {
	out := make(chan error)
	m.inCh <- createRequest{Context: ctx, Key: key, Out: out, Value: value}
	return <-out
}

// Destroy an existing worker
func (m *Master) Destroy(ctx context.Context, key string) error {
	out := make(chan response)
	m.inCh <- destroyRequest{Context: ctx, Key: key, Out: out}
	res := <-out
	if res.Error != nil {
		return res.Error
	}

	// wait for the worker to destroy
	c := res.Value.(<-chan error)
	err, ok := <-c
	if ok && err != nil {
		return err
	}

	return nil
}

// Exists returns true if the worker exists, false otherwise
func (m *Master) Exists(ctx context.Context, key string) bool {
	out := make(chan bool)
	m.inCh <- existsRequest{Context: ctx, Key: key, Out: out}
	return <-out
}

// Request sends a request to a specific worker and returns back
// the response
func (m *Master) Request(ctx context.Context, key string, req interface{}) (interface{}, error) {
	out := make(chan response)
	m.inCh <- workerRequest{Context: ctx, Key: key, Value: req, Out: out}
	res := <-out
	return res.Value, res.Error
}

// shutdown closes all the workers and frees the resources
// they are using. This method should only be called outside
// the event loop
func (m *Master) shutdown() {
	go func() {
		for ev := range m.doneCh {
			m.removeWorker(ev)
		}
	}()

	for key := range m.workers {
		m.shutdownWorker(key)
	}

	m.workerCount.Wait()
}

func (m *Master) shutdownWorker(key string) (<-chan error, bool) {
	w, ok := m.workers[key]
	if !ok {
		return nil, false
	}

	// remove the worker from the set of active workers and move it to the
	// set of workers which are being shutdown
	delete(m.workers, key)
	m.shutdownWorkers[key] = w
	close(w.C)
	return w.ShutdownC, true
}

func (m *Master) startLoop(ctx context.Context) {
	defer func() {
		m.shutdown()
	}()

	m.ctx = ctx
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdownCh:
			return
		case ev, ok := <-m.doneCh:
			if !ok {
				return
			}

			m.removeWorker(ev)
		case req, ok := <-m.inCh:
			if !ok {
				return
			}

			m.handleRequest(req)
		}
	}
}

func (m *Master) handleRequest(req request) {
	switch req := req.(type) {
	case workerRequest:
		m.handleWorkerRequest(req)
	case createRequest:
		m.handleCreateRequest(req)
	case destroyRequest:
		m.handleDestroyRequest(req)
	case existsRequest:
		m.handleExistsRequest(req)
	default:
		panic("received unexpected request")
	}
}

func (m *Master) handleWorkerRequest(req workerRequest) {
	w, ok := m.workers[req.Key]
	if !ok {
		req.Out <- response{Value: nil, Error: errors.New("worker does not exist")}
		return
	}

	w.C <- req
}

func (m *Master) handleCreateRequest(req createRequest) {
	defer func() {
		if r := recover(); r != nil {
			err := errorFromPanic(r)
			req.Out <- err
		}
	}()

	_, ok := m.workers[req.Key]
	if ok {
		req.Out <- errors.New("worker already exists")
		return
	}

	var props CreateWorkerProps
	err := m.handler.Handle(req.Context, CreateWorkerEvent{
		Context: req.Context,
		Value:   req.Value,
		Key:     req.Key,
		Props:   &props,
	})
	if err != nil {
		req.Out <- err
		return
	}

	ch := make(chan workerRequest, 64)
	m.workers[req.Key] = NewWorker(m.ctx, WorkerProps{
		Key:           req.Key,
		DoneC:         m.doneCh,
		WorkerHandler: props.WorkerHandler,
		UserData:      props.UserData,
		ErrC:          props.ErrC,
		C:             ch,
	})

	m.workerCount.Add(1)
	req.Out <- nil
}

func (m *Master) handleDestroyRequest(req destroyRequest) {
	defer func() {
		if r := recover(); r != nil {
			err := errorFromPanic(r)
			req.Out <- response{Error: err, Value: nil}
		}
	}()

	c, ok := m.shutdownWorker(req.Key)
	if !ok {
		req.Out <- response{Error: errors.New("worker does not exist"), Value: nil}
		return
	}

	req.Out <- response{Error: nil, Value: c}
}

func (m *Master) handleExistsRequest(req existsRequest) {
	_, ok := m.workers[req.Key]
	req.Out <- ok
}

func (m *Master) removeWorker(ev workerDestroyed) {
	var (
		ok  bool
		err error
		w   *Worker
	)

	defer func() {
		if r := recover(); r != nil {
			err = errorFromPanic(r)
		}

		if err != nil {
			// in case of an error raise it to the listener so it can be bubbled up
			// propertly
			w.ShutdownC <- err
		}

		close(w.ShutdownC)
	}()

	w, ok = m.shutdownWorkers[ev.Key]
	if !ok {
		return
	}

	delete(m.shutdownWorkers, ev.Key)
	m.workerCount.Done()

	err = m.handler.Handle(context.Background(), DestroyWorkerEvent{
		Context: ev.Context,
		Worker:  w,
		Key:     ev.Key,
	})
}
