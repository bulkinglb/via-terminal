package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	viaVendorID  = 0x342D
	viaProductID = 0xE4C2

	cmdGetProtocolVersion = 0x01
	cmdGetKeycode         = 0x04
	cmdGetLayerCount      = 0x11

	reportSize = 32
)

func main() {
	rows := flag.Int("rows", 0, "matrix rows from the board's VIA JSON (required to dump the keymap)")
	cols := flag.Int("cols", 0, "matrix cols from the board's VIA JSON (required to dump the keymap)")
	flag.Parse()

	f, path, version, err := findVIADevice(viaVendorID, viaProductID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Println("VIA interface:", path)
	fmt.Println("protocol version:", version)

	layers, err := getLayerCount(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("layers:", layers)

	if *rows == 0 || *cols == 0 {
		fmt.Println(`pass --rows and --cols (from the board's VIA JSON "matrix" field) to dump layer 0`)
		return
	}

	fmt.Println("layer 0:")
	for row := 0; row < *rows; row++ {
		vals := make([]string, *cols)
		for col := 0; col < *cols; col++ {
			kc, err := getKeycode(f, 0, row, col)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			vals[col] = fmt.Sprintf("0x%04X", kc)
		}
		fmt.Println(" ", strings.Join(vals, " "))
	}
}

// findVIADevice scans /dev/hidraw* for a matching vendor/product, then
// confirms which of the board's several HID interfaces is the VIA one
// by sending get-protocol-version: only that interface echoes it back.
func findVIADevice(vid, pid uint16) (f *os.File, path string, version uint16, err error) {
	candidates, _ := filepath.Glob("/dev/hidraw*")
	for _, p := range candidates {
		uevent, rerr := os.ReadFile("/sys/class/hidraw/" + filepath.Base(p) + "/device/uevent")
		if rerr != nil || !hasHIDID(string(uevent), vid, pid) {
			continue
		}
		dev, oerr := os.OpenFile(p, os.O_RDWR, 0)
		if oerr != nil {
			continue
		}
		resp, cerr := doCommand(dev, cmdGetProtocolVersion)
		if cerr == nil {
			return dev, p, binary.BigEndian.Uint16(resp[1:3]), nil
		}
		dev.Close()
	}
	return nil, "", 0, fmt.Errorf("no VIA interface found for %04X:%04X", vid, pid)
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

func getKeycode(f *os.File, layer, row, col int) (uint16, error) {
	resp, err := doCommand(f, cmdGetKeycode, byte(layer), byte(row), byte(col))
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(resp[4:6]), nil
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
