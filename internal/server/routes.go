package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/stellhub/stellar"
	stellarhttp "github.com/stellhub/stellar/transport/http"
	"github.com/stellhub/stellatlas-service/internal/cmdb"
)

type apiHandler struct {
	config  stellar.Config
	service *cmdb.Service
}

type statusResponse struct {
	Service     string             `json:"service"`
	Product     string             `json:"product"`
	Role        string             `json:"role"`
	Description string             `json:"description"`
	Framework   string             `json:"framework"`
	Environment string             `json:"environment"`
	Zone        string             `json:"zone,omitempty"`
	Storage     cmdb.ServiceStatus `json:"storage"`
	Timestamp   string             `json:"timestamp"`
}

type applicationListResponse struct {
	Items  []cmdb.ApplicationSummary `json:"items"`
	Count  int                       `json:"count"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

type ownerListResponse struct {
	Items []cmdb.PersonRelation `json:"items"`
	Count int                   `json:"count"`
}

type instanceListResponse struct {
	Items  []cmdb.InstanceSummary `json:"items"`
	Count  int                    `json:"count"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func registerRoutes(router *stellarhttp.Router, config stellar.Config, service *cmdb.Service) {
	handler := apiHandler{
		config:  config,
		service: service,
	}
	api := router.Group("/api/stellatlas/v1")
	api.GET("/status", handler.handleStatus)
	api.GET("/apps", handler.handleApplications)
	api.GET("/apps/detail", handler.handleApplication)
	api.POST("/apps", handler.handleCreateApplication)
	api.PUT("/apps", handler.handleUpdateApplication)
	api.DELETE("/apps", handler.handleDeleteApplication)
	api.GET("/app-owners", handler.handleApplicationOwners)
	api.GET("/app-instances", handler.handleApplicationInstances)
}

func (h apiHandler) handleStatus(context.Context, *stellarhttp.Request) (*stellarhttp.Response, error) {
	return stellarhttp.JSON(http.StatusOK, statusResponse{
		Service:     h.config.AppName,
		Product:     "StellAtlas",
		Role:        "Configuration Management Database service",
		Description: "Manages configuration items, asset inventory, topology relationships, ownership metadata, and lifecycle state for the Stell platform.",
		Framework:   "stellar",
		Environment: string(h.config.Environment),
		Zone:        h.config.Zone,
		Storage:     h.service.Status(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}), nil
}

func (h apiHandler) handleApplications(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	query := cmdb.ApplicationListQuery{
		Environment: firstQuery(request.Query, "environment", "env"),
		Status:      request.Query.Get("status"),
		Search:      request.Query.Get("search"),
		Limit:       intQuery(request.Query, "limit", 50),
		Offset:      intQuery(request.Query, "offset", 0),
	}
	items, err := h.service.ListApplications(ctx, query)
	if err != nil {
		return serviceError(err), nil
	}
	return stellarhttp.JSON(http.StatusOK, applicationListResponse{
		Items:  items,
		Count:  len(items),
		Limit:  query.Limit,
		Offset: query.Offset,
	}), nil
}

func (h apiHandler) handleApplication(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	item, err := h.service.GetApplication(ctx, firstQuery(request.Query, "app_id", "app_code", "ci_id"))
	if err != nil {
		return serviceError(err), nil
	}
	return stellarhttp.JSON(http.StatusOK, item), nil
}

func (h apiHandler) handleCreateApplication(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	var payload cmdb.CreateApplicationRequest
	if err := decodeJSON(request, &payload); err != nil {
		return apiError(http.StatusBadRequest, "INVALID_JSON", err.Error()), nil
	}
	item, err := h.service.CreateApplication(ctx, payload)
	if err != nil {
		return serviceError(err), nil
	}
	return stellarhttp.JSON(http.StatusCreated, item), nil
}

func (h apiHandler) handleUpdateApplication(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	var payload cmdb.UpdateApplicationRequest
	if err := decodeJSON(request, &payload); err != nil {
		return apiError(http.StatusBadRequest, "INVALID_JSON", err.Error()), nil
	}
	item, err := h.service.UpdateApplication(ctx, payload)
	if err != nil {
		return serviceError(err), nil
	}
	return stellarhttp.JSON(http.StatusOK, item), nil
}

func (h apiHandler) handleDeleteApplication(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	if err := h.service.DeleteApplication(ctx, firstQuery(request.Query, "app_id", "app_code", "ci_id")); err != nil {
		return serviceError(err), nil
	}
	return &stellarhttp.Response{Status: http.StatusNoContent}, nil
}

func (h apiHandler) handleApplicationOwners(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	items, err := h.service.ListApplicationOwners(ctx, firstQuery(request.Query, "app_id", "app_code", "ci_id"))
	if err != nil {
		return serviceError(err), nil
	}
	return stellarhttp.JSON(http.StatusOK, ownerListResponse{
		Items: items,
		Count: len(items),
	}), nil
}

func (h apiHandler) handleApplicationInstances(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	query := cmdb.InstanceListQuery{
		AppID:       request.Query.Get("app_id"),
		Environment: firstQuery(request.Query, "environment", "env"),
		Limit:       intQuery(request.Query, "limit", 50),
		Offset:      intQuery(request.Query, "offset", 0),
	}
	items, err := h.service.ListApplicationInstances(ctx, query)
	if err != nil {
		return serviceError(err), nil
	}
	return stellarhttp.JSON(http.StatusOK, instanceListResponse{
		Items:  items,
		Count:  len(items),
		Limit:  query.Limit,
		Offset: query.Offset,
	}), nil
}

func serviceError(err error) *stellarhttp.Response {
	switch {
	case errors.Is(err, cmdb.ErrAppIDRequired):
		return apiError(http.StatusBadRequest, "APP_ID_REQUIRED", err.Error())
	case errors.Is(err, cmdb.ErrInvalidAppID):
		return apiError(http.StatusBadRequest, "INVALID_APP_ID", err.Error())
	case errors.Is(err, cmdb.ErrApplicationNameRequired):
		return apiError(http.StatusBadRequest, "APP_NAME_REQUIRED", err.Error())
	case errors.Is(err, cmdb.ErrApplicationDuplicate):
		return apiError(http.StatusConflict, "APPLICATION_ALREADY_EXISTS", err.Error())
	case errors.Is(err, cmdb.ErrApplicationNotFound):
		return apiError(http.StatusNotFound, "APPLICATION_NOT_FOUND", err.Error())
	case errors.Is(err, cmdb.ErrRepositoryUnavailable):
		return apiError(http.StatusServiceUnavailable, "POSTGRESQL_UNAVAILABLE", err.Error())
	default:
		return apiError(http.StatusInternalServerError, "CMDB_QUERY_FAILED", err.Error())
	}
}

func apiError(status int, code string, message string) *stellarhttp.Response {
	return stellarhttp.JSON(status, errorResponse{
		Code:    code,
		Message: message,
	})
}

func decodeJSON(request *stellarhttp.Request, target any) error {
	if request.Body == nil {
		return errors.New("request body is required")
	}
	defer request.Body.Close()
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func firstQuery(values url.Values, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(values.Get(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func intQuery(values url.Values, key string, fallback int) int {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
