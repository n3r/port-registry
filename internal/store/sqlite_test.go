package store

import (
	"testing"

	"github.com/n3r/port-registry/internal/model"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	s.PortChecker = nil // skip real system checks in tests
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAllocateSpecificPort(t *testing.T) {
	s := newTestStore(t)

	a, err := s.Allocate(model.AllocateRequest{
		App: "myapp", Instance: "i1", Service: "web", Port: 3000,
	}, 3000, 9999)
	if err != nil {
		t.Fatal(err)
	}
	if a.Port != 3000 {
		t.Fatalf("expected port 3000, got %d", a.Port)
	}
}

func TestAllocateAutoAssign(t *testing.T) {
	s := newTestStore(t)

	a, err := s.Allocate(model.AllocateRequest{
		App: "myapp", Instance: "i1", Service: "web",
	}, 3000, 9999)
	if err != nil {
		t.Fatal(err)
	}
	if a.Port != 3000 {
		t.Fatalf("expected auto-assigned port 3000, got %d", a.Port)
	}

	b, err := s.Allocate(model.AllocateRequest{
		App: "myapp", Instance: "i1", Service: "db",
	}, 3000, 9999)
	if err != nil {
		t.Fatal(err)
	}
	if b.Port != 3001 {
		t.Fatalf("expected auto-assigned port 3001, got %d", b.Port)
	}
}

func TestAllocateDuplicatePort(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Allocate(model.AllocateRequest{
		App: "a", Instance: "i", Service: "s", Port: 5000,
	}, 3000, 9999)
	if err != nil {
		t.Fatal(err)
	}

	holder, err := s.Allocate(model.AllocateRequest{
		App: "b", Instance: "j", Service: "s", Port: 5000,
	}, 3000, 9999)
	if err != ErrPortTaken {
		t.Fatalf("expected ErrPortTaken, got %v", err)
	}
	if holder == nil || holder.App != "a" {
		t.Fatal("expected holder info on conflict")
	}
}

func TestAllocateDuplicateService(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Allocate(model.AllocateRequest{
		App: "a", Instance: "i", Service: "s",
	}, 3000, 9999)
	if err != nil {
		t.Fatal(err)
	}

	holder, err := s.Allocate(model.AllocateRequest{
		App: "a", Instance: "i", Service: "s",
	}, 3000, 9999)
	if err != ErrServiceAllocated {
		t.Fatalf("expected ErrServiceAllocated, got %v", err)
	}
	if holder == nil || holder.App != "a" {
		t.Fatal("expected holder info on service conflict")
	}
}

func TestList(t *testing.T) {
	s := newTestStore(t)

	s.Allocate(model.AllocateRequest{App: "a", Instance: "i1", Service: "web", Port: 3000}, 3000, 9999)
	s.Allocate(model.AllocateRequest{App: "a", Instance: "i1", Service: "db", Port: 3001}, 3000, 9999)
	s.Allocate(model.AllocateRequest{App: "b", Instance: "i2", Service: "web", Port: 3002}, 3000, 9999)

	all, err := s.List(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 allocations, got %d", len(all))
	}

	filtered, err := s.List(Filter{App: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 allocations for app=a, got %d", len(filtered))
	}
}

func TestGetByPort(t *testing.T) {
	s := newTestStore(t)

	s.Allocate(model.AllocateRequest{App: "a", Instance: "i", Service: "s", Port: 4000}, 3000, 9999)

	a, err := s.GetByPort(4000)
	if err != nil {
		t.Fatal(err)
	}
	if a.App != "a" {
		t.Fatalf("expected app=a, got %s", a.App)
	}

	_, err = s.GetByPort(9999)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteByID(t *testing.T) {
	s := newTestStore(t)

	a, _ := s.Allocate(model.AllocateRequest{App: "a", Instance: "i", Service: "s", Port: 4000}, 3000, 9999)

	if err := s.DeleteByID(a.ID); err != nil {
		t.Fatal(err)
	}

	_, err := s.GetByPort(4000)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	if err := s.DeleteByID(9999); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-existent ID, got %v", err)
	}
}

func TestDeleteByFilter(t *testing.T) {
	s := newTestStore(t)

	s.Allocate(model.AllocateRequest{App: "a", Instance: "i1", Service: "web", Port: 3000}, 3000, 9999)
	s.Allocate(model.AllocateRequest{App: "a", Instance: "i1", Service: "db", Port: 3001}, 3000, 9999)
	s.Allocate(model.AllocateRequest{App: "a", Instance: "i2", Service: "web", Port: 3002}, 3000, 9999)

	n, err := s.DeleteByFilter(Filter{App: "a", Instance: "i1"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 deleted, got %d", n)
	}

	all, _ := s.List(Filter{})
	if len(all) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(all))
	}
}

func TestDeleteByFilterRequiresFilter(t *testing.T) {
	s := newTestStore(t)

	_, err := s.DeleteByFilter(Filter{})
	if err == nil {
		t.Fatal("expected error for empty filter")
	}
}

func TestAllocatePortBusy(t *testing.T) {
	s := newTestStore(t)
	s.PortChecker = func(port int) bool { return port != 5000 }

	_, err := s.Allocate(model.AllocateRequest{
		App: "a", Instance: "i", Service: "s", Port: 5000,
	}, 3000, 9999)
	if err != ErrPortBusy {
		t.Fatalf("expected ErrPortBusy, got %v", err)
	}
}

func TestAllocateAutoAssignSkipsBusy(t *testing.T) {
	s := newTestStore(t)
	s.PortChecker = func(port int) bool { return port != 3000 }

	a, err := s.Allocate(model.AllocateRequest{
		App: "a", Instance: "i", Service: "s",
	}, 3000, 9999)
	if err != nil {
		t.Fatal(err)
	}
	if a.Port != 3001 {
		t.Fatalf("expected auto-assigned port 3001 (3000 busy), got %d", a.Port)
	}
}
