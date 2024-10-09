package ginjwt

import (
	"github.com/pkg/errors"

	"github.com/metal-automata/rivets/ginauth"
)

// NewMultiTokenMiddlewareFromConfigs builds a MultiTokenMiddleware object from multiple AuthConfigs.
func NewMultiTokenMiddlewareFromConfigs(cfgs ...AuthConfig) (*ginauth.MultiTokenMiddleware, error) {
	if len(cfgs) == 0 {
		return nil, errors.Wrap(ErrInvalidAuthConfig, "configuration empty")
	}

	mtm := &ginauth.MultiTokenMiddleware{}

	for i := range cfgs {
		middleware, err := NewAuthMiddleware(cfgs[i])
		if err != nil {
			return nil, err
		}

		if err := mtm.Add(middleware); err != nil {
			return nil, err
		}
	}

	return mtm, nil
}
