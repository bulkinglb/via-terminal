// Package keycodes names QMK keycodes as VIA protocol v12 numbers them.
package keycodes

import (
	"fmt"
	"slices"
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

// Name turns a v12 keycode into a QMK-style name. custom holds the
// definition's customKeycodes, which VIA maps to QK_KB_0 onwards.
func Name(code uint16, custom []string) string {
	basic := func(c uint16) string { return Name(c&0xFF, nil) }
	switch {
	case code <= 0xFF:
		if name, ok := names[code]; ok {
			return name
		}
	case code < 0x2000:
		name := basic(code)
		mods := modNames(code >> 8)
		for i := len(mods) - 1; i >= 0; i-- {
			name = mods[i] + "(" + name + ")"
		}
		return name
	case code < 0x4000:
		return fmt.Sprintf("MT(%s,%s)", strings.Join(modNames(code>>8), "|"), basic(code))
	case code < 0x5000:
		return fmt.Sprintf("LT(%d,%s)", code>>8&0xF, basic(code))
	case code < 0x5200:
		return fmt.Sprintf("LM(%d,%s)", code>>5&0xF, strings.Join(modNames(code), "|"))
	case code < 0x5300:
		fn := [...]string{"TO", "MO", "DF", "TG", "OSL", "OSM", "TT", "PDF"}[(code-0x5200)>>5]
		if fn == "OSM" {
			return "OSM(" + strings.Join(modNames(code), "|") + ")"
		}
		return fmt.Sprintf("%s(%d)", fn, code&0x1F)
	case code >= 0x5700 && code < 0x5800:
		return fmt.Sprintf("TD(%d)", code&0xFF)
	case code >= 0x7700 && code < 0x7780:
		return fmt.Sprintf("MC_%d", code&0x7F)
	case code >= 0x7E00 && code < 0x7E40:
		if i := int(code - 0x7E00); i < len(custom) {
			return custom[i]
		}
		return fmt.Sprintf("QK_KB_%d", code-0x7E00)
	case code >= 0x7E40 && code < 0x8000:
		return fmt.Sprintf("QK_USER_%d", code-0x7E40)
	default:
		if name, ok := names[code]; ok {
			return name
		}
	}
	return fmt.Sprintf("0x%04X", code)
}

// modNames decodes QMK's 5-bit mod mask: ctrl, shift, alt, gui, and a fifth
// bit that makes all of them right-hand.
func modNames(mask uint16) []string {
	side := "L"
	if mask&0x10 != 0 {
		side = "R"
	}
	var names []string
	for i, mod := range []string{"CTL", "SFT", "ALT", "GUI"} {
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

// Filter ranks exact matches first, then prefixes, then substrings,
// keeping table order within each group.
func Filter(list []Keycode, query string) []Keycode {
	q := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(query)), "KC_")
	if q == "" {
		return list
	}
	var exact, prefix, rest []Keycode
	for _, k := range list {
		switch {
		case k.Name == q || k.Long == q:
			exact = append(exact, k)
		case strings.HasPrefix(k.Name, q) || strings.HasPrefix(k.Long, q):
			prefix = append(prefix, k)
		case strings.Contains(k.Name, q) || strings.Contains(k.Long, q):
			rest = append(rest, k)
		}
	}
	return slices.Concat(exact, prefix, rest)
}
