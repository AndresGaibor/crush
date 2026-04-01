package cron

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Runner ejecuta un cron job y devuelve un resumen textual.
type Runner func(context.Context, *CronJob) (string, error)

// Scheduler administra la ejecución de jobs programados.
type Scheduler struct {
	mu          sync.RWMutex
	persistence *Persistence
	jobs        map[string]*CronJob
	tickers     map[string]*time.Ticker
	timers      map[string]*time.Timer
	stopCh      map[string]chan struct{}
	config      CronConfig
	runner      Runner
}

// NewScheduler crea un scheduler nuevo.
func NewScheduler(persistence *Persistence, cfg CronConfig, runner Runner) *Scheduler {
	return &Scheduler{
		persistence: persistence,
		jobs:        make(map[string]*CronJob),
		tickers:     make(map[string]*time.Ticker),
		timers:      make(map[string]*time.Timer),
		stopCh:      make(map[string]chan struct{}),
		config:      cfg,
		runner:      runner,
	}
}

// SetRunner reemplaza la función ejecutora.
func (s *Scheduler) SetRunner(runner Runner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runner = runner
}

// LoadJobs carga los jobs persistidos y agenda los activos.
func (s *Scheduler) LoadJobs() error {
	jobs, err := s.persistence.List(nil)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range jobs {
		s.jobs[job.ID] = job
		if job.Status == CronEnabled {
			if err := s.scheduleLocked(job); err != nil {
				slog.Warn("Failed to schedule cron job", "id", job.ID, "error", err)
			}
		}
	}

	slog.Info("Cron jobs loaded", "total", len(jobs))
	return nil
}

// AddJob persiste y agenda un job nuevo.
func (s *Scheduler) AddJob(job *CronJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if count, err := s.persistence.Count(); err == nil && count >= s.config.MaxJobs {
		return fmt.Errorf("maximum cron jobs reached (%d)", s.config.MaxJobs)
	}

	if job.Status == "" {
		job.Status = CronEnabled
	}
	if job.Schedule.TZ == "" {
		job.Schedule.TZ = "UTC"
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.UpdatedAt = time.Now()

	if err := validateSchedule(job.Schedule); err != nil {
		return err
	}
	if err := s.persistence.Create(job); err != nil {
		return err
	}

	s.jobs[job.ID] = job
	if job.Status == CronEnabled {
		if err := s.scheduleLocked(job); err != nil {
			return err
		}
	}

	return nil
}

// RemoveJob detiene y elimina un job.
func (s *Scheduler) RemoveJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopLocked(id)
	delete(s.jobs, id)
	return s.persistence.Delete(id)
}

// EnableJob habilita un job existente.
func (s *Scheduler) EnableJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("cron job %q not found", id)
	}

	job.Status = CronEnabled
	job.UpdatedAt = time.Now()
	if err := s.persistence.Update(job); err != nil {
		return err
	}
	return s.scheduleLocked(job)
}

// DisableJob deshabilita un job existente.
func (s *Scheduler) DisableJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("cron job %q not found", id)
	}

	job.Status = CronDisabled
	job.UpdatedAt = time.Now()
	s.stopLocked(id)
	return s.persistence.Update(job)
}

// ListJobs devuelve los jobs cargados en memoria.
func (s *Scheduler) ListJobs(status *CronStatus) []*CronJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*CronJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		if status != nil && job.Status != *status {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs
}

// Stop detiene todos los timers y tickers.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id := range s.tickers {
		s.stopLocked(id)
	}
	for id := range s.timers {
		s.stopLocked(id)
	}
}

func (s *Scheduler) scheduleLocked(job *CronJob) error {
	s.stopLocked(job.ID)

	switch job.Schedule.Kind {
	case ScheduleFixedRate:
		intervalSec, err := parsePositiveSeconds(job.Schedule.Expr)
		if err != nil {
			return err
		}
		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		stopCh := make(chan struct{})
		s.tickers[job.ID] = ticker
		s.stopCh[job.ID] = stopCh
		job.NextRunAt = ptrTime(time.Now().Add(time.Duration(intervalSec) * time.Second))

		go func() {
			for {
				select {
				case <-ticker.C:
					s.execute(job.ID)
				case <-stopCh:
					ticker.Stop()
					return
				}
			}
		}()
		return s.persistence.Update(job)

	case ScheduleCron:
		if err := validateCronExpr(job.Schedule.Expr); err != nil {
			return err
		}
		ticker := time.NewTicker(time.Minute)
		stopCh := make(chan struct{})
		s.tickers[job.ID] = ticker
		s.stopCh[job.ID] = stopCh
		if next := nextCronRun(job.Schedule.Expr, time.Now()); next != nil {
			job.NextRunAt = next
		}
		if err := s.persistence.Update(job); err != nil {
			return err
		}

		go func() {
			for {
				select {
				case <-ticker.C:
					if shouldRunCron(job.Schedule.Expr, time.Now()) {
						s.execute(job.ID)
					}
				case <-stopCh:
					ticker.Stop()
					return
				}
			}
		}()
		return nil

	case ScheduleOneTime:
		when, err := parseOneTime(job.Schedule.Expr)
		if err != nil {
			return err
		}

		delay := time.Until(when)
		if delay <= 0 {
			go func() {
				s.execute(job.ID)
				_ = s.DisableJob(job.ID)
			}()
			return nil
		}

		timer := time.NewTimer(delay)
		stopCh := make(chan struct{})
		s.timers[job.ID] = timer
		s.stopCh[job.ID] = stopCh
		job.NextRunAt = &when
		if err := s.persistence.Update(job); err != nil {
			return err
		}

		go func() {
			select {
			case <-timer.C:
				s.execute(job.ID)
				_ = s.DisableJob(job.ID)
			case <-stopCh:
				timer.Stop()
			}
		}()
		return nil

	default:
		return fmt.Errorf("unsupported schedule kind %q", job.Schedule.Kind)
	}
}

func (s *Scheduler) stopLocked(id string) {
	if ticker, ok := s.tickers[id]; ok {
		ticker.Stop()
		delete(s.tickers, id)
	}
	if timer, ok := s.timers[id]; ok {
		timer.Stop()
		delete(s.timers, id)
	}
	if stopCh, ok := s.stopCh[id]; ok {
		close(stopCh)
		delete(s.stopCh, id)
	}
}

func (s *Scheduler) execute(jobID string) {
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	runner := s.runner
	if runner == nil {
		job.Status = CronError
		job.ErrorCount++
		job.LastError = "cron runner is not configured"
		job.UpdatedAt = time.Now()
		_ = s.persistence.Update(job)
		s.mu.Unlock()
		return
	}

	job.Status = CronRunning
	job.UpdatedAt = time.Now()
	_ = s.persistence.Update(job)
	timeout := time.Duration(s.config.DefaultTimeoutSec) * time.Second
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	s.mu.Unlock()

	start := time.Now()
	result, err := runner(ctx, job)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	job.LastRunAt = &now
	job.RunCount++
	job.UpdatedAt = now
	if err != nil {
		job.Status = CronError
		job.ErrorCount++
		job.LastError = err.Error()
		slog.Warn("Cron job failed", "id", job.ID, "error", err, "duration", time.Since(start))
	} else {
		job.Status = CronEnabled
		job.LastError = ""
		slog.Info("Cron job executed", "id", job.ID, "result_length", len(result), "duration", time.Since(start))
	}

	switch job.Schedule.Kind {
	case ScheduleFixedRate:
		if intervalSec, err := parsePositiveSeconds(job.Schedule.Expr); err == nil {
			job.NextRunAt = ptrTime(now.Add(time.Duration(intervalSec) * time.Second))
		}
	case ScheduleCron:
		job.NextRunAt = nextCronRun(job.Schedule.Expr, now)
	}

	if err := s.persistence.Update(job); err != nil {
		slog.Warn("Failed to persist cron job after execution", "id", job.ID, "error", err)
	}
}

func parsePositiveSeconds(expr string) (int64, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return 0, fmt.Errorf("schedule expression is required")
	}
	secs, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || secs <= 0 {
		return 0, fmt.Errorf("fixed_rate expects a positive number of seconds")
	}
	return secs, nil
}

func parseOneTime(expr string) (time.Time, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("schedule expression is required")
	}
	if when, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return when, nil
	}
	if secs, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return time.Unix(secs, 0), nil
	}
	return time.Time{}, fmt.Errorf("one_time expects an RFC3339 timestamp or unix seconds")
}

func validateSchedule(schedule CronSchedule) error {
	if schedule.Kind == "" {
		return fmt.Errorf("schedule kind is required")
	}
	if strings.TrimSpace(schedule.Expr) == "" {
		return fmt.Errorf("schedule expression is required")
	}
	switch schedule.Kind {
	case ScheduleCron:
		return validateCronExpr(schedule.Expr)
	case ScheduleFixedRate:
		_, err := parsePositiveSeconds(schedule.Expr)
		return err
	case ScheduleOneTime:
		_, err := parseOneTime(schedule.Expr)
		return err
	default:
		return fmt.Errorf("unsupported schedule kind %q", schedule.Kind)
	}
}

func validateCronExpr(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 && len(fields) != 6 {
		return fmt.Errorf("cron expects 5 fields (or 6 with seconds)")
	}
	return nil
}

func shouldRunCron(expr string, now time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) == 6 {
		fields = fields[1:]
	}
	if len(fields) != 5 {
		return false
	}

	checks := []struct {
		field string
		value int
	}{
		{fields[0], now.Minute()},
		{fields[1], now.Hour()},
		{fields[2], now.Day()},
		{fields[3], int(now.Month())},
		{fields[4], int(now.Weekday())},
	}

	for _, check := range checks {
		if !matchCronField(check.field, check.value) {
			return false
		}
	}
	return true
}

func nextCronRun(expr string, now time.Time) *time.Time {
	rounded := now.Truncate(time.Minute)
	for i := 1; i <= 7*24*60; i++ {
		candidate := rounded.Add(time.Duration(i) * time.Minute)
		if shouldRunCron(expr, candidate) {
			return &candidate
		}
	}
	return nil
}

func matchCronField(field string, value int) bool {
	field = strings.TrimSpace(field)
	if field == "*" {
		return true
	}

	for _, item := range strings.Split(field, ",") {
		item = strings.TrimSpace(item)
		if item == "*" {
			return true
		}
		if strings.HasPrefix(item, "*/") {
			step, err := strconv.Atoi(strings.TrimPrefix(item, "*/"))
			if err == nil && step > 0 && value%step == 0 {
				return true
			}
			continue
		}
		if strings.Contains(item, "-") {
			parts := strings.SplitN(item, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && value >= start && value <= end {
				return true
			}
			continue
		}
		if n, err := strconv.Atoi(item); err == nil && n == value {
			return true
		}
	}
	return false
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
