package auth

import (
	"errors"

	"github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/auth/oauth"
)

func NewAuth(config *Config) (core.Auth, error) {
	switch config.Provider {
	case AuthOauth:
		return oauth.GoogleOauth{}, nil
	case AuthInsecure:
		return insecure.InsecureAuth{}, nil
	default:
		return nil, errors.New("A valid authenticator must be specified")
	}
}
