// Package via talks the VIA raw HID protocol to QMK keyboards through
// Linux hidraw.
package via

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	cmdGetProtocolVersion = 0x01
	cmdSetKeycode         = 0x05
	cmdResetKeymap        = 0x06
	cmdGetLayerCount      = 0x11
	cmdGetKeymapBuffer    = 0x12
	cmdSetKeymapBuffer    = 0x13
	cmdGetEncoder         = 0x14
	cmdSetEncoder         = 0x15
	cmdUnhandled          = 0xFF

	reportSize = 32
	chunkSize  = reportSize - 4
)

// ErrUnhandled means the firmware doesn't implement a command, e.g. encoder
// mapping on a board built without it.
var ErrUnhandled = errors.New("command not supported by the firmware")

// usage is QMK's raw HID descriptor prefix: usage page 0xFF60, usage 0x61.
var usage = []byte{0x06, 0x60, 0xFF, 0x09, 0x61}

// Info is a hidraw interface that carries the VIA usage page.
type Info struct {
	Path                string
	VendorID, ProductID uint16
}

// Devices lists VIA interfaces from sysfs without opening them. Filtering on
// the usage page matters: probing a board's other interfaces would wait out
// the read timeout and write VIA packets into their reports.
func Devices() []Info {
	var infos []Info
	paths, _ := filepath.Glob("/dev/hidraw*")
	for _, p := range paths {
		sys := "/sys/class/hidraw/" + filepath.Base(p) + "/device/"
		uevent, err := os.ReadFile(sys + "uevent")
		if err != nil {
			continue
		}
		vid, pid, ok := parseHIDID(string(uevent))
		if !ok {
			continue
		}
		desc, err := os.ReadFile(sys + "report_descriptor")
		if err != nil || !bytes.Contains(desc, usage) {
			continue
		}
		infos = append(infos, Info{p, vid, pid})
	}
	return infos
}

func parseHIDID(uevent string) (vid, pid uint16, ok bool) {
	for _, line := range strings.Split(uevent, "\n") {
		id, found := strings.CutPrefix(line, "HID_ID=")
		if !found {
			continue
		}
		parts := strings.Split(id, ":")
		if len(parts) != 3 {
			return 0, 0, false
		}
		v, verr := strconv.ParseUint(parts[1], 16, 16)
		p, perr := strconv.ParseUint(parts[2], 16, 16)
		return uint16(v), uint16(p), verr == nil && perr == nil
	}
	return 0, 0, false
}

type Device struct {
	f       *os.File
	Version uint16
}

// Open opens a VIA interface and reads its protocol version, which doubles
// as the handshake.
func Open(path string) (*Device, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	d := &Device{f: f}
	resp, err := d.command(cmdGetProtocolVersion)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	d.Version = binary.BigEndian.Uint16(resp[1:3])
	return d, nil
}

func (d *Device) Close() error { return d.f.Close() }

func (d *Device) LayerCount() (int, error) {
	resp, err := d.command(cmdGetLayerCount)
	if err != nil {
		return 0, err
	}
	return int(resp[1]), nil
}

// Keymap reads every layer in 28-byte chunks, which is far fewer round trips
// than asking for each key. The result is indexed by (layer*rows+row)*cols+col.
func (d *Device) Keymap(layers, rows, cols int) ([]uint16, error) {
	size := layers * rows * cols * 2
	buf := make([]byte, 0, size)
	for off := 0; off < size; off += chunkSize {
		n := min(chunkSize, size-off)
		resp, err := d.command(cmdGetKeymapBuffer, byte(off>>8), byte(off), byte(n))
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

// SetKeymap writes a whole keymap laid out like Keymap returns it.
func (d *Device) SetKeymap(codes []uint16) error {
	buf := make([]byte, 2*len(codes))
	for i, code := range codes {
		binary.BigEndian.PutUint16(buf[2*i:], code)
	}
	for off := 0; off < len(buf); off += chunkSize {
		chunk := buf[off:min(off+chunkSize, len(buf))]
		args := append([]byte{byte(off >> 8), byte(off), byte(len(chunk))}, chunk...)
		if _, err := d.command(cmdSetKeymapBuffer, args...); err != nil {
			return err
		}
	}
	return nil
}

func (d *Device) SetKeycode(layer, row, col int, code uint16) error {
	_, err := d.command(cmdSetKeycode, byte(layer), byte(row), byte(col), byte(code>>8), byte(code))
	return err
}

// Encoders reads the rotation keycodes of every encoder on every layer,
// indexed by (layer*count+encoder)*2, counter-clockwise first.
func (d *Device) Encoders(layers, count int) ([]uint16, error) {
	codes := make([]uint16, 0, layers*count*2)
	for layer := range layers {
		for enc := range count {
			for cw := range 2 {
				resp, err := d.command(cmdGetEncoder, byte(layer), byte(enc), byte(cw))
				if err != nil {
					return nil, err
				}
				codes = append(codes, binary.BigEndian.Uint16(resp[4:6]))
			}
		}
	}
	return codes, nil
}

// SetEncoders writes codes laid out like Encoders returns them.
func (d *Device) SetEncoders(count int, codes []uint16) error {
	for i, code := range codes {
		layer, enc, cw := i/(count*2), i/2%count, i%2
		if _, err := d.command(cmdSetEncoder, byte(layer), byte(enc), byte(cw), byte(code>>8), byte(code)); err != nil {
			return err
		}
	}
	return nil
}

// ResetKeymap restores the firmware's default keymap and encoder mapping.
// The firmware rewrites all of it in EEPROM before replying, so it gets a
// longer timeout than other commands.
func (d *Device) ResetKeymap() error {
	_, err := d.commandWithin(10*time.Second, cmdResetKeymap)
	return err
}

func (d *Device) command(cmd byte, args ...byte) ([reportSize]byte, error) {
	return d.commandWithin(time.Second, cmd, args...)
}

// commandWithin writes a 32-byte VIA report and reads the response, checking
// the command byte is echoed back.
func (d *Device) commandWithin(timeout time.Duration, cmd byte, args ...byte) ([reportSize]byte, error) {
	var req [reportSize]byte
	req[0] = cmd
	copy(req[1:], args)

	if _, err := d.f.Write(req[:]); err != nil {
		return [reportSize]byte{}, err
	}
	if err := d.f.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return [reportSize]byte{}, err
	}
	var resp [reportSize]byte
	n, err := d.f.Read(resp[:])
	if err != nil {
		return resp, err
	}
	if n != reportSize {
		return resp, fmt.Errorf("short read: %d bytes", n)
	}
	if resp[0] == cmdUnhandled {
		return resp, fmt.Errorf("command 0x%02X: %w", cmd, ErrUnhandled)
	}
	if resp[0] != cmd {
		return resp, fmt.Errorf("unexpected response id 0x%02X for command 0x%02X", resp[0], cmd)
	}
	return resp, nil
}
