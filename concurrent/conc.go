package concurrent

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	stderr "github.com/pkg/errors"
)

// ErrNoOccurrence is returned failing to wait for an
// event that never occurred
type ErrNoOccurrence struct{}

func (e ErrNoOccurrence) Error() string {
	return "the expected event never occurred"
}

// ErrCannotRecover is an error that can be passed by clients to
// retry mechanisms so that the attempted action is not retried
type ErrCannotRecover struct {
	Cause error
}

// Error implementation of error for ErrCannotRecover
func (e ErrCannotRecover) Error() string {
	return e.Cause.Error()
}

// ErrMaxAttemptsReached is an error that is returned after attempting
// an action multiple times with failures
type ErrMaxAttemptsReached struct {
	Causes []error
}

// Error implementation of error for ErrCannotRecover
func (e ErrMaxAttemptsReached) Error() string {
	return fmt.Sprintf("maximum number of attempts %d reached", len(e.Causes))
}

const (
	defaultConcurrency     uint8         = 2
	defaultBaseTimeout     time.Duration = 100 * time.Millisecond
	defaultBaseExp         uint8         = 2
	defaultMaxRetryTimeout time.Duration = 10 * time.Second
	defaultAttempts        uint8         = 10
)

var RandomConfig = RetryConfig{
	BaseTimeout:     defaultBaseTimeout,
	BaseExp:         defaultBaseExp,
	MaxRetryTimeout: defaultMaxRetryTimeout,
	Attempts:        defaultAttempts,
	Random:          true,
}

// Supplier is an interface for a type that provides a value. It is
// useful to abstract any operation into a generic method that can
// be run by Retry or Batch without knowing any specifics of what
// the Supplier actually does. The preferred method to use it
// might be through SupplierFunc with a closure
type Supplier interface {
	// Supply executes the operation the supplier is expected to perform
	// and returns the value and error related with the operation
	Supply() (interface{}, error)
}

// SupplierFunc is a type that implements Supplier that allows
// functions and closures to be passed as a Supplier.
type SupplierFunc func() (interface{}, error)

// Supply is the implementation of Supplier by calling the method
// itself
func (s SupplierFunc) Supply() (interface{}, error) {
	return s()
}

// RetryConfig is the configuration parameters for the Retry
// concurrent utility. Look at RetryWithConfig for more information
type RetryConfig struct {
	// Random sets the retry to wait a random time based on the
	// exponential back off
	Random bool

	// UnlimitedAttempts when set to true, Attempts will be ignored
	// and the action will be retried until it succeeds or the context
	// stops
	UnlimitedAttempts bool

	// Attempts is the maximum number of attempts allowed by a
	// Retry operation
	Attempts uint8

	// BaseExp is the base exponent for the calculation of the next
	// time an attempt must be triggered using exponential backoff
	BaseExp uint8

	// BaseTimeout is the initial timeout used after the first
	// attempt fails
	BaseTimeout time.Duration

	// MaxRetryTimeout sets an upper bound into the time that
	// the retry will wait until attempting an operation again.
	MaxRetryTimeout time.Duration
}

// RetryWithConfig is an implementation of an exponential back off
// retry operation for a supplier. It keeps retrying the operation
// until the maximum number of attempts has been reached, in which
// case it returns the associated error, or until it succeeds.
func RetryWithConfig(
	ctx context.Context,
	supplier Supplier,
	config RetryConfig,
) (interface{}, error) {
	var errs []error
	timeout := config.BaseTimeout.Nanoseconds()
	exp := int64(config.BaseExp)
	maxTimeout := config.MaxRetryTimeout.Nanoseconds()
	attempts := 0
	maxAttempts := int(config.Attempts)
	timer := time.NewTimer(0 * time.Second)

	if config.UnlimitedAttempts {
		maxAttempts = -1
	}

	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, stderr.WithStack(context.Canceled)

		case <-timer.C:
			v, err := supplier.Supply()
			if err == nil {
				return v, nil
			}

			if err, ok := err.(ErrCannotRecover); ok {
				return nil, stderr.WithStack(err.Cause)
			}

			errs = append(errs, err)
		}

		attempts++
		if attempts >= maxAttempts && maxAttempts >= 0 {
			return nil, ErrMaxAttemptsReached{Causes: errs}
		}

		timeout = (timeout * exp)
		multiplier := rand.Float64() + 0.5
		if timeout > maxTimeout {
			timeout = maxTimeout
			multiplier = rand.Float64() + 1
		}
		if config.Random {
			timeout = int64(multiplier*float64(timeout)) + 1
		}
		timer.Reset(time.Duration(timeout))
	}
}

// Retry is the same operation as RetryWithConfig but in this
// case the default values for RetryConfig are used
func Retry(ctx context.Context, supplier Supplier) (interface{}, error) {
	return RetryWithConfig(ctx, supplier, RetryConfig{
		BaseTimeout:     defaultBaseTimeout,
		BaseExp:         defaultBaseExp,
		MaxRetryTimeout: defaultMaxRetryTimeout,
		Attempts:        defaultAttempts,
	})
}

// RetryRandom is the same operation as RetryWithConfig but in this
// case the default values for RetryConfig are used with random
// exponential backoff
func RetryRandom(ctx context.Context, supplier Supplier) (interface{}, error) {
	return RetryWithConfig(ctx, supplier, RandomConfig)
}

// PoolConfig is the configuration for a PoolRunner
type PoolConfig struct {
	// Concurrency is the number of Suppliers that the pool can
	// run in parallel at most
	Concurrency uint8
}

// PoolRunner has a fixed number of goroutines that are used to run
// an arbitrary number of tasks. A PoolRunner is a convenient way to
// execute multiple operations in parallel having control on how
// many go routines are run in the system.
type PoolRunner struct {
	config  PoolConfig
	wg      sync.WaitGroup
	argCh   []chan argument
	counter uint64
}

// NewPoolRunner creates and starts a new PoolRunner with the
// default configuration parameters
func NewPoolRunner(ctx context.Context) *PoolRunner {
	return NewPoolRunnerWithConfig(ctx, PoolConfig{
		Concurrency: defaultConcurrency,
	})
}

// NewPoolRunnerWithConfig creates a new PoolRunner with the specified
// configuration
func NewPoolRunnerWithConfig(ctx context.Context, config PoolConfig) *PoolRunner {
	if config.Concurrency == 0 {
		config.Concurrency = defaultConcurrency
	}

	runner := PoolRunner{
		config: config,
		argCh:  make([]chan argument, config.Concurrency),
	}

	runner.wg.Add(int(config.Concurrency))
	for i := 0; i < int(config.Concurrency); i++ {
		runner.argCh[i] = make(chan argument, 128)
		go runner.run(ctx, runner.argCh[i])
	}

	return &runner
}

func (r *PoolRunner) run(ctx context.Context, inCh <-chan argument) {
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case arg, ok := <-inCh:
				if !ok {
					return
				}

				s := arg.Supplier
				arg.TimeExecuted = time.Now().UnixNano()
				v, err := s.Supply()
				arg.TimeDone = time.Now().UnixNano()

				if arg.Out != nil {
					arg.Out <- Result{
						Result:        v,
						Err:           err,
						Index:         arg.Index,
						TimeSubmitted: arg.TimeSubmitted,
						TimeExecuted:  arg.TimeExecuted,
						TimeDone:      arg.TimeDone,
					}
				}
			}
		}
	}()
}

// Run runs the provided supplier and returns the result in the
// out channel
func (r *PoolRunner) Run(out chan<- Result, supplier Supplier) {
	if out == nil {
		panic("channel cannot be nil")
	}

	current := atomic.AddUint64(&r.counter, 1)
	index := r.counter % uint64(r.config.Concurrency)
	r.argCh[index] <- argument{
		Out:           out,
		Supplier:      supplier,
		Index:         current,
		TimeSubmitted: time.Now().UnixNano(),
	}
}

// RunAndDiscard runs the provided supplier and discards the result. This may
// be used if an action needs to be executed that does some logging on
// failure for example
func (r *PoolRunner) RunAndDiscard(supplier Supplier) {
	current := atomic.AddUint64(&r.counter, 1)
	index := r.counter % uint64(r.config.Concurrency)
	r.argCh[index] <- argument{
		Out:           nil,
		Supplier:      supplier,
		Index:         current,
		TimeSubmitted: time.Now().UnixNano(),
	}
}

// Stop orderly stops all the goroutines in the PoolRunner
// and returns once all the goroutines have exited
func (r *PoolRunner) Stop() {
	for _, ch := range r.argCh {
		close(ch)
	}

	r.wg.Wait()
}

// BatchRunner is a runner of Suppliers that executes a batch of suppliers
// until all of them complete and return the result in the order in which
// the suppliers were provided. This is useful to execute a set of operations
// as a block, and there's a need to wait for the whole results of the block
// before moving forward. An alternative approach that may work better even
// though the interface would be slightly more complicated would be with
// a sliding window.
type BatchRunner struct {
	config BatchConfig
	wg     sync.WaitGroup
	argCh  []chan argument
	resCh  chan Result
}

// NewBatchRunner creates a new BatchRunner. Same as NewBatchRunnerConfig
// but using default BatchConfig values
func NewBatchRunner(ctx context.Context) *BatchRunner {
	return NewBatchRunnerWithConfig(ctx, BatchConfig{Concurrency: defaultConcurrency})
}

// NewBatchRunnerWithConfig creates a new BatchRunner using the configuration
// parameters provided in BatchConfig. If the ctx is Done the
// BatchRunner execution stops
func NewBatchRunnerWithConfig(
	ctx context.Context,
	config BatchConfig,
) *BatchRunner {
	if config.Concurrency == 0 {
		config.Concurrency = defaultConcurrency
	}

	runner := BatchRunner{
		config: config,
		argCh:  make([]chan argument, config.Concurrency),
		resCh:  make(chan Result, 64),
	}

	runner.wg.Add(int(config.Concurrency))
	for i := 0; i < int(config.Concurrency); i++ {
		runner.argCh[i] = make(chan argument, 64)
		go runner.run(ctx, runner.argCh[i])
	}

	return &runner
}

type argument struct {
	Index         uint64
	TimeSubmitted int64
	TimeExecuted  int64
	TimeDone      int64
	Out           chan<- Result
	Supplier      Supplier
}

// Result is the result of a Batch operation.
type Result struct {
	// Result is the value returned by the Supplier.Supply() call if any
	Result interface{}

	// Err is the err returned by the Supplier.Supply() call if any
	Err error

	// TimeSubmitted is the time at which the supplier was first submitted
	// to a worker to be processed
	TimeSubmitted int64

	// TimeExecuted is the time at which the supplier was executed by
	// a worker
	TimeExecuted int64

	// TimeDone is the time at which the supplier was done
	TimeDone int64

	// Index of the result within the submitted batch
	Index uint64
}

// Stop stops the execution of the BatchRunner and all the goroutines
// associated with it
func (r *BatchRunner) Stop() {
	for _, ch := range r.argCh {
		close(ch)
	}

	r.wg.Wait()
	close(r.resCh)
}

func (r *BatchRunner) run(
	ctx context.Context,
	inCh <-chan argument,
) {
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case arg, ok := <-inCh:
				if !ok {
					return
				}

				s := arg.Supplier
				arg.TimeExecuted = time.Now().UnixNano()
				v, err := s.Supply()
				arg.TimeDone = time.Now().UnixNano()
				arg.Out <- Result{
					Result:        v,
					Err:           err,
					Index:         arg.Index,
					TimeSubmitted: arg.TimeSubmitted,
					TimeExecuted:  arg.TimeExecuted,
					TimeDone:      arg.TimeDone,
				}
			}
		}
	}()
}

// Run executes the block of Suppliers as a batch and returns the
// result once all of them have completed
func (r *BatchRunner) Run(s []Supplier) []Result {
	result := make([]Result, len(s))
	length := len(s)

	go func() {
		// needs to run in a different goroutine because the batch may
		// be too big to fit all in the channel and would block the main
		// goroutine
		for i := 0; i < length; i++ {
			index := i % int(r.config.Concurrency)
			r.argCh[index] <- argument{
				Out:           r.resCh,
				Supplier:      s[i],
				Index:         uint64(i),
				TimeSubmitted: time.Now().UnixNano(),
			}
		}
	}()

	counter := 0
	for res := range r.resCh {
		result[res.Index] = res
		counter++
		if counter == len(s) {
			break
		}
	}

	return result
}

// BatchConfig is the configuration for a Batch* function
// or BatchRunner
type BatchConfig struct {
	// Concurrency is the number of Suppliers that will be
	// run in parallel
	Concurrency uint8
}

// Batch executes a block of Suppliers as a block using a BatchRunner
// as the underlying implementation.
func Batch(ctx context.Context, supplier []Supplier) []Result {
	return BatchWithConfig(ctx, supplier, BatchConfig{
		Concurrency: defaultConcurrency,
	})
}

// BatchWithConfig executes a block of Suppliers as a block using a BatchRunner
// as the underlying implementation.
func BatchWithConfig(
	ctx context.Context,
	supplier []Supplier,
	config BatchConfig,
) []Result {
	runner := NewBatchRunnerWithConfig(ctx, config)
	defer runner.Stop()
	return runner.Run(supplier)
}
