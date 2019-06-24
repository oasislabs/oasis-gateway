package ethtest

import "github.com/stretchr/testify/mock"

type MockSubscription struct {
	mock.Mock
}

func (s *MockSubscription) Unsubscribe() {
	_ = s.Called()
}

func (s *MockSubscription) Err() <-chan error {
	args := s.Called()
	return args.Get(0).(<-chan error)
}
