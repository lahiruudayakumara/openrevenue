// Package domain owns immutable return versions and filing lifecycle rules.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	calculation "github.com/opencorex-org/openrevenue/internal/calculation/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type ReturnTag struct{}
type TaxReturnID = id.ID[ReturnTag]
type Status string

const (
	Draft      Status = "DRAFT"
	Validated  Status = "VALIDATED"
	Calculated Status = "CALCULATED"
	Submitted  Status = "SUBMITTED"
)

const (
	DefaultFormVersion = "sample-income-v1"
	DefaultRuleVersion = "fictional-flat-rate-v1"
)

var versionPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,63}$`)

type Line struct {
	Code        string `json:"code"`
	AmountMinor int64  `json:"amountMinor"`
}

type FieldDefinition struct {
	Code     string `json:"code"`
	Required bool   `json:"required"`
}

type FormDefinition struct {
	Version     string            `json:"version"`
	RuleVersion string            `json:"ruleVersion"`
	Fields      []FieldDefinition `json:"fields"`
}

func (f FormDefinition) Validate() error {
	if !versionPattern.MatchString(f.Version) || !versionPattern.MatchString(f.RuleVersion) {
		return errors.New("form and rule versions must be safe immutable identifiers")
	}
	if len(f.Fields) == 0 {
		return errors.New("form must define at least one field")
	}
	seen := map[string]struct{}{}
	for _, field := range f.Fields {
		if !validFieldCode(field.Code) {
			return errors.New("form contains an invalid field code")
		}
		if _, exists := seen[field.Code]; exists {
			return errors.New("form contains a duplicate field code")
		}
		seen[field.Code] = struct{}{}
	}
	return nil
}

func DefaultForm() FormDefinition {
	return FormDefinition{
		Version: DefaultFormVersion, RuleVersion: DefaultRuleVersion,
		Fields: []FieldDefinition{{Code: "GROSS", Required: true}},
	}
}

type ValidationIssue struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

type ValidationResult struct {
	Valid       bool              `json:"valid"`
	FormVersion string            `json:"formVersion"`
	PayloadHash string            `json:"payloadHash"`
	Issues      []ValidationIssue `json:"issues"`
}

type TaxReturn struct {
	ID                TaxReturnID         `json:"id"`
	TenantID          string              `json:"tenantId"`
	Jurisdiction      string              `json:"jurisdiction"`
	TaxpayerID        string              `json:"taxpayerId"`
	RegistrationID    string              `json:"registrationId"`
	PeriodID          string              `json:"periodId"`
	FormVersion       string              `json:"formVersion"`
	RuleVersion       string              `json:"ruleVersion"`
	Lines             []Line              `json:"lines"`
	Status            Status              `json:"status"`
	Revision          int                 `json:"revision"`
	OriginalReturnID  string              `json:"originalReturnId"`
	SupersedesID      string              `json:"supersedesId,omitempty"`
	Validation        *ValidationResult   `json:"validation,omitempty"`
	Calculation       *calculation.Result `json:"calculation,omitempty"`
	FrozenPayloadHash string              `json:"frozenPayloadHash,omitempty"`
	SubmittedAt       *time.Time          `json:"submittedAt,omitempty"`
}

func New(scope foundation.Context, taxpayerID, registrationID, periodID string, lines []Line) (TaxReturn, error) {
	return NewVersioned(scope, taxpayerID, registrationID, periodID, DefaultFormVersion, DefaultRuleVersion, lines)
}

func NewVersioned(
	scope foundation.Context,
	taxpayerID, registrationID, periodID, formVersion, ruleVersion string,
	lines []Line,
) (TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return TaxReturn{}, err
	}
	if strings.TrimSpace(taxpayerID) == "" || strings.TrimSpace(registrationID) == "" || strings.TrimSpace(periodID) == "" {
		return TaxReturn{}, errors.New("taxpayer, registration, and period are required")
	}
	if !versionPattern.MatchString(formVersion) || !versionPattern.MatchString(ruleVersion) {
		return TaxReturn{}, errors.New("form and rule versions must be safe immutable identifiers")
	}
	value := TaxReturn{
		ID: id.New[ReturnTag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), TaxpayerID: taxpayerID,
		RegistrationID: registrationID, PeriodID: periodID,
		FormVersion: formVersion, RuleVersion: ruleVersion,
		Lines: cloneLines(lines), Status: Draft, Revision: 1,
	}
	value.OriginalReturnID = value.ID.String()
	return value, nil
}

func (r *TaxReturn) Validate() error {
	result := r.ValidateAgainst(DefaultForm())
	if !result.Valid {
		return errors.New("return validation failed")
	}
	return nil
}

func (r *TaxReturn) ValidateAgainst(form FormDefinition) ValidationResult {
	result := ValidationResult{
		FormVersion: r.FormVersion, PayloadHash: payloadHash(r.Lines),
		Issues: make([]ValidationIssue, 0),
	}
	if r.Status != Draft {
		result.Issues = append(result.Issues, ValidationIssue{Field: "$", Code: "RETURN_NOT_DRAFT"})
		r.Validation = &result
		return result
	}
	if err := form.Validate(); err != nil || form.Version != r.FormVersion || form.RuleVersion != r.RuleVersion {
		result.Issues = append(result.Issues, ValidationIssue{Field: "$", Code: "FORM_VERSION_INCOMPATIBLE"})
		r.Validation = &result
		return result
	}
	allowed := map[string]FieldDefinition{}
	for _, field := range form.Fields {
		allowed[field.Code] = field
	}
	present := map[string]struct{}{}
	for _, line := range r.Lines {
		switch {
		case !validFieldCode(line.Code):
			result.Issues = append(result.Issues, ValidationIssue{Field: "lines", Code: "FIELD_CODE_INVALID"})
		case line.AmountMinor < 0:
			result.Issues = append(result.Issues, ValidationIssue{Field: line.Code, Code: "AMOUNT_NEGATIVE"})
		default:
			if _, exists := allowed[line.Code]; !exists {
				result.Issues = append(result.Issues, ValidationIssue{Field: line.Code, Code: "FIELD_NOT_IN_FORM"})
			}
		}
		if _, duplicate := present[line.Code]; duplicate {
			result.Issues = append(result.Issues, ValidationIssue{Field: line.Code, Code: "FIELD_DUPLICATE"})
		}
		present[line.Code] = struct{}{}
	}
	for _, field := range form.Fields {
		if _, exists := present[field.Code]; field.Required && !exists {
			result.Issues = append(result.Issues, ValidationIssue{Field: field.Code, Code: "FIELD_REQUIRED"})
		}
	}
	sort.SliceStable(result.Issues, func(i, j int) bool {
		if result.Issues[i].Field == result.Issues[j].Field {
			return result.Issues[i].Code < result.Issues[j].Code
		}
		return result.Issues[i].Field < result.Issues[j].Field
	})
	result.Valid = len(result.Issues) == 0
	r.Validation = &result
	if result.Valid {
		r.Status = Validated
	}
	return result
}

func (r *TaxReturn) RecordCalculation(result calculation.Result) error {
	if r.Status != Validated {
		return errors.New("return must be validated before calculation")
	}
	if err := result.Validate(); err != nil {
		return err
	}
	if result.RuleVersion != r.RuleVersion || result.InputHash != payloadHash(r.Lines) {
		return errors.New("calculation result is stale or incompatible")
	}
	copy := result
	copy.Steps = append([]calculation.Step(nil), result.Steps...)
	r.Calculation = &copy
	r.Status = Calculated
	return nil
}

func (r *TaxReturn) Submit(now time.Time, currentVersions ...string) error {
	if r.Status != Calculated || r.Validation == nil || !r.Validation.Valid || r.Calculation == nil {
		return errors.New("return must be validated and calculated")
	}
	if now.IsZero() {
		return errors.New("submission time is required")
	}
	if len(currentVersions) == 2 &&
		(currentVersions[0] != r.FormVersion || currentVersions[1] != r.RuleVersion) {
		return errors.New("return uses stale or incompatible versions")
	}
	hash := payloadHash(r.Lines)
	if r.Validation.PayloadHash != hash || r.Calculation.InputHash != hash {
		return errors.New("return payload changed after validation or calculation")
	}
	submittedAt := now.UTC()
	r.Status = Submitted
	r.SubmittedAt = &submittedAt
	r.FrozenPayloadHash = hash
	r.Lines = cloneLines(r.Lines)
	return nil
}

func (r TaxReturn) Amend(scope foundation.Context) (TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return TaxReturn{}, err
	}
	if r.TenantID != scope.Tenant().String() || r.Jurisdiction != scope.Jurisdiction().String() {
		return TaxReturn{}, errors.New("return is outside the request scope")
	}
	if r.Status != Submitted {
		return TaxReturn{}, errors.New("only a submitted return can be amended")
	}
	amendment, err := NewVersioned(
		scope, r.TaxpayerID, r.RegistrationID, r.PeriodID,
		r.FormVersion, r.RuleVersion, r.Lines,
	)
	if err != nil {
		return TaxReturn{}, err
	}
	amendment.Revision = r.Revision + 1
	amendment.OriginalReturnID = r.OriginalReturnID
	amendment.SupersedesID = r.ID.String()
	return amendment, nil
}

func cloneLines(lines []Line) []Line {
	return append([]Line(nil), lines...)
}

func payloadHash(lines []Line) string {
	hash := sha256.New()
	for _, line := range lines {
		hash.Write([]byte(line.Code))
		hash.Write([]byte{0})
		for shift := 56; shift >= 0; shift -= 8 {
			hash.Write([]byte{byte(uint64(line.AmountMinor) >> shift)})
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validFieldCode(value string) bool {
	if len(value) < 2 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}
