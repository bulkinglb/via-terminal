package defs

import (
	"encoding/json"
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
	keys, err := parseKLE(rows)
	if err != nil {
		t.Fatal(err)
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
