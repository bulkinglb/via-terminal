package keycodes

import (
	"strings"
	"testing"
)

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

func TestOldProtocols(t *testing.T) {
	defer UseProtocol(12)
	custom := []string{"BT DEV1"}
	for _, tt := range []struct {
		protocol    int
		names       map[uint16]string
		unavailable []string
	}{
		{9, map[uint16]string{0x5011: "TO(1)", 0x5000: "0x5000"}, nil},
		{10, map[uint16]string{
			0x5000: "TO(0)", 0x5101: "MO(1)", 0x5302: "TG(2)", 0x5801: "TT(1)", 0x5502: "OSM(LSFT)",
			0x5913: "LM(1,LCTL|LSFT)", 0x6204: "MT(LSFT,A)", 0x4105: "LT(1,B)", 0x0204: "LSFT(A)",
			0x00F0: "MS_UP", 0x00F4: "MS_BTN1", 0x00FD: "MS_ACL0", 0x5C00: "QK_BOOT", 0x5CC2: "RGB_TOG",
			0x5CD7: "SC_LSPO", 0x5F12: "MC_0", 0x5F80: "BT DEV1", 0x5F81: "QK_KB_1",
			// Legacy action functions and VIA's old lighting keys have no v12 number.
			0x2105: "0x2105", 0x5F00: "0x5F00", 0x00CD: "0x00CD", 0x00C0: "0x00C0",
		}, []string{"CW_TOGG", "MS_BTN8", "MC_16", "LM(1,RCTL)", "QK_KB_16"}},
		{11, map[uint16]string{0x7700: "TL_LOWR", 0x7702: "MC_0", 0x7F00: "BT DEV1", 0x5221: "MO(1)", 0x7C73: "CW_TOGG", 0x7712: "0x7712"},
			[]string{"MC_16", "QK_KB_256"}},
	} {
		UseProtocol(tt.protocol)
		for code, want := range tt.names {
			if got := Name(code, custom); got != want {
				t.Errorf("v%d: Name(0x%04X) = %q, want %q", tt.protocol, code, got, want)
			}
		}
		for c := range 0x10000 {
			code := uint16(c)
			if got, err := Parse(Name(code, custom), custom); err != nil || got != code {
				t.Fatalf("v%d: 0x%04X named %q reads back as 0x%04X, %v", tt.protocol, code, Name(code, custom), got, err)
			}
		}
		for _, name := range tt.unavailable {
			if code, err := Parse(name, custom); err == nil {
				t.Errorf("v%d: %s should be unavailable, got 0x%04X", tt.protocol, name, code)
			}
		}
		for _, k := range Picker(4, custom) {
			if got := Name(k.Code, custom); got != k.Name && got != strings.ToUpper(k.Name) {
				t.Errorf("v%d: picker entry %s is 0x%04X, which is named %s", tt.protocol, k.Name, k.Code, got)
			}
		}
	}
}
