package hocon

import (
	"errors"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("parse simple object", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a:"b"}`))
		config, err := parser.parse()
		assertNoError(err, t)
		assertConfigEquals(config, "{a:b}", t)
	})

	t.Run("parse simple array", func(t *testing.T) {
		parser := newParser(strings.NewReader(`["a", "b"]`))
		config, err := parser.parse()
		assertNoError(err, t)
		assertConfigEquals(config, "[a,b]", t)
	})

	t.Run("parse nested object", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a: {c: "d"}}`))
		config, err := parser.parse()
		assertNoError(err, t)
		assertConfigEquals(config, "{a:{c:d}}", t)
	})

	t.Run("parse with the omitted root braces", func(t *testing.T) {
		parser := newParser(strings.NewReader("a=1"))
		config, err := parser.parse()
		assertNoError(err, t)
		assertConfigEquals(config, "{a:1}", t)
	})

	t.Run("parse the path key", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a.b:"c"}`))
		config, err := parser.parse()
		assertNoError(err, t)
		assertConfigEquals(config, "{a:{b:c}}", t)
	})
}

func TestMergeConfigObjects(t *testing.T) {
	t.Run("merge config objects", func(t *testing.T) {
		config1, _ := ParseString(`{b:5}`)
		obj1 := config1.root.(*ConfigObject)
		config2, _ := ParseString(`{c:3}`)
		obj2 := config2.root.(*ConfigObject)
		expected := `{b:5, c:3}`
		mergeConfigObjects(obj1.items, obj2)
		assertConfigEquals(obj1, expected, t)
	})

	t.Run("merge config objects recursively if both parameters contain the same key as of type *ConfigObject", func(t *testing.T) {
		config1, _ := ParseString(`{b:{e:5}}`)
		obj1 := config1.root.(*ConfigObject)
		config2, _ := ParseString(`{b:{f:7}, c:3}`)
		obj2 := config2.root.(*ConfigObject)
		expectedConfig, _ := ParseString(`{b:{e:5, f:7}, c:3}`)
		expected := expectedConfig.root.(*ConfigObject)
		mergeConfigObjects(obj1.items, obj2)
		assertDeepEqual(obj1, expected, t)
	})

	t.Run("merge config objects recursively, config from the second parameter should override the first one if any of them are not of type *ConfigObject", func(t *testing.T) {
		config1, _ := ParseString(`{b:{e:5}, c:3}`)
		obj1 := config1.root.(*ConfigObject)
		config2, _ := ParseString(`{b:7}`)
		obj2 := config2.root.(*ConfigObject)
		expectedConfig, _ := ParseString(`{b:7, c:3}`)
		expected := expectedConfig.root.(*ConfigObject)
		mergeConfigObjects(obj1.items, obj2)
		assertDeepEqual(obj1, expected, t)
	})
}

func TestResolveSubstitutions(t *testing.T) {
	t.Run("resolve valid substitution at the root level", func(t *testing.T) {
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": &Substitution{"a", false},
		})
		err := resolveSubstitutions(configObject)
		assertNoError(err, t)
	})

	t.Run("return an error for non-existing substitution path", func(t *testing.T) {
		substitution := &Substitution{"c", false}
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": substitution,
		})
		err := resolveSubstitutions(configObject)
		expectedError := errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		assertError(err, t, expectedError)
	})

	t.Run("ignore the optional substitution if it's path does not exist", func(t *testing.T) {
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": &Substitution{"c", true},
		})
		err := resolveSubstitutions(configObject)
		assertNoError(err, t)
	})

	t.Run("resolve valid substitution at the non-root level", func(t *testing.T) {
		subConfigObject := NewConfigObject(map[string]ConfigValue{"c": &Substitution{"a", false}})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": subConfigObject,
		})
		err := resolveSubstitutions(configObject, subConfigObject)
		assertNoError(err, t)
	})

	t.Run("resolve valid substitution inside an array", func(t *testing.T) {
		subConfigArray := NewConfigArray([]ConfigValue{&Substitution{"a", false}})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": subConfigArray,
		})
		err := resolveSubstitutions(configObject, subConfigArray)
		assertNoError(err, t)
	})

	t.Run("return error for non-existing substitution path inside an array", func(t *testing.T) {
		substitution := &Substitution{"c", false}
		subConfigArray := NewConfigArray([]ConfigValue{substitution})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": subConfigArray,
		})
		err := resolveSubstitutions(configObject, subConfigArray)
		expectedError := errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		assertError(err, t, expectedError)
	})

	t.Run("return error for non-existing substitution path inside an array", func(t *testing.T) {
		subConfigArray := NewConfigArray([]ConfigValue{&Substitution{"a", true}})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": subConfigArray,
		})
		err := resolveSubstitutions(configObject, subConfigArray)
		assertNoError(err, t)
	})

	t.Run("return array if subConfig is not an object or array", func(t *testing.T) {
		subConfigInt := NewConfigInt(42)
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigInt(5),
			"b": subConfigInt,
		})
		err := resolveSubstitutions(configObject, subConfigInt)
		expectedError := errors.New("invalid type for substitution, substitutions are only allowed in field values and array elements")
		assertError(err, t, expectedError)
	})
}
