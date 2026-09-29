package via

import "testing"

func TestParseHIDID(t *testing.T) {
	uevent := "DRIVER=hid-generic\nHID_ID=0003:0000342D:0000E4C2\nHID_NAME=Hangsheng MonsGeek Keyboard\n"

	if vid, pid, ok := parseHIDID(uevent); !ok || vid != 0x342D || pid != 0xE4C2 {
		t.Errorf("got %04X:%04X ok=%v, want 342D:E4C2", vid, pid, ok)
	}
	if _, _, ok := parseHIDID("no HID_ID line here"); ok {
		t.Error("expected no match when HID_ID is missing")
	}
}
