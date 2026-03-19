package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleHealthz_Returns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	HandleHealthz(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestHandleReadyz_NoChecks_Returns200(t *testing.T) {
	handler := HandleReadyz()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "ready", body["status"])
}

func TestHandleReadyz_AllChecksPass_Returns200(t *testing.T) {
	okCheck := func() error { return nil }
	handler := HandleReadyz(okCheck, okCheck)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleReadyz_CheckFails_Returns503(t *testing.T) {
	okCheck := func() error { return nil }
	failCheck := func() error { return errors.New("not ready") }
	handler := HandleReadyz(okCheck, failCheck)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "not_ready", body["status"])

	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, checks, "check_1")
}
