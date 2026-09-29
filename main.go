package main

import (
	"flag"
	"fmt"
	"os"

	"via-terminal/internal/defs"
	"via-terminal/internal/tui"
	"via-terminal/internal/via"
)

func main() {
	defPath := flag.String("def", "", "VIA v3 definition JSON for the board")
	flag.Parse()
	if *defPath == "" {
		fmt.Fprintln(os.Stderr, "usage: via-terminal --def board.json")
		os.Exit(2)
	}
	if err := run(*defPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(defPath string) error {
	def, err := defs.Load(defPath)
	if err != nil {
		return err
	}
	var path string
	for _, info := range via.Devices() {
		if info.VendorID == def.VendorID && info.ProductID == def.ProductID {
			path = info.Path
			break
		}
	}
	if path == "" {
		return fmt.Errorf("no VIA interface found for %04X:%04X", def.VendorID, def.ProductID)
	}

	dev, err := via.Open(path)
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
