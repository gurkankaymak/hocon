package hocon

import (
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("parse simple object", func(t *testing.T) {
		configStr := `{a:"b"}`
		expected := "{a:b}"
		checkParseResult(configStr, expected, t)
	})

	t.Run("parse simple array", func(t *testing.T) {
		configStr := `["a", "b"]`
		expected := "[a,b]"
		checkParseResult(configStr, expected, t)
	})

	t.Run("parse nested object", func(t *testing.T) {
		configStr := `{a: {c: "d"}}`
		expected := "{a:{c:d}}"
		checkParseResult(configStr, expected, t)
	})

	t.Run("parse with the omitted root braces", func(t *testing.T) {
		configStr := `a=1`
		expected := "{a:1}"
		checkParseResult(configStr, expected, t)
	})

	t.Run("parse the path key", func(t *testing.T) {
		configStr := `{a.b:"c"}`
		expected := "{a:{b:c}}"
		checkParseResult(configStr, expected, t)
	})
}

func checkParseResult(configStr, expected string, t *testing.T) {
	t.Helper()
	got, _ := Parse(configStr) // TODO gk: refactor this to be able to check errors
	if got.String() != expected {
		t.Errorf("expected: %s, got: %s", expected, got)
	}
}
