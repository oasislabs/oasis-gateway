package core

import (
	"encoding/json"
	"net/http"

	"github.com/oasislabs/developer-gateway/stats"
)

type MultiAuth struct {
	auths []Auth
}

func (m *MultiAuth) Add(a Auth) {
	if m.auths == nil {
		m.auths = make([]Auth, 0)
		m.auths = append(m.auths, a)
	}
}

func (*MultiAuth) Name() string {
	return "auth.MultiAuth"
}
func (m *MultiAuth) Stats() stats.Metrics {
	metrics := make(stats.Metrics)
	for _, auth := range m.auths {
		for k, val := range auth.Stats() {
			metrics[k] = val
		}
	}
	return metrics
}

func (m *MultiAuth) Authenticate(req *http.Request) (string, error) {
	strs := make([]string, 0, len(m.auths))
	for _, auth := range m.auths {
		s, err := auth.Authenticate(req)
		if err != nil {
			return "", err
		}
		strs = append(strs, s)
	}
	bytes, err := json.Marshal(strs)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (m *MultiAuth) Verify(data, expected string) error {
	var strs []string
	err := json.Unmarshal([]byte(data), &strs)
	if err != nil {
		return err
	}
	for i, auth := range m.auths {
		if err = auth.Verify(data, strs[i]); err != nil {
			return err
		}
	}
	return nil
}
