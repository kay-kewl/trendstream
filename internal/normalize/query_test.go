package normalize

import "testing"

func TestQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{
			name: "lowercases and trims latin query",
			raw:  "  IPhone   15 PRO ",
			want: "iphone 15 pro",
			ok:   true,
		},
		{
			name: "lowercases and trims cyrillic query",
			raw:  "\tКроссовки   Женские\n",
			want: "кроссовки женские",
			ok:   true,
		},
		{
			name: "collapses mixed whitespace",
			raw:  "one\t\t two\nthree",
			want: "one two three",
			ok:   true,
		},
		{
			name: "drops non-whitespace control characters",
			raw:  "abc\x00def",
			want: "abcdef",
			ok:   true,
		},
		{
			name: "empty query is rejected",
			raw:  "",
			want: "",
			ok:   false,
		},
		{
			name: "spaces only query is rejected",
			raw:  " \t\n ",
			want: "",
			ok:   false,
		},
		{
			name: "control characters only query is rejected",
			raw:  "\x00\x01\x02",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := Query(tt.raw)
			if ok != tt.ok {
				t.Fatalf("ok mismatch: got %v, want %v", ok, tt.ok)
			}

			if got != tt.want {
				t.Fatalf("query mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}
