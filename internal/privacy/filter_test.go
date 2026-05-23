package privacy

import "testing"

func TestInspectAllowsRegularProductQueries(t *testing.T) {
	t.Parallel()

	tests := []string{
		"iphone 15 pro",
		"samsung galaxy s23 256gb",
		"кроссовки женские 39 размер",
		"ноутбук 16gb ram 512gb ssd",
		"чехол iphone 14",
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt, func(t *testing.T) {
			t.Parallel()

			result := Inspect(tt)
			if result.Sensitive {
				t.Fatalf("expected query %q to be allowed, got sensitive rule %q", tt, result.Rule)
			}
		})
	}
}

func TestInspectDetectsEmail(t *testing.T) {
	t.Parallel()

	result := Inspect("john.doe@example.com")
	if !result.Sensitive {
		t.Fatalf("expected email query to be sensitive")
	}

	if result.Rule != RuleEmail {
		t.Fatalf("rule mismatch: got %q, want %q", result.Rule, RuleEmail)
	}
}

func TestInspectDetectsEmailInsideQuery(t *testing.T) {
	t.Parallel()

	result := Inspect("найти заказ john.doe@example.com")
	if !result.Sensitive {
		t.Fatalf("expected query with email to be sensitive")
	}

	if result.Rule != RuleEmail {
		t.Fatalf("rule mismatch: got %q, want %q", result.Rule, RuleEmail)
	}
}

func TestInspectDetectsLikelyCardNumber(t *testing.T) {
	t.Parallel()

	result := Inspect("4111 1111 1111 1111")
	if !result.Sensitive {
		t.Fatalf("expected card-like query to be sensitive")
	}

	if result.Rule != RuleLikelyCard {
		t.Fatalf("rule mismatch: got %q, want %q", result.Rule, RuleLikelyCard)
	}
}

func TestInspectDetectsLongDigitRun(t *testing.T) {
	t.Parallel()

	result := Inspect("123456789")
	if !result.Sensitive {
		t.Fatalf("expected long digit run to be sensitive")
	}

	if result.Rule != RuleLongDigitRun {
		t.Fatalf("rule mismatch: got %q, want %q", result.Rule, RuleLongDigitRun)
	}
}

func TestInspectDetectsLikelyPhoneNumber(t *testing.T) {
	t.Parallel()

	result := Inspect("+7 999 123 45 67")
	if !result.Sensitive {
		t.Fatalf("expected phone-like query to be sensitive")
	}

	if result.Rule != RuleLikelyPhone {
		t.Fatalf("rule mismatch: got %q, want %q", result.Rule, RuleLikelyPhone)
	}
}

func TestInspectDetectsPhoneWithMarker(t *testing.T) {
	t.Parallel()

	result := Inspect("телефон 8 999 123 45 67")
	if !result.Sensitive {
		t.Fatalf("expected phone-like query with marker to be sensitive")
	}

	if result.Rule != RuleLikelyPhone {
		t.Fatalf("rule mismatch: got %q, want %q", result.Rule, RuleLikelyPhone)
	}
}

func TestInspectDetectsHighDigitRatio(t *testing.T) {
	t.Parallel()

	result := Inspect("заказ 123 456 789 012")
	if !result.Sensitive {
		t.Fatalf("expected high digit ratio query to be sensitive")
	}

	if result.Rule != RuleHighDigitRatio {
		t.Fatalf("rule mismatch: got %q, want %q", result.Rule, RuleHighDigitRatio)
	}
}

func TestContainsSensitiveData(t *testing.T) {
	t.Parallel()

	if !ContainsSensitiveData("user@example.com") {
		t.Fatalf("expected sensitive query")
	}

	if ContainsSensitiveData("iphone 15 pro") {
		t.Fatalf("expected regular query to be allowed")
	}
}
