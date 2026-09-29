package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	cmdGetProtocolVersion = 0x01
	cmdSetKeycode         = 0x05
	cmdGetLayerCount      = 0x11
	cmdGetKeymapBuffer    = 0x12

	reportSize = 32
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
	def, err := loadDefinition(defPath)
	if err != nil {
		return err
	}
	f, version, err := findVIADevice(def.VendorID, def.ProductID)
	if err != nil {
		return err
	}
	defer f.Close()
	if version < 12 {
		return fmt.Errorf("VIA protocol v%d is not supported yet, only v12 and later", version)
	}

	layers, err := getLayerCount(f)
	if err != nil {
		return err
	}
	keymap, err := readKeymap(f, int(layers), def.Rows, def.Cols)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newModel(def, f, version, int(layers), keymap)).Run()
	return err
}

// viaUsage is QMK's raw HID descriptor prefix: usage page 0xFF60, usage 0x61.
var viaUsage = []byte{0x06, 0x60, 0xFF, 0x09, 0x61}

// findVIADevice scans /dev/hidraw* for the matching vendor/product and picks
// the interface with the VIA usage page. Probing the other interfaces would
// wait out the read timeout and write VIA packets into their reports.
func findVIADevice(vid, pid uint16) (f *os.File, version uint16, err error) {
	candidates, _ := filepath.Glob("/dev/hidraw*")
	for _, p := range candidates {
		sys := "/sys/class/hidraw/" + filepath.Base(p) + "/device/"
		uevent, rerr := os.ReadFile(sys + "uevent")
		if rerr != nil || !hasHIDID(string(uevent), vid, pid) {
			continue
		}
		desc, rerr := os.ReadFile(sys + "report_descriptor")
		if rerr != nil || !bytes.Contains(desc, viaUsage) {
			continue
		}
		dev, oerr := os.OpenFile(p, os.O_RDWR, 0)
		if oerr != nil {
			continue
		}
		resp, cerr := doCommand(dev, cmdGetProtocolVersion)
		if cerr == nil {
			return dev, binary.BigEndian.Uint16(resp[1:3]), nil
		}
		dev.Close()
	}
	return nil, 0, fmt.Errorf("no VIA interface found for %04X:%04X", vid, pid)
}

func hasHIDID(uevent string, vid, pid uint16) bool {
	for _, line := range strings.Split(uevent, "\n") {
		id, ok := strings.CutPrefix(line, "HID_ID=")
		if !ok {
			continue
		}
		parts := strings.Split(id, ":")
		if len(parts) != 3 {
			continue
		}
		v, _ := strconv.ParseUint(parts[1], 16, 16)
		p, _ := strconv.ParseUint(parts[2], 16, 16)
		return uint16(v) == vid && uint16(p) == pid
	}
	return false
}

func getLayerCount(f *os.File) (uint8, error) {
	resp, err := doCommand(f, cmdGetLayerCount)
	if err != nil {
		return 0, err
	}
	return resp[1], nil
}

// readKeymap reads every layer in 28-byte chunks, which is far fewer round
// trips than asking for each key. The result is indexed by
// (layer*rows+row)*cols+col.
func readKeymap(f *os.File, layers, rows, cols int) ([]uint16, error) {
	size := layers * rows * cols * 2
	buf := make([]byte, 0, size)
	for off := 0; off < size; off += 28 {
		n := min(28, size-off)
		resp, err := doCommand(f, cmdGetKeymapBuffer, byte(off>>8), byte(off), byte(n))
		if err != nil {
			return nil, err
		}
		buf = append(buf, resp[4:4+n]...)
	}
	codes := make([]uint16, size/2)
	for i := range codes {
		codes[i] = binary.BigEndian.Uint16(buf[2*i:])
	}
	return codes, nil
}

func setKeycode(f *os.File, layer, row, col int, code uint16) error {
	_, err := doCommand(f, cmdSetKeycode, byte(layer), byte(row), byte(col), byte(code>>8), byte(code))
	return err
}

// doCommand writes a 32-byte VIA HID report and reads the response,
// checking the command byte is echoed back.
func doCommand(f *os.File, cmd byte, args ...byte) ([reportSize]byte, error) {
	var req [reportSize]byte
	req[0] = cmd
	copy(req[1:], args)

	if _, err := f.Write(req[:]); err != nil {
		return [reportSize]byte{}, err
	}
	if err := f.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		return [reportSize]byte{}, err
	}
	var resp [reportSize]byte
	n, err := f.Read(resp[:])
	if err != nil {
		return resp, err
	}
	if n != reportSize {
		return resp, fmt.Errorf("short read: %d bytes", n)
	}
	if resp[0] != cmd {
		return resp, fmt.Errorf("unexpected response id 0x%02X for command 0x%02X", resp[0], cmd)
	}
	return resp, nil
}
