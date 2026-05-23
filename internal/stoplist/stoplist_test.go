package stoplist

import "testing"

func TestStopListContainsNormalizedTerm(t *testing.T) {
	t.Parallel()

	stopList := New([]string{
		"  Casino   Online ",
		"КАЗИНО",
	})

	if !stopList.Contains("casino online") {
		t.Fatalf("expected stop-list to contain normalized latin term")
	}

	if !stopList.Contains(" казино ") {
		t.Fatalf("expected stop-list to contain normalized cyrillic term")
	}
}

func TestStopListTermsAreSortedAndDeduplicated(t *testing.T) {
	t.Parallel()

	stopList := New([]string{
		"banana",
		"Apple",
		"apple",
		" banana ",
		"",
	})

	terms := stopList.Terms()

	want := []string{"apple", "banana"}

	if len(terms) != len(want) {
		t.Fatalf("terms length mismatch: got %d, want %d; terms=%#v", len(terms), len(want), terms)
	}

	for i := range want {
		if terms[i] != want[i] {
			t.Fatalf("term %d mismatch: got %q, want %q", i, terms[i], want[i])
		}
	}
}

func TestStopListReplace(t *testing.T) {
	t.Parallel()

	stopList := New([]string{"old"})
	if !stopList.Contains("old") {
		t.Fatalf("expected old term to exist before replace")
	}

	stopList.Replace([]string{"new"})

	if stopList.Contains("old") {
		t.Fatalf("old term should not exist after replace")
	}

	if !stopList.Contains("new") {
		t.Fatalf("new term should exist after replace")
	}
}

func TestNormalizeTermRejectsEmptyTerm(t *testing.T) {
	t.Parallel()

	_, err := NormalizeTerm(" \t\n ")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSnapshotReturnsCopy(t *testing.T) {
	t.Parallel()

	stopList := New([]string{"casino"})

	snapshot := stopList.Snapshot()
	delete(snapshot.Exact, "casino")

	if !stopList.Contains("casino") {
		t.Fatalf("external snapshot mutation should not affect stop-list")
	}
}
