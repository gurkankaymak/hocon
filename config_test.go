package hocon

import (
	"errors"
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

	t.Run("return the raw string values without adding quotes", func(t *testing.T) {
		conf, err := ParseString(`a = { b = "x y", c = 1 }`)
		assertNoError(t, err)
		assertDeepEqual(t, conf.GetStringMapString("a"), map[string]string{"b": "x y", "c": "1"})
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

	t.Run("return the raw strings without adding quotes", func(t *testing.T) {
		conf, err := ParseString(`a = ["x y", "http://z"]`)
		assertNoError(t, err)
		assertDeepEqual(t, conf.GetStringSlice("a"), []string{"x y", "http://z"})
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

	t.Run("return the quoted strings without adding extra quotes", func(t *testing.T) {
		conf, err := ParseString(`url = "https://example.com/path?q=1"`)
		assertNoError(t, err)
		assertEquals(t, conf.GetString("url"), "https://example.com/path?q=1")
	})

	t.Run("return the unquoted strings containing special characters without adding quotes", func(t *testing.T) {
		conf, err := ParseString("name = foo-bar")
		assertNoError(t, err)
		assertEquals(t, conf.GetString("name"), "foo-bar")
	})

	t.Run("keep the hocon quoting in the rendered configuration while returning the raw string", func(t *testing.T) {
		conf, err := ParseString(`greeting = "hello world"`)
		assertNoError(t, err)
		assertEquals(t, conf.GetString("greeting"), "hello world")
		assertEquals(t, conf.String(), `{greeting:"hello world"}`)
	})

	t.Run("return the flattened string of a string value concatenation", func(t *testing.T) {
		conf, err := ParseString(`greeting = "hello" "world"`)
		assertNoError(t, err)
		assertEquals(t, conf.GetString("greeting"), "hello world")
	})

	t.Run("return the flattened string of a concatenation with substitutions", func(t *testing.T) {
		conf, err := ParseString("host = localhost\nurl = \"https://\"${host}\"/api\"")
		assertNoError(t, err)
		assertEquals(t, conf.GetString("url"), "https://localhost/api")
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

func TestHasPath(t *testing.T) {
	config := &Config{Object{"a": Int(1), "b": Object{"c": String("d")}}}

	t.Run("return true if a value exists at the given path", func(t *testing.T) {
		assertEquals(t, config.HasPath("b.c"), true)
	})

	t.Run("return false if no value exists at the given path", func(t *testing.T) {
		assertEquals(t, config.HasPath("b.e"), false)
	})

	t.Run("return false if the path traverses through a non-object value", func(t *testing.T) {
		assertEquals(t, config.HasPath("a.b"), false)
	})
}

func TestGetObjectE(t *testing.T) {
	config := &Config{Object{"a": Object{"b": Int(1)}, "c": Int(2)}}

	t.Run("get object", func(t *testing.T) {
		got, err := config.GetObjectE("a")
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"b": Int(1)})
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetObjectE("d")
		assertPathNotFoundError(t, err)
		assertNil(t, got)
	})

	t.Run("return an error if the value is not an object", func(t *testing.T) {
		got, err := config.GetObjectE("c")
		assertError(t, err, errors.New("cannot parse value: 2 to Object!"))
		assertNil(t, got)
	})
}

func TestGetConfigE(t *testing.T) {
	config := &Config{Object{"a": Object{"b": Int(1)}, "c": Int(2)}}

	t.Run("get config", func(t *testing.T) {
		got, err := config.GetConfigE("a")
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"b": Int(1)}})
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetConfigE("d")
		assertPathNotFoundError(t, err)
		assertNil(t, got)
	})

	t.Run("return an error if the value is not an object", func(t *testing.T) {
		got, err := config.GetConfigE("c")
		assertError(t, err, errors.New("cannot parse value: 2 to Object!"))
		assertNil(t, got)
	})
}

func TestGetStringMapE(t *testing.T) {
	config := &Config{Object{"a": Object{"b": Int(1)}, "c": Int(2)}}

	t.Run("get object as map[string]Value", func(t *testing.T) {
		got, err := config.GetStringMapE("a")
		assertNoError(t, err)
		assertDeepEqual(t, got, map[string]Value{"b": Int(1)})
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetStringMapE("d")
		assertPathNotFoundError(t, err)
		assertNil(t, got)
	})

	t.Run("return an error if the value is not an object", func(t *testing.T) {
		got, err := config.GetStringMapE("c")
		assertError(t, err, errors.New("cannot parse value: 2 to Object!"))
		assertNil(t, got)
	})
}

func TestGetStringMapStringE(t *testing.T) {
	config := &Config{Object{"a": Object{"b": String("x y"), "e": Int(1)}, "c": Int(2)}}

	t.Run("get object as map[string]string with the raw string values", func(t *testing.T) {
		got, err := config.GetStringMapStringE("a")
		assertNoError(t, err)
		assertDeepEqual(t, got, map[string]string{"b": "x y", "e": "1"})
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetStringMapStringE("d")
		assertPathNotFoundError(t, err)
		assertNil(t, got)
	})

	t.Run("return an error if the value is not an object", func(t *testing.T) {
		got, err := config.GetStringMapStringE("c")
		assertError(t, err, errors.New("cannot parse value: 2 to Object!"))
		assertNil(t, got)
	})
}

func TestGetArrayE(t *testing.T) {
	config := &Config{Object{"a": Array{Int(1), Int(2)}, "b": Int(3)}}

	t.Run("get array", func(t *testing.T) {
		got, err := config.GetArrayE("a")
		assertNoError(t, err)
		assertDeepEqual(t, got, Array{Int(1), Int(2)})
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetArrayE("c")
		assertPathNotFoundError(t, err)
		assertNil(t, got)
	})

	t.Run("return an error if the value is not an array", func(t *testing.T) {
		got, err := config.GetArrayE("b")
		assertError(t, err, errors.New("cannot parse value: 3 to Array!"))
		assertNil(t, got)
	})
}

func TestGetIntSliceE(t *testing.T) {
	config := &Config{Object{"a": Array{Int(1), Int(2)}, "b": Array{String("c"), Int(1)}}}

	t.Run("get array as int slice", func(t *testing.T) {
		got, err := config.GetIntSliceE("a")
		assertNoError(t, err)
		assertDeepEqual(t, got, []int{1, 2})
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetIntSliceE("e")
		assertPathNotFoundError(t, err)
		assertNil(t, got)
	})

	t.Run("return an error if the array contains a non-int element", func(t *testing.T) {
		got, err := config.GetIntSliceE("b")
		assertError(t, err, errors.New("cannot parse value: c to int!"))
		assertNil(t, got)
	})
}

func TestGetStringSliceE(t *testing.T) {
	config := &Config{Object{"a": Array{String("x y"), Int(1)}, "b": Int(2)}}

	t.Run("get array as string slice with the raw string values", func(t *testing.T) {
		got, err := config.GetStringSliceE("a")
		assertNoError(t, err)
		assertDeepEqual(t, got, []string{"x y", "1"})
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetStringSliceE("c")
		assertPathNotFoundError(t, err)
		assertNil(t, got)
	})

	t.Run("return an error if the value is not an array", func(t *testing.T) {
		got, err := config.GetStringSliceE("b")
		assertError(t, err, errors.New("cannot parse value: 2 to Array!"))
		assertNil(t, got)
	})
}

func TestGetStringE(t *testing.T) {
	config := &Config{Object{"a": String("b"), "c": Int(2)}}

	t.Run("get string", func(t *testing.T) {
		got, err := config.GetStringE("a")
		assertNoError(t, err)
		assertEquals(t, got, "b")
	})

	t.Run("convert to string and return the value if it is not a string", func(t *testing.T) {
		got, err := config.GetStringE("c")
		assertNoError(t, err)
		assertEquals(t, got, "2")
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetStringE("d")
		assertPathNotFoundError(t, err)
		assertEquals(t, got, "")
	})
}

func TestGetIntE(t *testing.T) {
	config := &Config{Object{"a": Int(1), "b": String("2"), "c": String("xyz"), "d": Boolean(true)}}

	t.Run("get int", func(t *testing.T) {
		got, err := config.GetIntE("a")
		assertNoError(t, err)
		assertEquals(t, got, 1)
	})

	t.Run("convert the numeric string value to int", func(t *testing.T) {
		got, err := config.GetIntE("b")
		assertNoError(t, err)
		assertEquals(t, got, 2)
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetIntE("e")
		assertPathNotFoundError(t, err)
		assertEquals(t, got, 0)
	})

	t.Run("return an error if the value is a non-numeric string", func(t *testing.T) {
		got, err := config.GetIntE("c")
		assertError(t, err, errors.New(`strconv.Atoi: parsing "xyz": invalid syntax`))
		assertEquals(t, got, 0)
	})

	t.Run("return an error if the value cannot be converted to int", func(t *testing.T) {
		got, err := config.GetIntE("d")
		assertError(t, err, errors.New("cannot parse value: true to int!"))
		assertEquals(t, got, 0)
	})
}

func TestGetFloat32E(t *testing.T) {
	config := &Config{Object{"a": Float32(1.5), "b": Float64(2.5), "c": String("3.5"), "d": Boolean(true)}}

	t.Run("get float32", func(t *testing.T) {
		got, err := config.GetFloat32E("a")
		assertNoError(t, err)
		assertEquals(t, got, float32(1.5))
	})

	t.Run("convert the float64 value to float32", func(t *testing.T) {
		got, err := config.GetFloat32E("b")
		assertNoError(t, err)
		assertEquals(t, got, float32(2.5))
	})

	t.Run("convert the numeric string value to float32", func(t *testing.T) {
		got, err := config.GetFloat32E("c")
		assertNoError(t, err)
		assertEquals(t, got, float32(3.5))
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetFloat32E("e")
		assertPathNotFoundError(t, err)
		assertEquals(t, got, float32(0))
	})

	t.Run("return an error if the value cannot be converted to float32", func(t *testing.T) {
		got, err := config.GetFloat32E("d")
		assertError(t, err, errors.New("cannot parse value: true to float32!"))
		assertEquals(t, got, float32(0))
	})
}

func TestGetFloat64E(t *testing.T) {
	config := &Config{Object{"a": Float64(1.5), "b": Float32(2.5), "c": String("3.5"), "d": Boolean(true)}}

	t.Run("get float64", func(t *testing.T) {
		got, err := config.GetFloat64E("a")
		assertNoError(t, err)
		assertEquals(t, got, 1.5)
	})

	t.Run("convert the float32 value to float64", func(t *testing.T) {
		got, err := config.GetFloat64E("b")
		assertNoError(t, err)
		assertEquals(t, got, 2.5)
	})

	t.Run("convert the numeric string value to float64", func(t *testing.T) {
		got, err := config.GetFloat64E("c")
		assertNoError(t, err)
		assertEquals(t, got, 3.5)
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetFloat64E("e")
		assertPathNotFoundError(t, err)
		assertEquals(t, got, 0.0)
	})

	t.Run("return an error if the value cannot be converted to float64", func(t *testing.T) {
		got, err := config.GetFloat64E("d")
		assertError(t, err, errors.New("cannot parse value: true to float64!"))
		assertEquals(t, got, 0.0)
	})
}

func TestGetBooleanE(t *testing.T) {
	config := &Config{Object{"a": Boolean(true), "b": String("yes"), "c": String("off"), "d": String("xyz"), "e": Int(1)}}

	t.Run("get boolean", func(t *testing.T) {
		got, err := config.GetBooleanE("a")
		assertNoError(t, err)
		assertEquals(t, got, true)
	})

	t.Run("convert the truthy string value to boolean", func(t *testing.T) {
		got, err := config.GetBooleanE("b")
		assertNoError(t, err)
		assertEquals(t, got, true)
	})

	t.Run("convert the falsy string value to boolean", func(t *testing.T) {
		got, err := config.GetBooleanE("c")
		assertNoError(t, err)
		assertEquals(t, got, false)
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetBooleanE("f")
		assertPathNotFoundError(t, err)
		assertEquals(t, got, false)
	})

	t.Run("return an error if the value is a non-boolean string", func(t *testing.T) {
		got, err := config.GetBooleanE("d")
		assertError(t, err, errors.New("cannot parse value: xyz to boolean!"))
		assertEquals(t, got, false)
	})

	t.Run("return an error if the value cannot be converted to boolean", func(t *testing.T) {
		got, err := config.GetBooleanE("e")
		assertError(t, err, errors.New("cannot parse value: 1 to boolean!"))
		assertEquals(t, got, false)
	})
}

func TestGetDurationE(t *testing.T) {
	config := &Config{Object{"a": Duration(5 * time.Second), "b": Int(1)}}

	t.Run("get duration", func(t *testing.T) {
		got, err := config.GetDurationE("a")
		assertNoError(t, err)
		assertEquals(t, got, 5*time.Second)
	})

	t.Run("return ErrPathNotFound for a non-existing path", func(t *testing.T) {
		got, err := config.GetDurationE("c")
		assertPathNotFoundError(t, err)
		assertEquals(t, got, time.Duration(0))
	})

	t.Run("return an error if the value is not a duration", func(t *testing.T) {
		got, err := config.GetDurationE("b")
		assertError(t, err, errors.New("cannot parse value: 1 to Duration!"))
		assertEquals(t, got, time.Duration(0))
	})
}

func TestWithFallback(t *testing.T) {
	config1 := &Config{Object{"a": String("aa"), "b": String("bb")}}
	config2 := &Config{Object{"a": String("aaa"), "c": String("cc")}}
	config3 := &Config{Array{Int(1), Int(2)}}
	config4 := &Config{Object{"a": String("aa")}}
	config5 := &Config{Object{"a": nil}}
	config6 := &Config{Object{"a": Object{"a": String("aa"), "b": String("bb")}}}

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

	t.Run("return the current value if new value is nil", func(t *testing.T) {
		got := config4.WithFallback(config5)
		assertDeepEqual(t, got, config4)
	})

	t.Run("rewrite the current value if old value is nil and new value is not nil", func(t *testing.T) {
		got := config5.WithFallback(config4)
		assertDeepEqual(t, got, config4)
	})

	t.Run("rewrite the current value if old value is nil and new value is not nil and Object", func(t *testing.T) {
		got := config5.WithFallback(config6)
		assertDeepEqual(t, got, config6)
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

	t.Run("return nil if the path traverses through a non-object value", func(t *testing.T) {
		object := Object{"a": Int(1)}
		got := object.find("a.b")
		assertNil(t, got)
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
		if got != "{a:\"!@#$%^&*()_+{}[];:',./<>?\"\\\", b:2}" && got != "{b:2, a:\"!@#$%^&*()_+{}[];:',./<>?\"\\\"}" {
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

	t.Run("return true if the concatenation contains a nil element and an Object", func(t *testing.T) {
		concatenation := concatenation{nil, Object{"a": String("aa")}}
		got := concatenation.containsObject()
		assertEquals(t, got, true)
	})

	t.Run("return false if the concatenation contains only nil and non-object elements", func(t *testing.T) {
		concatenation := concatenation{nil, String("b")}
		got := concatenation.containsObject()
		assertEquals(t, got, false)
	})
}
