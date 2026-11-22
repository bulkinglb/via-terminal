package via

import (
	"errors"
	"slices"
	"testing"
)

// fakeBoard answers each report with handle(request), or echoes it.
type fakeBoard struct {
	sent    [][]byte
	replies chan []byte
	handle  func(req []byte) []byte
}

func (f *fakeBoard) SetOutputReport(_ byte, data []byte) error {
	req := slices.Clone(data)
	f.sent = append(f.sent, req)
	resp := slices.Clone(req)
	if f.handle != nil {
		resp = f.handle(resp)
	}
	f.replies <- resp
	return nil
}

func (f *fakeBoard) GetInputReport() (byte, []byte, error) { return 0, <-f.replies, nil }
func (f *fakeBoard) Close() error                          { return nil }

func newFake(handle func([]byte) []byte) (*Device, *fakeBoard) {
	f := &fakeBoard{replies: make(chan []byte, 1), handle: handle}
	return &Device{dev: f, Version: 12}, f
}

// buffers serves the get/set buffer commands from mem like the firmware does.
func buffers(mem []byte) func([]byte) []byte {
	return func(req []byte) []byte {
		off, n := int(req[1])<<8|int(req[2]), int(req[3])
		switch req[0] {
		case cmdGetKeymapBuffer, cmdGetMacroBuffer:
			copy(req[4:4+n], mem[off:])
		case cmdSetKeymapBuffer, cmdSetMacroBuffer:
			copy(mem[off:off+n], req[4:])
		}
		return req
	}
}

func TestKeymapRoundTrip(t *testing.T) {
	mem := make([]byte, 2*2*2*8) // 2 layers of 2x8 keys, 3 chunks
	dev, board := newFake(buffers(mem))
	want := make([]uint16, len(mem)/2)
	for i := range want {
		want[i] = uint16(0x0100*i + 4)
	}
	if err := dev.SetKeymap(want); err != nil {
		t.Fatal(err)
	}
	got, err := dev.Keymap(2, 2, 8)
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("got %v %v, want %v", got, err, want)
	}
	if len(board.sent) != 6 || board.sent[2][1] != 0 || board.sent[2][2] != 56 || board.sent[2][3] != 8 {
		t.Errorf("want 3 writes then 3 reads, the last of 8 bytes at 56, got %v", board.sent)
	}
}

func TestSetMacroBuffer(t *testing.T) {
	mem := make([]byte, 64)
	dev, board := newFake(buffers(mem))
	if err := dev.SetMacroBuffer(make([]byte, 65), 64); err == nil || len(board.sent) > 0 {
		t.Fatal("macros bigger than the buffer should fail before touching the board")
	}
	if err := dev.SetMacroBuffer([]byte("hi\x00"), 64); err != nil {
		t.Fatal(err)
	}
	// Reset, mark the last byte busy, write, clear the mark.
	var steps []string
	for _, r := range board.sent {
		steps = append(steps, string([]byte{r[0], r[2], r[4]}))
	}
	want := []string{"\x10\x00\x00", "\x0f\x3f\xff", "\x0f\x00h", "\x0f\x3f\x00"}
	if !slices.Equal(steps, want) || string(mem[:3]) != "hi\x00" || mem[63] != 0 {
		t.Errorf("steps %q, want %q; buffer %q", steps, want, mem)
	}
}

func TestCustomValue(t *testing.T) {
	dev, board := newFake(func(req []byte) []byte {
		req[3], req[4] = 0xAA, 0x80
		return req
	})
	if err := dev.SetCustomValue(3, 4, 2, 0xAA80); err != nil {
		t.Fatal(err)
	}
	if got := board.sent[0][:5]; !slices.Equal(got, []byte{cmdCustomSetValue, 3, 4, 0xAA, 0x80}) {
		t.Errorf("set sent %X", got)
	}
	if v, err := dev.CustomValue(3, 4, 2); err != nil || v != 0xAA80 {
		t.Errorf("got 0x%X %v, want 0xAA80", v, err)
	}
}

func TestProtocol10(t *testing.T) {
	mem := make([]byte, 16)
	dev, board := newFake(func(req []byte) []byte {
		if req[0] == cmdCustomGetValue {
			req[2] = 200
			return req
		}
		return buffers(mem)(req)
	})
	dev.Version = 10
	// RGB light brightness is lighting value 0x80, with no channel byte.
	if err := dev.SetCustomValue(2, 1, 1, 128); err != nil {
		t.Fatal(err)
	}
	if got := board.sent[0][:3]; !slices.Equal(got, []byte{cmdCustomSetValue, 0x80, 128}) {
		t.Errorf("set sent % X", got)
	}
	if v, err := dev.CustomValue(1, 2, 1); err != nil || v != 200 || board.sent[1][1] != 0x0A {
		t.Errorf("backlight effect: got %d %v from % X", v, err, board.sent[1][:2])
	}
	if _, err := dev.CustomValue(0, 1, 1); !errors.Is(err, ErrUnhandled) {
		t.Errorf("custom channels didn't exist yet, got %v", err)
	}

	// Macros go to the board without the action prefix and come back with it.
	if err := dev.SetMacroBuffer([]byte{'a', 1, 1, 0x04, 0}, 16); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(mem[:4], []byte{'a', 1, 0x04, 0}) {
		t.Errorf("stored % X", mem[:4])
	}
	if buf, err := dev.MacroBuffer(4); err != nil || !slices.Equal(buf, []byte{'a', 1, 1, 0x04, 0}) {
		t.Errorf("read back % X %v", buf, err)
	}
	if n, err := dev.MacroLen([]byte{1, 4, '9', '|', 0}); err == nil {
		t.Errorf("a delay should be refused, got %d bytes", n)
	}
}

func TestCommandErrors(t *testing.T) {
	unhandled, _ := newFake(func(req []byte) []byte { req[0] = cmdUnhandled; return req })
	if _, err := unhandled.LayerCount(); !errors.Is(err, ErrUnhandled) {
		t.Errorf("0xFF should be ErrUnhandled, got %v", err)
	}
	if n, err := unhandled.MacroCount(); n != 0 || err != nil {
		t.Errorf("firmware without macros should have 0, got %d %v", n, err)
	}
	wrong, _ := newFake(func(req []byte) []byte { req[0]++; return req })
	if _, err := wrong.MacroCount(); err == nil {
		t.Error("a reply to another command should be an error, not 0 macros")
	}
}
