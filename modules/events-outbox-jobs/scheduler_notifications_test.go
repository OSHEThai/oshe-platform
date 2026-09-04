package eventoutbox_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	eventoutbox "github.com/oshethai/oshe-platform/modules/events-outbox-jobs"
)

func newSchedulerClock(initial time.Time) (func() time.Time, func(d time.Duration)) {
	curr := initial
	var mu sync.Mutex
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return curr
	}
	advance := func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		curr = curr.Add(d)
	}
	return clock, advance
}

func TestScheduler_DelayedJob(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	clock, advance := newSchedulerClock(baseTime)

	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{Clock: clock})

	var jobExecuted bool
	_ = s.RegisterJobHandler("cleanup", func(job eventoutbox.ScheduledJob) error {
		jobExecuted = true
		return nil
	})

	job := eventoutbox.ScheduledJob{
		JobID:       "job_delayed_001",
		TenantID:    "ten_alpha",
		JobType:     "cleanup",
		ScheduledAt: baseTime,
		DueAt:       baseTime.Add(1 * time.Hour),
	}
	if err := s.ScheduleJob(job); err != nil {
		t.Fatalf("ScheduleJob failed: %v", err)
	}

	// At T0 (initial): not due
	executed, err := s.ExecuteDueJobs()
	if err != nil {
		t.Fatalf("ExecuteDueJobs failed: %v", err)
	}
	if len(executed) != 0 || jobExecuted {
		t.Fatalf("expected 0 executed jobs at T0, got %d", len(executed))
	}

	// At T0 + 30m: still not due
	advance(30 * time.Minute)
	executed, _ = s.ExecuteDueJobs()
	if len(executed) != 0 || jobExecuted {
		t.Fatalf("expected 0 executed jobs at T0+30m, got %d", len(executed))
	}

	// At T0 + 65m: due
	advance(35 * time.Minute)
	executed, err = s.ExecuteDueJobs()
	if err != nil {
		t.Fatalf("ExecuteDueJobs failed: %v", err)
	}
	if len(executed) != 1 || !jobExecuted {
		t.Fatalf("expected 1 executed job at T0+65m, got %d", len(executed))
	}

	snap, _ := s.GetJob("job_delayed_001")
	if snap.State != eventoutbox.JobStateCompleted {
		t.Errorf("expected JobStateCompleted, got %s", snap.State)
	}
	if snap.ExecutedAt == nil {
		t.Error("expected non-nil ExecutedAt")
	}
}

func TestScheduler_ClockSkew(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	clock, advance := newSchedulerClock(baseTime)

	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{Clock: clock})

	// DueAt before ScheduledAt -> clock skew rejection
	jobSkewed := eventoutbox.ScheduledJob{
		JobID:       "job_skew",
		TenantID:    "ten_alpha",
		JobType:     "backup",
		ScheduledAt: baseTime,
		DueAt:       baseTime.Add(-10 * time.Minute), // backwards in time
	}
	err := s.ScheduleJob(jobSkewed)
	if !errors.Is(err, eventoutbox.ErrClockSkewDetected) {
		t.Fatalf("expected ErrClockSkewDetected for DueAt before ScheduledAt, got: %v", err)
	}

	// Valid future job scheduled
	jobFuture := eventoutbox.ScheduledJob{
		JobID:       "job_future",
		TenantID:    "ten_alpha",
		JobType:     "backup",
		ScheduledAt: baseTime,
		DueAt:       baseTime.Add(2 * time.Hour),
	}
	_ = s.ScheduleJob(jobFuture)

	// Clock advances to T+1h, then jumps backwards by 30m
	advance(1 * time.Hour)
	advance(-30 * time.Minute)

	// Job at T+2h must NOT execute prematurely
	executed, _ := s.ExecuteDueJobs()
	if len(executed) != 0 {
		t.Fatalf("job executed prematurely despite clock skew backwards: %v", executed)
	}
}

func TestScheduler_DuplicateJobIdentity(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{})

	job := eventoutbox.ScheduledJob{
		JobID:       "job_unique_001",
		TenantID:    "ten_alpha",
		JobType:     "sync",
		ScheduledAt: time.Now(),
		DueAt:       time.Now().Add(10 * time.Minute),
	}

	if err := s.ScheduleJob(job); err != nil {
		t.Fatalf("first ScheduleJob failed: %v", err)
	}

	// Attempt duplicate schedule
	err := s.ScheduleJob(job)
	if !errors.Is(err, eventoutbox.ErrDuplicateJobID) {
		t.Fatalf("expected ErrDuplicateJobID on duplicate schedule, got: %v", err)
	}
}

func TestNotification_ProhibitionOfExternalDeliveryActivation(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{})

	prohibitedChannels := []eventoutbox.NotificationChannel{
		"EMAIL",
		"SMS",
		"PUSH",
		"WEBHOOK",
		"SLACK",
		"PAGERDUTY",
	}

	for _, ch := range prohibitedChannels {
		req := eventoutbox.NotificationRequest{
			RequestID: "notif_prohibited_" + string(ch),
			TenantID:  "ten_alpha",
			Recipient: "user@example.com",
			Subject:   "Alert",
			Body:      "Details",
			Channel:   ch,
		}

		_, err := s.SendNotification(req)
		if !errors.Is(err, eventoutbox.ErrUnsupportedExternalChannel) {
			t.Errorf("for channel %s: expected ErrUnsupportedExternalChannel, got: %v", ch, err)
		}
	}
}

func TestNotification_LocalSinkDelivery(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{})

	req := eventoutbox.NotificationRequest{
		RequestID: "notif_local_001",
		TenantID:  "ten_alpha",
		Recipient: "local-operator",
		Subject:   "Inspection Scheduled",
		Body:      "Inspection scheduled for platform 2",
		Channel:   eventoutbox.ChannelLocalMemory,
	}

	res, err := s.SendNotification(req)
	if err != nil {
		t.Fatalf("SendNotification failed: %v", err)
	}

	if res.Status != eventoutbox.NotificationStatusDelivered {
		t.Errorf("expected NotificationStatusDelivered, got %s", res.Status)
	}
	if res.Attempts != 1 {
		t.Errorf("expected Attempts=1, got %d", res.Attempts)
	}
	if res.DeliveredAt == nil {
		t.Error("expected non-nil DeliveredAt")
	}

	deliveries := s.GetLocalSinkDeliveries()
	if len(deliveries) != 1 || deliveries[0].RequestID != "notif_local_001" {
		t.Fatalf("expected 1 delivery in local sink, got %d", len(deliveries))
	}
}

func TestNotification_LocalSinkDeliveryFailureAndRetryProgression(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{MaxNotificationAttempts: 3})

	// Simulate sink failure
	s.SetLocalSinkFailure(errors.New("local memory buffer full"))

	req := eventoutbox.NotificationRequest{
		RequestID: "notif_fail_001",
		TenantID:  "ten_alpha",
		Recipient: "local-operator",
		Subject:   "Warning",
		Body:      "Message",
		Channel:   eventoutbox.ChannelLocalMemory,
	}

	// Attempt 1: fails -> retrying
	res, err := s.SendNotification(req)
	if err == nil {
		t.Fatal("expected error on attempt 1")
	}
	if res.Status != eventoutbox.NotificationStatusRetrying || res.Attempts != 1 {
		t.Errorf("expected RETRYING attempts=1, got status=%s attempts=%d", res.Status, res.Attempts)
	}
	if res.LastError == "" || res.Diagnostics == "" {
		t.Error("expected visible LastError and Diagnostics")
	}

	// Attempt 2: fails -> retrying
	res, err = s.SendNotification(req)
	if err == nil {
		t.Fatal("expected error on attempt 2")
	}
	if res.Status != eventoutbox.NotificationStatusRetrying || res.Attempts != 2 {
		t.Errorf("expected RETRYING attempts=2, got status=%s attempts=%d", res.Status, res.Attempts)
	}

	// Attempt 3: fails -> quarantined
	res, err = s.SendNotification(req)
	if !errors.Is(err, eventoutbox.ErrNotificationMaxRetries) {
		t.Fatalf("expected ErrNotificationMaxRetries on 3rd attempt, got: %v", err)
	}
	if res.Status != eventoutbox.NotificationStatusQuarantined || res.Attempts != 3 {
		t.Errorf("expected QUARANTINED attempts=3, got status=%s attempts=%d", res.Status, res.Attempts)
	}

	// Attempt 4 without replay: rejected
	_, err = s.SendNotification(req)
	if !errors.Is(err, eventoutbox.ErrNotificationRetryLimitReached) {
		t.Fatalf("expected ErrNotificationRetryLimitReached, got: %v", err)
	}
}

func TestNotification_DuplicateRequestRejection(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{})

	req := eventoutbox.NotificationRequest{
		RequestID: "notif_dup_001",
		TenantID:  "ten_alpha",
		Recipient: "local-op",
		Subject:   "Msg",
		Body:      "Body",
		Channel:   eventoutbox.ChannelLocalMemory,
	}

	_, err := s.SendNotification(req)
	if err != nil {
		t.Fatalf("first notification failed: %v", err)
	}

	// Duplicate send
	_, err = s.SendNotification(req)
	if !errors.Is(err, eventoutbox.ErrDuplicateNotificationRequest) {
		t.Fatalf("expected ErrDuplicateNotificationRequest, got: %v", err)
	}
}

func TestNotification_CrossTenantAndUnauthorized(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{})

	req := eventoutbox.NotificationRequest{
		RequestID: "notif_tenant_001",
		TenantID:  "ten_alpha",
		Recipient: "local-op",
		Subject:   "Msg",
		Body:      "Body",
		Channel:   eventoutbox.ChannelLocalMemory,
	}
	_, _ = s.SendNotification(req)

	// Cross-tenant access
	_, err := s.GetNotification("notif_tenant_001", "ten_bravo")
	if !errors.Is(err, eventoutbox.ErrCrossTenantNotification) {
		t.Fatalf("expected ErrCrossTenantNotification, got: %v", err)
	}

	// Replay under mismatched tenant
	_, err = s.ReplayNotification("notif_tenant_001", "ten_bravo", "admin")
	if !errors.Is(err, eventoutbox.ErrCrossTenantNotification) {
		t.Fatalf("expected ErrCrossTenantNotification on replay, got: %v", err)
	}

	// Replay with blank caller identity
	_, err = s.ReplayNotification("notif_tenant_001", "ten_alpha", "")
	if !errors.Is(err, eventoutbox.ErrUnauthorizedNotificationAccess) {
		t.Fatalf("expected ErrUnauthorizedNotificationAccess, got: %v", err)
	}
}

func TestNotification_ReplayProgression(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{MaxNotificationAttempts: 3})

	// Quarantine a notification
	s.SetLocalSinkFailure(errors.New("sink down"))
	req := eventoutbox.NotificationRequest{
		RequestID: "notif_replay_001",
		TenantID:  "ten_alpha",
		Recipient: "local-op",
		Subject:   "Notice",
		Body:      "Body",
		Channel:   eventoutbox.ChannelLocalLog,
	}

	for i := 0; i < 3; i++ {
		_, _ = s.SendNotification(req)
	}

	notif, _ := s.GetNotification("notif_replay_001", "ten_alpha")
	if notif.Status != eventoutbox.NotificationStatusQuarantined {
		t.Fatalf("expected quarantined notification, got %s", notif.Status)
	}

	// Clear sink failure and replay
	s.SetLocalSinkFailure(nil)
	res, err := s.ReplayNotification("notif_replay_001", "ten_alpha", "admin_alice")
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	if res.Status != eventoutbox.NotificationStatusDelivered {
		t.Errorf("expected StatusDelivered after replay, got %s", res.Status)
	}
}
