package domain

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	identifierSchemePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,31}$`)
	identifierValuePattern  = regexp.MustCompile(`^[A-Z0-9]{1,128}$`)
)

type IdentifierNormalizer interface {
	Normalize(string) (string, error)
}

type IdentifierNormalizerFunc func(string) (string, error)

func (f IdentifierNormalizerFunc) Normalize(value string) (string, error) {
	return f(value)
}

// UpperAlphanumericNormalizer is a conservative default. Country packs can
// supply a scheme-specific normalizer and validator at the extension boundary.
type UpperAlphanumericNormalizer struct{}

func (UpperAlphanumericNormalizer) Normalize(value string) (string, error) {
	var normalized strings.Builder
	for _, r := range strings.TrimSpace(value) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			if r > unicode.MaxASCII {
				return "", invalid("taxpayer identifier", "must contain ASCII letters and digits")
			}
			normalized.WriteRune(unicode.ToUpper(r))
		case r == ' ', r == '-':
			continue
		default:
			return "", invalid("taxpayer identifier", "contains an unsupported character")
		}
	}
	return normalized.String(), nil
}

type TaxpayerIdentifier struct {
	tenant       TenantID
	jurisdiction Jurisdiction
	scheme       string
	value        string
}

func NewTaxpayerIdentifier(
	tenant TenantID,
	jurisdiction Jurisdiction,
	scheme string,
	rawValue string,
	normalizer IdentifierNormalizer,
) (TaxpayerIdentifier, error) {
	if err := tenant.Validate(); err != nil {
		return TaxpayerIdentifier{}, err
	}
	if err := jurisdiction.Validate(); err != nil {
		return TaxpayerIdentifier{}, err
	}
	if !identifierSchemePattern.MatchString(scheme) {
		return TaxpayerIdentifier{}, invalid("identifier scheme", "must be 2-32 uppercase letters, digits, or underscores")
	}
	if normalizer == nil {
		return TaxpayerIdentifier{}, invalid("identifier normalizer", "is required")
	}
	value, err := normalizer.Normalize(rawValue)
	if err != nil {
		return TaxpayerIdentifier{}, err
	}
	if !identifierValuePattern.MatchString(value) {
		return TaxpayerIdentifier{}, invalid("taxpayer identifier", "normalizes to an invalid or empty value")
	}
	return TaxpayerIdentifier{
		tenant: tenant, jurisdiction: jurisdiction, scheme: scheme, value: value,
	}, nil
}

func (v TaxpayerIdentifier) Tenant() TenantID           { return v.tenant }
func (v TaxpayerIdentifier) Jurisdiction() Jurisdiction { return v.jurisdiction }
func (v TaxpayerIdentifier) Scheme() string             { return v.scheme }
func (v TaxpayerIdentifier) String() string             { return v.value }
