package hocon

import (
	"fmt"
	"testing"
	"time"
)

func TestGetRoot(t *testing.T) {
	root := Object{"a": Object{"b": String("c")}, "d": Array{}}
	config := &Config{root}

	t.Run("get root value", func(t *testing.T) {
		got := config.GetRoot()
		assertDeepEqual(t, got, root)
	})
}

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

func TestGetConfig(t *testing.T) {
	nestedConfig := &Config{Object{"b": String("c"), "d": Array{}}}
	config := &Config{Object{"a": nestedConfig.root}}

	t.Run("get nested config", func(t *testing.T) {
		got := config.GetConfig("a")
		assertDeepEqual(t, got, nestedConfig)
	})

	t.Run("return nil for non existing config", func(t *testing.T) {
		got := config.GetConfig("b")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
	})
}

func TestGetStringMap(t *testing.T) {
	object := Object{"b": Int(1)}
	config := &Config{Object{"a": object}}
	got := config.GetObject("a")
	assertDeepEqual(t, got, object)
}

func TestGetStringMapString(t *testing.T) {
	config := &Config{Object{"a": Object{"b": String("c"), "e": Int(1)}, "d": Array{}}}

	t.Run("get object as map[string]string", func(t *testing.T) {
		got := config.GetStringMapString("a")
		assertDeepEqual(t, got, map[string]string{"b": "c", "e": "1"})
	})

	t.Run("return nil for a non-existing string map", func(t *testing.T) {
		got := config.GetStringMapString("f")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
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

func TestGetIntSlice(t *testing.T) {
	config := &Config{Object{"a": Array{Int(1), Int(2)}, "b": Array{String("c"), Int(1)}}}

	t.Run("get array as int slice", func(t *testing.T) {
		got := config.GetIntSlice("a")
		assertDeepEqual(t, got, []int{1, 2})
	})

	t.Run("return nil for a non-existing int slice", func(t *testing.T) {
		got := config.GetIntSlice("e")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
	})

	t.Run("panic if there is a non-int element in the requested array", func(t *testing.T) {
		assertPanic(t, func() { config.GetIntSlice("b") })
	})
}

func TestGetStringSlice(t *testing.T) {
	config := &Config{Object{"a": Array{String("a"), String("b")}, "b": Array{Int(1), String("c")}}}

	t.Run("get array as string slice", func(t *testing.T) {
		got := config.GetStringSlice("a")
		assertDeepEqual(t, got, []string{"a", "b"})
	})

	t.Run("return nil for a non-existing string slice", func(t *testing.T) {
		got := config.GetStringSlice("e")
		if got != nil {
			t.Errorf("expected: nil, got: %v", got)
		}
	})

	t.Run("use string representations of non-string elements and return string slice", func(t *testing.T) {
		got := config.GetStringSlice("b")
		assertDeepEqual(t, got, []string{"1", "c"})
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
		assertEquals(t, got.String(), Duration(5*time.Second).String())
	})

	t.Run("return zero for non-existing duration", func(t *testing.T) {
		got := config.GetDuration("c")
		assertEquals(t, got.String(), Duration(0).String())
	})

	t.Run("panic if the value is not a duration", func(t *testing.T) {
		assertPanic(t, func() { config.GetDuration("b") })
	})
}

func TestWithFallback(t *testing.T) {
	config1 := &Config{Object{"a": String("aa"), "b": String("bb")}}
	config2 := &Config{Object{"a": String("aaa"), "c": String("cc")}}
	config3 := &Config{Array{Int(1), Int(2)}}

	t.Run("merge the given fallback config with the current config if the root of both of them are of type Object (for the same keys current config should override the fallback)", func(t *testing.T) {
		expected := &Config{Object{"a": String("aa"), "b": String("bb"), "c": String("cc")}}
		got := config1.WithFallback(config2)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the current config if the root of the given fallback config is not an Object", func(t *testing.T) {
		got := config1.WithFallback(config3)
		assertDeepEqual(t, got, config1)
	})

	t.Run("return the current config if the root of it is not an Object", func(t *testing.T) {
		got := config3.WithFallback(config1)
		assertDeepEqual(t, got, config3)
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

	t.Run("return the string of an object that contains a empty string", func(t *testing.T) {
		got := Object{"a": String("")}.String()
		assertEquals(t, got, "{a:\"\"}")
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

	t.Run("return the string of an object that contains a single element with the forbidden characters", func(t *testing.T) {
		got := Object{"a": String("!@#$%^&*()_+{}[];:',./<>?\"\\")}.String()
		assertEquals(t, got, "{a:\"!@#$%^&*()_+{}[];:',./<>?\"\\\"}")
	})

	t.Run("return the string of an object that contains multiple elements with the forbidden characters", func(t *testing.T) {
		got := Object{"a": String("!@#$%^&*()_+{}[];:',./<>?\"\\"), "b": Int(2)}.String()
		if got != "{a:\"!@#$%^&*()_+{}[];:',./<>?\"\\\", b:2}" && got != "{b:2, a:\"!@#$%^&*()_+{}[];:',./<>?\"}" {
			fail(t, got, "{a:\"!@#$%^&*()_+{}[];:',./<>?\"\\\", b:2}")
		}
	})
}

func TestArray_String(t *testing.T) {
	t.Run("return the string of an empty array", func(t *testing.T) {
		got := Array{}.String()
		assertEquals(t, got, "[]")
	})

	t.Run("return the string of an object that contains a empty string", func(t *testing.T) {
		got := Array{String("")}.String()
		assertEquals(t, got, "[\"\"]")
	})

	t.Run("return the string of an array that contains a single element", func(t *testing.T) {
		got := Array{Int(1)}.String()
		assertEquals(t, got, "[1]")
	})

	t.Run("return the string of an array that contains multiple elements", func(t *testing.T) {
		got := Array{Int(1), Int(2)}.String()
		assertEquals(t, got, "[1,2]")
	})

	t.Run("return the string of an array that contains a single elements with the ':' character", func(t *testing.T) {
		got := Array{String("!@#$%^&*()_+{}[];:',./<>?\"\\")}.String()
		assertEquals(t, got, "[\"!@#$%^&*()_+{}[];:',./<>?\"\\\"]")
	})

	t.Run("return the string of an array that contains multiple elements with the ':' character", func(t *testing.T) {
		got := Array{String("!@#$%^&*()_+"), String("{}[]|;':\",./<>?\\")}.String()
		assertEquals(t, got, "[\"!@#$%^&*()_+\",\"{}[]|;':\",./<>?\\\"]")
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

func TestValueWithAlternative_String(t *testing.T) {
	t.Run("return the string of valueWithAlternative", func(t *testing.T) {
		substitution := Substitution{path: "a", optional: false}
		withAlt := valueWithAlternative{value: String("value"), alternative: &substitution}
		got := withAlt.String()
		assertEquals(t, got, "(value | ${a})")
	})
}

func TestToConfig(t *testing.T) {
	object := Object{"a": Int(1)}
	got := object.ToConfig()
	assertDeepEqual(t, got.root, object)
}

func TestContainsObject(t *testing.T) {
	t.Run("return false if the concatenation does not contain an Object", func(t *testing.T) {
		concatenation := concatenation{String("a"), String("b")}
		got := concatenation.containsObject()
		assertEquals(t, got, false)
	})

	t.Run("return true if the concatenation contains an Object", func(t *testing.T) {
		concatenation := concatenation{Object{"a": String("aa")}, String("b")}
		got := concatenation.containsObject()
		assertEquals(t, got, true)
	})
}
