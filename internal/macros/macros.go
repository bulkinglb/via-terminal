// Package macros converts the board's macro buffer to text and back, in
// VIA's syntax: text is typed as-is, {A} taps a key, {LCTL,C} presses keys
// together, {+LSFT} and {-LSFT} hold and release one, {100} waits 100 ms.
// A backslash escapes a brace or backslash; \n and \t type Enter and Tab.
package macros

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bulkinglb/via-terminal/internal/keycodes"
)

// The buffer holds the macros one after another, each ending in a zero byte.
// A prefix byte starts a key action; any other byte is a character to type.
const (
	prefix   = 0x01
	tap      = 0x01
	down     = 0x02
	up       = 0x03
	delay    = 0x04
	delayEnd = '|'
)

type action struct {
	kind byte // tap, down, up, delay, or 0 for text
	code byte
	ms   int
	text string
}

// Decode splits a macro buffer into count macros as text.
func Decode(buf []byte, count int) ([]string, error) {
	var macros []string
	var acts []action
	for i := 0; i < len(buf) && len(macros) < count; i++ {
		switch c := buf[i]; c {
		case 0:
			macros = append(macros, format(acts))
			acts = nil
		case prefix:
			if i+2 >= len(buf) {
				return nil, fmt.Errorf("macro %d ends inside a key action", len(macros))
			}
			i++
			switch kind := buf[i]; kind {
			case tap, down, up:
				i++
				acts = append(acts, action{kind: kind, code: buf[i]})
			case delay:
				end := strings.IndexByte(string(buf[i+1:]), delayEnd)
				ms, err := strconv.Atoi(string(buf[i+1 : i+1+max(end, 0)]))
				if end < 0 || err != nil {
					return nil, fmt.Errorf("macro %d has a broken delay", len(macros))
				}
				acts = append(acts, action{kind: delay, ms: ms})
				i += end + 1
			default:
				return nil, fmt.Errorf("macro %d has unknown key action %d", len(macros), kind)
			}
		default:
			if n := len(acts); n > 0 && acts[n-1].kind == 0 {
				acts[n-1].text += string(c)
			} else {
				acts = append(acts, action{text: string(c)})
			}
		}
	}
	for len(macros) < count {
		macros = append(macros, "")
	}
	return macros, nil
}

// format writes actions as text, folding presses that are released in
// reverse order right after into one {A,B} block, as VIA writes them.
func format(acts []action) string {
	var b strings.Builder
	name := func(code byte) string { return keycodes.Name(uint16(code), nil) }
	for i := 0; i < len(acts); i++ {
		a := acts[i]
		switch a.kind {
		case 0:
			b.WriteString(escape(a.text))
		case tap:
			b.WriteString("{" + name(a.code) + "}")
		case delay:
			fmt.Fprintf(&b, "{%d}", a.ms)
		case down, up:
			n := 0
			for i+n < len(acts) && acts[i+n].kind == down {
				n++
			}
			if chord(acts[i:], n) {
				var names []string
				for _, d := range acts[i : i+n] {
					names = append(names, name(d.code))
				}
				b.WriteString("{" + strings.Join(names, ",") + "}")
				i += 2*n - 1
				continue
			}
			sign := map[byte]string{down: "+", up: "-"}[a.kind]
			b.WriteString("{" + sign + name(a.code) + "}")
		}
	}
	return b.String()
}

// chord reports whether acts starts with n presses followed by their
// releases in reverse order.
func chord(acts []action, n int) bool {
	if n == 0 || len(acts) < 2*n {
		return false
	}
	for j := range n {
		if acts[n+j].kind != up || acts[n+j].code != acts[n-1-j].code {
			return false
		}
	}
	return true
}

func escape(text string) string {
	var b strings.Builder
	for _, c := range []byte(text) {
		switch {
		case c == '{' || c == '\\':
			b.WriteString(`\` + string(c))
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\t':
			b.WriteString(`\t`)
		case c < 0x20 || c >= 0x7F:
			fmt.Fprintf(&b, `\x%02X`, c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// Encode turns macros written as text into buffer bytes.
func Encode(macros []string) ([]byte, error) {
	var out []byte
	for i, text := range macros {
		b, err := encode(text)
		if err != nil {
			return nil, fmt.Errorf("M%d: %w", i, err)
		}
		out = append(append(out, b...), 0)
	}
	return out, nil
}

func encode(s string) ([]byte, error) {
	var out []byte
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\\':
			i++
			if i >= len(s) {
				return nil, fmt.Errorf("a backslash needs something after it")
			}
			switch s[i] {
			case 'n':
				out = append(out, '\n')
			case 't':
				out = append(out, '\t')
			case 'x':
				n, err := strconv.ParseUint(s[i+1:min(i+3, len(s))], 16, 8)
				if err != nil || n == 0 || n == prefix {
					return nil, fmt.Errorf(`\x needs two hex digits for a character other than 00 and 01`)
				}
				out = append(out, byte(n))
				i += 2
			default:
				out = append(out, s[i])
			}
		case c == '{':
			end := strings.IndexByte(s[i:], '}')
			if end < 0 {
				return nil, fmt.Errorf("a { isn't closed; type \\{ for a literal brace")
			}
			b, err := block(strings.TrimSpace(s[i+1 : i+end]))
			if err != nil {
				return nil, err
			}
			out = append(out, b...)
			i += end
		case c >= 0x80:
			return nil, fmt.Errorf("only plain ASCII text can be typed, not %q", []rune(s[i:])[0])
		default:
			out = append(out, c)
		}
	}
	return out, nil
}

// block encodes the inside of {...}: a delay, a single press or release, or
// keys tapped together.
func block(inner string) ([]byte, error) {
	if inner == "" {
		return nil, fmt.Errorf("{} is empty; type \\{} for literal braces")
	}
	if ms, err := strconv.Atoi(inner); err == nil {
		if ms < 0 || ms > 9999 {
			return nil, fmt.Errorf("{%s}: delays go up to 9999 ms", inner)
		}
		return append(append([]byte{prefix, delay}, strconv.Itoa(ms)...), delayEnd), nil
	}
	if name, ok := strings.CutPrefix(inner, "+"); ok {
		code, err := basic(name)
		return []byte{prefix, down, code}, err
	}
	if name, ok := strings.CutPrefix(inner, "-"); ok {
		code, err := basic(name)
		return []byte{prefix, up, code}, err
	}
	var codes []byte
	for _, name := range strings.Split(inner, ",") {
		code, err := basic(name)
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	if len(codes) == 1 {
		return []byte{prefix, tap, codes[0]}, nil
	}
	var out []byte
	for _, code := range codes {
		out = append(out, prefix, down, code)
	}
	for i := len(codes) - 1; i >= 0; i-- {
		out = append(out, prefix, up, codes[i])
	}
	return out, nil
}

// basic parses a key for a macro. Macros only store one byte per key, and
// the firmware counts zero bytes to find where each macro starts, so NO
// can't be used either.
func basic(name string) (byte, error) {
	code, err := keycodes.Parse(name, nil)
	switch {
	case err != nil:
		return 0, err
	case code == 0 || code > 0xFF:
		return 0, fmt.Errorf("%s can't be used in a macro, only basic keys like A, ENT or LCTL", strings.TrimSpace(name))
	}
	return byte(code), nil
}
