package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONWritesContentType(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
}

func TestJSONWritesStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusCreated, "data")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestJSONWritesBody(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, "hello")

	var got string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected body %q, got %q", "hello", got)
	}
}

func TestErrorBodyStructure(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "bad_request", "something went wrong")

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
	w := httptest.NewRecorder()
	Error(w, http.StatusUnauthorized, "unauthorized", "no access")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestSuccessWrapsInData(t *testing.T) {
	w := httptest.NewRecorder()
	Success(w, http.StatusOK, "payload")

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
	w := httptest.NewRecorder()
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	Success(w, http.StatusOK, Item{ID: 1, Name: "test"})

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
		w := httptest.NewRecorder()
		Error(w, tt.status, "err", "msg")
		if w.Code != tt.status {
			t.Errorf("Error(%s): expected status %d, got %d", tt.name, tt.status, w.Code)
		}
	}
}
