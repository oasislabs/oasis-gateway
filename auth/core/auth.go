package core

type Auth interface {
	Key() string
	Verify(key, value string) (string, error)
}
