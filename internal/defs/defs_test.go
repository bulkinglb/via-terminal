package defs

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseKLE(t *testing.T) {
	// ISO Enter row from the M1 V5 definition, plus an alternate layout
	// choice that must be dropped.
	rows := [][]json.RawMessage{
		{json.RawMessage(`{"w": 1.5}`), json.RawMessage(`"2,0"`), json.RawMessage(`{"x": 0.25, "w": 1.5, "h": 2, "h2": 1, "x2": -0.25}`), json.RawMessage(`"3,13"`), json.RawMessage(`{"x": 0.5}`), json.RawMessage(`"2,14"`)},
		{json.RawMessage(`{"y": 0.25}`), json.RawMessage(`"4,0\n\n\n0,1"`), json.RawMessage(`"4,1\n\n\n0,0"`)},
	}
	keys, encoders, err := parseKLE(rows)
	if err != nil || encoders != 0 {
		t.Fatal(encoders, err)
	}
	want := []Key{
		{Row: 2, Col: 0, X: 0, Y: 0, W: 1.5, H: 1, W2: 1.5, H2: 1},
		{Row: 3, Col: 13, X: 1.75, Y: 0, W: 1.5, H: 2, X2: -0.25, W2: 1.5, H2: 1},
		{Row: 2, Col: 14, X: 3.75, Y: 0, W: 1, H: 1, W2: 1, H2: 1},
		{Row: 4, Col: 1, X: 1, Y: 1.25, W: 1, H: 1, W2: 1, H2: 1},
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("got  %+v\nwant %+v", keys, want)
	}
}

func TestFind(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sub, _ := fs.Sub(bundled, "keyboards")
	names, _ := fs.Glob(sub, "*.json")
	for _, name := range names {
		data, _ := fs.ReadFile(sub, name)
		if _, err := parse(data); err != nil {
			t.Errorf("bundled %s: %v", name, err)
		}
	}

	def, err := Find(0x342D, 0xE4C2)
	if err != nil || def.Rows != 6 || def.Cols != 15 || def.Encoders != 1 {
		t.Fatalf("bundled M1 V5 ISO: %+v, %v", def, err)
	}

	userDir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "via-terminal", "keyboards")
	os.MkdirAll(userDir, 0o755)
	os.WriteFile(filepath.Join(userDir, "broken.json"), []byte("{"), 0o644)
	os.WriteFile(filepath.Join(userDir, "mine.json"), []byte(`{"name": "Mine", "vendorId": "0x342D", "productId": "0xE4C2", "matrix": {"rows": 1, "cols": 1}, "layouts": {"keymap": [["0,0"]]}}`), 0o644)
	if def, err := Find(0x342D, 0xE4C2); err != nil || def.Name != "Mine" {
		t.Errorf("user folder should win: %+v, %v", def, err)
	}
	if _, err := Find(0xFFFF, 0x0001); err == nil {
		t.Error("expected an error for an unknown board")
	}
}
