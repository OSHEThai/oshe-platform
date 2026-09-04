package eventoutbox

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DeliveryStatus represents the deterministic progression of event delivery.
type DeliveryStatus string

const (
	StatusPending     DeliveryStatus = "PENDING"
	StatusDelivered   DeliveryStatus = "DELIVERED"
	StatusRetrying    DeliveryStatus = "RETRYING"
	StatusQuarantined DeliveryStatus = "QUARANTINED"
)

var (
	ErrBlankConsumerID        = errors.New("consumer ID cannot be blank")
	ErrBlankEventID           = errors.New("event ID cannot be blank")
	ErrConsumerNotRegistered  = errors.New("consumer handler not registered")
	ErrAlreadyDelivered       = errors.New("event already successfully delivered to consumer")
	ErrCrossTenantDispatch    = errors.New("cross-tenant dispatch denied")
	ErrCrossTenantReplay      = errors.New("cross-tenant replay denied")
	ErrEventNotQuarantined    = errors.New("cannot replay an event that is not quarantined")
	ErrUnauthorizedReplay     = errors.New("unauthorized replay request: actor identity required")
	ErrDeliveryRecordNotFound = errors.New("delivery record not found")
	ErrMaxRetriesExceeded     = errors.New("event exceeded maximum retry attempts and is quarantined")
	ErrRetryLimitReached      = errors.New("cannot retry event: maximum attempt limit reached")
)

// EventHandler processes an event envelope.
// Returns nil on success, or an error on failure.
type EventHandler func(envelope EventEnvelope) error

// DeliveryRecord captures the persistent delivery state and history for an event/consumer pair.
type DeliveryRecord struct {
	EventID        string         `json:"event_id"`
	ConsumerID     string         `json:"consumer_id"`
	TenantID       string         `json:"tenant_id"`
	Status         DeliveryStatus `json:"status"`
	Attempts       int            `json:"attempts"`
	MaxAttempts    int            `json:"max_attempts"`
	LastError      string         `json:"last_error,omitempty"`
	FirstAttemptAt time.Time      `json:"first_attempt_at"`
	LastAttemptAt  time.Time      `json:"last_attempt_at"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
	QuarantinedAt  *time.Time     `json:"quarantined_at,omitempty"`
	ReplayedCount  int            `json:"replayed_count"`
}

// DispatcherConfig provides runtime configuration for Dispatcher.
type DispatcherConfig struct {
	MaxAttempts int
	Clock       func() time.Time
}

// Dispatcher coordinates deterministic event dispatch, retries, quarantine, and controlled replay.
type Dispatcher struct {
	mu              sync.RWMutex
	maxAttempts     int
	clock           func() time.Time
	handlers        map[string]EventHandler
	consumerTenants map[string]string // consumerID -> bound tenantID (optional, if tenant-scoped)
	records         map[string]*DeliveryRecord
}

// NewDispatcher constructs a new Dispatcher.
func NewDispatcher(cfg DispatcherConfig) *Dispatcher {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	return &Dispatcher{
		maxAttempts:     maxAttempts,
		clock:           clock,
		handlers:        make(map[string]EventHandler),
		consumerTenants: make(map[string]string),
		records:         make(map[string]*DeliveryRecord),
	}
}

func recordKey(consumerID, eventID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(consumerID), strings.TrimSpace(eventID))
}

// RegisterConsumer registers an event processing handler for a consumer.
// Optional tenantID enforces strict tenant isolation if provided.
func (d *Dispatcher) RegisterConsumer(consumerID string, tenantID string, handler EventHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := strings.TrimSpace(consumerID)
	if id == "" {
		return ErrBlankConsumerID
	}
	if handler == nil {
		return errors.New("event handler cannot be nil")
	}

	d.handlers[id] = handler
	if strings.TrimSpace(tenantID) != "" {
		d.consumerTenants[id] = strings.TrimSpace(tenantID)
	}

	return nil
}

// Dispatch delivers an event to a registered consumer.
// Fails closed if:
// - event ID or consumer ID is blank
// - consumer is not registered
// - consumer is scoped to a different tenant than the event
// - event is already successfully delivered (idempotency rejection)
// - event is currently quarantined (requires explicit replay)
func (d *Dispatcher) Dispatch(consumerID string, envelope EventEnvelope) (*DeliveryRecord, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	cID := strings.TrimSpace(consumerID)
	if cID == "" {
		return nil, ErrBlankConsumerID
	}
	eID := strings.TrimSpace(envelope.EventID)
	if eID == "" {
		return nil, ErrBlankEventID
	}

	handler, registered := d.handlers[cID]
	if !registered {
		return nil, ErrConsumerNotRegistered
	}

	// Tenant isolation check
	if boundTenant, exists := d.consumerTenants[cID]; exists {
		if boundTenant != envelope.TenantID {
			return nil, ErrCrossTenantDispatch
		}
	}

	key := recordKey(cID, eID)
	rec, exists := d.records[key]
	now := d.clock().UTC()

	if exists {
		if rec.Status == StatusDelivered {
			return rec, ErrAlreadyDelivered
		}
		if rec.Status == StatusQuarantined {
			return rec, ErrRetryLimitReached
		}
	} else {
		rec = &DeliveryRecord{
			EventID:        eID,
			ConsumerID:     cID,
			TenantID:       envelope.TenantID,
			Status:         StatusPending,
			Attempts:       0,
			MaxAttempts:    d.maxAttempts,
			FirstAttemptAt: now,
		}
		d.records[key] = rec
	}

	// Execute attempt
	rec.Attempts++
	rec.LastAttemptAt = now

	err := handler(envelope)
	if err == nil {
		rec.Status = StatusDelivered
		rec.DeliveredAt = &now
		rec.LastError = ""
		return rec, nil
	}

	// Failure handling
	rec.LastError = err.Error()
	if rec.Attempts >= rec.MaxAttempts {
		rec.Status = StatusQuarantined
		rec.QuarantinedAt = &now
		return rec, fmt.Errorf("%w: %s", ErrMaxRetriesExceeded, err.Error())
	}

	rec.Status = StatusRetrying
	return rec, fmt.Errorf("dispatch attempt %d failed: %w", rec.Attempts, err)
}

// Replay re-executes delivery for a quarantined event.
// Fails closed if:
// - callerIdentity is blank (unauthorized)
// - delivery record does not exist
// - event is not in StatusQuarantined
// - callerTenantID does not match the event/record tenantID
func (d *Dispatcher) Replay(consumerID, eventID, callerTenantID, callerIdentity string, envelope EventEnvelope) (*DeliveryRecord, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if strings.TrimSpace(callerIdentity) == "" {
		return nil, ErrUnauthorizedReplay
	}

	cID := strings.TrimSpace(consumerID)
	eID := strings.TrimSpace(eventID)
	key := recordKey(cID, eID)

	rec, exists := d.records[key]
	if !exists {
		return nil, ErrDeliveryRecordNotFound
	}

	if rec.Status != StatusQuarantined {
		return nil, ErrEventNotQuarantined
	}

	if strings.TrimSpace(callerTenantID) != rec.TenantID {
		return nil, ErrCrossTenantReplay
	}

	handler, registered := d.handlers[cID]
	if !registered {
		return nil, ErrConsumerNotRegistered
	}

	now := d.clock().UTC()
	rec.ReplayedCount++
	rec.Attempts++
	rec.LastAttemptAt = now

	err := handler(envelope)
	if err == nil {
		rec.Status = StatusDelivered
		rec.DeliveredAt = &now
		rec.LastError = ""
		return rec, nil
	}

	rec.LastError = err.Error()
	rec.Status = StatusQuarantined
	rec.QuarantinedAt = &now
	return rec, fmt.Errorf("replay failed: %w", err)
}

// GetDeliveryRecord returns a copy of a delivery record if found.
func (d *Dispatcher) GetDeliveryRecord(consumerID, eventID string) (DeliveryRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := recordKey(consumerID, eventID)
	rec, exists := d.records[key]
	if !exists {
		return DeliveryRecord{}, ErrDeliveryRecordNotFound
	}
	return *rec, nil
}

// ExportState snapshots all current delivery records for persistence or restart reconstruction.
func (d *Dispatcher) ExportState() []DeliveryRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make([]DeliveryRecord, 0, len(d.records))
	for _, rec := range d.records {
		out = append(out, *rec)
	}
	return out
}

// RestoreState reconstructs in-memory delivery state from exported records.
func (d *Dispatcher) RestoreState(records []DeliveryRecord) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, rec := range records {
		r := rec
		key := recordKey(r.ConsumerID, r.EventID)
		d.records[key] = &r
	}
	return nil
}
