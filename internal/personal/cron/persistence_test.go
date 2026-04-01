package cron

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newCronTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestPersistenceCRUD(t *testing.T) {
	db := newCronTestDB(t)
	p, err := NewPersistence(db)
	if err != nil {
		t.Fatalf("NewPersistence() error: %v", err)
	}

	now := time.Now()
	job := &CronJob{
		ID:        "cron_1",
		Name:      "Daily check",
		Schedule:  CronSchedule{Kind: ScheduleFixedRate, Expr: "60", TZ: "UTC"},
		Prompt:    "Run a daily check",
		Status:    CronEnabled,
		Priority:  5,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := p.Create(job); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := p.Get("cron_1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Name != "Daily check" {
		t.Fatalf("unexpected name: %s", got.Name)
	}

	got.Name = "Updated"
	if err := p.Update(got); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	got, err = p.Get("cron_1")
	if err != nil {
		t.Fatalf("Get() after update error: %v", err)
	}
	if got.Name != "Updated" {
		t.Fatalf("expected updated name, got %s", got.Name)
	}

	if err := p.Delete("cron_1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if _, err := p.Get("cron_1"); err == nil {
		t.Fatal("expected error after delete")
	}
}
