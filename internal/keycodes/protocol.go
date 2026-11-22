package keycodes

import (
	"fmt"
	"strconv"
	"strings"
)

// Boards before VIA protocol 12 number some keys differently. The exported
// functions take and return the board's own numbers and convert to v12 for
// naming, so names stay the same on every board and a backup moves between
// protocols. A code v12 has no number for is shown and read back as hex.

// block moves n codes that start at old on the board to v12's start.
type block struct{ old, v12, n uint16 }

type codec struct {
	protocol int
	blocks   []block
	// same reports codes both numberings share, outside the blocks.
	same func(code uint16) bool
	// layerMod4: LM(layer, mods) was 0x5900 | layer<<4 | mods, left mods only.
	layerMod4 bool
}

// active is the connected board's numbering; nil means v12.
var active *codec

// UseProtocol picks the keycode numbering for the board's VIA protocol.
func UseProtocol(protocol int) {
	switch {
	case protocol >= 12:
		active = nil
	case protocol == 11:
		active = &codec{protocol: 11, blocks: v11Blocks, same: func(c uint16) bool {
			return !(c >= 0x7700 && c <= 0x77FF || c >= 0x7E00 && c <= 0x7FFF || c == 0x7C77 || c == 0x7C78)
		}}
	default:
		blocks := v10Blocks
		if protocol < 10 {
			// TO(n) was 0x5010 | n before protocol 10, the "on press" bit included.
			blocks = append([]block{{0x5010, 0x5200, 16}}, v10Blocks[1:]...)
		}
		active = &codec{protocol: protocol, blocks: blocks, same: sameV10, layerMod4: true}
	}
}

// From VIA's keycode tables for protocol 11 and for 10 and before.
var v11Blocks = []block{
	{0x7700, 0x7C77, 2},   // TL_LOWR, TL_UPPR
	{0x7702, 0x7700, 16},  // macros
	{0x7F00, 0x7E00, 256}, // keyboard keycodes
}

// v10Blocks starts with TO, which UseProtocol swaps out for protocol 9.
var v10Blocks = []block{
	{0x5000, 0x5200, 32},     // TO
	{0x00F0, 0x00CD, 9},      // mouse cursor and buttons 1-5
	{0x00F9, 0x00D9, 7},      // mouse wheel and acceleration
	{0x6000, 0x2000, 0x2000}, // mod-tap
	{0x5100, 0x5220, 32},     // MO
	{0x5200, 0x5240, 32},     // DF
	{0x5300, 0x5260, 32},     // TG
	{0x5400, 0x5280, 32},     // OSL
	{0x5500, 0x52A0, 32},     // OSM
	{0x5800, 0x52C0, 32},     // TT
	{0x5C00, 0x7C00, 1},      // QK_BOOT
	{0x5C01, 0x7C02, 1},      // DB_TOGG
	{0x5C14, 0x7013, 1},      // NK_TOGG
	{0x5C16, 0x7C16, 1},      // QK_GESC
	{0x5C1D, 0x7480, 3},      // audio
	{0x5C20, 0x748A, 6},      // clicky
	{0x5C26, 0x7490, 4},      // music
	{0x5CBB, 0x7800, 2},      // BL_ON, BL_OFF
	{0x5CBD, 0x7803, 2},      // BL_DOWN, BL_UP
	{0x5CBF, 0x7802, 1},      // BL_TOGG
	{0x5CC0, 0x7805, 2},      // BL_STEP, BL_BRTG
	{0x5CC2, 0x7820, 19},     // RGB_TOG to RGB_M_G
	{0x5CD7, 0x7C1A, 2},      // SC_LSPO, SC_RSPC
	{0x5CD9, 0x7C1E, 1},      // SC_SENT
	{0x5CF3, 0x7C18, 2},      // SC_LCPO, SC_RCPC
	{0x5CF5, 0x7C1C, 2},      // SC_LAPO, SC_RAPC
	{0x5F10, 0x7C77, 2},      // TL_LOWR, TL_UPPR
	{0x5F12, 0x7700, 16},     // macros
	{0x5F80, 0x7E00, 16},     // keyboard keycodes
}

// sameV10 is basic HID keys, modifiers, mods(kc), LT, swap hands, tap dance
// and unicode, which QMK numbered the same before protocol 11.
func sameV10(c uint16) bool {
	return c <= 0x00BE || c == 0x00C1 || c == 0x00C2 || c >= 0x00E0 && c <= 0x00E7 || c >= 0x0100 && c <= 0x1FFF ||
		c >= 0x4000 && c <= 0x4FFF || c >= 0x5600 && c <= 0x57FF || c >= 0x8000
}

func (k *codec) toV12(code uint16) (uint16, bool) {
	for _, b := range k.blocks {
		if code >= b.old && code-b.old < b.n {
			return b.v12 + code - b.old, true
		}
	}
	if k.layerMod4 && code >= 0x5900 && code <= 0x59FF {
		return 0x5000 | (code>>4&0xF)<<5 | code&0xF, true
	}
	return code, k.same(code)
}

func (k *codec) fromV12(code uint16) (uint16, bool) {
	for _, b := range k.blocks {
		if code >= b.v12 && code-b.v12 < b.n {
			return b.old + code - b.v12, true
		}
	}
	if k.layerMod4 && code >= 0x5000 && code <= 0x51FF {
		return 0x5900 | (code>>5&0xF)<<4 | code&0xF, code&0x10 == 0
	}
	return code, k.same(code)
}

// V12 converts a board keycode to v12 numbering; ok is false when v12 has no
// equivalent.
func V12(code uint16) (uint16, bool) {
	if active == nil {
		return code, true
	}
	return active.toV12(code)
}

// Native converts a v12 keycode to the board's numbering; ok is false when
// the board's protocol has no such key.
func Native(code uint16) (uint16, bool) {
	if active == nil {
		return code, true
	}
	return active.fromV12(code)
}

// Name turns a board keycode into a QMK-style name that Parse reads back.
// custom holds the definition's customKeycodes, which VIA maps to QK_KB_0
// onwards. Codes without a meaningful name come out as hex.
func Name(code uint16, custom []string) string {
	v, ok := V12(code)
	if !ok {
		return hex(code)
	}
	// Hex always means the board's own number, never v12's.
	if name := name12(v, custom); v == code || !strings.HasPrefix(name, "0x") {
		return name
	}
	return hex(code)
}

// Parse reads a keycode the way Name writes it, as a board keycode. It
// ignores case, allows a KC_ prefix and long names like SPACE, and takes
// hex such as 0x7E05.
func Parse(s string, custom []string) (uint16, error) {
	if h, ok := strings.CutPrefix(strings.ToUpper(strings.TrimSpace(s)), "0X"); ok {
		if code, err := strconv.ParseUint(h, 16, 16); err == nil {
			return uint16(code), nil
		}
	}
	v, err := parse12(s, custom)
	if err != nil {
		return 0, err
	}
	code, ok := Native(v)
	if !ok {
		return 0, fmt.Errorf("%s doesn't exist on VIA protocol %d boards", strings.ToUpper(strings.TrimSpace(s)), active.protocol)
	}
	return code, nil
}

// Describe says what a board keycode does, for keys in the table and the
// ones built from arguments like LT(1,A).
func Describe(code uint16) string {
	if v, ok := V12(code); ok {
		return describe12(v)
	}
	return ""
}

func hex(code uint16) string { return fmt.Sprintf("0x%04X", code) }
