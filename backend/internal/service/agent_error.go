package service

import (
	"errors"
	"net/http"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"google.golang.org/grpc/codes"
)

func mapAgentError(err error) error {
	var agentErr *grpcclient.AgentError
	if errors.As(err, &agentErr) {
		appCode := apperror.ErrAgentUnavailable.Code
		if agentErr.Code > 0 {
			appCode = int(agentErr.Code)
		}

		message := apperror.ErrAgentUnavailable.Message
		if agentErr.Message != "" {
			message = agentErr.Message
		}

		return apperror.Wrap(
			appCode,
			mapAgentHTTPStatus(agentErr),
			message,
			err,
		)
	}

	return apperror.Wrap(
		apperror.ErrAgentUnavailable.Code,
		apperror.ErrAgentUnavailable.HTTPStatus,
		apperror.ErrAgentUnavailable.Message,
		err,
	)
}

func mapAgentHTTPStatus(err *grpcclient.AgentError) int {
	if err == nil {
		return apperror.ErrAgentUnavailable.HTTPStatus
	}

	if err.IsTransport() {
		switch err.GRPCCode {
		case codes.InvalidArgument:
			return http.StatusBadRequest
		case codes.Unauthenticated:
			return http.StatusUnauthorized
		case codes.PermissionDenied:
			return http.StatusForbidden
		case codes.NotFound:
			return http.StatusNotFound
		case codes.DeadlineExceeded, codes.Unavailable:
			return http.StatusServiceUnavailable
		case codes.Unimplemented:
			return http.StatusBadGateway
		default:
			return apperror.ErrAgentUnavailable.HTTPStatus
		}
	}

	switch err.Code {
	case 4000, 4003, 4006, 5000, 6001, 7000:
		return http.StatusBadRequest
	case 4004:
		return http.StatusRequestEntityTooLarge
	case 4001, 4007, 5002:
		return http.StatusForbidden
	case 4002, 6002, 7001:
		return http.StatusNotFound
	case 4005, 5001, 5003, 6003, 7002:
		return http.StatusBadGateway
	case 6000:
		return http.StatusServiceUnavailable
	default:
		return apperror.ErrAgentUnavailable.HTTPStatus
	}
}
