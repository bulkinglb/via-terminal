package macros

import (
	"bytes"
	"slices"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	in := []string{
		"Hello world{ENT}",
		"{LCTL,C}{100}{LCTL,V}",
		"{+LSFT}abc{-LSFT}",
		`a\{b\\c\n`,
		"",
	}
	buf, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	chord := []byte{1, 2, 0xE0, 1, 2, 0x06, 1, 3, 0x06, 1, 3, 0xE0}
	if !bytes.Contains(buf, append(chord, 1, 4, '1', '0', '0', '|')) {
		t.Errorf("{LCTL,C}{100} should press, release in reverse, then wait: % X", buf)
	}
	// Room left in the buffer past the last macro reads as empty macros.
	got, err := Decode(append(buf, make([]byte, 10)...), 7)
	if err != nil {
		t.Fatal(err)
	}
	if want := append(slices.Clone(in), "", ""); !slices.Equal(got, want) {
		t.Errorf("round trip:\n got %q\nwant %q", got, want)
	}

	if got, _ := Decode(mustEncode(t, "{kc_lctl, kc_c}"), 1); got[0] != "{LCTL,C}" {
		t.Errorf("VIA's KC_ names and spaces should read back as %q, got %q", "{LCTL,C}", got[0])
	}
	for _, bad := range []string{"{A", "{}", "{NOPE}", "{MO(1)}", "{NO}", "{12345}", "ä", `\`} {
		if _, err := Encode([]string{bad}); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func mustEncode(t *testing.T, s string) []byte {
	b, err := Encode([]string{s})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
