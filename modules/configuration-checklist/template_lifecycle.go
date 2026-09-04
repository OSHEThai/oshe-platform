// Package checklist provides the core in-memory immutable template-version
// lifecycle foundation for v0.2.0 checklists under milestone topic V020-T04.
package checklist

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrEmptyTemplateID indicates that the template identifier is empty.
	ErrEmptyTemplateID = errors.New("template_id cannot be empty")
	// ErrEmptyVersionID indicates that the version identifier is empty.
	ErrEmptyVersionID = errors.New("version_id cannot be empty")
	// ErrEmptyTitle indicates that the template title is empty.
	ErrEmptyTitle = errors.New("template title cannot be empty")
	// ErrEmptyQuestionList indicates that a template has no defined questions.
	ErrEmptyQuestionList = errors.New("template question list cannot be empty")
	// ErrInvalidLifecycleTransition indicates an unauthorized forward or backward state transition.
	ErrInvalidLifecycleTransition = errors.New("invalid lifecycle state transition")
	// ErrMissingReview indicates an attempt to approve or publish a template without a recorded review marker.
	ErrMissingReview = errors.New("template requires recorded review before approval or publication")
	// ErrMissingApproval indicates an attempt to publish without an explicit approval marker.
	ErrMissingApproval = errors.New("template requires explicit approval before publication")
	// ErrImmutableContent indicates an attempt to modify content or questions on an immutable template version.
	ErrImmutableContent = errors.New("published or retired template content is immutable and cannot be modified")
	// ErrInvalidProvenance indicates missing source template or version identifiers during a copy operation.
	ErrInvalidProvenance = errors.New("source template_id and version_id are required for copy provenance")
	// ErrNotPublished indicates an attempt to instantiate an unapproved or unpublished template version.
	ErrNotPublished = errors.New("cannot instantiate checklist from unapproved or non-published template version")
	// ErrEmptyInstanceID indicates that the instance identifier is empty.
	ErrEmptyInstanceID = errors.New("instance_id cannot be empty")
	// ErrEmptyTenantID indicates that the tenant identifier is empty.
	ErrEmptyTenantID = errors.New("tenant_id cannot be empty")
)

// LifecycleState represents the discrete governance states of a checklist template version.
type LifecycleState string

const (
	// StateDraft is the initial mutable authoring state.
	StateDraft LifecycleState = "Draft"
	// StateInReview indicates the template is under active evaluation.
	StateInReview LifecycleState = "InReview"
	// StateApproved indicates the template has received formal approval.
	StateApproved LifecycleState = "Approved"
	// StatePublished indicates the template is active and immutable.
	StatePublished LifecycleState = "Published"
	// StateRetired indicates the template is superseded or retired and permanently immutable.
	StateRetired LifecycleState = "Retired"
)

// Question represents an individual inspection item or form prompt within a template version.
type Question struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// ReviewMarker records the identity and timestamp of formal review.
type ReviewMarker struct {
	ReviewedBy string    `json:"reviewed_by"`
	ReviewedAt time.Time `json:"reviewed_at"`
	Notes      string    `json:"notes,omitempty"`
}

// ApprovalMarker records the identity and timestamp of human or authority approval.
type ApprovalMarker struct {
	ApprovedBy string    `json:"approved_by"`
	ApprovedAt time.Time `json:"approved_at"`
	Authority  string    `json:"authority,omitempty"`
}

// Provenance records source lineage when a template is copied or derived.
type Provenance struct {
	SourceTemplateID string    `json:"source_template_id"`
	SourceVersionID  string    `json:"source_version_id"`
	DerivedAt        time.Time `json:"derived_at"`
}

// TemplateSnapshot captures an immutable deep copy of published template content.
type TemplateSnapshot struct {
	TemplateID  string         `json:"template_id"`
	VersionID   string         `json:"version_id"`
	Title       string         `json:"title"`
	Questions   []Question     `json:"questions"`
	Review      ReviewMarker   `json:"review"`
	Approval    ApprovalMarker `json:"approval"`
	PublishedAt time.Time      `json:"published_at"`
}

// TemplateVersion encapsulates an immutable or draft template version and its governance lifecycle.
type TemplateVersion struct {
	TemplateID        string            `json:"template_id"`
	VersionID         string            `json:"version_id"`
	Title             string            `json:"title"`
	State             LifecycleState    `json:"state"`
	Questions         []Question        `json:"questions"`
	Review            *ReviewMarker     `json:"review,omitempty"`
	Approval          *ApprovalMarker   `json:"approval,omitempty"`
	Provenance        *Provenance       `json:"provenance,omitempty"`
	PublishedSnapshot *TemplateSnapshot `json:"published_snapshot,omitempty"`
}

func cloneQuestions(qs []Question) []Question {
	if qs == nil {
		return []Question{}
	}
	out := make([]Question, len(qs))
	copy(out, qs)
	return out
}

// NewTemplateVersion creates a new template version in the initial Draft state.
func NewTemplateVersion(templateID, versionID, title string, questions []Question) (*TemplateVersion, error) {
	tTemplateID := strings.TrimSpace(templateID)
	if tTemplateID == "" {
		return nil, ErrEmptyTemplateID
	}
	tVersionID := strings.TrimSpace(versionID)
	if tVersionID == "" {
		return nil, ErrEmptyVersionID
	}
	tTitle := strings.TrimSpace(title)
	if tTitle == "" {
		return nil, ErrEmptyTitle
	}
	if len(questions) == 0 {
		return nil, ErrEmptyQuestionList
	}

	return &TemplateVersion{
		TemplateID: tTemplateID,
		VersionID:  tVersionID,
		Title:      tTitle,
		State:      StateDraft,
		Questions:  cloneQuestions(questions),
	}, nil
}

// SubmitForReview transitions a template from Draft to InReview.
func (t *TemplateVersion) SubmitForReview() error {
	if t.State != StateDraft {
		return fmt.Errorf("%w: cannot submit for review from state %q", ErrInvalidLifecycleTransition, t.State)
	}
	if len(t.Questions) == 0 {
		return ErrEmptyQuestionList
	}
	t.State = StateInReview
	return nil
}

// RecordReview records the review marker while the template is InReview.
func (t *TemplateVersion) RecordReview(reviewer, notes string) error {
	if t.State != StateInReview {
		return fmt.Errorf("%w: reviews can only be recorded in InReview state", ErrInvalidLifecycleTransition)
	}
	tReviewer := strings.TrimSpace(reviewer)
	if tReviewer == "" {
		return errors.New("reviewer identity cannot be empty")
	}
	t.Review = &ReviewMarker{
		ReviewedBy: tReviewer,
		ReviewedAt: time.Now().UTC(),
		Notes:      strings.TrimSpace(notes),
	}
	return nil
}

// RejectToDraft returns an in-review template back to Draft state for revisions.
func (t *TemplateVersion) RejectToDraft() error {
	if t.State != StateInReview {
		return fmt.Errorf("%w: cannot return to draft from state %q", ErrInvalidLifecycleTransition, t.State)
	}
	t.State = StateDraft
	return nil
}

// Approve transitions an InReview template to Approved. Requires a recorded review marker.
func (t *TemplateVersion) Approve(approver, authority string) error {
	if t.State != StateInReview {
		return fmt.Errorf("%w: cannot approve template in state %q", ErrInvalidLifecycleTransition, t.State)
	}
	if t.Review == nil || strings.TrimSpace(t.Review.ReviewedBy) == "" {
		return ErrMissingReview
	}
	tApprover := strings.TrimSpace(approver)
	if tApprover == "" {
		return errors.New("approver identity cannot be empty")
	}

	t.Approval = &ApprovalMarker{
		ApprovedBy: tApprover,
		ApprovedAt: time.Now().UTC(),
		Authority:  strings.TrimSpace(authority),
	}
	t.State = StateApproved
	return nil
}

// Publish transitions an Approved template to Published and creates an immutable snapshot.
// Requires explicit review and approval markers.
func (t *TemplateVersion) Publish() error {
	if t.State != StateApproved {
		return fmt.Errorf("%w: cannot publish template from state %q", ErrInvalidLifecycleTransition, t.State)
	}
	if t.Review == nil || strings.TrimSpace(t.Review.ReviewedBy) == "" {
		return ErrMissingReview
	}
	if t.Approval == nil || strings.TrimSpace(t.Approval.ApprovedBy) == "" {
		return ErrMissingApproval
	}

	now := time.Now().UTC()
	t.PublishedSnapshot = &TemplateSnapshot{
		TemplateID:  t.TemplateID,
		VersionID:   t.VersionID,
		Title:       t.Title,
		Questions:   cloneQuestions(t.Questions),
		Review:      *t.Review,
		Approval:    *t.Approval,
		PublishedAt: now,
	}
	t.State = StatePublished
	return nil
}

// Retire transitions a Published template to Retired.
func (t *TemplateVersion) Retire() error {
	if t.State != StatePublished {
		return fmt.Errorf("%w: only Published templates can be retired, current state %q", ErrInvalidLifecycleTransition, t.State)
	}
	t.State = StateRetired
	return nil
}

// UpdateContent updates the title and questions of a template version.
// Permitted strictly when State == Draft.
func (t *TemplateVersion) UpdateContent(title string, questions []Question) error {
	if t.State == StatePublished || t.State == StateRetired {
		return ErrImmutableContent
	}
	if t.State != StateDraft {
		return fmt.Errorf("%w: content modifications permitted only in Draft state, current: %q", ErrImmutableContent, t.State)
	}
	tTitle := strings.TrimSpace(title)
	if tTitle == "" {
		return ErrEmptyTitle
	}
	if len(questions) == 0 {
		return ErrEmptyQuestionList
	}

	t.Title = tTitle
	t.Questions = cloneQuestions(questions)
	return nil
}

// Copy derives a new template version from an existing template version.
// Requires valid newTemplateID and newVersionID, records provenance, resets state to Draft,
// and deep-copies questions while preserving any existing published snapshot on the source.
func (t *TemplateVersion) Copy(newTemplateID, newVersionID, newTitle string) (*TemplateVersion, error) {
	tNewTID := strings.TrimSpace(newTemplateID)
	if tNewTID == "" {
		return nil, ErrEmptyTemplateID
	}
	tNewVID := strings.TrimSpace(newVersionID)
	if tNewVID == "" {
		return nil, ErrEmptyVersionID
	}
	tNewTitle := strings.TrimSpace(newTitle)
	if tNewTitle == "" {
		tNewTitle = t.Title
	}
	if strings.TrimSpace(t.TemplateID) == "" || strings.TrimSpace(t.VersionID) == "" {
		return nil, ErrInvalidProvenance
	}

	return &TemplateVersion{
		TemplateID: tNewTID,
		VersionID:  tNewVID,
		Title:      tNewTitle,
		State:      StateDraft,
		Questions:  cloneQuestions(t.Questions),
		Provenance: &Provenance{
			SourceTemplateID: t.TemplateID,
			SourceVersionID:  t.VersionID,
			DerivedAt:        time.Now().UTC(),
		},
	}, nil
}

// VersionReference provides an immutable reference to a specific template version.
type VersionReference struct {
	TemplateID string `json:"template_id"`
	VersionID  string `json:"version_id"`
}

// ChecklistInstance represents an execution instance pinned to an exact published template snapshot.
// Prior instances retain the exact version used even after newer versions are published or prior versions are retired.
type ChecklistInstance struct {
	InstanceID  string           `json:"instance_id"`
	TenantID    string           `json:"tenant_id"`
	TemplateRef VersionReference `json:"template_ref"`
	Snapshot    TemplateSnapshot `json:"snapshot"`
	CreatedAt   time.Time        `json:"created_at"`
}

// Instantiate creates an operational checklist instance pinned to this published version snapshot.
// Requires StatePublished. Fails closed on Draft, InReview, Approved, or Retired.
func (t *TemplateVersion) Instantiate(instanceID, tenantID string) (*ChecklistInstance, error) {
	if t.State != StatePublished || t.PublishedSnapshot == nil {
		return nil, ErrNotPublished
	}
	tInstID := strings.TrimSpace(instanceID)
	if tInstID == "" {
		return nil, ErrEmptyInstanceID
	}
	tTenantID := strings.TrimSpace(tenantID)
	if tTenantID == "" {
		return nil, ErrEmptyTenantID
	}

	snap := *t.PublishedSnapshot
	snap.Questions = cloneQuestions(t.PublishedSnapshot.Questions)

	return &ChecklistInstance{
		InstanceID: tInstID,
		TenantID:   tTenantID,
		TemplateRef: VersionReference{
			TemplateID: t.TemplateID,
			VersionID:  t.VersionID,
		},
		Snapshot:  snap,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// IsStale evaluates whether this instance was created from a version preceding the latest active version.
func (ci *ChecklistInstance) IsStale(latestVersionID string) bool {
	return ci.TemplateRef.VersionID != strings.TrimSpace(latestVersionID)
}
