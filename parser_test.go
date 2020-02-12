package hocon

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("parse simple object", func(t *testing.T) {
		config, err := Parse(`{a:"b"}`)
		assertNoError(err, t)
		assertConfigEquals(config, "{a:b}", t)
	})

	t.Run("parse simple array", func(t *testing.T) {
		config, err := Parse(`["a", "b"]`)
		assertNoError(err, t)
		assertConfigEquals(config, "[a,b]", t)
	})

	t.Run("parse nested object", func(t *testing.T) {
		config, err := Parse(`{a: {c: "d"}}`)
		assertNoError(err, t)
		assertConfigEquals(config, "{a:{c:d}}", t)
	})

	t.Run("parse with the omitted root braces", func(t *testing.T) {
		config, err := Parse(`a=1`)
		assertNoError(err, t)
		assertConfigEquals(config, "{a:1}", t)
	})

	t.Run("parse the path key", func(t *testing.T) {
		config, err := Parse(`{a.b:"c"}`)
		assertNoError(err, t)
		assertConfigEquals(config, "{a:{b:c}}", t)
	})
}

func TestMergeConfigObjects(t *testing.T) {
	t.Run("merge config objects", func(t *testing.T) {
		config1, _ := Parse(`{b:5}`)
		obj1 := config1.root.(*ConfigObject)
		config2, _ := Parse(`{c:3}`)
		obj2 := config2.root.(*ConfigObject)
		expected := `{b:5, c:3}`
		got := mergeConfigObjects(obj1, obj2)
		assertConfigEquals(got, expected, t)
	})

	t.Run("merge config objects recursively if both parameters contain the same key as of type *ConfigObject", func(t *testing.T) {
		config1, _ := Parse(`{b:{e:5}}`)
		obj1 := config1.root.(*ConfigObject)
		config2, _ := Parse(`{b:{f:7}, c:3}`)
		obj2 := config2.root.(*ConfigObject)
		expectedConfig, _ := Parse(`{b:{e:5, f:7}, c:3}`)
		expected := expectedConfig.root.(*ConfigObject)
		got := mergeConfigObjects(obj1, obj2)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("expected: %s, got: %s", expected, got)
		}
	})

	t.Run("merge config objects recursively, config from the second parameter should override the first one if any of them are not of type *ConfigObject", func(t *testing.T) {
		config1, _ := Parse(`{b:{e:5}, c:3}`)
		obj1 := config1.root.(*ConfigObject)
		config2, _ := Parse(`{b:7}`)
		obj2 := config2.root.(*ConfigObject)
		expectedConfig, _ := Parse(`{b:7, c:3}`)
		expected := expectedConfig.root.(*ConfigObject)
		got := mergeConfigObjects(obj1, obj2)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("expected: %s, got: %s", expected, got)
		}
	})
}