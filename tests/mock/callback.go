package mock

import (
	"context"

	callback "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/stretchr/testify/mock"
)

type MockCallbackClient struct {
	mock.Mock
}

func (c *MockCallbackClient) WalletOutOfFunds(ctx context.Context, body callback.WalletOutOfFundsBody) {
	_ = c.Called(ctx, body)
}
