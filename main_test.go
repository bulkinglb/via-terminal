package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestHasHIDID(t *testing.T) {
	uevent := "DRIVER=hid-generic\nHID_ID=0003:0000342D:0000E4C2\nHID_NAME=Hangsheng MonsGeek Keyboard\n"

	if !hasHIDID(uevent, 0x342D, 0xE4C2) {
		t.Error("expected match for MonsGeek vendor/product")
	}
	if hasHIDID(uevent, 0x1532, 0x0099) {
		t.Error("expected no match for a different vendor/product")
	}
	if hasHIDID("no HID_ID line here", 0x342D, 0xE4C2) {
		t.Error("expected no match when HID_ID is missing")
	}
}

func TestParseKLE(t *testing.T) {
	// ISO Enter row from the M1 V5 definition, plus an alternate layout
	// choice that must be dropped.
	rows := [][]json.RawMessage{
		{json.RawMessage(`{"w": 1.5}`), json.RawMessage(`"2,0"`), json.RawMessage(`{"x": 0.25, "w": 1.5, "h": 2, "h2": 1, "x2": -0.25}`), json.RawMessage(`"3,13"`), json.RawMessage(`{"x": 0.5}`), json.RawMessage(`"2,14"`)},
		{json.RawMessage(`{"y": 0.25}`), json.RawMessage(`"4,0\n\n\n0,1"`), json.RawMessage(`"4,1\n\n\n0,0"`)},
	}
	keys, err := parseKLE(rows)
	if err != nil {
		t.Fatal(err)
	}
	want := []key{
		{Row: 2, Col: 0, X: 0, Y: 0, W: 1.5, H: 1, W2: 1.5, H2: 1},
		{Row: 3, Col: 13, X: 1.75, Y: 0, W: 1.5, H: 2, X2: -0.25, W2: 1.5, H2: 1},
		{Row: 2, Col: 14, X: 3.75, Y: 0, W: 1, H: 1, W2: 1, H2: 1},
		{Row: 4, Col: 1, X: 1, Y: 1.25, W: 1, H: 1, W2: 1, H2: 1},
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("got  %+v\nwant %+v", keys, want)
	}
}

func TestRender(t *testing.T) {
	keys := []key{
		{X: 0, Y: 0, W: 1, H: 1, W2: 1, H2: 1},
		{X: 1, Y: 0, W: 1, H: 1, W2: 1, H2: 1},
		{X: 0, Y: 1, W: 2, H: 1, W2: 2, H2: 1},
	}
	names := []string{"a", "b", "c"}
	i := 0
	got := render(keys, func(key) string {
		i++
		return names[i-1]
	}, -1)
	want := strings.Join([]string{
		"┌─────┬─────┐",
		"│  a  │  b  │",
		"│     │     │",
		"├─────┴─────┤",
		"│     c     │",
		"│           │",
		"└───────────┘",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	for in, want := range map[string]string{
		"RGB_MOD": "RGB_|MOD",
		"BT DEV1": "BT|DEV1",
		"LT(1,A)": "LT(1,|A)",
		"ABCDEFG": "ABCDE|FG",
		"ESC":     "ESC",
	} {
		var parts []string
		for _, line := range wrap([]rune(in), 5) {
			parts = append(parts, string(line))
		}
		if got := strings.Join(parts, "|"); got != want {
			t.Errorf("wrap(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKeyName(t *testing.T) {
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
		if got := keyName(code, custom); got != want {
			t.Errorf("keyName(0x%04X) = %q, want %q", code, got, want)
		}
	}

	got := filterKeycodes(pickerKeycodes(2, custom), "kc_esc")
	if len(got) == 0 || got[0].name != "ESC" {
		t.Errorf("ESC should rank first for kc_esc, got %v", got)
	}
	if got := filterKeycodes(pickerKeycodes(2, custom), "space"); len(got) == 0 || got[0].code != 0x2C {
		t.Errorf("long names should match, got %v", got)
	}
}

func TestNearestKey(t *testing.T) {
	keys := []key{
		{X: 0, Y: 0, W: 1, H: 1},
		{X: 1, Y: 0, W: 1, H: 1},
		{X: 0, Y: 1, W: 2, H: 1},
		{X: 2.5, Y: 1.25, W: 1, H: 1},
	}
	for _, c := range []struct{ from, dx, dy, want int }{
		{0, 1, 0, 1},
		{1, 0, 1, 2},
		{2, 1, 0, 3},
		{0, -1, 0, 0},
	} {
		if got := nearestKey(keys, c.from, c.dx, c.dy); got != c.want {
			t.Errorf("nearestKey(%d, %d, %d) = %d, want %d", c.from, c.dx, c.dy, got, c.want)
		}
	}
}
