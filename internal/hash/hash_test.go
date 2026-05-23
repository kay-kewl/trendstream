package hash

import "testing"

func TestString64IsDeterministic(t *testing.T) {
	t.Parallel()

	first := String64("iphone 15")
	second := String64("iphone 15")

	if first != second {
		t.Fatalf("hash should be deterministic: first=%d second=%d", first, second)
	}
}

func TestIndexIsDeterministic(t *testing.T) {
	t.Parallel()

	first := Index("iphone 15", 32)
	second := Index("iphone 15", 32)

	if first != second {
		t.Fatalf("index should be deterministic: first=%d second=%d", first, second)
	}
}

func TestIndexReturnsValueInsideRange(t *testing.T) {
	t.Parallel()

	const size = 32

	for _, value := range []string{
		"",
		"iphone 15",
		"кроссовки женские",
		"long query with several words",
	} {
		got := Index(value, size)

		if got < 0 || got >= size {
			t.Fatalf("index out of range for %q: got %d, size %d", value, got, size)
		}
	}
}

func TestIndexWithInvalidSizeReturnsZero(t *testing.T) {
	t.Parallel()

	if got := Index("iphone 15", 0); got != 0 {
		t.Fatalf("index mismatch for zero size: got %d, want 0", got)
	}

	if got := Index("iphone 15", -1); got != 0 {
		t.Fatalf("index mismatch for negative size: got %d, want 0", got)
	}
}
