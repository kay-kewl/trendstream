package privacy

import (
	"regexp"
	"strings"
	"unicode"
)

type Rule string

const (
	RuleNone           Rule = ""
	RuleEmail          Rule = "email"
	RuleLikelyCard     Rule = "likely_card"
	RuleLongDigitRun   Rule = "long_digit_run"
	RuleLikelyPhone    Rule = "likely_phone"
	RuleHighDigitRatio Rule = "high_digit_ratio"
)

type Result struct {
	Sensitive bool
	Rule      Rule
}

var (
	emailPattern        = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	longDigitRunPattern = regexp.MustCompile(`[0-9]{9,}`)
)

func ContainsSensitiveData(query string) bool {
	return Inspect(query).Sensitive
}

func Inspect(query string) Result {
	query = strings.TrimSpace(query)
	if query == "" {
		return Result{}
	}

	if emailPattern.MatchString(query) {
		return Result{
			Sensitive: true,
			Rule:      RuleEmail,
		}
	}

	digits := asciiDigits(query)

	if len(digits) >= 13 && len(digits) <= 19 && luhnValid(digits) {
		return Result{
			Sensitive: true,
			Rule:      RuleLikelyCard,
		}
	}

	if longDigitRunPattern.MatchString(query) {
		return Result{
			Sensitive: true,
			Rule:      RuleLongDigitRun,
		}
	}

	stats := characterStats(query)

	if looksLikePhone(query, stats) {
		return Result{
			Sensitive: true,
			Rule:      RuleLikelyPhone,
		}
	}

	if hasHighDigitRatio(stats) {
		return Result{
			Sensitive: true,
			Rule:      RuleHighDigitRatio,
		}
	}

	return Result{}
}

type charStats struct {
	runes   int
	digits  int
	letters int
}

func characterStats(s string) charStats {
	var stats charStats

	for _, r := range s {
		stats.runes++

		switch {
		case unicode.IsDigit(r):
			stats.digits++
		case unicode.IsLetter(r):
			stats.letters++
		}
	}

	return stats
}

func looksLikePhone(query string, stats charStats) bool {
	if stats.digits < 10 || stats.digits > 15 {
		return false
	}

	if hasPhoneMarker(query) {
		return true
	}

	if containsLetters(query) {
		return false
	}

	return containsOnlyPhoneNumberRunes(query)
}

func containsLetters(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}

	return false
}

func containsOnlyPhoneNumberRunes(s string) bool {
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			continue
		case unicode.IsSpace(r):
			continue
		case r == '+', r == '-', r == '(', r == ')', r == '.':
			continue
		default:
			return false
		}
	}

	return true
}

func hasPhoneMarker(query string) bool {
	markers := []string{
		"тел",
		"телефон",
		"phone",
		"mobile",
		"номер телефона",
	}

	for _, marker := range markers {
		if strings.Contains(query, marker) {
			return true
		}
	}

	return false
}

func hasHighDigitRatio(stats charStats) bool {
	if stats.runes == 0 {
		return false
	}

	if stats.digits < 10 {
		return false
	}

	digitRatio := float64(stats.digits) / float64(stats.runes)

	return digitRatio >= 0.40
}

func asciiDigits(s string) string {
	var builder strings.Builder

	for _, r := range s {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func luhnValid(digits string) bool {
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	sum := 0
	double := false

	for i := len(digits) - 1; i >= 0; i-- {
		digit := int(digits[i] - '0')

		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		double = !double
	}

	return sum%10 == 0
}
