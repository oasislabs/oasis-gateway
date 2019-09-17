package auth

import (
	"github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/auth/oauth"
)

type Factory interface {
	New(*Config) (core.Auth, error)
}

type FactoryFunc func(*Config) (core.Auth, error)

func (f FactoryFunc) New(config *Config) (core.Auth, error) {
	return f(config)
}

var NewAuth = FactoryFunc(func(config *Config) (core.Auth, error) {
	if len(config.Providers) == 0 {
		return &core.NilAuth{}, nil
	} else if len(config.Providers) == 1 {
		return config.Providers[0], nil
	}
	multiAuth := new(core.MultiAuth)
	for _, p := range config.Providers {
		multiAuth.Add(p)
	}
	return multiAuth, nil
})

func newAuthSingle(provider AuthProvider) core.Auth {
	switch provider {
	case AuthOauth:
		return oauth.NewGoogleOauth(oauth.NewGoogleIDTokenVerifier())
	case AuthInsecure:
		return insecure.InsecureAuth{}
	default:
		return nil
	}
}
