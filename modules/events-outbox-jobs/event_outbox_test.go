package eventoutbox_test

import (
	"errors"
	"testing"
	"time"

	eventoutbox "github.com/oshethai/oshe-platform/modules/events-outbox-jobs"
)

const (
	validDigest = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	validTenant = "ten_alpha"
)

func validEventInput(eventID string) eventoutbox.EventInput {
	return eventoutbox.EventInput{
		EventID:         eventID,
		TenantID:        validTenant,
		Producer:        "inspection-service",
		EventType:       "inspection.completed",
		EnvelopeVersion: eventoutbox.CurrentEnvelopeVersion,
		SchemaVersion:   eventoutbox.CurrentSchemaVersion,
		CorrelationID:   "corr_1234567890abcdef",
		CausationID:     "caus_fedcba0987654321",
		PayloadDigest:   validDigest,
		Timestamp:       time.Now().UTC(),
	}
}

func TestOutbox_CommitPublication(t *testing.T) {
	outbox := eventoutbox.NewOutbox()
	if len(outbox.CommittedEvents()) != 0 {
		t.Fatal("expected empty outbox upon creation")
	}

	tx, err := outbox.BeginTx(validTenant)
	if err != nil {
		t.Fatalf("unexpected BeginTx error: %v", err)
	}

	if err := tx.Stage(validEventInput("evt_001")); err != nil {
		t.Fatalf("stage evt_001 failed: %v", err)
	}
	if err := tx.Stage(validEventInput("evt_002")); err != nil {
		t.Fatalf("stage evt_002 failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	events := outbox.CommittedEvents()
	if len(events) != 2 {
		t.Fatalf("expected 2 committed events, got %d", len(events))
	}

	if events[0].EventID != "evt_001" || events[0].SequenceNumber != 1 {
		t.Errorf("unexpected event 0: %+v", events[0])
	}
	if events[1].EventID != "evt_002" || events[1].SequenceNumber != 2 {
		t.Errorf("unexpected event 1: %+v", events[1])
	}
	if events[0].TenantID != validTenant {
		t.Errorf("expected tenant %s, got %s", validTenant, events[0].TenantID)
	}
}

func TestOutbox_PreCommitInvisibility(t *testing.T) {
	outbox := eventoutbox.NewOutbox()
	tx, err := outbox.BeginTx(validTenant)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	if err := tx.Stage(validEventInput("evt_pre_001")); err != nil {
		t.Fatalf("stage failed: %v", err)
	}

	// Staged events must be completely invisible before commit
	if len(outbox.CommittedEvents()) != 0 {
		t.Fatal("expected 0 events in CommittedEvents before commit")
	}

	eventsTenant, err := outbox.CommittedEventsForTenant(validTenant)
	if err != nil {
		t.Fatalf("CommittedEventsForTenant failed: %v", err)
	}
	if len(eventsTenant) != 0 {
		t.Fatal("expected 0 events in CommittedEventsForTenant before commit")
	}
}

func TestOutbox_Rollback(t *testing.T) {
	outbox := eventoutbox.NewOutbox()
	tx, err := outbox.BeginTx(validTenant)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	if err := tx.Stage(validEventInput("evt_roll_001")); err != nil {
		t.Fatalf("stage failed: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	// Must remain empty
	if len(outbox.CommittedEvents()) != 0 {
		t.Fatal("expected empty outbox after rollback")
	}

	// Operations after rollback must return ErrTxClosed
	if err := tx.Stage(validEventInput("evt_roll_002")); !errors.Is(err, eventoutbox.ErrTxClosed) {
		t.Fatalf("expected ErrTxClosed on Stage after rollback, got %v", err)
	}
	if err := tx.Commit(); !errors.Is(err, eventoutbox.ErrTxClosed) {
		t.Fatalf("expected ErrTxClosed on Commit after rollback, got %v", err)
	}
	if err := tx.Rollback(); !errors.Is(err, eventoutbox.ErrTxClosed) {
		t.Fatalf("expected ErrTxClosed on Rollback after rollback, got %v", err)
	}
}

func TestOutbox_DuplicateEventID(t *testing.T) {
	outbox := eventoutbox.NewOutbox()

	// 1. Duplicate in same transaction
	tx1, _ := outbox.BeginTx(validTenant)
	_ = tx1.Stage(validEventInput("evt_dup_001"))
	err := tx1.Stage(validEventInput("evt_dup_001"))
	if !errors.Is(err, eventoutbox.ErrDuplicateEventID) {
		t.Fatalf("expected ErrDuplicateEventID on intra-tx duplicate, got %v", err)
	}
	_ = tx1.Commit()

	// 2. Duplicate across transactions
	tx2, _ := outbox.BeginTx(validTenant)
	err = tx2.Stage(validEventInput("evt_dup_001"))
	if !errors.Is(err, eventoutbox.ErrDuplicateEventID) {
		t.Fatalf("expected ErrDuplicateEventID on cross-tx duplicate, got %v", err)
	}
}

func TestOutbox_TenantMismatch(t *testing.T) {
	outbox := eventoutbox.NewOutbox()

	// 1. Invalid tenant prefix on BeginTx
	_, err := outbox.BeginTx("bad_tenant")
	if err == nil {
		t.Fatal("expected error on BeginTx with invalid prefix")
	}

	// 2. Cross-tenant event staging
	tx, _ := outbox.BeginTx("ten_alpha")
	input := validEventInput("evt_cross_001")
	input.TenantID = "ten_bravo" // conflicting tenant
	err = tx.Stage(input)
	if !errors.Is(err, eventoutbox.ErrCrossTenantAssociation) {
		t.Fatalf("expected ErrCrossTenantAssociation, got %v", err)
	}

	// 3. Invalid tenant query
	_, err = outbox.CommittedEventsForTenant("invalid_prefix")
	if err == nil {
		t.Fatal("expected error on CommittedEventsForTenant with invalid prefix")
	}
}

func TestOutbox_EnvelopeAndSchemaValidation(t *testing.T) {
	outbox := eventoutbox.NewOutbox()
	tx, _ := outbox.BeginTx(validTenant)

	// Blank event ID
	in1 := validEventInput("")
	if err := tx.Stage(in1); !errors.Is(err, eventoutbox.ErrEmptyID) {
		t.Errorf("expected ErrEmptyID for empty event ID, got %v", err)
	}

	// Malformed event ID (wrong prefix)
	in2 := validEventInput("wrong_123")
	if err := tx.Stage(in2); !errors.Is(err, eventoutbox.ErrPrefixMismatch) {
		t.Errorf("expected ErrPrefixMismatch for wrong prefix, got %v", err)
	}

	// Blank producer
	in3 := validEventInput("evt_val_001")
	in3.Producer = ""
	if err := tx.Stage(in3); !errors.Is(err, eventoutbox.ErrBlankField) {
		t.Errorf("expected ErrBlankField for blank producer, got %v", err)
	}

	// Blank event type
	in4 := validEventInput("evt_val_002")
	in4.EventType = ""
	if err := tx.Stage(in4); !errors.Is(err, eventoutbox.ErrBlankField) {
		t.Errorf("expected ErrBlankField for blank event type, got %v", err)
	}

	// Malformed correlation ID
	in5 := validEventInput("evt_val_003")
	in5.CorrelationID = "badcorr_123"
	if err := tx.Stage(in5); !errors.Is(err, eventoutbox.ErrPrefixMismatch) {
		t.Errorf("expected ErrPrefixMismatch for correlation ID, got %v", err)
	}

	// Malformed causation ID
	in6 := validEventInput("evt_val_004")
	in6.CausationID = "badcaus_123"
	if err := tx.Stage(in6); !errors.Is(err, eventoutbox.ErrPrefixMismatch) {
		t.Errorf("expected ErrPrefixMismatch for causation ID, got %v", err)
	}

	// Invalid payload digests
	badDigests := []string{
		"short",
		"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855", // uppercase
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855extra",
		"g3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // 'g' non-hex
	}
	for _, bd := range badDigests {
		in := validEventInput("evt_val_005")
		in.PayloadDigest = bd
		if err := tx.Stage(in); !errors.Is(err, eventoutbox.ErrInvalidDigest) {
			t.Errorf("for digest %q expected ErrInvalidDigest, got %v", bd, err)
		}
	}
}

func TestOutbox_CompatibilityRejection(t *testing.T) {
	outbox := eventoutbox.NewOutbox()
	tx, _ := outbox.BeginTx(validTenant)

	// Unsupported envelope version
	in1 := validEventInput("evt_comp_001")
	in1.EnvelopeVersion = "2.0.0"
	if err := tx.Stage(in1); !errors.Is(err, eventoutbox.ErrUnsupportedEnvelopeVersion) {
		t.Errorf("expected ErrUnsupportedEnvelopeVersion, got %v", err)
	}

	// Incompatible schema version
	in2 := validEventInput("evt_comp_002")
	in2.SchemaVersion = "2.0.0"
	if err := tx.Stage(in2); !errors.Is(err, eventoutbox.ErrIncompatibleSchemaVersion) {
		t.Errorf("expected ErrIncompatibleSchemaVersion, got %v", err)
	}

	in3 := validEventInput("evt_comp_003")
	in3.SchemaVersion = ""
	if err := tx.Stage(in3); !errors.Is(err, eventoutbox.ErrIncompatibleSchemaVersion) {
		t.Errorf("expected ErrIncompatibleSchemaVersion for blank, got %v", err)
	}
}

func TestOutbox_DeterministicOrdering(t *testing.T) {
	outbox := eventoutbox.NewOutbox()

	// Tx 1: tenant alpha
	tx1, _ := outbox.BeginTx("ten_alpha")
	_ = tx1.Stage(validEventInput("evt_ord_001"))
	_ = tx1.Stage(validEventInput("evt_ord_002"))
	_ = tx1.Commit()

	// Tx 2: tenant bravo
	tx2, _ := outbox.BeginTx("ten_bravo")
	inB1 := validEventInput("evt_ord_003")
	inB1.TenantID = "ten_bravo"
	_ = tx2.Stage(inB1)
	_ = tx2.Commit()

	// Tx 3: tenant alpha again
	tx3, _ := outbox.BeginTx("ten_alpha")
	_ = tx3.Stage(validEventInput("evt_ord_004"))
	_ = tx3.Commit()

	allEvents := outbox.CommittedEvents()
	if len(allEvents) != 4 {
		t.Fatalf("expected 4 events, got %d", len(allEvents))
	}

	for i, ev := range allEvents {
		expectedSeq := int64(i + 1)
		if ev.SequenceNumber != expectedSeq {
			t.Errorf("event %d: expected sequence %d, got %d", i, expectedSeq, ev.SequenceNumber)
		}
	}

	alphaEvents, err := outbox.CommittedEventsForTenant("ten_alpha")
	if err != nil {
		t.Fatalf("unexpected error for tenant alpha: %v", err)
	}
	if len(alphaEvents) != 3 {
		t.Fatalf("expected 3 events for tenant alpha, got %d", len(alphaEvents))
	}
	if alphaEvents[0].SequenceNumber != 1 || alphaEvents[1].SequenceNumber != 2 || alphaEvents[2].SequenceNumber != 4 {
		t.Errorf("unexpected sequence order for tenant alpha: %+v", alphaEvents)
	}
}
