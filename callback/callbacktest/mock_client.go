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

func (c *MockClient) WalletReachedFundsThreshold(
	ctx context.Context,
	body callback.WalletReachedFundsThresholdBody,
) {
	_ = c.Called(ctx, body)
}

func ImplementMock(client *MockClient) {
	client.On("WalletOutOfFunds", mock.Anything, mock.Anything).Return()
	client.On("WalletReachedFundsThreshold", mock.Anything, mock.Anything).Return()
}
