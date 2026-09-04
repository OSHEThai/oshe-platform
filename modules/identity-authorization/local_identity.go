package localidentity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

// IdentityState represents whether an identity record is actively permitted or disabled.
type IdentityState string

const (
	IdentityActive   IdentityState = "ACTIVE"
	IdentityDisabled IdentityState = "DISABLED"

	TokenPrefix       = "oshe_tok_"
	tokenEntropyBytes = 32
)

var (
	ErrBlankSubject     = errors.New("subject must not be blank")
	ErrBlankTenantID    = errors.New("tenant ID must not be blank")
	ErrInvalidState     = errors.New("invalid identity state")
	ErrMalformedToken   = errors.New("session token is malformed")
	ErrTokenExpired     = errors.New("session token has expired")
	ErrTokenRevoked     = errors.New("session token has been revoked")
	ErrTokenNotFound    = errors.New("session token not found")
	ErrIdentityNotFound = errors.New("identity not found")
	ErrIdentityDisabled = errors.New("identity is disabled")
	ErrTenantMismatch   = errors.New("token tenant does not match requested tenant scope")
	ErrIdentityExists   = errors.New("identity already registered")
)

// Identity is a dependency-free, immutable record of an identity bound to a tenant.
type Identity struct {
	subject  string
	tenantID string
	state    IdentityState
}

func (i Identity) Subject() string      { return i.subject }
func (i Identity) TenantID() string     { return i.tenantID }
func (i Identity) State() IdentityState { return i.state }
func (i Identity) IsActive() bool       { return i.state == IdentityActive }

// NewIdentity constructs and validates a new immutable Identity record.
func NewIdentity(subject, tenantID string, state IdentityState) (Identity, error) {
	trimmedSub := strings.TrimSpace(subject)
	if trimmedSub == "" {
		return Identity{}, ErrBlankSubject
	}
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return Identity{}, ErrBlankTenantID
	}
	if state != IdentityActive && state != IdentityDisabled {
		return Identity{}, ErrInvalidState
	}
	return Identity{
		subject:  trimmedSub,
		tenantID: trimmedTenant,
		state:    state,
	}, nil
}

// SessionRecord holds session metadata keyed strictly by SHA-256 token digest.
// The raw token is NEVER stored or serialized in this record.
type SessionRecord struct {
	TokenDigest [32]byte
	Subject     string
	TenantID    string
	ExpiresAt   time.Time
	Revoked     bool
}

// ValidatedIdentity is the scoped return type providing verified identity context.
type ValidatedIdentity struct {
	subject         string
	tenantID        string
	isAuthenticated bool
	tokenDigest     [32]byte
	validatedAt     time.Time
}

func (v ValidatedIdentity) Subject() string        { return v.subject }
func (v ValidatedIdentity) TenantID() string       { return v.tenantID }
func (v ValidatedIdentity) IsAuthenticated() bool  { return v.isAuthenticated }
func (v ValidatedIdentity) TokenDigest() [32]byte  { return v.tokenDigest }
func (v ValidatedIdentity) ValidatedAt() time.Time { return v.validatedAt }

// Clock is a deterministic injectable time provider.
type Clock func() time.Time

// IdentityManager coordinates local identities and session token verification.
type IdentityManager struct {
	mu         sync.RWMutex
	clock      Clock
	identities map[string]Identity
	sessions   map[[32]byte]SessionRecord
}

// NewIdentityManager constructs an IdentityManager using the supplied clock.
func NewIdentityManager(clock Clock) *IdentityManager {
	if clock == nil {
		clock = time.Now
	}
	return &IdentityManager{
		clock:      clock,
		identities: make(map[string]Identity),
		sessions:   make(map[[32]byte]SessionRecord),
	}
}

// RegisterIdentity registers an identity record.
func (m *IdentityManager) RegisterIdentity(identity Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if identity.Subject() == "" {
		return ErrBlankSubject
	}
	if identity.TenantID() == "" {
		return ErrBlankTenantID
	}
	if _, exists := m.identities[identity.Subject()]; exists {
		return ErrIdentityExists
	}
	m.identities[identity.Subject()] = identity
	return nil
}

// SetIdentityState updates the active/disabled status of a registered identity.
func (m *IdentityManager) SetIdentityState(subject string, state IdentityState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, exists := m.identities[subject]
	if !exists {
		return ErrIdentityNotFound
	}
	if state != IdentityActive && state != IdentityDisabled {
		return ErrInvalidState
	}
	id.state = state
	m.identities[subject] = id
	return nil
}

// GenerateOpaqueToken generates cryptographically random bytes and formats the opaque token and its SHA-256 digest.
func GenerateOpaqueToken() (rawToken string, digest [32]byte, err error) {
	buf := make([]byte, tokenEntropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", [32]byte{}, err
	}
	rawToken = TokenPrefix + hex.EncodeToString(buf)
	digest = sha256.Sum256([]byte(rawToken))
	return rawToken, digest, nil
}

// IssueSession creates an authenticated session token for the given subject.
// Returns the raw token ONCE to the caller; the manager stores ONLY the SHA-256 digest.
func (m *IdentityManager) IssueSession(subject string, ttl time.Duration) (rawToken string, digest [32]byte, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, exists := m.identities[subject]
	if !exists {
		return "", [32]byte{}, ErrIdentityNotFound
	}
	if !id.IsActive() {
		return "", [32]byte{}, ErrIdentityDisabled
	}

	raw, dig, err := GenerateOpaqueToken()
	if err != nil {
		return "", [32]byte{}, err
	}

	expiresAt := m.clock().Add(ttl)
	record := SessionRecord{
		TokenDigest: dig,
		Subject:     id.Subject(),
		TenantID:    id.TenantID(),
		ExpiresAt:   expiresAt,
		Revoked:     false,
	}

	m.sessions[dig] = record
	return raw, dig, nil
}

// RevokeSession revokes a session by its token digest.
func (m *IdentityManager) RevokeSession(digest [32]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.sessions[digest]
	if !exists {
		return ErrTokenNotFound
	}
	rec.Revoked = true
	m.sessions[digest] = rec
	return nil
}

// RevokeToken hashes the raw token and revokes the matching session.
func (m *IdentityManager) RevokeToken(rawToken string) error {
	digest, err := parseTokenDigest(rawToken)
	if err != nil {
		return err
	}
	return m.RevokeSession(digest)
}

func parseTokenDigest(rawToken string) ([32]byte, error) {
	trimmed := strings.TrimSpace(rawToken)
	if !strings.HasPrefix(trimmed, TokenPrefix) {
		return [32]byte{}, ErrMalformedToken
	}
	hexPart := strings.TrimPrefix(trimmed, TokenPrefix)
	if len(hexPart) != tokenEntropyBytes*2 {
		return [32]byte{}, ErrMalformedToken
	}
	if _, err := hex.DecodeString(hexPart); err != nil {
		return [32]byte{}, ErrMalformedToken
	}
	return sha256.Sum256([]byte(trimmed)), nil
}

// ValidateSession verifies a raw token and confirms scope alignment.
// Fails closed on malformed, expired, revoked, unknown, disabled-identity, or tenant-mismatched tokens.
func (m *IdentityManager) ValidateSession(rawToken string, expectedTenantID string) (ValidatedIdentity, error) {
	digest, err := parseTokenDigest(rawToken)
	if err != nil {
		return ValidatedIdentity{}, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, exists := m.sessions[digest]
	if !exists {
		return ValidatedIdentity{}, ErrTokenNotFound
	}

	if rec.Revoked {
		return ValidatedIdentity{}, ErrTokenRevoked
	}

	now := m.clock()
	if now.After(rec.ExpiresAt) || now.Equal(rec.ExpiresAt) {
		return ValidatedIdentity{}, ErrTokenExpired
	}

	if expectedTenantID != "" && strings.TrimSpace(expectedTenantID) != rec.TenantID {
		return ValidatedIdentity{}, ErrTenantMismatch
	}

	id, exists := m.identities[rec.Subject]
	if !exists {
		return ValidatedIdentity{}, ErrIdentityNotFound
	}
	if !id.IsActive() {
		return ValidatedIdentity{}, ErrIdentityDisabled
	}

	return ValidatedIdentity{
		subject:         rec.Subject,
		tenantID:        rec.TenantID,
		isAuthenticated: true,
		tokenDigest:     digest,
		validatedAt:     now,
	}, nil
}
