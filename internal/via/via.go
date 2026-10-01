// Package via talks the VIA raw HID protocol to QMK keyboards. HID access
// goes through rafaelmartins.com/p/usbhid, which covers Linux, macOS and
// Windows in pure Go, so every platform cross-compiles without cgo.
package via

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"rafaelmartins.com/p/usbhid"
)

const (
	cmdGetProtocolVersion = 0x01
	cmdSetKeycode         = 0x05
	cmdResetKeymap        = 0x06
	cmdCustomSetValue     = 0x07
	cmdCustomGetValue     = 0x08
	cmdCustomSave         = 0x09
	cmdGetLayerCount      = 0x11
	cmdMacroCount         = 0x0C
	cmdMacroBufferSize    = 0x0D
	cmdGetMacroBuffer     = 0x0E
	cmdSetMacroBuffer     = 0x0F
	cmdResetMacros        = 0x10
	cmdGetKeymapBuffer    = 0x12
	cmdSetKeymapBuffer    = 0x13
	cmdGetEncoder         = 0x14
	cmdSetEncoder         = 0x15
	cmdUnhandled          = 0xFF

	reportSize = 32
	chunkSize  = reportSize - 4

	// QMK's raw HID interface, which VIA uses.
	usagePage = 0xFF60
	usage     = 0x61
)

// ErrUnhandled means the firmware doesn't implement a command, e.g. encoder
// mapping on a board built without it.
var ErrUnhandled = errors.New("command not supported by the firmware")

// Info is a keyboard's VIA interface, found but not opened yet.
type Info struct {
	Path                string
	VendorID, ProductID uint16
	dev                 *usbhid.Device
}

// Devices lists VIA interfaces without opening them. Filtering on the usage
// page matters: probing a board's other interfaces would wait out the read
// timeout and write VIA packets into their reports.
func Devices() []Info {
	devs, _ := usbhid.Enumerate(func(d *usbhid.Device) bool {
		return d.UsagePage() == usagePage && d.Usage() == usage
	})
	var infos []Info
	for _, d := range devs {
		infos = append(infos, Info{d.Path(), d.VendorId(), d.ProductId(), d})
	}
	return infos
}

type Device struct {
	dev     *usbhid.Device
	Version uint16
}

// Open opens a VIA interface and reads its protocol version, which doubles
// as the handshake.
func Open(info Info) (*Device, error) {
	if err := info.dev.Open(false); err != nil {
		return nil, err
	}
	d := &Device{dev: info.dev}
	resp, err := d.command(cmdGetProtocolVersion)
	if err != nil {
		info.dev.Close()
		return nil, fmt.Errorf("%s: %w", info.Path, err)
	}
	d.Version = binary.BigEndian.Uint16(resp[1:3])
	return d, nil
}

func (d *Device) Close() error { return d.dev.Close() }

func (d *Device) LayerCount() (int, error) {
	resp, err := d.command(cmdGetLayerCount)
	if err != nil {
		return 0, err
	}
	return int(resp[1]), nil
}

// Keymap reads every layer at once. The result is indexed by
// (layer*rows+row)*cols+col.
func (d *Device) Keymap(layers, rows, cols int) ([]uint16, error) {
	buf, err := d.readBuffer(cmdGetKeymapBuffer, layers*rows*cols*2)
	if err != nil {
		return nil, err
	}
	codes := make([]uint16, len(buf)/2)
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
	return d.writeBuffer(cmdSetKeymapBuffer, 0, buf)
}

func (d *Device) MacroCount() (int, error) {
	resp, err := d.command(cmdMacroCount)
	return int(resp[1]), err
}

func (d *Device) MacroBufferSize() (int, error) {
	resp, err := d.command(cmdMacroBufferSize)
	return int(binary.BigEndian.Uint16(resp[1:3])), err
}

func (d *Device) MacroBuffer(size int) ([]byte, error) {
	return d.readBuffer(cmdGetMacroBuffer, size)
}

// SetMacroBuffer replaces all macros the way VIA does: clear the buffer,
// mark its last byte as a write in progress, write, then clear the mark.
func (d *Device) SetMacroBuffer(data []byte, size int) error {
	if len(data) > size {
		return fmt.Errorf("macros need %d bytes, the board has %d", len(data), size)
	}
	// Clearing rewrites the whole buffer in EEPROM before the board replies.
	if _, err := d.commandWithin(10*time.Second, cmdResetMacros); err != nil {
		return err
	}
	if err := d.writeBuffer(cmdSetMacroBuffer, size-1, []byte{0xFF}); err != nil {
		return err
	}
	err := d.writeBuffer(cmdSetMacroBuffer, 0, data)
	return errors.Join(err, d.writeBuffer(cmdSetMacroBuffer, size-1, []byte{0x00}))
}

// readBuffer reads size bytes in 28-byte chunks, which is far fewer round
// trips than reading item by item.
func (d *Device) readBuffer(cmd byte, size int) ([]byte, error) {
	buf := make([]byte, 0, size)
	for off := 0; off < size; off += chunkSize {
		n := min(chunkSize, size-off)
		resp, err := d.command(cmd, byte(off>>8), byte(off), byte(n))
		if err != nil {
			return nil, err
		}
		buf = append(buf, resp[4:4+n]...)
	}
	return buf, nil
}

func (d *Device) writeBuffer(cmd byte, start int, data []byte) error {
	for i := 0; i < len(data); i += chunkSize {
		chunk := data[i:min(i+chunkSize, len(data))]
		off := start + i
		args := append([]byte{byte(off >> 8), byte(off), byte(len(chunk))}, chunk...)
		if _, err := d.command(cmd, args...); err != nil {
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
		if err := d.SetEncoder(i/(count*2), i/2%count, i%2, code); err != nil {
			return err
		}
	}
	return nil
}

// SetEncoder sets what turning one encoder does on a layer; cw is 0 for
// counter-clockwise and 1 for clockwise.
func (d *Device) SetEncoder(layer, encoder, cw int, code uint16) error {
	_, err := d.command(cmdSetEncoder, byte(layer), byte(encoder), byte(cw), byte(code>>8), byte(code))
	return err
}

// ResetKeymap restores the firmware's default keymap and encoder mapping.
// The firmware rewrites all of it in EEPROM before replying, so it gets a
// longer timeout than other commands.
func (d *Device) ResetKeymap() error {
	_, err := d.commandWithin(10*time.Second, cmdResetKeymap)
	return err
}

// CustomValue reads a menu setting of size bytes, big-endian.
func (d *Device) CustomValue(channel, id byte, size int) (int, error) {
	resp, err := d.command(cmdCustomGetValue, channel, id)
	if err != nil {
		return 0, err
	}
	v := 0
	for _, b := range resp[3 : 3+size] {
		v = v<<8 | int(b)
	}
	return v, nil
}

// SetCustomValue applies a menu setting right away but only in RAM; call
// SaveCustom to keep it across unplugging.
func (d *Device) SetCustomValue(channel, id byte, size, value int) error {
	args := []byte{channel, id}
	for i := size - 1; i >= 0; i-- {
		args = append(args, byte(value>>(8*i)))
	}
	_, err := d.command(cmdCustomSetValue, args...)
	return err
}

func (d *Device) SaveCustom(channel byte) error {
	_, err := d.command(cmdCustomSave, channel)
	return err
}

func (d *Device) command(cmd byte, args ...byte) ([reportSize]byte, error) {
	return d.commandWithin(time.Second, cmd, args...)
}

// commandWithin writes a 32-byte VIA report and reads the response, checking
// the command byte is echoed back. Reads block until a report arrives, so
// they run aside with a deadline; a board that never answers leaves one
// goroutine waiting, which only happens once something is already broken.
func (d *Device) commandWithin(timeout time.Duration, cmd byte, args ...byte) ([reportSize]byte, error) {
	var req, resp [reportSize]byte
	req[0] = cmd
	copy(req[1:], args)
	if err := d.dev.SetOutputReport(0, req[:]); err != nil {
		return resp, err
	}

	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		_, data, err := d.dev.GetInputReport()
		done <- result{data, err}
	}()
	var r result
	select {
	case r = <-done:
	case <-time.After(timeout):
		return resp, fmt.Errorf("command 0x%02X: the board didn't answer", cmd)
	}
	if r.err != nil {
		return resp, r.err
	}
	if n := copy(resp[:], r.data); n != reportSize {
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
