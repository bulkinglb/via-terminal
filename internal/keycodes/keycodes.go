// Package keycodes names QMK keycodes as VIA protocol v12 numbers them.
package keycodes

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Keycode names follow QMK's shortest alias without the KC_ prefix. Long is
// the spelled-out name, only there so the picker finds "space" or "shift".
type Keycode struct {
	Code       uint16
	Name, Long string
}

// table holds the VIA protocol v12 numbering (QMK 0.19+), checked against
// QMK's data/constants/keycodes specs. 0x7820-0x782A use the RGB_* names
// because VIA-era firmware drives the RGB matrix with them; newer QMK calls
// them UG_* and moved matrix control to RM_*.
var table = slices.Concat(series(), []Keycode{
	{0x0028, "ENT", "ENTER"},
	{0x0029, "ESC", "ESCAPE"},
	{0x002A, "BSPC", "BACKSPACE"},
	{0x002B, "TAB", ""},
	{0x002C, "SPC", "SPACE"},
	{0x002D, "MINS", "MINUS"},
	{0x002E, "EQL", "EQUAL"},
	{0x002F, "LBRC", "LEFT_BRACKET"},
	{0x0030, "RBRC", "RIGHT_BRACKET"},
	{0x0031, "BSLS", "BACKSLASH"},
	{0x0032, "NUHS", "NONUS_HASH"},
	{0x0033, "SCLN", "SEMICOLON"},
	{0x0034, "QUOT", "QUOTE"},
	{0x0035, "GRV", "GRAVE"},
	{0x0036, "COMM", "COMMA"},
	{0x0037, "DOT", ""},
	{0x0038, "SLSH", "SLASH"},
	{0x0039, "CAPS", "CAPS_LOCK"},
	{0x0046, "PSCR", "PRINT_SCREEN"},
	{0x0047, "SCRL", "SCROLL_LOCK"},
	{0x0048, "BRK", "PAUSE"},
	{0x0049, "INS", "INSERT"},
	{0x004A, "HOME", ""},
	{0x004B, "PGUP", "PAGE_UP"},
	{0x004C, "DEL", "DELETE"},
	{0x004D, "END", ""},
	{0x004E, "PGDN", "PAGE_DOWN"},
	{0x004F, "RGHT", "RIGHT"},
	{0x0050, "LEFT", ""},
	{0x0051, "DOWN", ""},
	{0x0052, "UP", ""},
	{0x0053, "NUM", "NUM_LOCK"},
	{0x0054, "PSLS", "KP_SLASH"},
	{0x0055, "PAST", "KP_ASTERISK"},
	{0x0056, "PMNS", "KP_MINUS"},
	{0x0057, "PPLS", "KP_PLUS"},
	{0x0058, "PENT", "KP_ENTER"},
	{0x0059, "P1", "KP_1"},
	{0x005A, "P2", "KP_2"},
	{0x005B, "P3", "KP_3"},
	{0x005C, "P4", "KP_4"},
	{0x005D, "P5", "KP_5"},
	{0x005E, "P6", "KP_6"},
	{0x005F, "P7", "KP_7"},
	{0x0060, "P8", "KP_8"},
	{0x0061, "P9", "KP_9"},
	{0x0062, "P0", "KP_0"},
	{0x0063, "PDOT", "KP_DOT"},
	{0x0064, "NUBS", "NONUS_BACKSLASH"},
	{0x0065, "APP", "APPLICATION"},
	{0x0066, "KB_POWER", ""},
	{0x0067, "PEQL", "KP_EQUAL"},
	{0x0074, "EXEC", "EXECUTE"},
	{0x0075, "HELP", ""},
	{0x0076, "MENU", ""},
	{0x0077, "SLCT", "SELECT"},
	{0x0078, "STOP", ""},
	{0x0079, "AGIN", "AGAIN"},
	{0x007A, "UNDO", ""},
	{0x007B, "CUT", ""},
	{0x007C, "COPY", ""},
	{0x007D, "PSTE", "PASTE"},
	{0x007E, "FIND", ""},
	{0x007F, "KB_MUTE", ""},
	{0x0080, "KB_VOLUME_UP", ""},
	{0x0081, "KB_VOLUME_DOWN", ""},
	{0x0082, "LCAP", "LOCKING_CAPS_LOCK"},
	{0x0083, "LNUM", "LOCKING_NUM_LOCK"},
	{0x0084, "LSCR", "LOCKING_SCROLL_LOCK"},
	{0x0085, "PCMM", "KP_COMMA"},
	{0x0086, "KP_EQUAL_AS400", ""},
	{0x0087, "INT1", "INTERNATIONAL_1"},
	{0x0088, "INT2", "INTERNATIONAL_2"},
	{0x0089, "INT3", "INTERNATIONAL_3"},
	{0x008A, "INT4", "INTERNATIONAL_4"},
	{0x008B, "INT5", "INTERNATIONAL_5"},
	{0x008C, "INT6", "INTERNATIONAL_6"},
	{0x008D, "INT7", "INTERNATIONAL_7"},
	{0x008E, "INT8", "INTERNATIONAL_8"},
	{0x008F, "INT9", "INTERNATIONAL_9"},
	{0x0090, "LNG1", "LANGUAGE_1"},
	{0x0091, "LNG2", "LANGUAGE_2"},
	{0x0092, "LNG3", "LANGUAGE_3"},
	{0x0093, "LNG4", "LANGUAGE_4"},
	{0x0094, "LNG5", "LANGUAGE_5"},
	{0x0095, "LNG6", "LANGUAGE_6"},
	{0x0096, "LNG7", "LANGUAGE_7"},
	{0x0097, "LNG8", "LANGUAGE_8"},
	{0x0098, "LNG9", "LANGUAGE_9"},
	{0x0099, "ERAS", "ALTERNATE_ERASE"},
	{0x009A, "SYRQ", "SYSTEM_REQUEST"},
	{0x009B, "CNCL", "CANCEL"},
	{0x009C, "CLR", "CLEAR"},
	{0x009D, "PRIR", "PRIOR"},
	{0x009E, "RETN", "RETURN"},
	{0x009F, "SEPR", "SEPARATOR"},
	{0x00A0, "OUT", ""},
	{0x00A1, "OPER", ""},
	{0x00A2, "CLAG", "CLEAR_AGAIN"},
	{0x00A3, "CRSL", "CRSEL"},
	{0x00A4, "EXSL", "EXSEL"},
	{0x00A5, "PWR", "SYSTEM_POWER"},
	{0x00A6, "SLEP", "SYSTEM_SLEEP"},
	{0x00A7, "WAKE", "SYSTEM_WAKE"},
	{0x00A8, "MUTE", "AUDIO_MUTE"},
	{0x00A9, "VOLU", "AUDIO_VOL_UP"},
	{0x00AA, "VOLD", "AUDIO_VOL_DOWN"},
	{0x00AB, "MNXT", "MEDIA_NEXT_TRACK"},
	{0x00AC, "MPRV", "MEDIA_PREV_TRACK"},
	{0x00AD, "MSTP", "MEDIA_STOP"},
	{0x00AE, "MPLY", "MEDIA_PLAY_PAUSE"},
	{0x00AF, "MSEL", "MEDIA_SELECT"},
	{0x00B0, "EJCT", "MEDIA_EJECT"},
	{0x00B1, "MAIL", ""},
	{0x00B2, "CALC", "CALCULATOR"},
	{0x00B3, "MYCM", "MY_COMPUTER"},
	{0x00B4, "WSCH", "WWW_SEARCH"},
	{0x00B5, "WHOM", "WWW_HOME"},
	{0x00B6, "WBAK", "WWW_BACK"},
	{0x00B7, "WFWD", "WWW_FORWARD"},
	{0x00B8, "WSTP", "WWW_STOP"},
	{0x00B9, "WREF", "WWW_REFRESH"},
	{0x00BA, "WFAV", "WWW_FAVORITES"},
	{0x00BB, "MFFD", "MEDIA_FAST_FORWARD"},
	{0x00BC, "MRWD", "MEDIA_REWIND"},
	{0x00BD, "BRIU", "BRIGHTNESS_UP"},
	{0x00BE, "BRID", "BRIGHTNESS_DOWN"},
	{0x00BF, "CPNL", "CONTROL_PANEL"},
	{0x00C0, "ASST", "ASSISTANT"},
	{0x00C1, "MCTL", "MISSION_CONTROL"},
	{0x00C2, "LPAD", "LAUNCHPAD"},
	{0x00CD, "MS_UP", "MOUSE_CURSOR_UP"},
	{0x00CE, "MS_DOWN", "MOUSE_CURSOR_DOWN"},
	{0x00CF, "MS_LEFT", "MOUSE_CURSOR_LEFT"},
	{0x00D0, "MS_RGHT", "MOUSE_CURSOR_RIGHT"},
	{0x00D1, "MS_BTN1", "MOUSE_BUTTON_1"},
	{0x00D2, "MS_BTN2", "MOUSE_BUTTON_2"},
	{0x00D3, "MS_BTN3", "MOUSE_BUTTON_3"},
	{0x00D4, "MS_BTN4", "MOUSE_BUTTON_4"},
	{0x00D5, "MS_BTN5", "MOUSE_BUTTON_5"},
	{0x00D6, "MS_BTN6", "MOUSE_BUTTON_6"},
	{0x00D7, "MS_BTN7", "MOUSE_BUTTON_7"},
	{0x00D8, "MS_BTN8", "MOUSE_BUTTON_8"},
	{0x00D9, "MS_WHLU", "MOUSE_WHEEL_UP"},
	{0x00DA, "MS_WHLD", "MOUSE_WHEEL_DOWN"},
	{0x00DB, "MS_WHLL", "MOUSE_WHEEL_LEFT"},
	{0x00DC, "MS_WHLR", "MOUSE_WHEEL_RIGHT"},
	{0x00DD, "MS_ACL0", "MOUSE_ACCELERATION_0"},
	{0x00DE, "MS_ACL1", "MOUSE_ACCELERATION_1"},
	{0x00DF, "MS_ACL2", "MOUSE_ACCELERATION_2"},
	{0x00E0, "LCTL", "LEFT_CTRL"},
	{0x00E1, "LSFT", "LEFT_SHIFT"},
	{0x00E2, "LALT", "LEFT_ALT"},
	{0x00E3, "LGUI", "LEFT_GUI"},
	{0x00E4, "RCTL", "RIGHT_CTRL"},
	{0x00E5, "RSFT", "RIGHT_SHIFT"},
	{0x00E6, "RALT", "RIGHT_ALT"},
	{0x00E7, "RGUI", "RIGHT_GUI"},
	{0x7000, "CL_SWAP", "MAGIC_SWAP_CONTROL_CAPS_LOCK"},
	{0x7001, "CL_NORM", "MAGIC_UNSWAP_CONTROL_CAPS_LOCK"},
	{0x7002, "CL_TOGG", "MAGIC_TOGGLE_CONTROL_CAPS_LOCK"},
	{0x7003, "CL_CAPS", "MAGIC_CAPS_LOCK_AS_CONTROL_OFF"},
	{0x7004, "CL_CTRL", "MAGIC_CAPS_LOCK_AS_CONTROL_ON"},
	{0x7005, "AG_LSWP", "MAGIC_SWAP_LALT_LGUI"},
	{0x7006, "AG_LNRM", "MAGIC_UNSWAP_LALT_LGUI"},
	{0x7007, "AG_RSWP", "MAGIC_SWAP_RALT_RGUI"},
	{0x7008, "AG_RNRM", "MAGIC_UNSWAP_RALT_RGUI"},
	{0x7009, "GU_ON", "MAGIC_GUI_ON"},
	{0x700A, "GU_OFF", "MAGIC_GUI_OFF"},
	{0x700B, "GU_TOGG", "MAGIC_TOGGLE_GUI"},
	{0x700C, "GE_SWAP", "MAGIC_SWAP_GRAVE_ESC"},
	{0x700D, "GE_NORM", "MAGIC_UNSWAP_GRAVE_ESC"},
	{0x700E, "BS_SWAP", "MAGIC_SWAP_BACKSLASH_BACKSPACE"},
	{0x700F, "BS_NORM", "MAGIC_UNSWAP_BACKSLASH_BACKSPACE"},
	{0x7010, "BS_TOGG", "MAGIC_TOGGLE_BACKSLASH_BACKSPACE"},
	{0x7011, "NK_ON", "MAGIC_NKRO_ON"},
	{0x7012, "NK_OFF", "MAGIC_NKRO_OFF"},
	{0x7013, "NK_TOGG", "MAGIC_TOGGLE_NKRO"},
	{0x7014, "AG_SWAP", "MAGIC_SWAP_ALT_GUI"},
	{0x7015, "AG_NORM", "MAGIC_UNSWAP_ALT_GUI"},
	{0x7016, "AG_TOGG", "MAGIC_TOGGLE_ALT_GUI"},
	{0x7017, "CG_LSWP", "MAGIC_SWAP_LCTL_LGUI"},
	{0x7018, "CG_LNRM", "MAGIC_UNSWAP_LCTL_LGUI"},
	{0x7019, "CG_RSWP", "MAGIC_SWAP_RCTL_RGUI"},
	{0x701A, "CG_RNRM", "MAGIC_UNSWAP_RCTL_RGUI"},
	{0x701B, "CG_SWAP", "MAGIC_SWAP_CTL_GUI"},
	{0x701C, "CG_NORM", "MAGIC_UNSWAP_CTL_GUI"},
	{0x701D, "CG_TOGG", "MAGIC_TOGGLE_CTL_GUI"},
	{0x701E, "EH_LEFT", "MAGIC_EE_HANDS_LEFT"},
	{0x701F, "EH_RGHT", "MAGIC_EE_HANDS_RIGHT"},
	{0x7020, "EC_SWAP", "MAGIC_SWAP_ESCAPE_CAPS_LOCK"},
	{0x7021, "EC_NORM", "MAGIC_UNSWAP_ESCAPE_CAPS_LOCK"},
	{0x7022, "EC_TOGG", "MAGIC_TOGGLE_ESCAPE_CAPS_LOCK"},
	{0x7800, "BL_ON", "BACKLIGHT_ON"},
	{0x7801, "BL_OFF", "BACKLIGHT_OFF"},
	{0x7802, "BL_TOGG", "BACKLIGHT_TOGGLE"},
	{0x7803, "BL_DOWN", "BACKLIGHT_DOWN"},
	{0x7804, "BL_UP", "BACKLIGHT_UP"},
	{0x7805, "BL_STEP", "BACKLIGHT_STEP"},
	{0x7806, "BL_BRTG", "BACKLIGHT_TOGGLE_BREATHING"},
	{0x7820, "RGB_TOG", "RGB_TOGGLE"},
	{0x7821, "RGB_MOD", "RGB_MODE_NEXT"},
	{0x7822, "RGB_RMOD", "RGB_MODE_PREVIOUS"},
	{0x7823, "RGB_HUI", "RGB_HUE_UP"},
	{0x7824, "RGB_HUD", "RGB_HUE_DOWN"},
	{0x7825, "RGB_SAI", "RGB_SATURATION_UP"},
	{0x7826, "RGB_SAD", "RGB_SATURATION_DOWN"},
	{0x7827, "RGB_VAI", "RGB_BRIGHTNESS_UP"},
	{0x7828, "RGB_VAD", "RGB_BRIGHTNESS_DOWN"},
	{0x7829, "RGB_SPI", "RGB_SPEED_UP"},
	{0x782A, "RGB_SPD", "RGB_SPEED_DOWN"},
	{0x7840, "RM_ON", "RGB_MATRIX_ON"},
	{0x7841, "RM_OFF", "RGB_MATRIX_OFF"},
	{0x7842, "RM_TOGG", "RGB_MATRIX_TOGGLE"},
	{0x7843, "RM_NEXT", "RGB_MATRIX_MODE_NEXT"},
	{0x7844, "RM_PREV", "RGB_MATRIX_MODE_PREVIOUS"},
	{0x7845, "RM_HUEU", "RGB_MATRIX_HUE_UP"},
	{0x7846, "RM_HUED", "RGB_MATRIX_HUE_DOWN"},
	{0x7847, "RM_SATU", "RGB_MATRIX_SATURATION_UP"},
	{0x7848, "RM_SATD", "RGB_MATRIX_SATURATION_DOWN"},
	{0x7849, "RM_VALU", "RGB_MATRIX_VALUE_UP"},
	{0x784A, "RM_VALD", "RGB_MATRIX_VALUE_DOWN"},
	{0x784B, "RM_SPDU", "RGB_MATRIX_SPEED_UP"},
	{0x784C, "RM_SPDD", "RGB_MATRIX_SPEED_DOWN"},
	{0x7C00, "QK_BOOT", "BOOTLOADER"},
	{0x7C01, "QK_RBT", "REBOOT"},
	{0x7C02, "DB_TOGG", "DEBUG_TOGGLE"},
	{0x7C03, "EE_CLR", "CLEAR_EEPROM"},
	{0x7C16, "QK_GESC", "GRAVE_ESCAPE"},
	{0x7C73, "CW_TOGG", "CAPS_WORD_TOGGLE"},
	{0x7C77, "TL_LOWR", "TRI_LAYER_LOWER"},
	{0x7C78, "TL_UPPR", "TRI_LAYER_UPPER"},
	{0x7C79, "QK_REP", "REPEAT_KEY"},
	{0x7C7B, "QK_LLCK", "LAYER_LOCK"},
})

var names = func() map[uint16]string {
	m := make(map[uint16]string, len(table))
	for _, k := range table {
		m[k.Code] = k.Name
	}
	return m
}()

func series() []Keycode {
	list := []Keycode{{0x0000, "NO", ""}, {0x0001, "TRNS", "TRANSPARENT"}}
	for i := range 26 {
		list = append(list, Keycode{0x04 + uint16(i), string(rune('A' + i)), ""})
	}
	for i := range 10 {
		list = append(list, Keycode{0x1E + uint16(i), string("1234567890"[i]), ""})
	}
	for i := range 24 {
		code := 0x3A + uint16(i)
		if i >= 12 {
			code = 0x68 + uint16(i-12)
		}
		list = append(list, Keycode{code, fmt.Sprintf("F%d", i+1), ""})
	}
	return list
}

// layerFns are the layer keycodes from 0x5200 on, 32 codes each.
var layerFns = [...]string{"TO", "MO", "DF", "TG", "OSL", "OSM", "TT", "PDF"}

// numbered are the ranges QMK names PREFIX_n.
var numbered = []struct {
	prefix      string
	base, count uint16
}{
	{"MC_", 0x7700, 0x80},
	{"QK_KB_", 0x7E00, 0x40},
	{"QK_USER_", 0x7E40, 0x1C0},
}

// Name turns a v12 keycode into a QMK-style name that Parse reads back.
// custom holds the definition's customKeycodes, which VIA maps to QK_KB_0
// onwards. Codes without a meaningful name come out as hex.
func Name(code uint16, custom []string) string {
	if name, ok := names[code]; ok {
		return name
	}
	if i := int(code) - 0x7E00; i >= 0 && i < len(custom) {
		return custom[i]
	}
	for _, r := range numbered {
		if code >= r.base && code < r.base+r.count {
			return fmt.Sprintf("%s%d", r.prefix, code-r.base)
		}
	}
	basic := func(c uint16) string { return Name(c&0xFF, nil) }
	mods := func(mask uint16) string { return strings.Join(modNames(mask), "|") }
	switch {
	case code >= 0x0100 && code < 0x2000 && code>>8&0xF != 0:
		name := basic(code)
		for _, mod := range slices.Backward(modNames(code >> 8)) {
			name = mod + "(" + name + ")"
		}
		return name
	case code >= 0x2000 && code < 0x4000 && code>>8&0xF != 0:
		return fmt.Sprintf("MT(%s,%s)", mods(code>>8), basic(code))
	case code >= 0x4000 && code < 0x5000:
		return fmt.Sprintf("LT(%d,%s)", code>>8&0xF, basic(code))
	case code >= 0x5000 && code < 0x5200 && code&0xF != 0:
		return fmt.Sprintf("LM(%d,%s)", code>>5&0xF, mods(code))
	case code >= 0x5200 && code < 0x5300:
		fn := layerFns[(code-0x5200)>>5]
		if fn != "OSM" {
			return fmt.Sprintf("%s(%d)", fn, code&0x1F)
		}
		if code&0xF != 0 {
			return "OSM(" + mods(code) + ")"
		}
	case code >= 0x5700 && code < 0x5800:
		return fmt.Sprintf("TD(%d)", code&0xFF)
	}
	return fmt.Sprintf("0x%04X", code)
}

var byName = func() map[string]uint16 {
	m := make(map[string]uint16, 2*len(table))
	for _, k := range table {
		if k.Long != "" {
			m[k.Long] = k.Code
		}
	}
	// Short names win if one collides with another key's long name.
	for _, k := range table {
		m[k.Name] = k.Code
	}
	return m
}()

// Parse reads a keycode the way Name writes it. It ignores case, allows a
// KC_ prefix and long names like SPACE, and takes hex such as 0x7E05.
func Parse(s string, custom []string) (uint16, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	for i, name := range custom {
		if strings.ToUpper(name) == s {
			return 0x7E00 + uint16(i), nil
		}
	}
	if code, ok := byName[strings.TrimPrefix(s, "KC_")]; ok {
		return code, nil
	}
	if hex, ok := strings.CutPrefix(s, "0X"); ok {
		if code, err := strconv.ParseUint(hex, 16, 16); err == nil {
			return uint16(code), nil
		}
	}
	for _, r := range numbered {
		if n, ok := strings.CutPrefix(s, r.prefix); ok {
			if i, err := strconv.Atoi(n); err == nil && i >= 0 && i < int(r.count) {
				return r.base + uint16(i), nil
			}
		}
	}

	unknown := fmt.Errorf("unknown keycode %q", s)
	fn, args, open := strings.Cut(s, "(")
	args, closed := strings.CutSuffix(args, ")")
	if !open || !closed {
		return 0, unknown
	}
	first, second, _ := strings.Cut(args, ",")
	switch fn {
	case "TD":
		n, err := number(args, 0xFF)
		return 0x5700 | n, err
	case "OSM":
		mask, err := parseMods(args)
		return 0x52A0 | mask, err
	case "LM":
		layer, err := number(first, 0xF)
		mask, merr := parseMods(second)
		return 0x5000 | layer<<5 | mask, cmp.Or(err, merr)
	case "LT":
		layer, err := number(first, 0xF)
		kc, kerr := parseBasic(second)
		return 0x4000 | layer<<8 | kc, cmp.Or(err, kerr)
	case "MT":
		mask, err := parseMods(first)
		kc, kerr := parseBasic(second)
		return 0x2000 | mask<<8 | kc, cmp.Or(err, kerr)
	}
	if i := slices.Index(layerFns[:], fn); i >= 0 {
		layer, err := number(args, 0x1F)
		return 0x5200 + uint16(i)<<5 | layer, err
	}

	// Modifier wrappers such as LSFT(A) or LCTL(LSFT(A)).
	mask, err := parseMods(fn)
	if err != nil {
		return 0, unknown
	}
	inner, err := Parse(args, nil)
	if err != nil {
		return 0, err
	}
	if inner >= 0x2000 || inner >= 0x0100 && inner>>12 != mask>>4 {
		return 0, fmt.Errorf("%q: modifiers must wrap a basic key and be all left or all right", s)
	}
	return inner | mask<<8, nil
}

func number(s string, max uint16) (uint16, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 || n > int(max) {
		return 0, fmt.Errorf("%q: want a number from 0 to %d", s, max)
	}
	return uint16(n), nil
}

func parseBasic(s string) (uint16, error) {
	code, err := Parse(s, nil)
	if err == nil && code > 0xFF {
		err = fmt.Errorf("%q: only basic keys fit here", s)
	}
	return code, err
}

// parseMods reads "LCTL|LSFT" back into QMK's 5-bit mod mask.
func parseMods(s string) (uint16, error) {
	var mask uint16
	side := ""
	for _, name := range strings.Split(s, "|") {
		name = strings.TrimSpace(name)
		i := slices.Index(modList, strings.TrimLeft(name, "LR"))
		if len(name) != 4 || i < 0 || side != "" && side != name[:1] {
			return 0, fmt.Errorf("%q: want mods like LCTL|LSFT, all left or all right", s)
		}
		side = name[:1]
		mask |= 1 << i
	}
	if side == "R" {
		mask |= 0x10
	}
	return mask, nil
}

var modList = []string{"CTL", "SFT", "ALT", "GUI"}

// modNames decodes QMK's 5-bit mod mask: ctrl, shift, alt, gui, and a fifth
// bit that makes all of them right-hand.
func modNames(mask uint16) []string {
	side := "L"
	if mask&0x10 != 0 {
		side = "R"
	}
	var names []string
	for i, mod := range modList {
		if mask&(1<<i) != 0 {
			names = append(names, side+mod)
		}
	}
	return names
}

// Picker is everything the remap picker offers: the named table,
// layer switches for the board's layers and its custom keycodes.
func Picker(layers int, custom []string) []Keycode {
	list := slices.Clone(table)
	for _, base := range []uint16{0x5220, 0x5260, 0x52C0, 0x5280, 0x5200, 0x5240} {
		for layer := range layers {
			list = append(list, Keycode{base | uint16(layer), Name(base|uint16(layer), nil), ""})
		}
	}
	for i, name := range custom {
		list = append(list, Keycode{0x7E00 + uint16(i), strings.ToUpper(name), ""})
	}
	return list
}
