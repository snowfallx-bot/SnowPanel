package service

import (
	"errors"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

func mapAgentError(err error) error {
	if errors.Is(err, grpcclient.ErrNotImplemented) {
		return apperror.ErrAgentUnavailable
	}
	return apperror.Wrap(
		apperror.ErrAgentUnavailable.Code,
		apperror.ErrAgentUnavailable.HTTPStatus,
		apperror.ErrAgentUnavailable.Message,
		err,
	)
}
