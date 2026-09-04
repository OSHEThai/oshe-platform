package eventoutbox

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// JobState represents the execution lifecycle state of a scheduled job.
type JobState string

const (
	JobStateScheduled JobState = "SCHEDULED"
	JobStateRunning   JobState = "RUNNING"
	JobStateCompleted JobState = "COMPLETED"
	JobStateFailed    JobState = "FAILED"
	JobStateCanceled  JobState = "CANCELED"
)

// NotificationChannel enumerates permitted notification channels.
// Strictly local sinks only; external network channels are prohibited.
type NotificationChannel string

const (
	ChannelLocalMemory NotificationChannel = "LOCAL_MEMORY"
	ChannelLocalLog    NotificationChannel = "LOCAL_LOG"
)

// NotificationStatus tracks delivery status of a notification request.
type NotificationStatus string

const (
	NotificationStatusPending     NotificationStatus = "PENDING"
	NotificationStatusDelivered   NotificationStatus = "DELIVERED"
	NotificationStatusRetrying    NotificationStatus = "RETRYING"
	NotificationStatusQuarantined NotificationStatus = "QUARANTINED"
)

var (
	ErrBlankJobID                     = errors.New("job ID cannot be blank")
	ErrBlankTenantID                  = errors.New("tenant ID cannot be blank")
	ErrBlankJobType                   = errors.New("job type cannot be blank")
	ErrDuplicateJobID                 = errors.New("duplicate job ID: job already scheduled or executed")
	ErrJobNotFound                    = errors.New("job not found")
	ErrClockSkewDetected              = errors.New("clock skew detected: due date precedes scheduled date")
	ErrUnsupportedExternalChannel     = errors.New("unsupported external notification channel: only local sink channels are permitted")
	ErrCrossTenantNotification        = errors.New("cross-tenant notification denied")
	ErrDuplicateNotificationRequest   = errors.New("duplicate notification request ID")
	ErrNotificationNotFound           = errors.New("notification request not found")
	ErrUnauthorizedNotificationAccess = errors.New("unauthorized notification status or replay access")
	ErrNotificationNotQuarantined     = errors.New("notification is not in quarantined state")
	ErrNotificationMaxRetries         = errors.New("notification exceeded maximum retry attempts and is quarantined")
	ErrNotificationRetryLimitReached  = errors.New("notification retry limit reached: requires manual replay")
)

// JobHandler processes a due scheduled job.
type JobHandler func(job ScheduledJob) error

// ScheduledJob represents a deterministic background job.
type ScheduledJob struct {
	JobID       string     `json:"job_id"`
	TenantID    string     `json:"tenant_id"`
	JobType     string     `json:"job_type"`
	Payload     string     `json:"payload"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	DueAt       time.Time  `json:"due_at"`
	State       JobState   `json:"state"`
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"last_error,omitempty"`
}

// NotificationRequest captures a notification command directed to a local sink.
type NotificationRequest struct {
	RequestID    string              `json:"request_id"`
	TenantID     string              `json:"tenant_id"`
	Recipient    string              `json:"recipient"`
	Subject      string              `json:"subject"`
	Body         string              `json:"body"`
	Channel      NotificationChannel `json:"channel"`
	CreatedAt    time.Time           `json:"created_at"`
	Status       NotificationStatus  `json:"status"`
	Attempts     int                 `json:"attempts"`
	MaxAttempts  int                 `json:"max_attempts"`
	DeliveredAt  *time.Time          `json:"delivered_at,omitempty"`
	LastError    string              `json:"last_error,omitempty"`
	Diagnostics  string              `json:"diagnostics,omitempty"`
}

// SchedulerConfig provides configuration for Scheduler.
type SchedulerConfig struct {
	Clock                   func() time.Time
	MaxNotificationAttempts int
}

// Scheduler coordinates deterministic job scheduling and local notification sink delivery.
type Scheduler struct {
	mu                   sync.RWMutex
	clock                func() time.Time
	maxNotifAttempts     int
	jobs                 map[string]*ScheduledJob
	jobHandlers          map[string]JobHandler
	notifications        map[string]*NotificationRequest
	localSinkDeliveries  []NotificationRequest
	simulatedSinkFailure error
}

// NewScheduler constructs a new deterministic Scheduler.
func NewScheduler(cfg SchedulerConfig) *Scheduler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	maxAttempts := cfg.MaxNotificationAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	return &Scheduler{
		clock:               clock,
		maxNotifAttempts:    maxAttempts,
		jobs:                make(map[string]*ScheduledJob),
		jobHandlers:         make(map[string]JobHandler),
		notifications:       make(map[string]*NotificationRequest),
		localSinkDeliveries: make([]NotificationRequest, 0),
	}
}

// RegisterJobHandler binds an execution handler for a specific jobType.
func (s *Scheduler) RegisterJobHandler(jobType string, handler JobHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jt := strings.TrimSpace(jobType)
	if jt == "" {
		return ErrBlankJobType
	}
	if handler == nil {
		return errors.New("job handler cannot be nil")
	}

	s.jobHandlers[jt] = handler
	return nil
}

// ScheduleJob schedules a future or due job.
// Fails closed if IDs are blank, duplicate, or clock skew is detected (DueAt < ScheduledAt).
func (s *Scheduler) ScheduleJob(job ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(job.JobID)
	if jobID == "" {
		return ErrBlankJobID
	}
	tenantID := strings.TrimSpace(job.TenantID)
	if tenantID == "" {
		return ErrBlankTenantID
	}
	jobType := strings.TrimSpace(job.JobType)
	if jobType == "" {
		return ErrBlankJobType
	}

	if _, exists := s.jobs[jobID]; exists {
		return ErrDuplicateJobID
	}

	if job.DueAt.Before(job.ScheduledAt) {
		return ErrClockSkewDetected
	}

	now := s.clock().UTC()
	schedAt := job.ScheduledAt
	if schedAt.IsZero() {
		schedAt = now
	}
	dueAt := job.DueAt
	if dueAt.IsZero() {
		dueAt = now
	}

	j := &ScheduledJob{
		JobID:       jobID,
		TenantID:    tenantID,
		JobType:     jobType,
		Payload:     job.Payload,
		ScheduledAt: schedAt,
		DueAt:       dueAt,
		State:       JobStateScheduled,
		Attempts:    0,
	}

	s.jobs[jobID] = j
	return nil
}

// ExecuteDueJobs scans for scheduled jobs whose DueAt <= current clock time,
// executing their registered handlers.
func (s *Scheduler) ExecuteDueJobs() ([]ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock().UTC()
	var executed []ScheduledJob

	for _, job := range s.jobs {
		if job.State != JobStateScheduled {
			continue
		}

		if now.Before(job.DueAt) {
			// Not yet due
			continue
		}

		handler, exists := s.jobHandlers[job.JobType]
		if !exists {
			job.State = JobStateFailed
			job.LastError = fmt.Sprintf("no handler registered for job type: %s", job.JobType)
			executed = append(executed, *job)
			continue
		}

		job.State = JobStateRunning
		job.Attempts++
		execTime := now
		job.ExecutedAt = &execTime

		err := handler(*job)
		if err != nil {
			job.State = JobStateFailed
			job.LastError = err.Error()
		} else {
			job.State = JobStateCompleted
			job.LastError = ""
		}

		executed = append(executed, *job)
	}

	return executed, nil
}

// GetJob returns a snapshot of a scheduled job.
func (s *Scheduler) GetJob(jobID string) (ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	j, exists := s.jobs[strings.TrimSpace(jobID)]
	if !exists {
		return ScheduledJob{}, ErrJobNotFound
	}
	return *j, nil
}

// SendNotification dispatches a notification request strictly to the configured local sink.
// Prohibits external channels, rejects duplicates, and enforces bounded retry progression.
func (s *Scheduler) SendNotification(req NotificationRequest) (*NotificationRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reqID := strings.TrimSpace(req.RequestID)
	if reqID == "" {
		return nil, errors.New("request ID cannot be blank")
	}
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return nil, ErrBlankTenantID
	}

	// Reject unsupported external channels
	switch req.Channel {
	case ChannelLocalMemory, ChannelLocalLog:
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedExternalChannel, req.Channel)
	}

	record, exists := s.notifications[reqID]
	now := s.clock().UTC()

	if exists {
		if record.Status == NotificationStatusDelivered {
			return record, ErrDuplicateNotificationRequest
		}
		if record.Status == NotificationStatusQuarantined {
			return record, ErrNotificationRetryLimitReached
		}
	} else {
		createdAt := req.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		record = &NotificationRequest{
			RequestID:   reqID,
			TenantID:    tenantID,
			Recipient:   strings.TrimSpace(req.Recipient),
			Subject:     strings.TrimSpace(req.Subject),
			Body:        strings.TrimSpace(req.Body),
			Channel:     req.Channel,
			CreatedAt:   createdAt,
			Status:      NotificationStatusPending,
			Attempts:    0,
			MaxAttempts: s.maxNotifAttempts,
		}
		s.notifications[reqID] = record
	}

	record.Attempts++

	// Attempt local delivery
	if s.simulatedSinkFailure != nil {
		err := s.simulatedSinkFailure
		record.LastError = err.Error()
		record.Diagnostics = fmt.Sprintf("local sink delivery failed on attempt %d: %s", record.Attempts, err.Error())

		if record.Attempts >= record.MaxAttempts {
			record.Status = NotificationStatusQuarantined
			return record, fmt.Errorf("%w: %s", ErrNotificationMaxRetries, err.Error())
		}

		record.Status = NotificationStatusRetrying
		return record, fmt.Errorf("local sink attempt %d failed: %w", record.Attempts, err)
	}

	record.Status = NotificationStatusDelivered
	record.DeliveredAt = &now
	record.LastError = ""
	record.Diagnostics = fmt.Sprintf("successfully delivered to %s on attempt %d", record.Channel, record.Attempts)

	s.localSinkDeliveries = append(s.localSinkDeliveries, *record)
	return record, nil
}

// ReplayNotification re-executes delivery for a quarantined notification.
// Requires non-blank caller identity, matching tenant scope, and quarantined state.
func (s *Scheduler) ReplayNotification(requestID, callerTenantID, callerIdentity string) (*NotificationRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(callerIdentity) == "" {
		return nil, ErrUnauthorizedNotificationAccess
	}

	reqID := strings.TrimSpace(requestID)
	record, exists := s.notifications[reqID]
	if !exists {
		return nil, ErrNotificationNotFound
	}

	if record.TenantID != strings.TrimSpace(callerTenantID) {
		return nil, ErrCrossTenantNotification
	}

	if record.Status != NotificationStatusQuarantined {
		return nil, ErrNotificationNotQuarantined
	}

	record.Attempts++
	now := s.clock().UTC()

	if s.simulatedSinkFailure != nil {
		err := s.simulatedSinkFailure
		record.LastError = err.Error()
		record.Diagnostics = fmt.Sprintf("replay delivery failed: %s", err.Error())
		record.Status = NotificationStatusQuarantined
		return record, fmt.Errorf("replay failed: %w", err)
	}

	record.Status = NotificationStatusDelivered
	record.DeliveredAt = &now
	record.LastError = ""
	record.Diagnostics = fmt.Sprintf("replayed by %s and delivered to %s", callerIdentity, record.Channel)

	s.localSinkDeliveries = append(s.localSinkDeliveries, *record)
	return record, nil
}

// GetNotification returns a copy of a notification record, enforcing tenant isolation.
func (s *Scheduler) GetNotification(requestID, callerTenantID string) (NotificationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.notifications[strings.TrimSpace(requestID)]
	if !exists {
		return NotificationRequest{}, ErrNotificationNotFound
	}

	if record.TenantID != strings.TrimSpace(callerTenantID) {
		return NotificationRequest{}, ErrCrossTenantNotification
	}

	return *record, nil
}

// SetLocalSinkFailure injects or clears a simulated local sink failure for testing.
func (s *Scheduler) SetLocalSinkFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.simulatedSinkFailure = err
}

// GetLocalSinkDeliveries returns a slice copy of all notifications successfully delivered to the local sink.
func (s *Scheduler) GetLocalSinkDeliveries() []NotificationRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]NotificationRequest, len(s.localSinkDeliveries))
	copy(out, s.localSinkDeliveries)
	return out
}
