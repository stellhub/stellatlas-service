package cmdb

import (
	"context"
	"errors"
	"strings"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

var (
	ErrRepositoryUnavailable   = errors.New("cmdb: postgresql repository is unavailable")
	ErrAppIDRequired           = errors.New("cmdb: app_id is required")
	ErrInvalidAppID            = errors.New("cmdb: invalid app_id")
	ErrApplicationNameRequired = errors.New("cmdb: app_name is required")
	ErrApplicationNotFound     = errors.New("cmdb: application not found")
	ErrApplicationDuplicate    = errors.New("cmdb: application already exists")
)

type Repository interface {
	ListApplications(context.Context, ApplicationListQuery) ([]ApplicationSummary, error)
	GetApplication(context.Context, string) (*ApplicationDetail, error)
	CreateApplication(context.Context, CreateApplicationRequest) (*ApplicationDetail, error)
	UpdateApplication(context.Context, UpdateApplicationRequest) (*ApplicationDetail, error)
	DeleteApplication(context.Context, string) error
	ListApplicationOwners(context.Context, string) ([]PersonRelation, error)
	ListApplicationInstances(context.Context, InstanceListQuery) ([]InstanceSummary, error)
}

type Cache interface {
	GetApplicationList(context.Context, ApplicationListQuery) ([]ApplicationSummary, bool, error)
	SetApplicationList(context.Context, ApplicationListQuery, []ApplicationSummary) error
	GetApplicationOwners(context.Context, string) ([]PersonRelation, bool, error)
	SetApplicationOwners(context.Context, string, []PersonRelation) error
	InvalidateApplications(context.Context, ...string) error
}

type Service struct {
	repository Repository
	cache      Cache
}

func NewService(repository Repository, cache Cache) *Service {
	return &Service{
		repository: repository,
		cache:      cache,
	}
}

func (s *Service) Status() ServiceStatus {
	return ServiceStatus{
		PostgreSQLConfigured: s != nil && s.repository != nil,
		RedisConfigured:      s != nil && s.cache != nil,
		HighFrequencyReads: []string{
			"application_list",
			"application_owner_relation",
		},
	}
}

func (s *Service) ListApplications(ctx context.Context, query ApplicationListQuery) ([]ApplicationSummary, error) {
	query = normalizeApplicationListQuery(query)
	if s != nil && s.cache != nil {
		items, ok, err := s.cache.GetApplicationList(ctx, query)
		if err == nil && ok {
			return items, nil
		}
	}
	if s == nil || s.repository == nil {
		return nil, ErrRepositoryUnavailable
	}
	items, err := s.repository.ListApplications(ctx, query)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.SetApplicationList(ctx, query, items)
	}
	return items, nil
}

func (s *Service) GetApplication(ctx context.Context, appID string) (*ApplicationDetail, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, ErrAppIDRequired
	}
	if s == nil || s.repository == nil {
		return nil, ErrRepositoryUnavailable
	}
	return s.repository.GetApplication(ctx, appID)
}

func (s *Service) CreateApplication(ctx context.Context, request CreateApplicationRequest) (*ApplicationDetail, error) {
	normalized, err := normalizeCreateApplicationRequest(request)
	if err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil {
		return nil, ErrRepositoryUnavailable
	}
	item, err := s.repository.CreateApplication(ctx, normalized)
	if err != nil {
		return nil, err
	}
	s.invalidateApplicationCache(ctx, item.CIID, item.AppID, item.AppCode)
	return item, nil
}

func (s *Service) UpdateApplication(ctx context.Context, request UpdateApplicationRequest) (*ApplicationDetail, error) {
	normalized, err := normalizeUpdateApplicationRequest(request)
	if err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil {
		return nil, ErrRepositoryUnavailable
	}
	item, err := s.repository.UpdateApplication(ctx, normalized)
	if err != nil {
		return nil, err
	}
	s.invalidateApplicationCache(ctx, item.CIID, item.AppID, item.AppCode)
	return item, nil
}

func (s *Service) DeleteApplication(ctx context.Context, appID string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return ErrAppIDRequired
	}
	if s == nil || s.repository == nil {
		return ErrRepositoryUnavailable
	}
	if err := s.repository.DeleteApplication(ctx, appID); err != nil {
		return err
	}
	s.invalidateApplicationCache(ctx, appID)
	return nil
}

func (s *Service) ListApplicationOwners(ctx context.Context, appID string) ([]PersonRelation, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, ErrAppIDRequired
	}
	if s != nil && s.cache != nil {
		items, ok, err := s.cache.GetApplicationOwners(ctx, appID)
		if err == nil && ok {
			return items, nil
		}
	}
	if s == nil || s.repository == nil {
		return nil, ErrRepositoryUnavailable
	}
	items, err := s.repository.ListApplicationOwners(ctx, appID)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.SetApplicationOwners(ctx, appID, items)
	}
	return items, nil
}

func (s *Service) ListApplicationInstances(ctx context.Context, query InstanceListQuery) ([]InstanceSummary, error) {
	query.AppID = strings.TrimSpace(query.AppID)
	if query.AppID == "" {
		return nil, ErrAppIDRequired
	}
	query.Environment = strings.TrimSpace(query.Environment)
	query.Limit = normalizeLimit(query.Limit)
	if query.Offset < 0 {
		query.Offset = 0
	}
	if s == nil || s.repository == nil {
		return nil, ErrRepositoryUnavailable
	}
	return s.repository.ListApplicationInstances(ctx, query)
}

func normalizeCreateApplicationRequest(request CreateApplicationRequest) (CreateApplicationRequest, error) {
	appID := StandardAppIDFromRequest(request.AppID, request.AppCode)
	if appID == "" {
		return CreateApplicationRequest{}, ErrAppIDRequired
	}
	if _, err := ValidateStandardAppID(appID); err != nil {
		return CreateApplicationRequest{}, err
	}
	request.AppID = appID
	request.AppCode = appID
	request.AppName = strings.TrimSpace(request.AppName)
	if request.AppName == "" {
		return CreateApplicationRequest{}, ErrApplicationNameRequired
	}
	request.Environment = normalizeEnvironment(request.Environment)
	request.Status = normalizeStatus(request.Status)
	request.Lifecycle = normalizeLifecycle(request.Lifecycle)
	request.OwnerTeamCode = strings.TrimSpace(request.OwnerTeamCode)
	request.OwnerTeamName = strings.TrimSpace(request.OwnerTeamName)
	request.Language = strings.TrimSpace(request.Language)
	request.RepositoryURL = strings.TrimSpace(request.RepositoryURL)
	return request, nil
}

func normalizeUpdateApplicationRequest(request UpdateApplicationRequest) (UpdateApplicationRequest, error) {
	request.AppID = strings.TrimSpace(request.AppID)
	if request.AppID == "" {
		return UpdateApplicationRequest{}, ErrAppIDRequired
	}
	if strings.Contains(request.AppID, ".") {
		if _, err := ValidateStandardAppID(request.AppID); err != nil {
			return UpdateApplicationRequest{}, err
		}
	}
	request.AppName = strings.TrimSpace(request.AppName)
	if request.AppName == "" {
		return UpdateApplicationRequest{}, ErrApplicationNameRequired
	}
	request.Environment = normalizeEnvironment(request.Environment)
	request.Status = normalizeStatus(request.Status)
	request.Lifecycle = normalizeLifecycle(request.Lifecycle)
	request.OwnerTeamCode = strings.TrimSpace(request.OwnerTeamCode)
	request.OwnerTeamName = strings.TrimSpace(request.OwnerTeamName)
	request.Language = strings.TrimSpace(request.Language)
	request.RepositoryURL = strings.TrimSpace(request.RepositoryURL)
	return request, nil
}

func normalizeApplicationListQuery(query ApplicationListQuery) ApplicationListQuery {
	query.Environment = strings.TrimSpace(query.Environment)
	query.Status = strings.TrimSpace(query.Status)
	query.Search = strings.TrimSpace(query.Search)
	query.Limit = normalizeLimit(query.Limit)
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func normalizeEnvironment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "prod"
	}
	return value
}

func normalizeStatus(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "active"
	}
	return value
}

func normalizeLifecycle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "managed"
	}
	return value
}

func (s *Service) invalidateApplicationCache(ctx context.Context, identifiers ...string) {
	if s == nil || s.cache == nil {
		return
	}
	_ = s.cache.InvalidateApplications(ctx, identifiers...)
}
