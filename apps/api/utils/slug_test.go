package utils

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Panduan Onboarding": "panduan-onboarding",
		"  SOP   Keuangan  ": "sop-keuangan",
		"API v2.1 — Auth":    "api-v2-1-auth",
		"Tanya/Jawab (FAQ)":  "tanya-jawab-faq",
		"---":                "",
		"Ünïcödé":            "n-c-d",
		"2026 Q1":            "2026-q1",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueSlugAppendsSuffix(t *testing.T) {
	taken := map[string]bool{"sop": true, "sop-2": true}
	exists := func(c string) (bool, error) { return taken[c], nil }

	got, err := UniqueSlug("sop", exists)
	if err != nil {
		t.Fatal(err)
	}
	if got != "sop-3" {
		t.Errorf("got %q, want sop-3", got)
	}

	// An empty base is the all-non-ASCII title Slugify cannot represent.
	got, err = UniqueSlug("", exists)
	if err != nil {
		t.Fatal(err)
	}
	if got != "untitled" {
		t.Errorf("got %q, want untitled", got)
	}
}
