package cron

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSchedulerFixedRateExecution(t *testing.T) {
	db := newCronTestDB(t)
	p, err := NewPersistence(db)
	if err != nil {
		t.Fatalf("NewPersistence() error: %v", err)
	}

	var runs int32
	sched := NewScheduler(p, DefaultCronConfig(), func(ctx context.Context, job *CronJob) (string, error) {
		atomic.AddInt32(&runs, 1)
		return "ok", nil
	})

	job := &CronJob{
		ID:        "cron_fast",
		Name:      "Fast",
		Schedule:  CronSchedule{Kind: ScheduleFixedRate, Expr: "1", TZ: "UTC"},
		Prompt:    "Ping",
		Status:    CronEnabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := sched.AddJob(job); err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	defer sched.Stop()

	time.Sleep(1300 * time.Millisecond)

	if atomic.LoadInt32(&runs) == 0 {
		t.Fatal("expected at least one execution")
	}
}

func TestSchedulerDisableAndEnable(t *testing.T) {
	db := newCronTestDB(t)
	p, err := NewPersistence(db)
	if err != nil {
		t.Fatalf("NewPersistence() error: %v", err)
	}

	sched := NewScheduler(p, DefaultCronConfig(), func(ctx context.Context, job *CronJob) (string, error) {
		return "ok", nil
	})
	defer sched.Stop()

	job := &CronJob{
		ID:        "cron_toggle",
		Name:      "Toggle",
		Schedule:  CronSchedule{Kind: ScheduleFixedRate, Expr: "60", TZ: "UTC"},
		Prompt:    "Ping",
		Status:    CronEnabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sched.AddJob(job); err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	if err := sched.DisableJob(job.ID); err != nil {
		t.Fatalf("DisableJob() error: %v", err)
	}
	if err := sched.EnableJob(job.ID); err != nil {
		t.Fatalf("EnableJob() error: %v", err)
	}
}
