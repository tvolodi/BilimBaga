package questions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// #514: two repository helpers with real behaviour: lowercasing a copy (the input is not changed) and sorting map
// keys, which keeps the translation list order stable across requests.

func TestLowercaseAll_ReturnsALowercasedCopy(t *testing.T) {
	in := []string{"Safety", "ÄÖ", "already"}
	out := lowercaseAll(in)
	assert.Equal(t, []string{"safety", "äö", "already"}, out)
	assert.Equal(t, []string{"Safety", "ÄÖ", "already"}, in, "the input must not be changed")
}

func TestLowercaseAll_EmptyInputGivesEmptyOutput(t *testing.T) {
	assert.Empty(t, lowercaseAll(nil))
}

func TestSortedKeys_ReturnsKeysInOrder(t *testing.T) {
	m := map[string]TranslationDetail{"ru": {}, "en": {}, "kk": {}}
	assert.Equal(t, []string{"en", "kk", "ru"}, sortedKeys(m))
}

func TestSortedKeys_EmptyMapGivesNoKeys(t *testing.T) {
	assert.Empty(t, sortedKeys(map[string]TranslationDetail{}))
}
