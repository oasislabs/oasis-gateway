package core

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
)

type subscription struct {
	ctx    context.Context
	logger log.Logger
	C      chan interface{}
	key    string
	mqueue mqueue.MQueue
}

func newSubscription(ctx context.Context, logger log.Logger, mqueue mqueue.MQueue, key string) *subscription {
	return &subscription{
		ctx:    ctx,
		logger: logger.ForClass("backend", "subscription"),
		C:      make(chan interface{}),
		key:    key,
		mqueue: mqueue,
	}
}

func (s *subscription) Start() {
	// TODO(stan):
	// - a subscription should have a context deriving from the
	// manager's context.
	// - a subscription should notify the key manager when it
	// exits to make sure state is tracked correctly
	for {
		select {
		case ev, ok := <-s.C:
			if !ok {
				return
			}

			id, err := s.mqueue.Next(s.key)
			if err != nil {
				s.logger.Warn(s.ctx, "failed to find next resource for event", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"err":       err.Error(),
				})
				continue
			}

			data, ok := ev.(types.Log)
			if !ok {
				s.logger.Warn(s.ctx, "received event of unexpected type", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"type":      fmt.Sprintf("%+v", ev),
				})
				continue
			}

			if err := s.mqueue.Insert(s.key, mqueue.Element{
				Value: DataEvent{
					ID:   id,
					Data: hexutil.Encode(data.Data),
				},
				Offset: id,
			}); err != nil {
				s.logger.Warn(s.ctx, "failed to insert event to resource", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"err":       err.Error(),
				})
			}
		}
	}
}
