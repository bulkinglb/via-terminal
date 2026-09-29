package keycodes

import "testing"

func TestName(t *testing.T) {
	custom := []string{"BT DEV1"}
	for code, want := range map[uint16]string{
		0x0029: "ESC",
		0x003A: "F1",
		0x0068: "F13",
		0x5221: "MO(1)",
		0x0204: "LSFT(A)",
		0x0304: "LCTL(LSFT(A))",
		0x1204: "RSFT(A)",
		0x2204: "MT(LSFT,A)",
		0x4105: "LT(1,B)",
		0x52A2: "OSM(LSFT)",
		0x7E00: "BT DEV1",
		0x7E07: "QK_KB_7",
		0x7C00: "QK_BOOT",
		0x7820: "RGB_TOG",
		0x6000: "0x6000",
	} {
		if got := Name(code, custom); got != want {
			t.Errorf("Name(0x%04X) = %q, want %q", code, got, want)
		}
	}

	got := Filter(Picker(2, custom), "kc_esc")
	if len(got) == 0 || got[0].Name != "ESC" {
		t.Errorf("ESC should rank first for kc_esc, got %v", got)
	}
	if got := Filter(Picker(2, custom), "space"); len(got) == 0 || got[0].Code != 0x2C {
		t.Errorf("long names should match, got %v", got)
	}
}

func TestParseRoundTrip(t *testing.T) {
	custom := []string{"BT DEV1", "USB"}
	for c := range 0x10000 {
		code := uint16(c)
		name := Name(code, custom)
		if got, err := Parse(name, custom); err != nil || got != code {
			t.Errorf("Parse(%q) = 0x%04X, %v; want 0x%04X", name, got, err, code)
		}
	}
	for in, want := range map[string]uint16{"kc_esc": 0x29, "space": 0x2C, "lt(1, a)": 0x4104, "0x7e05": 0x7E05} {
		if got, err := Parse(in, nil); err != nil || got != want {
			t.Errorf("Parse(%q) = 0x%04X, %v; want 0x%04X", in, got, err, want)
		}
	}
	for _, bad := range []string{"NOPE", "MO(32)", "LCTL(RSFT(A))", "LT(1,MO(2))", "MT(LCTL,"} {
		if _, err := Parse(bad, nil); err == nil {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}
