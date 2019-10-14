package mailboxtest

import (
	"context"

	"github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/oasislabs/oasis-gateway/stats"
	"github.com/stretchr/testify/mock"
)

type Mailbox struct {
	mock.Mock
}

func (m *Mailbox) Name() string {
	return "mqueue.mailboxtest.Mailbox"
}

func (m *Mailbox) Stats() stats.Metrics {
	return nil
}

func (m *Mailbox) Insert(ctx context.Context, req core.InsertRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *Mailbox) Exists(ctx context.Context, req core.ExistsRequest) (bool, error) {
	args := m.Called(ctx, req)
	return args.Bool(0), args.Error(1)
}

func (m *Mailbox) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.Elements, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(core.Elements), args.Error(1)
}

func (m *Mailbox) Discard(ctx context.Context, req core.DiscardRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *Mailbox) Next(ctx context.Context, req core.NextRequest) (uint64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *Mailbox) Remove(ctx context.Context, req core.RemoveRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}
