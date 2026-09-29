package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"via-terminal/internal/defs"
	"via-terminal/internal/tui"
	"via-terminal/internal/via"
)

func main() {
	defPath := flag.String("def", "", "VIA v3 definition JSON (default: look up by vendor/product ID)")
	flag.Parse()
	if err := run(*defPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if errors.Is(err, fs.ErrPermission) {
			fmt.Fprintln(os.Stderr, "hint: add the hidraw udev rule from docs/project-notes.md")
		}
		os.Exit(1)
	}
}

func run(defPath string) error {
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
	keymap, err := dev.Keymap(layers, def.Rows, def.Cols)
	if err != nil {
		return err
	}
	return tui.Run(def, dev, layers, keymap)
}

// pick pairs a connected board with its definition: the --def file if given,
// otherwise whatever defs.Find knows for each board.
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
