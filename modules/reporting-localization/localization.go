package reporting

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Locale identifies an IETF BCP 47 language tag for supported test locales.
type Locale string

const (
	LocaleEnUS            Locale = "en-US"
	LocaleThTH            Locale = "th-TH"
	DefaultFallbackLocale Locale = LocaleEnUS
)

// FallbackDisposition records how a translation key was resolved.
type FallbackDisposition string

const (
	DispositionExactMatch FallbackDisposition = "EXACT_MATCH"
	DispositionFallback   FallbackDisposition = "FALLBACK"
	DispositionMissing    FallbackDisposition = "MISSING"
)

var (
	ErrBlankLocale       = errors.New("locale cannot be blank")
	ErrBlankKey          = errors.New("translation key cannot be blank")
	ErrInvalidTimeZone   = errors.New("invalid or unsupported time zone")
	ErrUnsupportedLocale = errors.New("unsupported test locale")
)

// LocalizedMessage represents the resolved text with explicit fallback visibility.
type LocalizedMessage struct {
	Key           string              `json:"key"`
	Locale        Locale              `json:"locale"`
	Text          string              `json:"text"`
	Disposition   FallbackDisposition `json:"disposition"`
	FallbackUsed  bool                `json:"fallback_used"`
	MissingNotice string              `json:"missing_notice,omitempty"`
}

// LocalizationCatalog coordinates in-memory translation keys, bundles, and visible fallback resolution.
type LocalizationCatalog struct {
	mu           sync.RWMutex
	translations map[Locale]map[string]string
	fallback     Locale
}

// NewLocalizationCatalog constructs an initialized translation catalog.
func NewLocalizationCatalog(fallback Locale) *LocalizationCatalog {
	if fallback == "" {
		fallback = DefaultFallbackLocale
	}
	cat := &LocalizationCatalog{
		translations: make(map[Locale]map[string]string),
		fallback:     fallback,
	}
	cat.translations[LocaleEnUS] = make(map[string]string)
	cat.translations[LocaleThTH] = make(map[string]string)
	return cat
}

// RegisterTranslation registers a single translation key for a locale.
func (c *LocalizationCatalog) RegisterTranslation(loc Locale, key, text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	loc = Locale(strings.TrimSpace(string(loc)))
	if loc == "" {
		return ErrBlankLocale
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return ErrBlankKey
	}

	if c.translations[loc] == nil {
		c.translations[loc] = make(map[string]string)
	}
	c.translations[loc][k] = text
	return nil
}

// RegisterBundle registers a map of translation keys for a locale.
func (c *LocalizationCatalog) RegisterBundle(loc Locale, bundle map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	loc = Locale(strings.TrimSpace(string(loc)))
	if loc == "" {
		return ErrBlankLocale
	}

	if c.translations[loc] == nil {
		c.translations[loc] = make(map[string]string)
	}
	for k, v := range bundle {
		trimmedKey := strings.TrimSpace(k)
		if trimmedKey != "" {
			c.translations[loc][trimmedKey] = v
		}
	}
	return nil
}

// Resolve resolves a translation key for the target locale.
// Fails open to visible fallback:
// - Exact match in target locale -> DispositionExactMatch
// - Missing in target, found in fallback -> DispositionFallback with FallbackUsed = true
// - Missing in both -> DispositionMissing with explicit "[MISSING: key]" placeholder
func (c *LocalizationCatalog) Resolve(loc Locale, key string) LocalizedMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	k := strings.TrimSpace(key)
	if k == "" {
		return LocalizedMessage{
			Key:           k,
			Locale:        loc,
			Text:          "[EMPTY_KEY]",
			Disposition:   DispositionMissing,
			MissingNotice: "translation key was empty",
		}
	}

	// 1. Try exact match in target locale
	if targetMap, exists := c.translations[loc]; exists {
		if val, found := targetMap[k]; found {
			return LocalizedMessage{
				Key:          k,
				Locale:       loc,
				Text:         val,
				Disposition:  DispositionExactMatch,
				FallbackUsed: false,
			}
		}
	}

	// 2. Try fallback locale
	if loc != c.fallback {
		if fallbackMap, exists := c.translations[c.fallback]; exists {
			if val, found := fallbackMap[k]; found {
				return LocalizedMessage{
					Key:           k,
					Locale:        c.fallback,
					Text:          val,
					Disposition:   DispositionFallback,
					FallbackUsed:  true,
					MissingNotice: fmt.Sprintf("key %q not found in %s; fallen back to %s", k, loc, c.fallback),
				}
			}
		}
	}

	// 3. Completely missing: return visible missing indicator
	return LocalizedMessage{
		Key:           k,
		Locale:        loc,
		Text:          fmt.Sprintf("[MISSING: %s]", k),
		Disposition:   DispositionMissing,
		FallbackUsed:  false,
		MissingNotice: fmt.Sprintf("translation key %q is missing in both %s and fallback %s", k, loc, c.fallback),
	}
}

// FormatNumber formats an integer or float with locale-specific digit separators.
func FormatNumber(val float64, loc Locale) string {
	intPart := int64(val)
	fracPart := val - float64(intPart)

	// Format integer with comma thousands separator
	sign := ""
	if intPart < 0 {
		sign = "-"
		intPart = -intPart
	}

	str := fmt.Sprintf("%d", intPart)
	var result []byte
	n := len(str)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, str[i])
	}

	formattedInt := sign + string(result)
	if fracPart == 0 {
		return formattedInt
	}
	fracStr := fmt.Sprintf("%.2f", fracPart)
	if len(fracStr) > 2 {
		return formattedInt + fracStr[1:] // append ".xx"
	}
	return formattedInt
}

// FormatUnit formats a value with its localized unit label.
func FormatUnit(val float64, unit string, loc Locale) string {
	numStr := FormatNumber(val, loc)
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "celsius", "c":
		return numStr + " °C"
	case "meter", "m":
		if loc == LocaleThTH {
			return numStr + " ม."
		}
		return numStr + " m"
	case "kilogram", "kg":
		if loc == LocaleThTH {
			return numStr + " กก."
		}
		return numStr + " kg"
	case "decibel", "db":
		if loc == LocaleThTH {
			return numStr + " เดซิเบล"
		}
		return numStr + " dB"
	default:
		return numStr + " " + unit
	}
}

// FormatDateTime formats a timestamp for the specified locale and time zone.
// Supports "Asia/Bangkok" (UTC+7) and "UTC".
// For th-TH, dates use the Buddhist Era (BE = CE + 543).
func FormatDateTime(t time.Time, loc Locale, timeZoneName string) (string, error) {
	var zone *time.Location
	switch strings.TrimSpace(timeZoneName) {
	case "Asia/Bangkok":
		zone = time.FixedZone("Asia/Bangkok", 7*3600)
	case "UTC", "":
		zone = time.UTC
	default:
		var err error
		zone, err = time.LoadLocation(timeZoneName)
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrInvalidTimeZone, timeZoneName)
		}
	}

	zonedTime := t.In(zone)

	if loc == LocaleThTH {
		beYear := zonedTime.Year() + 543
		return fmt.Sprintf("%02d/%02d/%04d %02d:%02d:%02d (%s)",
			zonedTime.Day(), zonedTime.Month(), beYear,
			zonedTime.Hour(), zonedTime.Minute(), zonedTime.Second(),
			timeZoneName), nil
	}

	// Default en-US ISO-style
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d (%s)",
		zonedTime.Year(), zonedTime.Month(), zonedTime.Day(),
		zonedTime.Hour(), zonedTime.Minute(), zonedTime.Second(),
		timeZoneName), nil
}

// CalculateTextExpansion evaluates string byte and character length differences
// between English and Thai representations to verify UI layout expansion tolerances.
func CalculateTextExpansion(englishText, thaiText string) float64 {
	enLen := len([]rune(englishText))
	if enLen == 0 {
		return 0.0
	}
	thLen := len([]rune(thaiText))
	return float64(thLen) / float64(enLen)
}
