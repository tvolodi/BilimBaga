package tenant

import (
	"fmt"
	"math"
	"strconv"
)

// minPrimaryContrast is the WCAG 2.x AA minimum contrast for normal text (FR-BB320 AC-1).
// A tenant primary colour must reach it against white, the page text colour on primary.
const minPrimaryContrast = 4.5

// ContrastRatio returns the WCAG 2.x contrast ratio between two colours given as
// #rgb or #rrggbb. The result is in [1, 21].
func ContrastRatio(a, b string) (float64, error) {
	ra, ga, ba, err := parseHexColor(a)
	if err != nil {
		return 0, err
	}
	rb, gb, bb, err := parseHexColor(b)
	if err != nil {
		return 0, err
	}
	la := relativeLuminance(ra, ga, ba)
	lb := relativeLuminance(rb, gb, bb)
	lighter, darker := math.Max(la, lb), math.Min(la, lb)
	return (lighter + 0.05) / (darker + 0.05), nil
}

// parseHexColor parses #rgb or #rrggbb (the leading '#' is required).
func parseHexColor(s string) (r, g, b uint8, err error) {
	if len(s) != 4 && len(s) != 7 || s[0] != '#' {
		return 0, 0, 0, fmt.Errorf("tenant.parseHexColor: %q is not a #rgb or #rrggbb colour", s)
	}
	digits := s[1:]
	if len(digits) == 3 {
		digits = string([]byte{digits[0], digits[0], digits[1], digits[1], digits[2], digits[2]})
	}
	v, perr := strconv.ParseUint(digits, 16, 32)
	if perr != nil {
		return 0, 0, 0, fmt.Errorf("tenant.parseHexColor: %q is not a hex colour: %w", s, perr)
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), nil
}

// relativeLuminance is the WCAG 2.x relative luminance of an sRGB colour.
func relativeLuminance(r, g, b uint8) float64 {
	lin := func(c uint8) float64 {
		s := float64(c) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}
