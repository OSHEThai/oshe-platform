package recordsaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrActiveLegalHold       = errors.New("cannot delete record: active legal hold is in effect")
	ErrRetentionPeriodActive = errors.New("cannot delete record: mandatory retention period has not expired")
	ErrBlankHoldID           = errors.New("hold ID cannot be blank")
	ErrBlankPolicyID         = errors.New("policy ID cannot be blank")
	ErrHoldNotFound          = errors.New("legal hold not found")
	ErrBackupTampered        = errors.New("synthetic backup integrity verification failed: tamper detected")
	ErrExportTampered        = errors.New("export package integrity verification failed: tamper detected")
	ErrTamperDetected        = errors.New("record integrity verification failed: payload digest mismatch")
	ErrPolicyNotFound        = errors.New("retention policy not found")
	ErrDuplicateHoldID       = errors.New("duplicate legal hold ID")
	ErrDuplicatePolicyID     = errors.New("duplicate retention policy ID")
)

// RetentionPolicy defines minimum required data retention for a record type.
type RetentionPolicy struct {
	PolicyID        string        `json:"policy_id"`
	TenantID        string        `json:"tenant_id"`
	RecordType      string        `json:"record_type"`
	RetentionPeriod time.Duration `json:"retention_period"`
	CreatedAt       time.Time     `json:"created_at"`
}

// LegalHold represents a binding hold preventing record deletion or purging.
type LegalHold struct {
	HoldID     string     `json:"hold_id"`
	TenantID   string     `json:"tenant_id"`
	RecordID   string     `json:"record_id"`
	Reason     string     `json:"reason"`
	PlacedBy   string     `json:"placed_by"`
	PlacedAt   time.Time  `json:"placed_at"`
	Active     bool       `json:"active"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`
	ReleasedBy string     `json:"released_by,omitempty"`
}

// ExportItem captures a single record and its full historical snapshots and audit events.
type ExportItem struct {
	RecordID       string           `json:"record_id"`
	RecordType     string           `json:"record_type"`
	CurrentVersion string           `json:"current_version"`
	State          RecordLifecycleState `json:"state"`
	CreatedAt      time.Time        `json:"created_at"`
	Snapshots      []RecordSnapshot `json:"snapshots"`
	AuditEntries   []AuditEntry     `json:"audit_entries"`
	PayloadDigest  string           `json:"payload_digest"`
}

// ExportPackage is a deterministic, tamper-evident bundle of tenant records.
type ExportPackage struct {
	ManifestID    string       `json:"manifest_id"`
	TenantID      string       `json:"tenant_id"`
	ExportedAt    time.Time    `json:"exported_at"`
	ExportedBy    string       `json:"exported_by"`
	Items         []ExportItem `json:"items"`
	ItemCount     int          `json:"item_count"`
	PackageDigest string       `json:"package_digest"`
}

// BackupArchive represents a synthetic in-memory backup snapshot.
type BackupArchive struct {
	BackupID      string                     `json:"backup_id"`
	CreatedAt     time.Time                  `json:"created_at"`
	Records       map[string]DeclaredRecord  `json:"records"`
	Snapshots     map[string][]RecordSnapshot `json:"snapshots"`
	AuditLog      []AuditEntry               `json:"audit_log"`
	Policies      map[string]RetentionPolicy `json:"policies"`
	LegalHolds    map[string]LegalHold       `json:"legal_holds"`
	ArchiveDigest string                     `json:"archive_digest"`
}

// RetentionManager coordinates retention policies, legal holds, export packaging,
// and synthetic backup/restore operations alongside a RecordStore.
type RetentionManager struct {
	mu         sync.RWMutex
	store      *RecordStore
	policies   map[string]RetentionPolicy // key: tenantID:recordType
	legalHolds map[string]LegalHold       // key: holdID
	recordHolds map[string][]string       // key: tenantID:recordID -> []holdID
}

// NewRetentionManager constructs a new RetentionManager.
func NewRetentionManager(store *RecordStore) *RetentionManager {
	if store == nil {
		store = NewRecordStore()
	}
	return &RetentionManager{
		store:       store,
		policies:    make(map[string]RetentionPolicy),
		legalHolds:  make(map[string]LegalHold),
		recordHolds: make(map[string][]string),
	}
}

func makePolicyKey(tenantID, recordType string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(recordType))
}

// RegisterRetentionPolicy registers a retention rule for a record type within tenant scope.
func (m *RetentionManager) RegisterRetentionPolicy(policy RetentionPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pID := strings.TrimSpace(policy.PolicyID)
	if pID == "" {
		return ErrBlankPolicyID
	}
	tID := strings.TrimSpace(policy.TenantID)
	if tID == "" {
		return ErrBlankTenantID
	}
	rType := strings.TrimSpace(policy.RecordType)
	if rType == "" {
		return ErrBlankRecordType
	}

	key := makePolicyKey(tID, rType)
	if _, exists := m.policies[key]; exists {
		return ErrDuplicatePolicyID
	}

	m.policies[key] = RetentionPolicy{
		PolicyID:        pID,
		TenantID:        tID,
		RecordType:      rType,
		RetentionPeriod: policy.RetentionPeriod,
		CreatedAt:       policy.CreatedAt,
	}
	return nil
}

// PlaceLegalHold activates a legal hold on a specific record, blocking deletion.
func (m *RetentionManager) PlaceLegalHold(hold LegalHold) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hID := strings.TrimSpace(hold.HoldID)
	if hID == "" {
		return ErrBlankHoldID
	}
	tID := strings.TrimSpace(hold.TenantID)
	if tID == "" {
		return ErrBlankTenantID
	}
	rID := strings.TrimSpace(hold.RecordID)
	if rID == "" {
		return ErrBlankID
	}

	if _, exists := m.legalHolds[hID]; exists {
		return ErrDuplicateHoldID
	}

	// Verify record exists in caller tenant scope
	m.store.mu.RLock()
	recKey := makeRecordKey(tID, rID)
	_, recExists := m.store.records[recKey]
	m.store.mu.RUnlock()
	if !recExists {
		return ErrRecordNotFound
	}

	h := LegalHold{
		HoldID:   hID,
		TenantID: tID,
		RecordID: rID,
		Reason:   strings.TrimSpace(hold.Reason),
		PlacedBy: strings.TrimSpace(hold.PlacedBy),
		PlacedAt: hold.PlacedAt,
		Active:   true,
	}

	m.legalHolds[hID] = h
	m.recordHolds[recKey] = append(m.recordHolds[recKey], hID)
	return nil
}

// ReleaseLegalHold deactivates an active legal hold.
func (m *RetentionManager) ReleaseLegalHold(tenantID, holdID, actorID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hID := strings.TrimSpace(holdID)
	h, exists := m.legalHolds[hID]
	if !exists {
		return ErrHoldNotFound
	}

	if h.TenantID != strings.TrimSpace(tenantID) {
		return ErrCrossTenantAccess
	}

	now := time.Now().UTC()
	h.Active = false
	h.ReleasedAt = &now
	h.ReleasedBy = strings.TrimSpace(actorID)
	m.legalHolds[hID] = h

	return nil
}

// CanDelete evaluates whether a record can be deleted under active retention and legal hold rules.
func (m *RetentionManager) CanDelete(tenantID, recordID string, checkTime time.Time) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tID := strings.TrimSpace(tenantID)
	rID := strings.TrimSpace(recordID)

	recKey := makeRecordKey(tID, rID)
	m.store.mu.RLock()
	rec, exists := m.store.records[recKey]
	m.store.mu.RUnlock()
	if !exists {
		return false, ErrRecordNotFound
	}

	// 1. Check legal holds
	if holdIDs, hasHolds := m.recordHolds[recKey]; hasHolds {
		for _, hid := range holdIDs {
			if h, found := m.legalHolds[hid]; found && h.Active {
				return false, ErrActiveLegalHold
			}
		}
	}

	// 2. Check retention policy
	policyKey := makePolicyKey(tID, rec.RecordType)
	if policy, found := m.policies[policyKey]; found && policy.RetentionPeriod > 0 {
		expiration := rec.CreatedAt.Add(policy.RetentionPeriod)
		if checkTime.Before(expiration) {
			return false, ErrRetentionPeriodActive
		}
	}

	return true, nil
}

// CalculatePackageDigest computes a deterministic SHA-256 hash over an export package's items.
func CalculatePackageDigest(items []ExportItem) string {
	h := sha256.New()
	for _, item := range items {
		h.Write([]byte(item.RecordID))
		h.Write([]byte(item.RecordType))
		h.Write([]byte(item.CurrentVersion))
		h.Write([]byte(item.PayloadDigest))
		h.Write([]byte(item.State))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateExportPackage produces a deterministic, tenant-isolated export package.
func (m *RetentionManager) GenerateExportPackage(tenantID, actorID string) (ExportPackage, error) {
	tID := strings.TrimSpace(tenantID)
	if tID == "" {
		return ExportPackage{}, ErrBlankTenantID
	}
	aID := strings.TrimSpace(actorID)
	if aID == "" {
		return ExportPackage{}, errors.New("actor ID cannot be blank")
	}

	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	var items []ExportItem
	for key, rec := range m.store.records {
		if rec.TenantID != tID {
			continue // strictly tenant-isolated
		}

		snaps := m.store.snapshots[key]
		snapsCopy := make([]RecordSnapshot, len(snaps))
		copy(snapsCopy, snaps)

		// Latest payload digest
		var latestDigest string
		if len(snaps) > 0 {
			latestDigest = snaps[len(snaps)-1].PayloadDigest
		}

		// Filter audit entries for this record
		var audits []AuditEntry
		for _, entry := range m.store.auditLog {
			if entry.TenantID == tID && entry.RecordID == rec.RecordID {
				audits = append(audits, entry)
			}
		}

		items = append(items, ExportItem{
			RecordID:       rec.RecordID,
			RecordType:     rec.RecordType,
			CurrentVersion: rec.CurrentVersion,
			State:          rec.State,
			CreatedAt:      rec.CreatedAt,
			Snapshots:      snapsCopy,
			AuditEntries:   audits,
			PayloadDigest:  latestDigest,
		})
	}

	// Sort items deterministically by RecordID
	sort.Slice(items, func(i, j int) bool {
		return items[i].RecordID < items[j].RecordID
	})

	now := time.Now().UTC()
	digest := CalculatePackageDigest(items)

	return ExportPackage{
		ManifestID:    fmt.Sprintf("exp_%s_%d", tID, now.Unix()),
		TenantID:      tID,
		ExportedAt:    now,
		ExportedBy:    aID,
		Items:         items,
		ItemCount:     len(items),
		PackageDigest: digest,
	}, nil
}

// VerifyExportIntegrity validates that an ExportPackage has not been tampered with.
func VerifyExportIntegrity(pkg ExportPackage) error {
	expectedDigest := CalculatePackageDigest(pkg.Items)
	if pkg.PackageDigest != expectedDigest {
		return ErrExportTampered
	}
	if pkg.ItemCount != len(pkg.Items) {
		return ErrExportTampered
	}
	return nil
}

// CreateSyntheticBackup generates an in-memory backup snapshot with SHA-256 integrity digest.
func (m *RetentionManager) CreateSyntheticBackup() (BackupArchive, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	recordsCopy := make(map[string]DeclaredRecord)
	for k, v := range m.store.records {
		recordsCopy[k] = v
	}

	snapshotsCopy := make(map[string][]RecordSnapshot)
	for k, v := range m.store.snapshots {
		sCopy := make([]RecordSnapshot, len(v))
		copy(sCopy, v)
		snapshotsCopy[k] = sCopy
	}

	auditCopy := make([]AuditEntry, len(m.store.auditLog))
	copy(auditCopy, m.store.auditLog)

	policiesCopy := make(map[string]RetentionPolicy)
	for k, v := range m.policies {
		policiesCopy[k] = v
	}

	holdsCopy := make(map[string]LegalHold)
	for k, v := range m.legalHolds {
		holdsCopy[k] = v
	}

	now := time.Now().UTC()
	payload, err := json.Marshal(struct {
		Records    map[string]DeclaredRecord  `json:"records"`
		Snapshots  map[string][]RecordSnapshot `json:"snapshots"`
		AuditLog   []AuditEntry               `json:"audit_log"`
		Policies   map[string]RetentionPolicy `json:"policies"`
		LegalHolds map[string]LegalHold       `json:"legal_holds"`
	}{
		Records:    recordsCopy,
		Snapshots:  snapshotsCopy,
		AuditLog:   auditCopy,
		Policies:   policiesCopy,
		LegalHolds: holdsCopy,
	})
	if err != nil {
		return BackupArchive{}, err
	}

	hash := sha256.Sum256(payload)
	digest := hex.EncodeToString(hash[:])

	return BackupArchive{
		BackupID:      fmt.Sprintf("backup_%d", now.UnixNano()),
		CreatedAt:     now,
		Records:       recordsCopy,
		Snapshots:     snapshotsCopy,
		AuditLog:      auditCopy,
		Policies:      policiesCopy,
		LegalHolds:    holdsCopy,
		ArchiveDigest: digest,
	}, nil
}

// RestoreSyntheticBackup restores in-memory state after verifying the archive's digest.
func (m *RetentionManager) RestoreSyntheticBackup(archive BackupArchive) error {
	// 1. Verify archive digest before mutating any state
	payload, err := json.Marshal(struct {
		Records    map[string]DeclaredRecord  `json:"records"`
		Snapshots  map[string][]RecordSnapshot `json:"snapshots"`
		AuditLog   []AuditEntry               `json:"audit_log"`
		Policies   map[string]RetentionPolicy `json:"policies"`
		LegalHolds map[string]LegalHold       `json:"legal_holds"`
	}{
		Records:    archive.Records,
		Snapshots:  archive.Snapshots,
		AuditLog:   archive.AuditLog,
		Policies:   archive.Policies,
		LegalHolds: archive.LegalHolds,
	})
	if err != nil {
		return err
	}

	hash := sha256.Sum256(payload)
	calculatedDigest := hex.EncodeToString(hash[:])
	if calculatedDigest != archive.ArchiveDigest {
		return ErrBackupTampered
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	// 2. State restore
	m.store.records = make(map[string]DeclaredRecord)
	for k, v := range archive.Records {
		m.store.records[k] = v
	}

	m.store.snapshots = make(map[string][]RecordSnapshot)
	for k, v := range archive.Snapshots {
		sCopy := make([]RecordSnapshot, len(v))
		copy(sCopy, v)
		m.store.snapshots[k] = sCopy
	}

	m.store.auditLog = make([]AuditEntry, len(archive.AuditLog))
	copy(m.store.auditLog, archive.AuditLog)

	m.policies = make(map[string]RetentionPolicy)
	for k, v := range archive.Policies {
		m.policies[k] = v
	}

	m.legalHolds = make(map[string]LegalHold)
	m.recordHolds = make(map[string][]string)
	for k, v := range archive.LegalHolds {
		m.legalHolds[k] = v
		if v.Active {
			recKey := makeRecordKey(v.TenantID, v.RecordID)
			m.recordHolds[recKey] = append(m.recordHolds[recKey], v.HoldID)
		}
	}

	return nil
}

// VerifyRecordIntegrity verifies the content digest of a record's latest snapshot.
func (m *RetentionManager) VerifyRecordIntegrity(tenantID, recordID, expectedDigest string) error {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	recKey := makeRecordKey(tenantID, recordID)
	snaps, exists := m.store.snapshots[recKey]
	if !exists || len(snaps) == 0 {
		return ErrRecordNotFound
	}

	latest := snaps[len(snaps)-1]
	normExpected := strings.ToLower(strings.TrimSpace(expectedDigest))
	if latest.PayloadDigest != normExpected {
		return ErrTamperDetected
	}

	return nil
}
