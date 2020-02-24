package hocon

import (
	"testing"
)

func TestGetConfigObject(t *testing.T) {
	config, _ := ParseString(`{a:{b:"c"}, d:[]}`)

	t.Run("get config object", func(t *testing.T) {
		got := config.GetConfigObject("a")
		assertConfigEquals(got, "{b:c}", t)
	})

	t.Run("return nil for a non-existing config object", func(t *testing.T) {
		got := config.GetConfigObject("e")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
	})

	t.Run("panic if non-object type is requested as ConfigObject", func(t *testing.T) {
		assertPanic(func() { config.GetConfigObject("d") }, t)
	})
}

func TestGetConfigArray(t *testing.T) {
	config, _ := ParseString(`{a: [1, 2], b: {c: "d"}}`)

	t.Run("get config array", func(t *testing.T) {
		got := config.GetConfigArray("a")
		assertConfigEquals(got, "[1,2]", t)
	})

	t.Run("return nil for a non-existing config array", func(t *testing.T) {
		got := config.GetConfigArray("e")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
	})

	t.Run("panic if non-array type is requested as ConfigArray", func(t *testing.T) {
		assertPanic(func() { config.GetConfigArray("b") }, t)
	})
}

func TestGetString(t *testing.T) {
	config, _ := ParseString(`{a: "b", c: 2}`)

	t.Run("get string", func(t *testing.T) {
		assertEquals(config.GetString("a"), "b", t)
	})

	t.Run("return zero value(empty string) for a non-existing config string", func(t *testing.T) {
		assertEquals(config.GetString("d"), "", t)
	})

	t.Run("convert to string and return the value if it is not a string", func(t *testing.T) {
		assertEquals(config.GetString("c"), "2", t)
	})
}

func TestGetInt(t *testing.T) {
	config, _ := ParseString(`{a: "aa", b: "3", c: 2, d: [5]}`)

	t.Run("get int", func(t *testing.T) {
		assertEquals(config.GetInt("c"), 2, t)
	})

	t.Run("return zero for a non-existing config int", func(t *testing.T) {
		assertEquals(config.GetInt("e"), 0, t)
	})

	t.Run("convert to int and return if the value is a string that can be converted to int", func(t *testing.T) {
		assertEquals(config.GetInt("b"), 3, t)
	})

	t.Run("panic if the value is a string that can not be converted to int", func(t *testing.T) {
		assertPanic(func() { config.GetInt("a") }, t)
	})

	t.Run("panic if the value is not an int or a string", func(t *testing.T) {
		assertPanic(func() { config.GetInt("d") }, t)
	})
}

func TestGetFloat32(t *testing.T) {
	config, _ := ParseString(`{a: "aa", b: "3.2", c: 2.4, d: [5]}`)

	t.Run("get float32", func(t *testing.T) {
		assertEquals(config.GetFloat32("c"), float32(2.4), t)
	})

	t.Run("return float32(0.0) for a non-existing config float32", func(t *testing.T) {
		assertEquals(config.GetFloat32("e"), float32(0.0), t)
	})

	t.Run("convert to float32 and return if the value is a string that can be converted to float32", func(t *testing.T) {
		assertEquals(config.GetFloat32("b"), float32(3.2), t)
	})

	t.Run("panic if the value is a string that can not be converted to float32", func(t *testing.T) {
		assertPanic(func() { config.GetFloat32("a") }, t)
	})

	t.Run("panic if the value is not a float32 or a string", func(t *testing.T) {
		assertPanic(func() { config.GetFloat32("d") }, t)
	})
}

func TestGetBoolean(t *testing.T) {
	config, _ := ParseString(`{a: true, b: yes, c: on, d: false, e: no, f: off, g: "true", h: "yes", i: "on", j: "false", k: "no", l: "off", m: "aa", n: [5]}`)

	t.Run("return zero value(false) for a non-existing boolean", func(t *testing.T) {
		assertEquals(config.GetBoolean("z"), false, t)
	})

	t.Run("panic if the value is a string that can not be converted to boolean", func(t *testing.T) {
		assertPanic(func() { config.GetBoolean("m") }, t)
	})

	t.Run("panic if the value is not a boolean or string", func(t *testing.T) {
		assertPanic(func() { config.GetBoolean("n") }, t)
	})

	var booleanTestCases = []struct {
		path     string
		expected bool
	}{
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", false},
		{"e", false},
		{"f", false},
		{"g", true},
		{"h", true},
		{"i", true},
		{"j", false},
		{"k", false},
		{"l", false},
	}

	for _, tc := range booleanTestCases {
		t.Run(tc.path, func(t *testing.T) {
			assertEquals(config.GetBoolean(tc.path), tc.expected, t)
		})
	}
}
