package ethtest

type MockSubscription struct {
	ErrC chan error
}

func (s *MockSubscription) Unsubscribe() {}

func (s *MockSubscription) Err() <-chan error {
	return s.ErrC
}
