# via-terminal

A terminal UI for VIA keyboards: remap keys, change lighting, back up your keymap. Linux only for now.

## Build

    go build

## Setup

Allow access to the keyboard, then replug it:

    echo 'KERNEL=="hidraw*", SUBSYSTEM=="hidraw", MODE="0660", TAG+="uaccess"' | sudo tee /etc/udev/rules.d/99-via.rules

## Use

    via-terminal                      # open the editor
    via-terminal export keymap.json   # save the keymap
    via-terminal import keymap.json   # load it back
    via-terminal reset                # restore the default keymap

Boards without a bundled definition need `--def board.json`.

MIT license.
