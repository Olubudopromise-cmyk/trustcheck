package scoring

import "testing"

func TestClamp(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{-5, 0},
		{120, 100},
		{55, 55},
	}
	for _, tc := range cases {
		if got := Clamp(tc.in); got != tc.want {
			t.Errorf("Clamp(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestBuilder(t *testing.T) {
	b := New()
	if got := b.Score(); got != 0 {
		t.Fatalf("new builder score = %d, want 0", got)
	}
	b.Add(40)
	if got := b.Score(); got != 40 {
		t.Errorf("after Add(40) = %d, want 40", got)
	}
	b.Sub(10)
	if got := b.Score(); got != 30 {
		t.Errorf("after Sub(10) = %d, want 30", got)
	}
}

func TestBuilder_Overflow(t *testing.T) {
	b := New()
	b.Add(120)
	if got := b.Score(); got != 100 {
		t.Errorf("overflow score = %d, want 100", got)
	}
}

func TestBuilder_Underflow(t *testing.T) {
	b := New()
	b.Sub(120)
	if got := b.Score(); got != 0 {
		t.Errorf("underflow score = %d, want 0", got)
	}
}

func TestStatusFromScore(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{100, "verified"},
		{90, "verified"},
		{89, "good"},
		{70, "good"},
		{69, "warning"},
		{50, "warning"},
		{49, "poor"},
		{20, "poor"},
		{19, "invalid"},
		{10, "invalid"},
		{0, "invalid"},
	}
	for _, tc := range cases {
		if got := StatusFromScore(tc.in); got != tc.want {
			t.Errorf("StatusFromScore(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
