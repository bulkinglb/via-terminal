package keymap

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/bulkinglb/via-terminal/internal/defs"
)

func TestRoundTrip(t *testing.T) {
	def := defs.Definition{Name: "Test", VendorID: 0x342D, ProductID: 0xE4C2, Rows: 2, Cols: 3, Encoders: 1, Custom: []string{"USB", "USB"}}
	backup := Backup{
		Keys:     []uint16{0x29, 0x5221, 0x0204, 0x0001, 0x7E00, 0x7E01, 0x4104, 0x0000, 0x7C00, 0x6000, 0x68, 0x7E05},
		Encoders: []uint16{0xAA, 0xA9, 0x7E22, 0x7E23},
		Macros:   []string{`Hello "you"{ENT}`, "", "{LCTL,C}"},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, def, backup); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("invalid JSON:\n%s", buf.String())
	}
	// The second "USB" can't be told apart by name, so it must fall back to hex.
	if !strings.Contains(buf.String(), `["ESC","MO(1)","LSFT(A)"]`) || !strings.Contains(buf.String(), `"0x7E01"`) {
		t.Errorf("unexpected encoding:\n%s", buf.String())
	}

	got, err := Decode(bytes.NewReader(buf.Bytes()), def, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Keys, backup.Keys) || !slices.Equal(got.Encoders, backup.Encoders) || !slices.Equal(got.Macros, backup.Macros) {
		t.Errorf("round trip changed the backup:\n%+v\n%+v", got, backup)
	}

	other := def
	other.ProductID = 0xE4C5
	if _, err := Decode(bytes.NewReader(buf.Bytes()), other, 2); err == nil {
		t.Error("a backup for another board should be rejected")
	}
	bad := strings.Replace(buf.String(), `"{LCTL,C}"`, `"{NOPE}"`, 1)
	if _, err := Decode(strings.NewReader(bad), def, 2); err == nil {
		t.Error("a macro with an unknown key should be rejected before anything is written")
	}
	old := buf.String()[:strings.Index(buf.String(), `,
  "macros"`)] + "\n}\n"
	if got, err := Decode(strings.NewReader(old), def, 2); err != nil || got.Macros != nil {
		t.Errorf("a backup without macros should still load and leave macros alone: %v, %q", err, got.Macros)
	}
}
