package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Status    string `json:"status"`
		Framework string `json:"framework"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if response.Framework != "stellar" {
		t.Fatalf("expected framework stellar, got %q", response.Framework)
	}
	if response.Status != "UP" {
		t.Fatalf("expected status UP, got %q", response.Status)
	}
}

func TestStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/stellatlas/v1/status", nil)

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response statusResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode status response: %v", err)
	}

	if response.Product != "StellAtlas" {
		t.Fatalf("expected product StellAtlas, got %q", response.Product)
	}
	if response.Framework != "stellar" {
		t.Fatalf("expected framework stellar, got %q", response.Framework)
	}
	if response.Role == "" {
		t.Fatal("expected non-empty role")
	}
}

func TestApplicationsWithoutRepository(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/stellatlas/v1/apps", nil)

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
}

func TestCreateApplicationRejectsInvalidAppID(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := []byte(`{"app_id":"payment.risk.antifraud.api","app_name":"Antifraud API"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/stellatlas/v1/apps", bytes.NewReader(body))

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != "INVALID_APP_ID" {
		t.Fatalf("expected INVALID_APP_ID, got %q", response.Code)
	}
}
