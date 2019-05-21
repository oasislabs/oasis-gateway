package conc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryError(t *testing.T) {
	ctx := context.Background()
	runs := 0

	res, err := RetryWithConfig(ctx, SupplierFunc(func() (interface{}, error) {
		runs++
		return runs, errors.New("error")
	}), RetryConfig{
		Attempts:        10,
		BaseExp:         2,
		BaseTimeout:     1 * time.Millisecond,
		MaxRetryTimeout: 10 * time.Millisecond,
	})

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, runs, 10)
}

func TestRetry(t *testing.T) {
	ctx := context.Background()
	runs := 0

	res, err := RetryWithConfig(ctx, SupplierFunc(func() (interface{}, error) {
		runs++
		return 0, nil
	}), RetryConfig{
		Attempts:        10,
		BaseExp:         2,
		BaseTimeout:     1 * time.Millisecond,
		MaxRetryTimeout: 10 * time.Millisecond,
	})

	assert.Nil(t, err)
	assert.Equal(t, res, 0)
	assert.Equal(t, runs, 1)
}

func TestRetryWithSomeErrors(t *testing.T) {
	ctx := context.Background()
	runs := 0

	res, err := RetryWithConfig(ctx, SupplierFunc(func() (interface{}, error) {
		runs++
		if runs < 9 {
			return nil, errors.New("error")
		}

		return 0, nil
	}), RetryConfig{
		Attempts:        10,
		BaseExp:         2,
		BaseTimeout:     1 * time.Millisecond,
		MaxRetryTimeout: 10 * time.Millisecond,
	})

	assert.Nil(t, err)
	assert.Equal(t, res, 0)
	assert.Equal(t, runs, 9)
}

func TestBatchRun(t *testing.T) {
	len := 10
	var suppliers []Supplier

	for i := 0; i < len; i++ {
		index := i
		// force a wait time to make sure that even though the first tasks
		// take longer to process, the Batch() still returns the results
		// in the order provided
		wait := time.Duration(len-i) * time.Millisecond
		suppliers = append(suppliers, SupplierFunc(func() (interface{}, error) {
			<-time.After(wait)
			return index, nil
		}))
	}

	res := BatchWithConfig(context.Background(), suppliers, BatchConfig{
		Concurrency: uint8(len),
	})

	for i := 0; i < len; i++ {
		assert.Nil(t, res[i].Err)
		assert.Equal(t, i, res[i].Result)
	}
}

func TestPoolRun(t *testing.T) {
	len := 10
	res := make(chan Result)
	pool := NewPoolRunner(context.Background())
	defer pool.Stop()

	go func() {
		for i := 0; i < len; i++ {
			wait := time.Millisecond
			pool.Run(res, SupplierFunc(func() (interface{}, error) {
				<-time.After(wait)
				return nil, nil
			}))
		}
	}()

	counter := 0
	for r := range res {
		assert.Nil(t, r.Err)
		assert.Nil(t, r.Result)
		counter++
		if counter == len {
			close(res)
		}
	}

	assert.Equal(t, counter, len)
}

func BenchmarkPoolRunner(b *testing.B) {
	res := make(chan Result, 64)
	supplier := SupplierFunc(func() (interface{}, error) {
		return nil, nil
	})
	pool := NewPoolRunnerWithConfig(context.Background(), PoolConfig{
		Concurrency: 8,
	})
	defer pool.Stop()

	go func() {
		for i := 0; i < b.N; i++ {
			pool.Run(res, supplier)
		}
	}()

	count := 0
	for range res {
		count++

		if count == b.N {
			close(res)
		}
	}

	assert.Equal(b, b.N, count)
}

func BenchmarkPoolRunnerWithConstantWait(b *testing.B) {
	res := make(chan Result, 128)
	supplier := SupplierFunc(func() (interface{}, error) {
		<-time.After(1 * time.Microsecond)
		return nil, nil
	})
	pool := NewPoolRunnerWithConfig(context.Background(), PoolConfig{
		Concurrency: 8,
	})
	defer pool.Stop()

	go func() {
		for i := 0; i < b.N; i++ {
			pool.Run(res, supplier)
		}
	}()

	count := 0
	for range res {
		count++

		if count == b.N {
			close(res)
		}
	}

	assert.Equal(b, b.N, count)
}

func BenchmarkBatchRunner(b *testing.B) {
	batchSize := 1024
	base := 0
	batch := make([]Supplier, 0, batchSize)
	s := SupplierFunc(func() (interface{}, error) {
		return 0, nil
	})
	runner := NewBatchRunnerWithConfig(context.Background(), BatchConfig{
		Concurrency: 8,
	})
	defer runner.Stop()

	for counter := 0; counter < b.N; counter += base {
		batch = batch[:0]

		for base = 0; base+counter < b.N && base < batchSize; base++ {
			batch = append(batch, s)
		}

		_ = runner.Run(batch)
	}
}

func BenchmarkBatchRunnerWithConstantWait(b *testing.B) {
	batchSize := 1024
	base := 0
	batch := make([]Supplier, 0, batchSize)
	s := SupplierFunc(func() (interface{}, error) {
		<-time.After(1 * time.Microsecond)
		return 0, nil
	})
	runner := NewBatchRunnerWithConfig(context.Background(), BatchConfig{
		Concurrency: 8,
	})
	defer runner.Stop()

	for counter := 0; counter < b.N; counter += base {
		batch = batch[:0]

		for base = 0; base+counter < b.N && base < batchSize; base++ {
			batch = append(batch, s)
		}

		_ = runner.Run(batch)
	}
}

func BenchmarkBatch(b *testing.B) {
	batchSize := 1024
	base := 0
	batch := make([]Supplier, 0, batchSize)
	s := SupplierFunc(func() (interface{}, error) {
		return 0, nil
	})

	for counter := 0; counter < b.N; counter += base {
		batch = batch[:0]

		for base = 0; base+counter < b.N && base < batchSize; base++ {
			batch = append(batch, s)
		}

		_ = BatchWithConfig(context.Background(), batch, BatchConfig{Concurrency: 16})
	}
}

func BenchmarkBatchWithoutRunner(b *testing.B) {
	batchSize := 1024
	base := 0
	batch := make([]Supplier, 0, batchSize)
	s := SupplierFunc(func() (interface{}, error) {
		return 0, nil
	})

	for counter := 0; counter < b.N; counter += base {
		batch = batch[:0]

		for base = 0; base+counter < b.N && base < batchSize; base++ {
			batch = append(batch, s)
		}

		for _, s := range batch {
			_, _ = s.Supply()
		}
	}
}

func BenchmarkBatchWithoutRunnerWithConstantWait(b *testing.B) {
	batchSize := 1024
	base := 0
	batch := make([]Supplier, 0, batchSize)
	s := SupplierFunc(func() (interface{}, error) {
		<-time.After(1 * time.Microsecond)
		return 0, nil
	})

	for counter := 0; counter < b.N; counter += base {
		batch = batch[:0]

		for base = 0; base+counter < b.N && base < batchSize; base++ {
			batch = append(batch, s)
		}

		for _, s := range batch {
			_, _ = s.Supply()
		}
	}
}

func BenchmarkExecuteWithoutBatch(b *testing.B) {
	s := SupplierFunc(func() (interface{}, error) {
		return 0, nil
	})

	for counter := 0; counter < b.N; counter++ {
		_, _ = s.Supply()
	}
}
