package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/nfedorov/port_server/internal/model"
	"github.com/nfedorov/port_server/internal/store"
)

func setup(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return New(s).Routes()
}

func TestHealthz(t *testing.T) {
	srv := setup(t)
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAllocateAndList(t *testing.T) {
	srv := setup(t)

	// Allocate
	body, _ := json.Marshal(model.AllocateRequest{App: "myapp", Instance: "i1", Service: "web", Port: 3000})
	req := httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var alloc model.Allocation
	json.NewDecoder(w.Body).Decode(&alloc)
	if alloc.Port != 3000 {
		t.Fatalf("expected port 3000, got %d", alloc.Port)
	}

	// List all
	req = httptest.NewRequest("GET", "/v1/allocations", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var allocs []model.Allocation
	json.NewDecoder(w.Body).Decode(&allocs)
	if len(allocs) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(allocs))
	}

	// List with filter
	req = httptest.NewRequest("GET", "/v1/allocations?app=myapp", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&allocs)
	if len(allocs) != 1 {
		t.Fatalf("expected 1 allocation for app=myapp, got %d", len(allocs))
	}
}

func TestAllocateAutoAssign(t *testing.T) {
	srv := setup(t)

	body, _ := json.Marshal(model.AllocateRequest{App: "a", Instance: "i", Service: "s"})
	req := httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var alloc model.Allocation
	json.NewDecoder(w.Body).Decode(&alloc)
	if alloc.Port < 3000 || alloc.Port > 9999 {
		t.Fatalf("auto-assigned port %d out of range", alloc.Port)
	}
}

func TestAllocateConflict(t *testing.T) {
	srv := setup(t)

	body, _ := json.Marshal(model.AllocateRequest{App: "a", Instance: "i", Service: "s", Port: 5000})
	req := httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	body, _ = json.Marshal(model.AllocateRequest{App: "b", Instance: "j", Service: "s", Port: 5000})
	req = httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp model.ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Holder == nil {
		t.Fatal("expected holder info in conflict response")
	}
}

func TestAllocateValidation(t *testing.T) {
	srv := setup(t)

	body, _ := json.Marshal(model.AllocateRequest{App: "a"})
	req := httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCheckPort(t *testing.T) {
	srv := setup(t)

	// Check available port
	req := httptest.NewRequest("GET", "/v1/ports/4000", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var status model.PortStatus
	json.NewDecoder(w.Body).Decode(&status)
	if !status.Available {
		t.Fatal("expected port to be available")
	}

	// Allocate it
	body, _ := json.Marshal(model.AllocateRequest{App: "a", Instance: "i", Service: "s", Port: 4000})
	req = httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Check taken port
	req = httptest.NewRequest("GET", "/v1/ports/4000", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&status)
	if status.Available {
		t.Fatal("expected port to be taken")
	}
	if status.Holder == nil {
		t.Fatal("expected holder info")
	}
}

func TestReleaseByID(t *testing.T) {
	srv := setup(t)

	body, _ := json.Marshal(model.AllocateRequest{App: "a", Instance: "i", Service: "s", Port: 3000})
	req := httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var alloc model.Allocation
	json.NewDecoder(w.Body).Decode(&alloc)

	req = httptest.NewRequest("DELETE", "/v1/allocations/"+strconv.FormatInt(alloc.ID, 10), nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	req = httptest.NewRequest("GET", "/v1/allocations", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var allocs []model.Allocation
	json.NewDecoder(w.Body).Decode(&allocs)
	if len(allocs) != 0 {
		t.Fatalf("expected 0 allocations, got %d", len(allocs))
	}
}

func TestReleaseByIDNotFound(t *testing.T) {
	srv := setup(t)

	req := httptest.NewRequest("DELETE", "/v1/allocations/999", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestReleaseByFilter(t *testing.T) {
	srv := setup(t)

	// Create two allocations
	for _, svc := range []string{"web", "db"} {
		body, _ := json.Marshal(model.AllocateRequest{App: "a", Instance: "i1", Service: svc})
		req := httptest.NewRequest("POST", "/v1/allocations", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
	}

	// Delete by app+instance
	body, _ := json.Marshal(model.ReleaseRequest{App: "a", Instance: "i1"})
	req := httptest.NewRequest("DELETE", "/v1/allocations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]int64
	json.NewDecoder(w.Body).Decode(&result)
	if result["deleted"] != 2 {
		t.Fatalf("expected 2 deleted, got %d", result["deleted"])
	}
}

func TestListEmpty(t *testing.T) {
	srv := setup(t)

	req := httptest.NewRequest("GET", "/v1/allocations", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var allocs []model.Allocation
	json.NewDecoder(w.Body).Decode(&allocs)
	if len(allocs) != 0 {
		t.Fatalf("expected empty list, got %d", len(allocs))
	}
}
