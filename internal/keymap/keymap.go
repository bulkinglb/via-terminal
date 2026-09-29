// Package keymap reads and writes backup files: every layer of the keymap,
// encoder rotations and macros, as text a person can edit.
package keymap

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bulkinglb/via-terminal/internal/defs"
	"github.com/bulkinglb/via-terminal/internal/keycodes"
	"github.com/bulkinglb/via-terminal/internal/macros"
)

// Backup is everything a backup file holds. Keys and Encoders are laid out
// like via.Device returns them; Macros are in the macros package's text
// form. Encoders and Macros are nil when a board or file has none.
type Backup struct {
	Keys, Encoders []uint16
	Macros         []string
}

// Encode writes the backup with one keyboard row per line so it diffs well
// in dotfiles.
func Encode(w io.Writer, def defs.Definition, backup Backup) error {
	keys, encoders := backup.Keys, backup.Encoders
	name := func(code uint16) string {
		n := keycodes.Name(code, def.Custom)
		// Duplicate custom names could make a name ambiguous; hex never is.
		if back, err := keycodes.Parse(n, def.Custom); err != nil || back != code {
			n = fmt.Sprintf("0x%04X", code)
		}
		return n
	}
	line := func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	}
	sep := func(i int) string {
		if i == 0 {
			return ""
		}
		return ","
	}

	var b strings.Builder
	fmt.Fprintf(&b, "{\n  \"name\": %s,\n  \"vendorId\": \"0x%04X\",\n  \"productId\": \"0x%04X\",\n  \"layers\": [",
		line(def.Name), def.VendorID, def.ProductID)
	layers := len(keys) / (def.Rows * def.Cols)
	for l := range layers {
		b.WriteString(sep(l) + "\n    [")
		for r := range def.Rows {
			start := (l*def.Rows + r) * def.Cols
			row := make([]string, def.Cols)
			for c := range row {
				row[c] = name(keys[start+c])
			}
			fmt.Fprintf(&b, "%s\n      %s", sep(r), line(row))
		}
		b.WriteString("\n    ]")
	}
	b.WriteString("\n  ]")
	if len(encoders) > 0 {
		b.WriteString(",\n  \"encoders\": [")
		for l := range layers {
			pairs := make([][2]string, def.Encoders)
			for e := range pairs {
				i := (l*def.Encoders + e) * 2
				pairs[e] = [2]string{name(encoders[i]), name(encoders[i+1])}
			}
			fmt.Fprintf(&b, "%s\n    %s", sep(l), line(pairs))
		}
		b.WriteString("\n  ]")
	}
	if backup.Macros != nil {
		b.WriteString(",\n  \"macros\": [")
		for i, text := range backup.Macros {
			fmt.Fprintf(&b, "%s\n    %s", sep(i), line(text))
		}
		b.WriteString("\n  ]")
	}
	b.WriteString("\n}\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// Decode reads a backup for def's board with the given layer count. It
// checks everything it can before returning, so nothing half-valid gets
// written; whether the macros fit the board's buffer is up to the caller.
func Decode(r io.Reader, def defs.Definition, layers int) (Backup, error) {
	var f file
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		return Backup{}, err
	}
	if _, err := macros.Encode(f.Macros); err != nil {
		return Backup{}, err
	}
	keys, encoders, err := f.codes(def, layers)
	if err != nil {
		return Backup{}, err
	}
	return Backup{keys, encoders, f.Macros}, nil
}

type file struct {
	VendorID, ProductID string
	Layers              [][][]string
	Encoders            [][][2]string
	Macros              []string
}

// codes parses the keymap and encoder names after checking they fit def.
func (f file) codes(def defs.Definition, layers int) (keys, encoders []uint16, err error) {
	vid, _ := strconv.ParseUint(f.VendorID, 0, 16)
	pid, _ := strconv.ParseUint(f.ProductID, 0, 16)
	if uint16(vid) != def.VendorID || uint16(pid) != def.ProductID {
		return nil, nil, fmt.Errorf("backup is for %s:%s, this board is %04X:%04X", f.VendorID, f.ProductID, def.VendorID, def.ProductID)
	}
	if len(f.Layers) != layers {
		return nil, nil, fmt.Errorf("backup has %d layers, the board has %d", len(f.Layers), layers)
	}
	for l, rows := range f.Layers {
		if len(rows) != def.Rows {
			return nil, nil, fmt.Errorf("layer %d has %d rows, want %d", l, len(rows), def.Rows)
		}
		for r, cols := range rows {
			if len(cols) != def.Cols {
				return nil, nil, fmt.Errorf("layer %d row %d has %d keys, want %d", l, r, len(cols), def.Cols)
			}
			for c, s := range cols {
				code, err := keycodes.Parse(s, def.Custom)
				if err != nil {
					return nil, nil, fmt.Errorf("layer %d row %d col %d: %w", l, r, c, err)
				}
				keys = append(keys, code)
			}
		}
	}

	if len(f.Encoders) == 0 {
		return keys, nil, nil
	}
	if len(f.Encoders) != layers {
		return nil, nil, fmt.Errorf("backup has encoders for %d layers, the board has %d", len(f.Encoders), layers)
	}
	for l, pairs := range f.Encoders {
		if len(pairs) != def.Encoders {
			return nil, nil, fmt.Errorf("layer %d has %d encoders, want %d", l, len(pairs), def.Encoders)
		}
		for e, pair := range pairs {
			for _, s := range pair {
				code, err := keycodes.Parse(s, def.Custom)
				if err != nil {
					return nil, nil, fmt.Errorf("layer %d encoder %d: %w", l, e, err)
				}
				encoders = append(encoders, code)
			}
		}
	}
	return keys, encoders, nil
}
