package core

import (
	"context"

	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
)

type SubscriptionEvent struct {
	Context context.Context
	Value   interface{}
}

type subscription struct {
	logger log.Logger
	C      chan SubscriptionEvent
	key    string
	mqueue mqueue.MQueue
}

func newSubscription(logger log.Logger, mqueue mqueue.MQueue, key string) *subscription {
	return &subscription{
		logger: logger.ForClass("backend", "subscription"),
		C:      make(chan SubscriptionEvent),
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
				s.logger.Warn(ev.Context, "failed to find next resource for event", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"err":       err.Error(),
				})
				continue
			}

			if err := s.mqueue.Insert(s.key, mqueue.Element{
				Value:  ev,
				Offset: id,
			}); err != nil {
				s.logger.Warn(ev.Context, "failed to insert event to resource", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"err":       err.Error(),
				})
			}
		}
	}
}
