package keymap

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"via-terminal/internal/defs"
)

func TestRoundTrip(t *testing.T) {
	def := defs.Definition{Name: "Test", VendorID: 0x342D, ProductID: 0xE4C2, Rows: 2, Cols: 3, Encoders: 1, Custom: []string{"USB", "USB"}}
	keys := []uint16{0x29, 0x5221, 0x0204, 0x0001, 0x7E00, 0x7E01, 0x4104, 0x0000, 0x7C00, 0x6000, 0x68, 0x7E05}
	encoders := []uint16{0xAA, 0xA9, 0x7E22, 0x7E23}

	var buf bytes.Buffer
	if err := Encode(&buf, def, keys, encoders); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("invalid JSON:\n%s", buf.String())
	}
	// The second "USB" can't be told apart by name, so it must fall back to hex.
	if !strings.Contains(buf.String(), `["ESC","MO(1)","LSFT(A)"]`) || !strings.Contains(buf.String(), `"0x7E01"`) {
		t.Errorf("unexpected encoding:\n%s", buf.String())
	}

	gotKeys, gotEnc, err := Decode(&buf, def, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(gotKeys, keys) || !slices.Equal(gotEnc, encoders) {
		t.Errorf("round trip changed the keymap:\n%04X\n%04X", gotKeys, gotEnc)
	}

	other := def
	other.ProductID = 0xE4C5
	buf.Reset()
	Encode(&buf, def, keys, encoders)
	if _, _, err := Decode(&buf, other, 2); err == nil {
		t.Error("a backup for another board should be rejected")
	}
}
