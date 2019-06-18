package auth

import (
	"errors"
	"fmt"

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
		return core.NilAuth{}, nil
	} else if len(config.Providers) == 1 {
		a := newAuthSingle(config.Providers[0])
		if a == nil {
			return nil, errors.New("A valid authenticator must be specified")
		} else {
			return a, nil
		}
	}
	multiAuth := new(core.MultiAuth)
	for _, p := range config.Providers {
		auth := newAuthSingle(p)
		if auth == nil {
			return nil, fmt.Errorf("Unable to create auth provider %v", p)
		}
		multiAuth.Add(auth)
	}
	return multiAuth, nil
})

func newAuthSingle(provider AuthProvider) core.Auth {
	switch provider {
	case AuthOauth:
		return oauth.GoogleOauth{}
	case AuthInsecure:
		return insecure.InsecureAuth{}
	default:
		return nil
	}
}
