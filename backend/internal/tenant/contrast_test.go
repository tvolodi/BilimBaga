package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContrastRatio_KnownValues(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{name: "black on white", a: "#000000", b: "#ffffff", want: 21},
		{name: "white on white", a: "#ffffff", b: "#ffffff", want: 1},
		{name: "shorthand equals long form", a: "#fff", b: "#ffffff", want: 1},
		// Classic AA boundary: #767676 is the lightest neutral grey that reaches 4.5:1 on white.
		{name: "AA boundary grey", a: "#767676", b: "#ffffff", want: 4.54},
		{name: "just below boundary", a: "#777777", b: "#ffffff", want: 4.48},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ContrastRatio(tc.a, tc.b)
			require.NoError(t, err)
			assert.InDelta(t, tc.want, got, 0.01)
		})
	}
}

func TestContrastRatio_IsSymmetric(t *testing.T) {
	ab, err := ContrastRatio("#0ea5e9", "#ffffff")
	require.NoError(t, err)
	ba, err := ContrastRatio("#ffffff", "#0ea5e9")
	require.NoError(t, err)
	assert.InDelta(t, ab, ba, 1e-12)
}

func TestContrastRatio_DesignSystemPrimaryPasses(t *testing.T) {
	got, err := ContrastRatio("#2E6DB4", "#ffffff")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, got, minPrimaryContrast)
}

func TestContrastRatio_RejectsMalformedColours(t *testing.T) {
	for _, bad := range []string{"", "#", "0ea5e9", "#0ea5e", "#0ea5e9ff", "#gggggg", "red", " #ffffff"} {
		t.Run(bad, func(t *testing.T) {
			_, err := ContrastRatio(bad, "#ffffff")
			assert.Error(t, err)
		})
	}
}
