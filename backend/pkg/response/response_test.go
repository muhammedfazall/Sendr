package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestJSONWritesContentType(t *testing.T) {
	c, w := newCtx()
	JSON(c, http.StatusOK, map[string]string{"hello": "world"})

	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
}

func TestJSONWritesStatusCode(t *testing.T) {
	c, w := newCtx()
	JSON(c, http.StatusCreated, "data")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestJSONWritesBody(t *testing.T) {
	c, w := newCtx()
	JSON(c, http.StatusOK, "hello")

	var got string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected body %q, got %q", "hello", got)
	}
}

func TestErrorBodyStructure(t *testing.T) {
	c, w := newCtx()
	Error(c, http.StatusBadRequest, "bad_request", "something went wrong")

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "bad_request" {
		t.Fatalf("expected code 'bad_request', got %q", body.Code)
	}
	if body.Message != "something went wrong" {
		t.Fatalf("expected message 'something went wrong', got %q", body.Message)
	}
}

func TestErrorStatusCode(t *testing.T) {
	c, w := newCtx()
	Error(c, http.StatusUnauthorized, "unauthorized", "no access")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestSuccessWrapsInData(t *testing.T) {
	c, w := newCtx()
	Success(c, http.StatusOK, "payload")

	var body struct {
		Data interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data != "payload" {
		t.Fatalf("expected data 'payload', got %v", body.Data)
	}
}

func TestSuccessWithStruct(t *testing.T) {
	c, w := newCtx()
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	Success(c, http.StatusOK, Item{ID: 1, Name: "test"})

	var body struct {
		Data Item `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data.ID != 1 || body.Data.Name != "test" {
		t.Fatalf("got %+v", body.Data)
	}
}

func TestErrorStatusText(t *testing.T) {
	tests := []struct {
		status int
		name   string
	}{
		{http.StatusBadRequest, "400"},
		{http.StatusUnauthorized, "401"},
		{http.StatusForbidden, "403"},
		{http.StatusNotFound, "404"},
		{http.StatusTooManyRequests, "429"},
		{http.StatusInternalServerError, "500"},
		{http.StatusServiceUnavailable, "503"},
	}
	for _, tt := range tests {
		c, w := newCtx()
		Error(c, tt.status, "err", "msg")
		if w.Code != tt.status {
			t.Errorf("Error(%s): expected status %d, got %d", tt.name, tt.status, w.Code)
		}
	}
}
