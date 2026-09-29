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
// the spelled-out name, so searching finds "space" or "shift", and Desc says
// what the key does in plain words.
type Keycode struct {
	Code             uint16
	Name, Long, Desc string
}

// table holds the VIA protocol v12 numbering (QMK 0.19+), checked against
// QMK's data/constants/keycodes specs. 0x7820-0x782A use the RGB_* names
// because VIA-era firmware drives the RGB matrix with them; newer QMK calls
// them UG_* and moved matrix control to RM_*.
var table = slices.Concat(series(), []Keycode{
	{0x0028, "ENT", "ENTER", "Enter"},
	{0x0029, "ESC", "ESCAPE", "Esc"},
	{0x002A, "BSPC", "BACKSPACE", "Backspace"},
	{0x002B, "TAB", "", "Tab"},
	{0x002C, "SPC", "SPACE", "Spacebar"},
	{0x002D, "MINS", "MINUS", "-"},
	{0x002E, "EQL", "EQUAL", "="},
	{0x002F, "LBRC", "LEFT_BRACKET", "["},
	{0x0030, "RBRC", "RIGHT_BRACKET", "]"},
	{0x0031, "BSLS", "BACKSLASH", "Backslash \\ and |"},
	{0x0032, "NUHS", "NONUS_HASH", "ISO # key, left of Enter on ISO boards"},
	{0x0033, "SCLN", "SEMICOLON", ";"},
	{0x0034, "QUOT", "QUOTE", "'"},
	{0x0035, "GRV", "GRAVE", "Grave ` and tilde ~"},
	{0x0036, "COMM", "COMMA", ","},
	{0x0037, "DOT", "", "."},
	{0x0038, "SLSH", "SLASH", "/"},
	{0x0039, "CAPS", "CAPS_LOCK", "Caps Lock"},
	{0x0046, "PSCR", "PRINT_SCREEN", "Print Screen"},
	{0x0047, "SCRL", "SCROLL_LOCK", "Scroll Lock"},
	{0x0048, "BRK", "PAUSE", "Pause"},
	{0x0049, "INS", "INSERT", "Insert"},
	{0x004A, "HOME", "", "Home"},
	{0x004B, "PGUP", "PAGE_UP", "Page Up"},
	{0x004C, "DEL", "DELETE", "Delete"},
	{0x004D, "END", "", "End"},
	{0x004E, "PGDN", "PAGE_DOWN", "Page Down"},
	{0x004F, "RGHT", "RIGHT", "Right"},
	{0x0050, "LEFT", "", "Left"},
	{0x0051, "DOWN", "", "Down"},
	{0x0052, "UP", "", "Up"},
	{0x0053, "NUM", "NUM_LOCK", "Num Lock"},
	{0x0054, "PSLS", "KP_SLASH", "/"},
	{0x0055, "PAST", "KP_ASTERISK", "*"},
	{0x0056, "PMNS", "KP_MINUS", "-"},
	{0x0057, "PPLS", "KP_PLUS", "+"},
	{0x0058, "PENT", "KP_ENTER", "Enter"},
	{0x0059, "P1", "KP_1", "1"},
	{0x005A, "P2", "KP_2", "2"},
	{0x005B, "P3", "KP_3", "3"},
	{0x005C, "P4", "KP_4", "4"},
	{0x005D, "P5", "KP_5", "5"},
	{0x005E, "P6", "KP_6", "6"},
	{0x005F, "P7", "KP_7", "7"},
	{0x0060, "P8", "KP_8", "8"},
	{0x0061, "P9", "KP_9", "9"},
	{0x0062, "P0", "KP_0", "0"},
	{0x0063, "PDOT", "KP_DOT", "."},
	{0x0064, "NUBS", "NONUS_BACKSLASH", "ISO \\ key, right of left Shift on ISO boards"},
	{0x0065, "APP", "APPLICATION", "Menu key"},
	{0x0066, "KB_POWER", "", "Power (keyboard page, PWR is more common)"},
	{0x0067, "PEQL", "KP_EQUAL", "="},
	{0x0074, "EXEC", "EXECUTE", "Execute"},
	{0x0075, "HELP", "", "Help"},
	{0x0076, "MENU", "", "Menu"},
	{0x0077, "SLCT", "SELECT", "Select"},
	{0x0078, "STOP", "", "Stop"},
	{0x0079, "AGIN", "AGAIN", "Again"},
	{0x007A, "UNDO", "", "Undo"},
	{0x007B, "CUT", "", "Cut"},
	{0x007C, "COPY", "", "Copy"},
	{0x007D, "PSTE", "PASTE", "Paste"},
	{0x007E, "FIND", "", "Find"},
	{0x007F, "KB_MUTE", "", "Mute (keyboard page, MUTE is more common)"},
	{0x0080, "KB_VOLUME_UP", "", "Volume up (keyboard page, VOLU is more common)"},
	{0x0081, "KB_VOLUME_DOWN", "", "Volume down (keyboard page, VOLD is more common)"},
	{0x0082, "LCAP", "LOCKING_CAPS_LOCK", "Caps Lock"},
	{0x0083, "LNUM", "LOCKING_NUM_LOCK", "Num Lock"},
	{0x0084, "LSCR", "LOCKING_SCROLL_LOCK", "Scroll Lock"},
	{0x0085, "PCMM", "KP_COMMA", ","},
	{0x0086, "KP_EQUAL_AS400", "", "="},
	{0x0087, "INT1", "INTERNATIONAL_1", "INT 1"},
	{0x0088, "INT2", "INTERNATIONAL_2", "INT 2"},
	{0x0089, "INT3", "INTERNATIONAL_3", "INT 3"},
	{0x008A, "INT4", "INTERNATIONAL_4", "INT 4"},
	{0x008B, "INT5", "INTERNATIONAL_5", "INT 5"},
	{0x008C, "INT6", "INTERNATIONAL_6", "INT 6"},
	{0x008D, "INT7", "INTERNATIONAL_7", "INT 7"},
	{0x008E, "INT8", "INTERNATIONAL_8", "INT 8"},
	{0x008F, "INT9", "INTERNATIONAL_9", "INT 9"},
	{0x0090, "LNG1", "LANGUAGE_1", "LANG 1"},
	{0x0091, "LNG2", "LANGUAGE_2", "LANG 2"},
	{0x0092, "LNG3", "LANGUAGE_3", "LANG 3"},
	{0x0093, "LNG4", "LANGUAGE_4", "LANG 4"},
	{0x0094, "LNG5", "LANGUAGE_5", "LANG 5"},
	{0x0095, "LNG6", "LANGUAGE_6", "LANG 6"},
	{0x0096, "LNG7", "LANGUAGE_7", "LANG 7"},
	{0x0097, "LNG8", "LANGUAGE_8", "LANG 8"},
	{0x0098, "LNG9", "LANGUAGE_9", "LANG 9"},
	{0x0099, "ERAS", "ALTERNATE_ERASE", "Alternate Erase"},
	{0x009A, "SYRQ", "SYSTEM_REQUEST", "SysReq/Attention"},
	{0x009B, "CNCL", "CANCEL", "Cancel"},
	{0x009C, "CLR", "CLEAR", "Clear"},
	{0x009D, "PRIR", "PRIOR", "Prior"},
	{0x009E, "RETN", "RETURN", "Return"},
	{0x009F, "SEPR", "SEPARATOR", "Separator"},
	{0x00A0, "OUT", "", "Out"},
	{0x00A1, "OPER", "", "Oper"},
	{0x00A2, "CLAG", "CLEAR_AGAIN", "Clear/Again"},
	{0x00A3, "CRSL", "CRSEL", "CrSel/Props"},
	{0x00A4, "EXSL", "EXSEL", "ExSel"},
	{0x00A5, "PWR", "SYSTEM_POWER", "System power"},
	{0x00A6, "SLEP", "SYSTEM_SLEEP", "Sleep"},
	{0x00A7, "WAKE", "SYSTEM_WAKE", "Wake"},
	{0x00A8, "MUTE", "AUDIO_MUTE", "Mute"},
	{0x00A9, "VOLU", "AUDIO_VOL_UP", "Volume Up"},
	{0x00AA, "VOLD", "AUDIO_VOL_DOWN", "Volume Down"},
	{0x00AB, "MNXT", "MEDIA_NEXT_TRACK", "Next"},
	{0x00AC, "MPRV", "MEDIA_PREV_TRACK", "Previous"},
	{0x00AD, "MSTP", "MEDIA_STOP", "Stop"},
	{0x00AE, "MPLY", "MEDIA_PLAY_PAUSE", "Play/Pause"},
	{0x00AF, "MSEL", "MEDIA_SELECT", "Media"},
	{0x00B0, "EJCT", "MEDIA_EJECT", "Eject"},
	{0x00B1, "MAIL", "", "Mail"},
	{0x00B2, "CALC", "CALCULATOR", "Calculator"},
	{0x00B3, "MYCM", "MY_COMPUTER", "My Computer"},
	{0x00B4, "WSCH", "WWW_SEARCH", "Search"},
	{0x00B5, "WHOM", "WWW_HOME", "Home"},
	{0x00B6, "WBAK", "WWW_BACK", "Back"},
	{0x00B7, "WFWD", "WWW_FORWARD", "Forward"},
	{0x00B8, "WSTP", "WWW_STOP", "Stop"},
	{0x00B9, "WREF", "WWW_REFRESH", "Refresh"},
	{0x00BA, "WFAV", "WWW_FAVORITES", "Favorites"},
	{0x00BB, "MFFD", "MEDIA_FAST_FORWARD", "Fast Forward"},
	{0x00BC, "MRWD", "MEDIA_REWIND", "Rewind"},
	{0x00BD, "BRIU", "BRIGHTNESS_UP", "Brightness Up"},
	{0x00BE, "BRID", "BRIGHTNESS_DOWN", "Brightness Down"},
	{0x00BF, "CPNL", "CONTROL_PANEL", "Control Panel"},
	{0x00C0, "ASST", "ASSISTANT", "Assistant"},
	{0x00C1, "MCTL", "MISSION_CONTROL", "Mission Control"},
	{0x00C2, "LPAD", "LAUNCHPAD", "Launchpad"},
	{0x00CD, "MS_UP", "MOUSE_CURSOR_UP", "Mouse Up"},
	{0x00CE, "MS_DOWN", "MOUSE_CURSOR_DOWN", "Mouse Down"},
	{0x00CF, "MS_LEFT", "MOUSE_CURSOR_LEFT", "Mouse Left"},
	{0x00D0, "MS_RGHT", "MOUSE_CURSOR_RIGHT", "Mouse Right"},
	{0x00D1, "MS_BTN1", "MOUSE_BUTTON_1", "Mouse Button 1"},
	{0x00D2, "MS_BTN2", "MOUSE_BUTTON_2", "Mouse Button 2"},
	{0x00D3, "MS_BTN3", "MOUSE_BUTTON_3", "Mouse Button 3"},
	{0x00D4, "MS_BTN4", "MOUSE_BUTTON_4", "Mouse Button 4"},
	{0x00D5, "MS_BTN5", "MOUSE_BUTTON_5", "Mouse Button 5"},
	{0x00D6, "MS_BTN6", "MOUSE_BUTTON_6", "Mouse Button 6"},
	{0x00D7, "MS_BTN7", "MOUSE_BUTTON_7", "Mouse Button 7"},
	{0x00D8, "MS_BTN8", "MOUSE_BUTTON_8", "Mouse Button 8"},
	{0x00D9, "MS_WHLU", "MOUSE_WHEEL_UP", "Mouse Wheel Up"},
	{0x00DA, "MS_WHLD", "MOUSE_WHEEL_DOWN", "Mouse Wheel Down"},
	{0x00DB, "MS_WHLL", "MOUSE_WHEEL_LEFT", "Mouse Wheel Left"},
	{0x00DC, "MS_WHLR", "MOUSE_WHEEL_RIGHT", "Mouse Wheel Right"},
	{0x00DD, "MS_ACL0", "MOUSE_ACCELERATION_0", "Acceleration 0"},
	{0x00DE, "MS_ACL1", "MOUSE_ACCELERATION_1", "Acceleration 1"},
	{0x00DF, "MS_ACL2", "MOUSE_ACCELERATION_2", "Acceleration 2"},
	{0x00E0, "LCTL", "LEFT_CTRL", "Left Control"},
	{0x00E1, "LSFT", "LEFT_SHIFT", "Left Shift"},
	{0x00E2, "LALT", "LEFT_ALT", "Left Alt"},
	{0x00E3, "LGUI", "LEFT_GUI", "Left GUI"},
	{0x00E4, "RCTL", "RIGHT_CTRL", "Right Control"},
	{0x00E5, "RSFT", "RIGHT_SHIFT", "Right Shift"},
	{0x00E6, "RALT", "RIGHT_ALT", "Right Alt"},
	{0x00E7, "RGUI", "RIGHT_GUI", "Right GUI"},
	{0x7000, "CL_SWAP", "MAGIC_SWAP_CONTROL_CAPS_LOCK", "Swap LCtl⇄Caps"},
	{0x7001, "CL_NORM", "MAGIC_UNSWAP_CONTROL_CAPS_LOCK", "Unswap LCtl⇄Caps"},
	{0x7002, "CL_TOGG", "MAGIC_TOGGLE_CONTROL_CAPS_LOCK", "Toggle LCtl⇄Caps"},
	{0x7003, "CL_CAPS", "MAGIC_CAPS_LOCK_AS_CONTROL_OFF", "Caps≠LCtl"},
	{0x7004, "CL_CTRL", "MAGIC_CAPS_LOCK_AS_CONTROL_ON", "Caps=LCtl"},
	{0x7005, "AG_LSWP", "MAGIC_SWAP_LALT_LGUI", "Swap LAlt⇄LGUI"},
	{0x7006, "AG_LNRM", "MAGIC_UNSWAP_LALT_LGUI", "Unswap LAlt⇄LGUI"},
	{0x7007, "AG_RSWP", "MAGIC_SWAP_RALT_RGUI", "Swap RAlt⇄RGUI"},
	{0x7008, "AG_RNRM", "MAGIC_UNSWAP_RALT_RGUI", "Unswap RAlt⇄RGUI"},
	{0x7009, "GU_ON", "MAGIC_GUI_ON", "GUI On"},
	{0x700A, "GU_OFF", "MAGIC_GUI_OFF", "GUI Off"},
	{0x700B, "GU_TOGG", "MAGIC_TOGGLE_GUI", "Toggle the GUI/Windows key on and off"},
	{0x700C, "GE_SWAP", "MAGIC_SWAP_GRAVE_ESC", "Swap `⇄Esc"},
	{0x700D, "GE_NORM", "MAGIC_UNSWAP_GRAVE_ESC", "Unswap `⇄Esc"},
	{0x700E, "BS_SWAP", "MAGIC_SWAP_BACKSLASH_BACKSPACE", "Swap \\⇄Bspc"},
	{0x700F, "BS_NORM", "MAGIC_UNSWAP_BACKSLASH_BACKSPACE", "Unswap \\⇄Bspc"},
	{0x7010, "BS_TOGG", "MAGIC_TOGGLE_BACKSLASH_BACKSPACE", "Toggle \\⇄Bspc"},
	{0x7011, "NK_ON", "MAGIC_NKRO_ON", "NKRO On"},
	{0x7012, "NK_OFF", "MAGIC_NKRO_OFF", "NKRO Off"},
	{0x7013, "NK_TOGG", "MAGIC_TOGGLE_NKRO", "Toggle N-key rollover"},
	{0x7014, "AG_SWAP", "MAGIC_SWAP_ALT_GUI", "Swap Alt⇄GUI"},
	{0x7015, "AG_NORM", "MAGIC_UNSWAP_ALT_GUI", "Unswap Alt⇄GUI"},
	{0x7016, "AG_TOGG", "MAGIC_TOGGLE_ALT_GUI", "Toggle Alt⇄GUI"},
	{0x7017, "CG_LSWP", "MAGIC_SWAP_LCTL_LGUI", "Swap LCtl⇄LGUI"},
	{0x7018, "CG_LNRM", "MAGIC_UNSWAP_LCTL_LGUI", "Unswap LCtl⇄LGUI"},
	{0x7019, "CG_RSWP", "MAGIC_SWAP_RCTL_RGUI", "Swap RCtl⇄RGUI"},
	{0x701A, "CG_RNRM", "MAGIC_UNSWAP_RCTL_RGUI", "Unswap RCtl⇄RGUI"},
	{0x701B, "CG_SWAP", "MAGIC_SWAP_CTL_GUI", "Swap Ctl⇄GUI"},
	{0x701C, "CG_NORM", "MAGIC_UNSWAP_CTL_GUI", "Unswap Ctl⇄GUI"},
	{0x701D, "CG_TOGG", "MAGIC_TOGGLE_CTL_GUI", "Toggle Ctl⇄GUI"},
	{0x701E, "EH_LEFT", "MAGIC_EE_HANDS_LEFT", "EE Hands Left"},
	{0x701F, "EH_RGHT", "MAGIC_EE_HANDS_RIGHT", "EE Hands Right"},
	{0x7020, "EC_SWAP", "MAGIC_SWAP_ESCAPE_CAPS_LOCK", "Swap Esc⇄Caps"},
	{0x7021, "EC_NORM", "MAGIC_UNSWAP_ESCAPE_CAPS_LOCK", "Unswap Esc⇄Caps"},
	{0x7022, "EC_TOGG", "MAGIC_TOGGLE_ESCAPE_CAPS_LOCK", "Toggle Esc⇄Caps"},
	{0x7800, "BL_ON", "BACKLIGHT_ON", "Backlight On"},
	{0x7801, "BL_OFF", "BACKLIGHT_OFF", "Backlight Off"},
	{0x7802, "BL_TOGG", "BACKLIGHT_TOGGLE", "Toggle Backlight"},
	{0x7803, "BL_DOWN", "BACKLIGHT_DOWN", "Backlight Down"},
	{0x7804, "BL_UP", "BACKLIGHT_UP", "Backlight Up"},
	{0x7805, "BL_STEP", "BACKLIGHT_STEP", "Backlight Step"},
	{0x7806, "BL_BRTG", "BACKLIGHT_TOGGLE_BREATHING", "Toggle Breathing"},
	{0x7820, "RGB_TOG", "RGB_TOGGLE", "Lighting on/off"},
	{0x7821, "RGB_MOD", "RGB_MODE_NEXT", "Next lighting effect"},
	{0x7822, "RGB_RMOD", "RGB_MODE_PREVIOUS", "Previous lighting effect"},
	{0x7823, "RGB_HUI", "RGB_HUE_UP", "Lighting hue up"},
	{0x7824, "RGB_HUD", "RGB_HUE_DOWN", "Lighting hue down"},
	{0x7825, "RGB_SAI", "RGB_SATURATION_UP", "Lighting saturation up"},
	{0x7826, "RGB_SAD", "RGB_SATURATION_DOWN", "Lighting saturation down"},
	{0x7827, "RGB_VAI", "RGB_BRIGHTNESS_UP", "Lighting brightness up"},
	{0x7828, "RGB_VAD", "RGB_BRIGHTNESS_DOWN", "Lighting brightness down"},
	{0x7829, "RGB_SPI", "RGB_SPEED_UP", "Lighting effect speed up"},
	{0x782A, "RGB_SPD", "RGB_SPEED_DOWN", "Lighting effect speed down"},
	{0x7840, "RM_ON", "RGB_MATRIX_ON", "RGB Matrix On"},
	{0x7841, "RM_OFF", "RGB_MATRIX_OFF", "RGB Matrix Off"},
	{0x7842, "RM_TOGG", "RGB_MATRIX_TOGGLE", "Toggle RGB Matrix"},
	{0x7843, "RM_NEXT", "RGB_MATRIX_MODE_NEXT", "RGB Matrix Next"},
	{0x7844, "RM_PREV", "RGB_MATRIX_MODE_PREVIOUS", "RGB Matrix Previous"},
	{0x7845, "RM_HUEU", "RGB_MATRIX_HUE_UP", "RGB Matrix Hue Up"},
	{0x7846, "RM_HUED", "RGB_MATRIX_HUE_DOWN", "RGB Matrix Hue Down"},
	{0x7847, "RM_SATU", "RGB_MATRIX_SATURATION_UP", "RGB Matrix Saturation Up"},
	{0x7848, "RM_SATD", "RGB_MATRIX_SATURATION_DOWN", "RGB Matrix Saturation Down"},
	{0x7849, "RM_VALU", "RGB_MATRIX_VALUE_UP", "RGB Matrix Value Up"},
	{0x784A, "RM_VALD", "RGB_MATRIX_VALUE_DOWN", "RGB Matrix Value Down"},
	{0x784B, "RM_SPDU", "RGB_MATRIX_SPEED_UP", "RGB Matrix Speed Up"},
	{0x784C, "RM_SPDD", "RGB_MATRIX_SPEED_DOWN", "RGB Matrix Speed Down"},
	{0x7C00, "QK_BOOT", "BOOTLOADER", "Restart into the bootloader, for flashing firmware"},
	{0x7C01, "QK_RBT", "REBOOT", "Restart the keyboard"},
	{0x7C02, "DB_TOGG", "DEBUG_TOGGLE", "Toggle QMK debug output"},
	{0x7C03, "EE_CLR", "CLEAR_EEPROM", "Wipe the board's saved settings and keymap (EEPROM)"},
	{0x7C16, "QK_GESC", "GRAVE_ESCAPE", "Esc, or ` while Shift or GUI is held"},
	{0x7C73, "CW_TOGG", "CAPS_WORD_TOGGLE", "Caps Word: capitals until the end of the word"},
	{0x7C77, "TL_LOWR", "TRI_LAYER_LOWER", "Lower"},
	{0x7C78, "TL_UPPR", "TRI_LAYER_UPPER", "Upper"},
	{0x7C79, "QK_REP", "REPEAT_KEY", "Repeat the last key"},
	{0x7C7B, "QK_LLCK", "LAYER_LOCK", "Layer Lock: keep the current layer on"},
})

var names, descs = func() (map[uint16]string, map[uint16]string) {
	names, descs := make(map[uint16]string, len(table)), make(map[uint16]string, len(table))
	for _, k := range table {
		names[k.Code], descs[k.Code] = k.Name, k.Desc
	}
	return names, descs
}()

func series() []Keycode {
	list := []Keycode{
		{0x0000, "NO", "", "Does nothing"},
		{0x0001, "TRNS", "TRANSPARENT", "Transparent: uses the key from the next active layer below"},
	}
	for i := range 26 {
		letter := string(rune('A' + i))
		list = append(list, Keycode{0x04 + uint16(i), letter, "", "Letter " + letter})
	}
	for i := range 10 {
		digit := string("1234567890"[i])
		list = append(list, Keycode{0x1E + uint16(i), digit, "", "Number " + digit})
	}
	for i := range 24 {
		code := 0x3A + uint16(i)
		if i >= 12 {
			code = 0x68 + uint16(i-12)
		}
		list = append(list, Keycode{code, fmt.Sprintf("F%d", i+1), "", "Function key"})
	}
	return list
}

// Describe says what a keycode does, for keys in the table and the ones
// built from arguments like LT(1,A).
func Describe(code uint16) string {
	if d, ok := descs[code]; ok {
		return d
	}
	basic := func(c uint16) string { return Name(c&0xFF, nil) }
	mods := func(mask uint16) string { return strings.Join(modNames(mask), "+") }
	layer := code & 0x1F
	switch {
	case code >= 0x0100 && code < 0x2000 && code>>8&0xF != 0:
		return fmt.Sprintf("%s with %s held", basic(code), mods(code>>8))
	case code >= 0x2000 && code < 0x4000 && code>>8&0xF != 0:
		return fmt.Sprintf("%s when tapped, %s while held", basic(code), mods(code>>8))
	case code >= 0x4000 && code < 0x5000:
		return fmt.Sprintf("%s when tapped, layer %d while held", basic(code), code>>8&0xF)
	case code >= 0x5000 && code < 0x5200 && code&0xF != 0:
		return fmt.Sprintf("Layer %d with %s while held", code>>5&0xF, mods(code))
	case code >= 0x5200 && code < 0x5300:
		switch layerFns[(code-0x5200)>>5] {
		case "TO":
			return fmt.Sprintf("Switch to layer %d, turning the others off", layer)
		case "MO":
			return fmt.Sprintf("Layer %d while held", layer)
		case "DF":
			return fmt.Sprintf("Make layer %d the base layer until unplugged", layer)
		case "TG":
			return fmt.Sprintf("Turn layer %d on or off", layer)
		case "OSL":
			return fmt.Sprintf("Layer %d for the next key press only", layer)
		case "OSM":
			if code&0xF != 0 {
				return mods(code) + " for the next key press only"
			}
		case "TT":
			return fmt.Sprintf("Layer %d while held, tap it a few times to keep it on", layer)
		case "PDF":
			return fmt.Sprintf("Make layer %d the base layer and remember it", layer)
		}
	case code >= 0x5700 && code < 0x5800:
		return "Tap dance: does different things depending on how often it's tapped"
	case code >= 0x7700 && code < 0x7780:
		return fmt.Sprintf("Macro %d", code&0x7F)
	case code >= 0x7E00 && code < 0x7E40:
		return "Keyboard specific, set by the firmware"
	case code >= 0x7E40 && code < 0x8000:
		return "User keycode from the firmware's keymap"
	}
	return ""
}

// Forms explains how to read keycodes that take arguments.
var Forms = []Keycode{
	{Name: "▽", Desc: "Transparent (TRNS): uses the key from the next active layer below"},
	{Name: "MO(n)", Desc: "Layer n while held"},
	{Name: "TG(n)", Desc: "Turn layer n on or off"},
	{Name: "TO(n)", Desc: "Switch to layer n, turning the others off"},
	{Name: "TT(n)", Desc: "Layer n while held, tap it a few times to keep it on"},
	{Name: "OSL(n)", Desc: "Layer n for the next key press only"},
	{Name: "DF(n)", Desc: "Make layer n the base layer until unplugged"},
	{Name: "LT(n,kc)", Desc: "kc when tapped, layer n while held"},
	{Name: "MT(mods,kc)", Desc: "kc when tapped, the modifiers while held"},
	{Name: "LSFT(kc)", Desc: "kc with Shift held; also LCTL, LALT, LGUI and the right-hand RCTL, RSFT, RALT, RGUI"},
	{Name: "OSM(mods)", Desc: "The modifiers for the next key press only"},
	{Name: "LM(n,mods)", Desc: "Layer n with the modifiers while held"},
	{Name: "TD(n)", Desc: "Tap dance: does different things depending on how often it's tapped"},
	{Name: "MC_n", Desc: "Macro n"},
	{Name: "QK_KB_n", Desc: "Keyboard specific key; the definition may name it"},
	{Name: "0x1234", Desc: "A keycode without a known name, shown as its number"},
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
			code := base | uint16(layer)
			list = append(list, Keycode{code, Name(code, nil), "", Describe(code)})
		}
	}
	for i, name := range custom {
		code := 0x7E00 + uint16(i)
		list = append(list, Keycode{code, strings.ToUpper(name), "", Describe(code)})
	}
	return list
}
