package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type CronService interface {
	ListTasks(ctx context.Context) (dto.ListCronTasksResult, error)
	CreateTask(ctx context.Context, req dto.CreateCronTaskRequest) (dto.CreateCronTaskResult, error)
	UpdateTask(ctx context.Context, id string, req dto.UpdateCronTaskRequest) (dto.UpdateCronTaskResult, error)
	DeleteTask(ctx context.Context, id string) (dto.DeleteCronTaskResult, error)
	SetEnabled(ctx context.Context, id string, enabled bool) (dto.ToggleCronTaskResult, error)
}

type cronService struct {
	agentClient grpcclient.AgentClient
}

func NewCronService(agentClient grpcclient.AgentClient) CronService {
	return &cronService{agentClient: agentClient}
}

func (s *cronService) ListTasks(ctx context.Context) (dto.ListCronTasksResult, error) {
	result, err := s.agentClient.ListCronTasks(ctx)
	if err != nil {
		return dto.ListCronTasksResult{}, mapAgentError(err)
	}

	items := make([]dto.CronTask, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		items = append(items, mapCronTask(task))
	}
	s.emitAudit(ctx, "cron", "list", "", true)
	return dto.ListCronTasksResult{Tasks: items}, nil
}

func (s *cronService) CreateTask(
	ctx context.Context,
	req dto.CreateCronTaskRequest,
) (dto.CreateCronTaskResult, error) {
	if err := validateCronCommand(req.Command); err != nil {
		return dto.CreateCronTaskResult{}, err
	}

	result, err := s.agentClient.CreateCronTask(ctx, grpcclient.CreateCronTaskRequest{
		Expression: req.Expression,
		Command:    req.Command,
		Enabled:    req.Enabled,
	})
	if err != nil {
		s.emitAudit(ctx, "cron", "create", req.Expression, false)
		return dto.CreateCronTaskResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "cron", "create", req.Expression, true)
	return dto.CreateCronTaskResult{Task: mapCronTask(result.Task)}, nil
}

func (s *cronService) UpdateTask(
	ctx context.Context,
	id string,
	req dto.UpdateCronTaskRequest,
) (dto.UpdateCronTaskResult, error) {
	if err := validateCronCommand(req.Command); err != nil {
		return dto.UpdateCronTaskResult{}, err
	}

	result, err := s.agentClient.UpdateCronTask(ctx, grpcclient.UpdateCronTaskRequest{
		ID:         id,
		Expression: req.Expression,
		Command:    req.Command,
		Enabled:    req.Enabled,
	})
	if err != nil {
		s.emitAudit(ctx, "cron", "update", id, false)
		return dto.UpdateCronTaskResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "cron", "update", id, true)
	return dto.UpdateCronTaskResult{Task: mapCronTask(result.Task)}, nil
}

func (s *cronService) DeleteTask(ctx context.Context, id string) (dto.DeleteCronTaskResult, error) {
	result, err := s.agentClient.DeleteCronTask(ctx, grpcclient.DeleteCronTaskRequest{ID: id})
	if err != nil {
		s.emitAudit(ctx, "cron", "delete", id, false)
		return dto.DeleteCronTaskResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "cron", "delete", id, true)
	return dto.DeleteCronTaskResult{ID: result.ID}, nil
}

func (s *cronService) SetEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (dto.ToggleCronTaskResult, error) {
	result, err := s.agentClient.SetCronTaskEnabled(ctx, grpcclient.SetCronTaskEnabledRequest{
		ID:      id,
		Enabled: enabled,
	})
	if err != nil {
		s.emitAudit(ctx, "cron", "set_enabled", id, false)
		return dto.ToggleCronTaskResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "cron", "set_enabled", id, true)
	return dto.ToggleCronTaskResult{Task: mapCronTask(result.Task)}, nil
}

func (s *cronService) emitAudit(_ context.Context, _ string, _ string, _ string, _ bool) {
	// Reserved for audit integration in stage 19.
}

func mapCronTask(task grpcclient.CronTask) dto.CronTask {
	return dto.CronTask{
		ID:         task.ID,
		Expression: task.Expression,
		Command:    task.Command,
		Enabled:    task.Enabled,
	}
}

func validateCronCommand(command string) error {
	normalized := strings.TrimSpace(command)
	if normalized == "" {
		return apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			"invalid cron command template",
			fmt.Errorf("command is empty"),
		)
	}

	blockedTokens := []string{"|", "&", ";", ">", "<", "`", "$", "\\", "(", ")"}
	for _, token := range blockedTokens {
		if strings.Contains(normalized, token) {
			return apperror.Wrap(
				apperror.ErrBadRequest.Code,
				apperror.ErrBadRequest.HTTPStatus,
				"invalid cron command template",
				fmt.Errorf("blocked shell token '%s' in command", token),
			)
		}
	}

	for _, ch := range normalized {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.' || ch == '/' {
			continue
		}
		return apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			"invalid cron command template",
			fmt.Errorf("unsupported character '%c' in command", ch),
		)
	}

	return nil
}
