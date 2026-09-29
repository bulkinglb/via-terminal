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
	"via-terminal/internal/tui"
	"via-terminal/internal/via"
)

const usage = `usage: via-terminal [--def board.json] [command]

commands:
  (none)          open the keymap editor
  export [file]   save the keymap to file, or print it
  import file     load a keymap saved by export
  reset           restore the firmware's default keymap

import and reset save a backup of the current keymap first.

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

	encoders, err := readEncoders(dev, def, layers)
	if err != nil {
		return err
	}
	switch cmd {
	case "export":
		var buf bytes.Buffer
		if err := keymap.Encode(&buf, def, keys, encoders); err != nil {
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
		newKeys, newEncoders, err := keymap.Decode(f, def, layers)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		if err := backup(def, keys, encoders); err != nil {
			return err
		}
		if err := dev.SetKeymap(newKeys); err != nil {
			return err
		}
		if newEncoders != nil && encoders != nil {
			if err := dev.SetEncoders(def.Encoders, newEncoders); err != nil {
				return err
			}
		}
		fmt.Println("imported", file)

	case "reset":
		if err := backup(def, keys, encoders); err != nil {
			return err
		}
		if err := dev.ResetKeymap(); err != nil {
			return err
		}
		fmt.Println("keymap reset to the firmware default")
	}
	return nil
}

// readEncoders returns nil when the board has no encoders or its firmware
// wasn't built with encoder mapping.
func readEncoders(dev *via.Device, def defs.Definition, layers int) ([]uint16, error) {
	if def.Encoders == 0 {
		return nil, nil
	}
	codes, err := dev.Encoders(layers, def.Encoders)
	if errors.Is(err, via.ErrUnhandled) {
		return nil, nil
	}
	return codes, err
}

// backup saves the current keymap next to the user's definitions so an
// import or reset can always be undone.
func backup(def defs.Definition, keys, encoders []uint16) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "via-terminal", "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := keymap.Encode(&buf, def, keys, encoders); err != nil {
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
	fmt.Println("backed up the current keymap, undo with: via-terminal import", f.Name())
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
