package eventoutbox_test

import (
	"errors"
	"testing"
	"time"

	eventoutbox "github.com/oshethai/oshe-platform/modules/events-outbox-jobs"
)

func sampleEnvelope(eventID, tenantID string) eventoutbox.EventEnvelope {
	return eventoutbox.EventEnvelope{
		EventID:         eventID,
		TenantID:        tenantID,
		Producer:        "safety-service",
		EventType:       "inspection.flagged",
		EnvelopeVersion: eventoutbox.CurrentEnvelopeVersion,
		SchemaVersion:   eventoutbox.CurrentSchemaVersion,
		CorrelationID:   "corr_1111222233334444",
		CausationID:     "caus_5555666677778888",
		PayloadDigest:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		SequenceNumber:  1,
		Timestamp:       time.Now().UTC(),
	}
}

func TestDuplicateDispatch_Idempotency(t *testing.T) {
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_notification"

	var callCount int
	err := d.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		callCount++
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterConsumer failed: %v", err)
	}

	env := sampleEnvelope("evt_001", "ten_alpha")

	// First dispatch: success
	rec, err := d.Dispatch(consumerID, env)
	if err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}
	if rec.Status != eventoutbox.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", rec.Status)
	}
	if callCount != 1 {
		t.Errorf("expected 1 handler invocation, got %d", callCount)
	}

	// Second dispatch of same event: idempotency check
	_, err = d.Dispatch(consumerID, env)
	if !errors.Is(err, eventoutbox.ErrAlreadyDelivered) {
		t.Fatalf("expected ErrAlreadyDelivered on duplicate dispatch, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected handler call count to remain 1, got %d", callCount)
	}
}

func TestRetrySequence(t *testing.T) {
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_retry"

	var attempts int
	_ = d.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient network failure")
		}
		return nil
	})

	env := sampleEnvelope("evt_retry", "ten_alpha")

	// Attempt 1: fails
	rec, err := d.Dispatch(consumerID, env)
	if err == nil {
		t.Fatal("expected failure on attempt 1")
	}
	if rec.Status != eventoutbox.StatusRetrying || rec.Attempts != 1 {
		t.Errorf("expected StatusRetrying attempts=1, got status=%s attempts=%d", rec.Status, rec.Attempts)
	}

	// Attempt 2: fails
	rec, err = d.Dispatch(consumerID, env)
	if err == nil {
		t.Fatal("expected failure on attempt 2")
	}
	if rec.Status != eventoutbox.StatusRetrying || rec.Attempts != 2 {
		t.Errorf("expected StatusRetrying attempts=2, got status=%s attempts=%d", rec.Status, rec.Attempts)
	}

	// Attempt 3: succeeds
	rec, err = d.Dispatch(consumerID, env)
	if err != nil {
		t.Fatalf("unexpected failure on attempt 3: %v", err)
	}
	if rec.Status != eventoutbox.StatusDelivered || rec.Attempts != 3 {
		t.Errorf("expected StatusDelivered attempts=3, got status=%s attempts=%d", rec.Status, rec.Attempts)
	}
}

func TestPoisonEventQuarantine(t *testing.T) {
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_poison"

	_ = d.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		return errors.New("poison payload parse error")
	})

	env := sampleEnvelope("evt_poison", "ten_alpha")

	// Attempts 1 and 2
	_, _ = d.Dispatch(consumerID, env)
	_, _ = d.Dispatch(consumerID, env)

	// Attempt 3: exceeds max attempts -> quarantined
	rec, err := d.Dispatch(consumerID, env)
	if !errors.Is(err, eventoutbox.ErrMaxRetriesExceeded) {
		t.Fatalf("expected ErrMaxRetriesExceeded on 3rd failure, got %v", err)
	}
	if rec.Status != eventoutbox.StatusQuarantined {
		t.Fatalf("expected StatusQuarantined, got %s", rec.Status)
	}
	if rec.QuarantinedAt == nil {
		t.Fatal("expected non-nil QuarantinedAt")
	}

	// Attempt 4 without replay: rejected
	_, err = d.Dispatch(consumerID, env)
	if !errors.Is(err, eventoutbox.ErrRetryLimitReached) {
		t.Fatalf("expected ErrRetryLimitReached on dispatch of quarantined event, got %v", err)
	}
}

func TestPermittedReplay(t *testing.T) {
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_replay_perm"

	failHandler := true
	_ = d.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		if failHandler {
			return errors.New("temporary bug")
		}
		return nil
	})

	env := sampleEnvelope("evt_to_replay", "ten_alpha")

	// Exhaust retries to quarantine
	for i := 0; i < 3; i++ {
		_, _ = d.Dispatch(consumerID, env)
	}

	rec, _ := d.GetDeliveryRecord(consumerID, "evt_to_replay")
	if rec.Status != eventoutbox.StatusQuarantined {
		t.Fatalf("expected quarantined event, got %s", rec.Status)
	}

	// Fix handler and replay
	failHandler = false
	recReplay, err := d.Replay(consumerID, "evt_to_replay", "ten_alpha", "admin_ops", env)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	if recReplay.Status != eventoutbox.StatusDelivered {
		t.Errorf("expected StatusDelivered after successful replay, got %s", recReplay.Status)
	}
	if recReplay.ReplayedCount != 1 {
		t.Errorf("expected ReplayedCount=1, got %d", recReplay.ReplayedCount)
	}
}

func TestRejectedReplay(t *testing.T) {
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_replay_rej"

	_ = d.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		return errors.New("failure")
	})

	env := sampleEnvelope("evt_quar", "ten_alpha")
	for i := 0; i < 3; i++ {
		_, _ = d.Dispatch(consumerID, env)
	}

	// 1. Unauthorized replay (blank identity)
	_, err := d.Replay(consumerID, "evt_quar", "ten_alpha", "", env)
	if !errors.Is(err, eventoutbox.ErrUnauthorizedReplay) {
		t.Errorf("expected ErrUnauthorizedReplay, got %v", err)
	}

	// 2. Cross-tenant replay
	_, err = d.Replay(consumerID, "evt_quar", "ten_bravo", "admin_ops", env)
	if !errors.Is(err, eventoutbox.ErrCrossTenantReplay) {
		t.Errorf("expected ErrCrossTenantReplay, got %v", err)
	}

	// 3. Non-existent delivery record
	_, err = d.Replay(consumerID, "evt_nonexistent", "ten_alpha", "admin_ops", env)
	if !errors.Is(err, eventoutbox.ErrDeliveryRecordNotFound) {
		t.Errorf("expected ErrDeliveryRecordNotFound, got %v", err)
	}

	// 4. Replay of non-quarantined (delivered) event
	dDeliv := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	_ = dDeliv.RegisterConsumer("cons_ok", "ten_alpha", func(env eventoutbox.EventEnvelope) error { return nil })
	envOk := sampleEnvelope("evt_ok", "ten_alpha")
	_, _ = dDeliv.Dispatch("cons_ok", envOk)

	_, err = dDeliv.Replay("cons_ok", "evt_ok", "ten_alpha", "admin_ops", envOk)
	if !errors.Is(err, eventoutbox.ErrEventNotQuarantined) {
		t.Errorf("expected ErrEventNotQuarantined on delivered event replay, got %v", err)
	}
}

func TestTenantIsolation(t *testing.T) {
	d := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_isolated"

	var invoked bool
	_ = d.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		invoked = true
		return nil
	})

	// Dispatch event belonging to ten_bravo to ten_alpha consumer
	envBravo := sampleEnvelope("evt_bravo", "ten_bravo")
	_, err := d.Dispatch(consumerID, envBravo)
	if !errors.Is(err, eventoutbox.ErrCrossTenantDispatch) {
		t.Fatalf("expected ErrCrossTenantDispatch, got %v", err)
	}
	if invoked {
		t.Fatal("handler must NEVER be invoked for cross-tenant event")
	}
}

func TestCrashRestartStateReconstruction(t *testing.T) {
	d1 := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	consumerID := "cons_recon"

	_ = d1.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		if env.EventID == "evt_fail" {
			return errors.New("boom")
		}
		return nil
	})

	// Event 1: Delivered
	_, _ = d1.Dispatch(consumerID, sampleEnvelope("evt_success", "ten_alpha"))

	// Event 2: Quarantined
	envFail := sampleEnvelope("evt_fail", "ten_alpha")
	for i := 0; i < 3; i++ {
		_, _ = d1.Dispatch(consumerID, envFail)
	}

	// Export state
	exported := d1.ExportState()
	if len(exported) != 2 {
		t.Fatalf("expected 2 exported records, got %d", len(exported))
	}

	// Reconstruct state into a fresh dispatcher
	d2 := eventoutbox.NewDispatcher(eventoutbox.DispatcherConfig{MaxAttempts: 3})
	var d2CallCount int
	_ = d2.RegisterConsumer(consumerID, "ten_alpha", func(env eventoutbox.EventEnvelope) error {
		d2CallCount++
		return nil
	})

	if err := d2.RestoreState(exported); err != nil {
		t.Fatalf("RestoreState failed: %v", err)
	}

	// Verify reconstructed records
	recSuccess, err := d2.GetDeliveryRecord(consumerID, "evt_success")
	if err != nil || recSuccess.Status != eventoutbox.StatusDelivered {
		t.Fatalf("reconstructed success record mismatch: status=%s, err=%v", recSuccess.Status, err)
	}

	recQuar, err := d2.GetDeliveryRecord(consumerID, "evt_fail")
	if err != nil || recQuar.Status != eventoutbox.StatusQuarantined {
		t.Fatalf("reconstructed quarantined record mismatch: status=%s, err=%v", recQuar.Status, err)
	}

	// Verify idempotency preserved across restart
	_, err = d2.Dispatch(consumerID, sampleEnvelope("evt_success", "ten_alpha"))
	if !errors.Is(err, eventoutbox.ErrAlreadyDelivered) {
		t.Fatalf("expected ErrAlreadyDelivered in reconstructed dispatcher, got %v", err)
	}
	if d2CallCount != 0 {
		t.Fatalf("handler must not be called for already delivered event in reconstructed dispatcher")
	}

	// Verify replay works in reconstructed dispatcher
	_, err = d2.Replay(consumerID, "evt_fail", "ten_alpha", "admin", envFail)
	if err != nil {
		t.Fatalf("replay failed in reconstructed dispatcher: %v", err)
	}
	if d2CallCount != 1 {
		t.Fatalf("expected exactly 1 call from replay, got %d", d2CallCount)
	}
}
