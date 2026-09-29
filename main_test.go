package main

import "testing"

func TestHasHIDID(t *testing.T) {
	uevent := "DRIVER=hid-generic\nHID_ID=0003:0000342D:0000E4C2\nHID_NAME=Hangsheng MonsGeek Keyboard\n"

	if !hasHIDID(uevent, 0x342D, 0xE4C2) {
		t.Error("expected match for MonsGeek vendor/product")
	}
	if hasHIDID(uevent, 0x1532, 0x0099) {
		t.Error("expected no match for a different vendor/product")
	}
	if hasHIDID("no HID_ID line here", 0x342D, 0xE4C2) {
		t.Error("expected no match when HID_ID is missing")
	}
}
