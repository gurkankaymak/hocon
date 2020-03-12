package hocon

import (
	"fmt"
	"testing"
	"time"
)

func TestGetObject(t *testing.T) {
	config := &Config{Object{"a": Object{"b": String("c")}, "d": Array{}}}

	t.Run("get object", func(t *testing.T) {
		got := config.GetObject("a")
		assertDeepEqual(t, got, Object{"b": String("c")})
	})

	t.Run("return nil for a non-existing object", func(t *testing.T) {
		got := config.GetObject("e")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
	})

	t.Run("panic if non-object type is requested as Object", func(t *testing.T) {
		assertPanic(t, func() { config.GetObject("d") })
	})
}

func TestGetArray(t *testing.T) {
	config := &Config{Object{"a": Array{Int(1), Int(2)}, "b": Object{"c": String("d")}}}

	t.Run("get array", func(t *testing.T) {
		got := config.GetArray("a")
		assertDeepEqual(t, got, Array{Int(1), Int(2)})
	})

	t.Run("return nil for a non-existing array", func(t *testing.T) {
		got := config.GetArray("e")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
	})

	t.Run("panic if non-array type is requested as Array", func(t *testing.T) {
		assertPanic(t, func() { config.GetArray("b") })
	})
}

func TestGetString(t *testing.T) {
	config := &Config{Object{"a": String("b"), "c": Int(2)}}

	t.Run("get string", func(t *testing.T) {
		assertEquals(t, config.GetString("a"), "b")
	})

	t.Run("return zero value(empty string) for a non-existing string", func(t *testing.T) {
		assertEquals(t, config.GetString("d"), "")
	})

	t.Run("convert to string and return the value if it is not a string", func(t *testing.T) {
		assertEquals(t, config.GetString("c"), "2")
	})
}

func TestGetInt(t *testing.T) {
	config := &Config{Object{"a": String("aa"), "b": String("3"), "c": Int(2), "d": Array{Int(5)}}}

	t.Run("get int", func(t *testing.T) {
		assertEquals(t, config.GetInt("c"), 2)
	})

	t.Run("return zero for a non-existing int", func(t *testing.T) {
		assertEquals(t, config.GetInt("e"), 0)
	})

	t.Run("convert to int and return if the value is a string that can be converted to int", func(t *testing.T) {
		assertEquals(t, config.GetInt("b"), 3)
	})

	t.Run("panic if the value is a string that can not be converted to int", func(t *testing.T) {
		assertPanic(t, func() { config.GetInt("a") })
	})

	t.Run("panic if the value is not an int or a string", func(t *testing.T) {
		assertPanic(t, func() { config.GetInt("d") })
	})
}

func TestGetFloat32(t *testing.T) {
	config := &Config{Object{"a": String("aa"), "b": String("3.2"), "c": Float32(2.4), "d": Array{Int(5)}, "e": Float64(2.5)}}

	t.Run("get float32", func(t *testing.T) {
		assertEquals(t, config.GetFloat32("c"), float32(2.4))
	})

	t.Run("convert to float32 and return if the value is float64", func(t *testing.T) {
		assertEquals(t, config.GetFloat32("e"), float32(2.5))
	})

	t.Run("return float32(0.0) for a non-existing float32", func(t *testing.T) {
		assertEquals(t, config.GetFloat32("z"), float32(0.0))
	})

	t.Run("convert to float32 and return if the value is a string that can be converted to float32", func(t *testing.T) {
		assertEquals(t, config.GetFloat32("b"), float32(3.2))
	})

	t.Run("panic if the value is a string that can not be converted to float32", func(t *testing.T) {
		assertPanic(t, func() { config.GetFloat32("a") })
	})

	t.Run("panic if the value is not a float32 or a string", func(t *testing.T) {
		assertPanic(t, func() { config.GetFloat32("d") })
	})
}

func TestGetFloat64(t *testing.T) {
	config := &Config{Object{"a": String("aa"), "b": String("3.2"), "c": Float32(2.4), "d": Array{Int(5)}, "e": Float64(2.5)}}

	t.Run("get float64", func(t *testing.T) {
		assertEquals(t, config.GetFloat64("e"), 2.5)
	})

	t.Run("convert to float64 and return if the value is float32", func(t *testing.T) {
		assertEquals(t, config.GetFloat64("c"), float64(float32(2.4)))
	})

	t.Run("return float64(0.0) for a non-existing float64", func(t *testing.T) {
		assertEquals(t, config.GetFloat64("z"), 0.0)
	})

	t.Run("convert to float64 and return if the value is a string that can be converted to float64", func(t *testing.T) {
		assertEquals(t, config.GetFloat64("b"), 3.2)
	})

	t.Run("panic if the value is a string that can not be converted to float64", func(t *testing.T) {
		assertPanic(t, func() { config.GetFloat64("a") })
	})

	t.Run("panic if the value is not a float64 or a string", func(t *testing.T) {
		assertPanic(t, func() { config.GetFloat64("d") })
	})
}

func TestGetBoolean(t *testing.T) {
	config := &Config{Object{
		"a": Boolean(true),
		"b": Boolean(false),
		"c": String("true"),
		"d": String("yes"),
		"e": String("on"),
		"f": String("false"),
		"g": String("no"),
		"h": String("off"),
		"i": String("aa"),
		"j": Array{Int(5)},
	}}

	t.Run("return zero value(false) for a non-existing boolean", func(t *testing.T) {
		assertEquals(t, config.GetBoolean("z"), false)
	})

	t.Run("panic if the value is a string that can not be converted to boolean", func(t *testing.T) {
		assertPanic(t, func() { config.GetBoolean("i") })
	})

	t.Run("panic if the value is not a boolean or string", func(t *testing.T) {
		assertPanic(t, func() { config.GetBoolean("j") })
	})

	var booleanTestCases = []struct {
		path     string
		expected bool
	}{
		{"a", true},
		{"b", false},
		{"c", true},
		{"d", true},
		{"e", true},
		{"f", false},
		{"g", false},
		{"h", false},
	}

	for _, tc := range booleanTestCases {
		t.Run(tc.path, func(t *testing.T) {
			assertEquals(t, config.GetBoolean(tc.path), tc.expected)
		})
	}
}

func TestGetDuration(t *testing.T) {
	config := &Config{Object{"a": Duration(5 * time.Second), "b": String("bb")}}

	t.Run("get Duration at the given path", func(t *testing.T) {
		got := config.GetDuration("a")
		assertEquals(t, got.String(), Duration(5 * time.Second).String())
	})

	t.Run("return zero for non-existing duration", func(t *testing.T) {
		got := config.GetDuration("c")
		assertEquals(t, got.String(), Duration(0).String())
	})

	t.Run("panic if the value is not a duration", func(t *testing.T) {
		assertPanic(t, func() { config.GetDuration("b") })
	})
}

func TestFind(t *testing.T) {
	t.Run("return nil if path does not contain any dot and there is no value with the given path", func(t *testing.T) {
		object := Object{"a": Int(1)}
		got := object.find("b")
		assertNil(t, got)
	})

	t.Run("find the value with the path that does not contain any dot", func(t *testing.T) {
		object := Object{"a": Int(1)}
		got := object.find("a")
		assertEquals(t, got, Int(1))
	})

	t.Run("return nil if path contains dot and there is no value with the sub-path", func(t *testing.T) {
		object := Object{"a": Object{"b": Int(1)}}
		got := object.find("c.b")
		assertNil(t, got)
	})

	t.Run("find the value with the path that contains dots", func(t *testing.T) {
		object := Object{"a": Object{"b": Int(1)}}
		got := object.find("a.b")
		assertEquals(t, got, Int(1))
	})
}

func TestObject_String(t *testing.T) {
	t.Run("return the string of an empty object", func(t *testing.T) {
		got := Object{}.String()
		assertEquals(t, got, "{}")
	})

	t.Run("return the string of an object that contains a single element", func(t *testing.T) {
		got := Object{"a": Int(1)}.String()
		assertEquals(t, got, "{a:1}")
	})

	t.Run("return the string of an object that contains multiple elements", func(t *testing.T) {
		got := Object{"a": Int(1), "b": Int(2)}.String()
		if got != "{a:1, b:2}" && got != "{b:2, a:1}" {
			fail(t, got, "{a:1, b:2}")
		}
	})
}

func TestArray_String(t *testing.T) {
	t.Run("return the string of an empty array", func(t *testing.T) {
		got := Array{}.String()
		assertEquals(t, got, "[]")
	})

	t.Run("return the string of an array that contains a single element", func(t *testing.T) {
		got := Array{Int(1)}.String()
		assertEquals(t, got, "[1]")
	})

	t.Run("return the string of an array that contains multiple elements", func(t *testing.T) {
		got := Array{Int(1), Int(2)}.String()
		assertEquals(t, got, "[1,2]")
	})
}

func TestGet(t *testing.T) {
	t.Run("return nil if the root of config is not an Object", func(t *testing.T) {
		config := &Config{Array{Int(1)}}
		got := config.Get("a")
		assertNil(t, got)
	})

	t.Run("find the value if the root of config is an object and a value exist with the given path", func(t *testing.T) {
		config := &Config{Object{"a": Int(1)}}
		got := config.Get("a")
		assertEquals(t, got, Int(1))
	})

	t.Run("return nil if the root of config is an object but value with the given path does not exist", func(t *testing.T) {
		config := &Config{Object{"a": Int(1)}}
		got := config.Get("b")
		assertNil(t, got)
	})
}

func TestNewBooleanFromString(t *testing.T) {
	var testCases = []struct {
		input    string
		expected Boolean
	}{
		{"true", Boolean(true)},
		{"yes", Boolean(true)},
		{"on", Boolean(true)},
		{"false", Boolean(false)},
		{"no", Boolean(false)},
		{"off", Boolean(false)},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("create the Boolean(%s) from the input string: %s", tc.expected, tc.input), func(t *testing.T) {
			got := newBooleanFromString(tc.input)
			assertEquals(t, got, tc.expected)
		})
	}

	t.Run("panic if the given string is not a boolean string", func(t *testing.T) {
		nonBooleanString := "nonBooleanString"
		assertPanic(t, func() { newBooleanFromString(nonBooleanString) }, fmt.Sprintf("cannot parse value: %s to Boolean!", nonBooleanString))
	})
}

func TestSubstitution_String(t *testing.T) {
	t.Run("return the string of required substitution", func(t *testing.T) {
		substitution := &Substitution{path: "a", optional: false}
		got := substitution.String()
		assertEquals(t, got, "${a}")
	})

	t.Run("return the string of optional substitution", func(t *testing.T) {
		substitution := &Substitution{path: "a", optional: true}
		got := substitution.String()
		assertEquals(t, got, "${?a}")
	})
}
