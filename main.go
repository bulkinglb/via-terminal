package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"via-terminal/internal/defs"
	"via-terminal/internal/keymap"
	"via-terminal/internal/macros"
	"via-terminal/internal/tui"
	"via-terminal/internal/via"
)

const usage = `usage: via-terminal [--def board.json] [command]

commands:
  (none)          open the keymap editor
  export [file]   save the keymap and macros to file, or print them
  import file     load a file saved by export
  reset           restore the firmware's default keymap

import and reset save a backup of the board first.

`

func main() {
	defPath := flag.String("def", "", "VIA v3 definition JSON (default: look up by vendor/product ID)")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := run(*defPath, flag.Arg(0), flag.Arg(1)); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if errors.Is(err, fs.ErrPermission) {
			fmt.Fprintln(os.Stderr, `hint: allow hidraw access with a udev rule, e.g. in /etc/udev/rules.d/99-via.rules:
  KERNEL=="hidraw*", SUBSYSTEM=="hidraw", MODE="0660", TAG+="uaccess"`)
		}
		os.Exit(1)
	}
}

func run(defPath, cmd, file string) error {
	switch cmd {
	case "", "export", "reset":
	case "import":
		if file == "" {
			return errors.New("import needs a file")
		}
	default:
		flag.Usage()
		os.Exit(2)
	}

	def, info, err := pick(defPath, via.Devices())
	if err != nil {
		return err
	}
	dev, err := via.Open(info.Path)
	if err != nil {
		return err
	}
	defer dev.Close()
	if dev.Version < 12 {
		return fmt.Errorf("VIA protocol v%d is not supported yet, only v12 and later", dev.Version)
	}
	layers, err := dev.LayerCount()
	if err != nil {
		return err
	}
	keys, err := dev.Keymap(layers, def.Rows, def.Cols)
	if err != nil {
		return err
	}
	if cmd == "" {
		return tui.Run(def, dev, layers, keys)
	}

	current, macroSize, err := readBackup(dev, def, layers, keys)
	if err != nil {
		return err
	}
	switch cmd {
	case "export":
		var buf bytes.Buffer
		if err := keymap.Encode(&buf, def, current); err != nil {
			return err
		}
		if file == "" {
			_, err := os.Stdout.Write(buf.Bytes())
			return err
		}
		return os.WriteFile(file, buf.Bytes(), 0o644)

	case "import":
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		next, err := keymap.Decode(f, def, layers)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		var macroBuf []byte
		if next.Macros != nil {
			if len(next.Macros) != len(current.Macros) {
				return fmt.Errorf("%s has %d macros, the board has %d", file, len(next.Macros), len(current.Macros))
			}
			macroBuf, _ = macros.Encode(next.Macros)
			if len(macroBuf) > macroSize {
				return fmt.Errorf("%s: the macros need %d bytes, the board has %d", file, len(macroBuf), macroSize)
			}
		}
		if err := backup(def, current); err != nil {
			return err
		}
		if err := dev.SetKeymap(next.Keys); err != nil {
			return err
		}
		if next.Encoders != nil && current.Encoders != nil {
			if err := dev.SetEncoders(def.Encoders, next.Encoders); err != nil {
				return err
			}
		}
		if macroBuf != nil {
			if err := dev.SetMacroBuffer(macroBuf, macroSize); err != nil {
				return err
			}
		}
		fmt.Println("imported", file)

	case "reset":
		if err := backup(def, current); err != nil {
			return err
		}
		if err := dev.ResetKeymap(); err != nil {
			return err
		}
		fmt.Println("keymap reset to the firmware default, macros are unchanged")
	}
	return nil
}

// readBackup reads what export and backups save besides the keymap: encoder
// rotations and macros. Either is left out when the board or its firmware
// doesn't have it. It also returns the macro buffer size.
func readBackup(dev *via.Device, def defs.Definition, layers int, keys []uint16) (keymap.Backup, int, error) {
	b := keymap.Backup{Keys: keys}
	if def.Encoders > 0 {
		codes, err := dev.Encoders(layers, def.Encoders)
		if err != nil && !errors.Is(err, via.ErrUnhandled) {
			return b, 0, err
		}
		b.Encoders = codes
	}
	count, err := dev.MacroCount()
	if err != nil || count == 0 {
		return b, 0, nil
	}
	size, err := dev.MacroBufferSize()
	if err != nil {
		return b, 0, err
	}
	buf, err := dev.MacroBuffer(size)
	if err != nil {
		return b, 0, err
	}
	b.Macros, err = macros.Decode(buf, count)
	return b, size, err
}

// backup saves the board's current state next to the user's definitions so
// an import or reset can always be undone.
func backup(def defs.Definition, current keymap.Backup) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "via-terminal", "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := keymap.Encode(&buf, def, current); err != nil {
		return err
	}
	// CreateTemp adds a random suffix, so two backups in the same second
	// can't overwrite each other.
	f, err := os.CreateTemp(dir, fmt.Sprintf("%04x-%04x-%s-*.json", def.VendorID, def.ProductID, time.Now().Format("20060102-150405")))
	if err != nil {
		return err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Println("backed up the board, undo with: via-terminal import", f.Name())
	return nil
}

// pick pairs a connected board with its definition: the --def file if given,
// otherwise the first connected board that defs.Find knows.
func pick(defPath string, infos []via.Info) (defs.Definition, via.Info, error) {
	if defPath != "" {
		def, err := defs.Load(defPath)
		if err != nil {
			return def, via.Info{}, err
		}
		for _, info := range infos {
			if info.VendorID == def.VendorID && info.ProductID == def.ProductID {
				return def, info, nil
			}
		}
		return def, via.Info{}, fmt.Errorf("%s is for %04X:%04X, which isn't connected", defPath, def.VendorID, def.ProductID)
	}
	err := errors.New("no VIA keyboard found")
	for _, info := range infos {
		def, ferr := defs.Find(info.VendorID, info.ProductID)
		if ferr == nil {
			return def, info, nil
		}
		err = ferr
	}
	return defs.Definition{}, via.Info{}, err
}
