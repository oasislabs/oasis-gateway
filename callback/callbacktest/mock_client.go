package callbacktest

import (
	"context"

	callback "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (c *MockClient) WalletOutOfFunds(
	ctx context.Context,
	body callback.WalletOutOfFundsBody,
) {
	_ = c.Called(ctx, body)
}
