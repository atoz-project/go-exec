package util

import (
	"regexp"
	"strings"
	"testing"
)

func TestRandomHostname_MatchesRegex(t *testing.T) {
	for i := 0; i < 100; i++ {
		h := RandomHostname()
		if !randHostnameRegex.MatchString(h) {
			t.Fatalf("RandomHostname()=%q does not match regex %q", h, randHostnameRegex.String())
		}
		if len(h) > 15 {
			t.Fatalf("RandomHostname()=%q length=%d > 15", h, len(h))
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("abc", 3); got != "abc" {
		t.Fatalf("Truncate len<=n got=%q", got)
	}
	if got := Truncate("abcd", 3); got != "abc..." {
		t.Fatalf("Truncate len>n got=%q", got)
	}
	if got := Truncate("abcd", 0); got != "..." {
		t.Fatalf("Truncate n=0 got=%q", got)
	}
}

func TestRandomWindowsTempFile_Format(t *testing.T) {
	p := RandomWindowsTempFile()
	prefix := `\Windows\Temp\`
	if !strings.HasPrefix(p, prefix) {
		t.Fatalf("RandomWindowsTempFile()=%q missing expected prefix %q", p, prefix)
	}

	rest := strings.TrimPrefix(p, prefix)
	if rest == "" {
		t.Fatalf("RandomWindowsTempFile()=%q has empty suffix", p)
	}

	// UUID uppercase form: 8-4-4-4-12 hex (upper) with dashes.
	re := regexp.MustCompile(`^[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}$`)
	if !re.MatchString(rest) {
		t.Fatalf("RandomWindowsTempFile() suffix=%q not an uppercase UUID", rest)
	}
}

func TestRandomStringFromCharset_LengthAndCharset(t *testing.T) {
	charset := "ab01"
	s := RandomStringFromCharset(charset, 64)
	if len(s) != 64 {
		t.Fatalf("len(RandomStringFromCharset)=%d, want 64", len(s))
	}
	for _, ch := range s {
		if !strings.ContainsRune(charset, ch) {
			t.Fatalf("RandomStringFromCharset produced rune %q not in charset %q", ch, charset)
		}
	}
}
