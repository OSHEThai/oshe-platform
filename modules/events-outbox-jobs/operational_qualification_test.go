package eventoutbox_test

import (
	"errors"
	"testing"
	"time"

	eventoutbox "github.com/oshethai/oshe-platform/modules/events-outbox-jobs"
)

func TestOperationalQualification_RollbackNoSilentStateMutation(t *testing.T) {
	outbox := eventoutbox.NewOutbox()
	tx, err := outbox.BeginTx("ten_alpha")
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	ev1 := validEventInput("evt_rollback_001")
	if err := tx.Stage(ev1); err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	ev2 := validEventInput("evt_rollback_002")
	if err := tx.Stage(ev2); err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// Staged events are completely invisible before commit
	if len(outbox.CommittedEvents()) != 0 {
		t.Fatalf("pre-commit leakage: expected 0 committed events, got %d", len(outbox.CommittedEvents()))
	}

	// Rollback transaction
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Ensure zero events committed or visible in outbox
	if len(outbox.CommittedEvents()) != 0 {
		t.Fatalf("silent state mutation: expected 0 events after rollback, got %d", len(outbox.CommittedEvents()))
	}
	tenantEvents, err := outbox.CommittedEventsForTenant("ten_alpha")
	if err != nil {
		t.Fatalf("CommittedEventsForTenant failed: %v", err)
	}
	if len(tenantEvents) != 0 {
		t.Fatalf("expected 0 tenant events after rollback, got %d", len(tenantEvents))
	}

	// Post-rollback calls must fail closed
	if err := tx.Stage(validEventInput("evt_rollback_003")); !errors.Is(err, eventoutbox.ErrTxClosed) {
		t.Errorf("expected ErrTxClosed on Stage after rollback, got: %v", err)
	}
	if err := tx.Commit(); !errors.Is(err, eventoutbox.ErrTxClosed) {
		t.Errorf("expected ErrTxClosed on Commit after rollback, got: %v", err)
	}
}

func TestOperationalQualification_DuplicateAndPoisonReplayHandling(t *testing.T) {
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_qual_poison"

	failHandler := true
	_ = d.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		if failHandler {
			return errors.New("poison payload: unparseable format")
		}
		return nil
	})

	env := sampleEnvelope("evt_qual_poison_001", "ten_alpha")

	// Attempt 1, 2, 3: handler fails, event quarantined
	for i := 1; i <= 3; i++ {
		_, err := d.Dispatch(consumerID, env)
		if i < 3 && err == nil {
			t.Fatalf("expected failure on attempt %d", i)
		}
	}

	rec, err := d.GetDeliveryRecord(consumerID, "evt_qual_poison_001")
	if err != nil {
		t.Fatalf("GetDeliveryRecord failed: %v", err)
	}
	if rec.Status != eventoutbox.StatusQuarantined {
		t.Fatalf("expected StatusQuarantined, got %s", rec.Status)
	}

	// Further normal dispatch attempts without replay are blocked
	_, err = d.Dispatch(consumerID, env)
	if !errors.Is(err, eventoutbox.ErrRetryLimitReached) {
		t.Fatalf("expected ErrRetryLimitReached on quarantined event, got: %v", err)
	}

	// Fix poison condition and perform authorized replay
	failHandler = false
	recReplay, err := d.Replay(consumerID, "evt_qual_poison_001", "ten_alpha", "admin_operator", env)
	if err != nil {
		t.Fatalf("authorized replay failed: %v", err)
	}
	if recReplay.Status != eventoutbox.StatusDelivered {
		t.Errorf("expected StatusDelivered after replay, got %s", recReplay.Status)
	}
	if recReplay.ReplayedCount != 1 {
		t.Errorf("expected ReplayedCount 1, got %d", recReplay.ReplayedCount)
	}

	// Subsequent replay on delivered event must be rejected
	_, err = d.Replay(consumerID, "evt_qual_poison_001", "ten_alpha", "admin_operator", env)
	if !errors.Is(err, eventoutbox.ErrEventNotQuarantined) {
		t.Fatalf("expected ErrEventNotQuarantined on delivered replay, got: %v", err)
	}
}

func TestOperationalQualification_SchemaMismatchDenial(t *testing.T) {
	outbox := eventoutbox.NewOutbox()
	tx, _ := outbox.BeginTx("ten_alpha")

	// 1. Incompatible schema version
	incompatibleSchema := validEventInput("evt_schema_bad")
	incompatibleSchema.SchemaVersion = "2.0.0"
	err := tx.Stage(incompatibleSchema)
	if !errors.Is(err, eventoutbox.ErrIncompatibleSchemaVersion) {
		t.Fatalf("expected ErrIncompatibleSchemaVersion, got: %v", err)
	}

	// 2. Blank schema version
	blankSchema := validEventInput("evt_schema_blank")
	blankSchema.SchemaVersion = ""
	err = tx.Stage(blankSchema)
	if !errors.Is(err, eventoutbox.ErrIncompatibleSchemaVersion) {
		t.Fatalf("expected ErrIncompatibleSchemaVersion for blank schema, got: %v", err)
	}

	// 3. Unsupported envelope version
	badEnvelope := validEventInput("evt_env_bad")
	badEnvelope.EnvelopeVersion = "9.9.9"
	err = tx.Stage(badEnvelope)
	if !errors.Is(err, eventoutbox.ErrUnsupportedEnvelopeVersion) {
		t.Fatalf("expected ErrUnsupportedEnvelopeVersion, got: %v", err)
	}

	// Zero events staged
	_ = tx.Commit()
	if len(outbox.CommittedEvents()) != 0 {
		t.Fatalf("expected 0 committed events following schema rejections, got %d", len(outbox.CommittedEvents()))
	}
}

func TestOperationalQualification_DelayedJobBehavior(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC)
	clock, advance := newSchedulerClock(baseTime)

	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{Clock: clock})

	var executionCount int
	_ = s.RegisterJobHandler("report_generation", func(job eventoutbox.ScheduledJob) error {
		executionCount++
		return nil
	})

	job := eventoutbox.ScheduledJob{
		JobID:       "job_qual_delayed",
		TenantID:    "ten_alpha",
		JobType:     "report_generation",
		ScheduledAt: baseTime,
		DueAt:       baseTime.Add(2 * time.Hour),
	}
	if err := s.ScheduleJob(job); err != nil {
		t.Fatalf("ScheduleJob failed: %v", err)
	}

	// Sub-checks: T0, T+1h, T+1h59m must not execute
	checkpoints := []time.Duration{0, 60 * time.Minute, 59 * time.Minute}
	for _, step := range checkpoints {
		advance(step)
		executed, _ := s.ExecuteDueJobs()
		if len(executed) != 0 || executionCount != 0 {
			t.Fatalf("job executed prematurely before DueAt: count=%d", executionCount)
		}
	}

	// At DueAt (T+2h = T0 + 119m + 1m): job executes exactly once
	advance(1 * time.Minute)
	executed, err := s.ExecuteDueJobs()
	if err != nil {
		t.Fatalf("ExecuteDueJobs failed: %v", err)
	}
	if len(executed) != 1 || executionCount != 1 {
		t.Fatalf("expected exactly 1 execution at DueAt, got %d", executionCount)
	}

	// Subsequent runs at T+3h must not re-execute completed job
	advance(1 * time.Hour)
	executed, _ = s.ExecuteDueJobs()
	if len(executed) != 0 || executionCount != 1 {
		t.Fatalf("completed job re-executed on subsequent tick: count=%d", executionCount)
	}
}

func TestOperationalQualification_BoundedRetryAndQuarantine(t *testing.T) {
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{MaxNotificationAttempts: 3})

	// Inject delivery sink failure
	s.SetLocalSinkFailure(errors.New("connection reset by peer"))

	req := eventoutbox.NotificationRequest{
		RequestID: "notif_qual_retry_001",
		TenantID:  "ten_alpha",
		Recipient: "ops-duty",
		Subject:   "High Priority Incident",
		Body:      "Incident details",
		Channel:   eventoutbox.ChannelLocalMemory,
	}

	// Attempt 1: RETRYING
	res, err := s.SendNotification(req)
	if err == nil || res.Status != eventoutbox.NotificationStatusRetrying || res.Attempts != 1 {
		t.Fatalf("unexpected state attempt 1: status=%s attempts=%d", res.Status, res.Attempts)
	}

	// Attempt 2: RETRYING
	res, err = s.SendNotification(req)
	if err == nil || res.Status != eventoutbox.NotificationStatusRetrying || res.Attempts != 2 {
		t.Fatalf("unexpected state attempt 2: status=%s attempts=%d", res.Status, res.Attempts)
	}

	// Attempt 3: QUARANTINED
	res, err = s.SendNotification(req)
	if !errors.Is(err, eventoutbox.ErrNotificationMaxRetries) {
		t.Fatalf("expected ErrNotificationMaxRetries on 3rd attempt, got: %v", err)
	}
	if res.Status != eventoutbox.NotificationStatusQuarantined || res.Attempts != 3 {
		t.Fatalf("expected QUARANTINED attempts=3, got status=%s attempts=%d", res.Status, res.Attempts)
	}
	if res.LastError == "" || res.Diagnostics == "" {
		t.Error("expected visible LastError and Diagnostics upon quarantine")
	}

	// Originating business state remains uncorrupted: local sink has 0 deliveries
	if len(s.GetLocalSinkDeliveries()) != 0 {
		t.Fatalf("local sink received corrupted delivery: count=%d", len(s.GetLocalSinkDeliveries()))
	}
}

func TestOperationalQualification_TenantAndAuthorizationNonBypass(t *testing.T) {
	// 1. Outbox cross-tenant staging denial
	outbox := eventoutbox.NewOutbox()
	tx, _ := outbox.BeginTx("ten_alpha")
	evBravo := validEventInput("evt_cross_tenant_001")
	evBravo.TenantID = "ten_bravo"
	if err := tx.Stage(evBravo); !errors.Is(err, eventoutbox.ErrCrossTenantAssociation) {
		t.Fatalf("expected ErrCrossTenantAssociation on cross-tenant event staging, got: %v", err)
	}

	// 2. Dispatcher cross-tenant dispatch denial
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{})
	_ = d.RegisterConsumer("cons_ten_a", "ten_alpha", func(env eventoutbox.EventEnvelope) error { return nil })
	_, err := d.Dispatch("cons_ten_a", sampleEnvelope("evt_cross_002", "ten_bravo"))
	if !errors.Is(err, eventoutbox.ErrCrossTenantDispatch) {
		t.Fatalf("expected ErrCrossTenantDispatch, got: %v", err)
	}

	// 3. Dispatcher cross-tenant replay denial
	_ = d.RegisterConsumer("cons_ten_quar", "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		return errors.New("quar")
	})
	envAlpha := sampleEnvelope("evt_quar_003", "ten_alpha")
	for i := 0; i < 3; i++ {
		_, _ = d.Dispatch("cons_ten_quar", envAlpha)
	}
	_, err = d.Replay("cons_ten_quar", "evt_quar_003", "ten_bravo", "admin_alice", envAlpha)
	if !errors.Is(err, eventoutbox.ErrCrossTenantReplay) {
		t.Fatalf("expected ErrCrossTenantReplay on mismatched tenant, got: %v", err)
	}

	// 4. Dispatcher unauthorized replay denial (blank identity)
	_, err = d.Replay("cons_ten_quar", "evt_quar_003", "ten_alpha", "", envAlpha)
	if !errors.Is(err, eventoutbox.ErrUnauthorizedReplay) {
		t.Fatalf("expected ErrUnauthorizedReplay on blank actor, got: %v", err)
	}

	// 5. Notification cross-tenant status check denial
	s := eventoutbox.NewScheduler(eventoutbox.SchedulerConfig{})
	req := eventoutbox.NotificationRequest{
		RequestID: "notif_auth_001",
		TenantID:  "ten_alpha",
		Recipient: "ops",
		Subject:   "Notice",
		Body:      "Body",
		Channel:   eventoutbox.ChannelLocalMemory,
	}
	_, _ = s.SendNotification(req)
	_, err = s.GetNotification("notif_auth_001", "ten_bravo")
	if !errors.Is(err, eventoutbox.ErrCrossTenantNotification) {
		t.Fatalf("expected ErrCrossTenantNotification, got: %v", err)
	}

	// 6. Notification unauthorized replay denial (blank identity)
	s.SetLocalSinkFailure(errors.New("fail"))
	reqQuar := eventoutbox.NotificationRequest{
		RequestID: "notif_auth_002",
		TenantID:  "ten_alpha",
		Recipient: "ops",
		Subject:   "Notice",
		Body:      "Body",
		Channel:   eventoutbox.ChannelLocalMemory,
	}
	for i := 0; i < 3; i++ {
		_, _ = s.SendNotification(reqQuar)
	}
	_, err = s.ReplayNotification("notif_auth_002", "ten_alpha", "")
	if !errors.Is(err, eventoutbox.ErrUnauthorizedNotificationAccess) {
		t.Fatalf("expected ErrUnauthorizedNotificationAccess on blank caller, got: %v", err)
	}
}
