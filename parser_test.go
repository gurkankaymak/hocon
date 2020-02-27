package hocon

import (
	"errors"
	"fmt"
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
		existingItems := map[string]ConfigValue{"b": NewConfigInt(5)}
		configObject := NewConfigObject(map[string]ConfigValue{"c": NewConfigInt(3)})
		expected := map[string]ConfigValue{"b": NewConfigInt(5), "c": NewConfigInt(3)}
		mergeConfigObjects(existingItems, configObject)
		assertDeepEqual(existingItems, expected, t)
	})

	t.Run("merge config objects recursively if both parameters contain the same key as of type *ConfigObject", func(t *testing.T) {
		existingItems := map[string]ConfigValue{"b": NewConfigObject(map[string]ConfigValue{"e": NewConfigInt(5)})} // {b:{e:5}}
		configObject := NewConfigObject(map[string]ConfigValue{ // {b:{f:7}, c:3}
			"b": NewConfigObject(map[string]ConfigValue{"f": NewConfigInt(7)}),
			"c": NewConfigInt(3),
		})
		expected := map[string]ConfigValue{ // {b:{e:5, f:7}, c:3}
			"b": NewConfigObject(map[string]ConfigValue{
				"e": NewConfigInt(5),
				"f": NewConfigInt(7),
			}),
			"c": NewConfigInt(3),
		}
		mergeConfigObjects(existingItems, configObject)
		assertDeepEqual(existingItems, expected, t)
	})

	t.Run("merge config objects recursively, config from the second parameter should override the first one if any of them are not of type *ConfigObject", func(t *testing.T) {
		existingItems := map[string]ConfigValue{ // {b:{e:5}, c:3}
			"b": NewConfigObject(map[string]ConfigValue{"e": NewConfigInt(5)}),
			"c": NewConfigInt(3),
		}
		configObject := NewConfigObject(map[string]ConfigValue{"b": NewConfigInt(7)})
		expected := map[string]ConfigValue{"b": NewConfigInt(7), "c": NewConfigInt(3)} // {b:7, c:3}
		mergeConfigObjects(existingItems, configObject)
		assertDeepEqual(existingItems, expected, t)
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

func TestParsePlusEqualsValue(t *testing.T) {
	t.Run("create an array that contains the value if the existingItems map does not contain a value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a += 42"))
		var currentRune int32
		for parser.scanner.TokenText() != "42" { // move the scanner to the position for the test case
			currentRune = parser.scanner.Scan()
		}
		existingItems := map[string]ConfigValue{}
		expected := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{NewConfigInt(42)})}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		assertNoError(err, t)
		assertDeepEqual(existingItems, expected, t)
	})

	t.Run("return the error received from extractConfigValue method if any, if the existingItems map does not contain a value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a += [42"))
		var currentRune int32
		for parser.scanner.TokenText() != "[" {
			currentRune = parser.scanner.Scan()
		}
		err := parser.parsePlusEqualsValue(map[string]ConfigValue{}, "a", currentRune)
		expectedError := invalidConfigArray("parenthesis do not match", 1, 7)
		assertError(err, t, expectedError)
	})

	t.Run("return an error if the existingItems map contains non-array value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: 1, a += 42"))
		var currentRune int32
		for parser.scanner.TokenText() != "42" { // move the scanner to the position for the test case
			currentRune = parser.scanner.Scan()
		}
		existingItems := map[string]ConfigValue{"a": NewConfigInt(1)}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		expectedError := fmt.Errorf("value of the key: %q is not an array", "a")
		assertError(err, t, expectedError)
	})

	t.Run("return the error received from extractConfigValue method if any, if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += {42"))
		var currentRune int32
		for parser.scanner.TokenText() != "{" {
			currentRune = parser.scanner.Scan()
		}
		existingItems := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{NewConfigInt(5)})}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		expectedError := invalidConfigObject("parenthesis do not match", 1, 15)
		assertError(err, t, expectedError)
	})

	t.Run("append the value if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += 42"))
		var currentRune int32
		for parser.scanner.TokenText() != "42" {
			currentRune = parser.scanner.Scan()
		}
		existingItems := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{NewConfigInt(5)})}
		expected := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{NewConfigInt(5), NewConfigInt(42)})}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		assertNoError(err, t)
		assertDeepEqual(existingItems, expected, t)
	})
}

func TestValidateIncludeValue(t *testing.T) {
	t.Run("return error if the include value starts with 'file' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include file[abc.conf]"))
		for ; parser.scanner.TokenText() != "file"; parser.scanner.Scan() {}
		expectedError := errors.New("invalid include value! missing opening parenthesis")
		path, err := parser.validateIncludeValue()
		assertError(err, t, expectedError)
		assertEquals(path, "", t)
	})

	t.Run("return error if the include value starts with 'file' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include file(abc.conf"))
		for ; parser.scanner.TokenText() != "file"; parser.scanner.Scan() {}
		expectedError := errors.New("invalid include value! missing closing parenthesis")
		path, err := parser.validateIncludeValue()
		assertError(err, t, expectedError)
		assertEquals(path, "", t)
	})

	t.Run("return error if the include value starts with 'classpath' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include classpath[abc.conf]"))
		for ; parser.scanner.TokenText() != "classpath"; parser.scanner.Scan() {}
		expectedError := errors.New("invalid include value! missing opening parenthesis")
		path, err := parser.validateIncludeValue()
		assertError(err, t, expectedError)
		assertEquals(path, "", t)
	})

	t.Run("return error if the include value starts with 'classpath' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include classpath(abc.conf"))
		for ; parser.scanner.TokenText() != "classpath"; parser.scanner.Scan() {}
		expectedError := errors.New("invalid include value! missing closing parenthesis")
		path, err := parser.validateIncludeValue()
		assertError(err, t, expectedError)
		assertEquals(path, "", t)
	})

	t.Run("return error if the include value is not a quoted string", func(t *testing.T) {
		parser := newParser(strings.NewReader("include abc.conf"))
		for ; parser.scanner.TokenText() != "abc"; parser.scanner.Scan() {}
		expectedError := errors.New(`invalid include value! expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)' `)
		path, err := parser.validateIncludeValue()
		assertError(err, t, expectedError)
		assertEquals(path, "", t)
	})
	
	t.Run("return the path with quotes removed", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "abc.conf"`))
		for ; parser.scanner.TokenText() != `"abc.conf"`; parser.scanner.Scan() {}
		path, err := parser.validateIncludeValue()
		assertNoError(err, t)
		assertEquals(path, "abc.conf", t)
	})

	t.Run("return the path in file(...) with quotes removed", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include file("abc.conf")`))
		for ; parser.scanner.TokenText() != "file"; parser.scanner.Scan() {}
		path, err := parser.validateIncludeValue()
		assertNoError(err, t)
		assertEquals(path, "abc.conf", t)
	})

	t.Run("return the path in classpath(...) with quotes removed", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include classpath("abc.conf")`))
		for ; parser.scanner.TokenText() != "classpath"; parser.scanner.Scan() {}
		path, err := parser.validateIncludeValue()
		assertNoError(err, t)
		assertEquals(path, "abc.conf", t)
	})
}

func TestParseIncludedResource(t *testing.T) {
	t.Run("return the error from the validateIncludeValue method if it returns an error", func(t *testing.T) {
		parser := newParser(strings.NewReader("include abc.conf"))
		for ; parser.scanner.TokenText() != "abc"; parser.scanner.Scan() {}
		expectedError := errors.New(`invalid include value! expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)' `)
		configObject, err := parser.parseIncludedResource()
		assertError(err, t, expectedError)
		assertNil(t, configObject)
	})

	t.Run("return an empty object if the file does not exist", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "nonExistFile.conf"`))
		for ; parser.scanner.TokenText() != `"nonExistFile.conf"`; parser.scanner.Scan() {}
		configObject, err := parser.parseIncludedResource()
		assertNil(t, err)
		assertConfigEquals(configObject, "{}", t)
	})

	t.Run("return an error if the included file contains an array as the value", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/array.conf"`))
		for ; parser.scanner.TokenText() != `"testdata/array.conf"`; parser.scanner.Scan() {}
		expectedError := errors.New("invalid included file! included file cannot contain an array as the root value")
		configObject, err := parser.parseIncludedResource()
		assertError(err, t, expectedError)
		assertNil(t, configObject)
	})

	t.Run("parse the included resource and return the parsed object if there is no error", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/a.conf"`))
		for ; parser.scanner.TokenText() != `"testdata/a.conf"`; parser.scanner.Scan() {}
		configObject, err := parser.parseIncludedResource()
		assertNoError(err, t)
		assertConfigEquals(configObject, "{a:1}", t)
	})
}

func TestExtractSubstitution(t *testing.T) {
	t.Run("return invalidSubstitutionError if the path expression is empty", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := invalidSubstitutionError("path expression cannot be empty", 1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(err, t, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return leadingPeriodError if the path expression starts with a period '.' ", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${.a}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := leadingPeriodError(1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(err, t, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return adjacentPeriodsError if the substitution path contains two adjacent periods", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b..c}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := adjacentPeriodsError(1,7)
		substitution, err := parser.extractSubstitution()
		assertError(err, t, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return invalidSubstitutionError if the closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := invalidSubstitutionError("missing closing parenthesis", 1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(err, t, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return trailingPeriodError if the path expression starts with a period '.' ", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${a.}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := trailingPeriodError(1, 6)
		substitution, err := parser.extractSubstitution()
		assertError(err, t, expectedError)
		assertNil(t, substitution)
	})

	t.Run("parse and return a pointer to the substitution", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b.c}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expected := &Substitution{path: "b.c", optional: false}
		substitution, err := parser.extractSubstitution()
		assertNoError(err, t)
		assertDeepEqual(substitution, expected, t)
	})

	t.Run("parse and return a pointer to the optional substitution", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${?b.c}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expected := &Substitution{path: "b.c", optional: true}
		substitution, err := parser.extractSubstitution()
		assertNoError(err, t)
		assertDeepEqual(substitution, expected, t)
	})

	for forbiddenChar, _ := range forbiddenCharacters {
		t.Run(fmt.Sprintf("return error for the forbidden character: %q", forbiddenChar), func(t *testing.T) {
			if forbiddenChar != "`" && forbiddenChar != `"` && forbiddenChar != "}" { // TODO gk: add test cases for '`' and '"' characters
				parser := newParser(strings.NewReader(fmt.Sprintf("a:${b%s}", forbiddenChar)))
				for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
				expectedError := fmt.Errorf("invalid key! %q is a forbidden character in keys", forbiddenChar)
				substitution, err := parser.extractSubstitution()
				assertError(err, t, expectedError)
				assertNil(t, substitution)
			}
		})
	}
}

func TestIsSubstitution(t *testing.T) {
	var testCases = []struct {
		token string
		peekedToken rune
		expected bool
	}{
		{"$", '{', true},
		{"a", '{', false},
		{"$", 'a', false},
		{"a", 'b', false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("return %v if the token is %q and the peekedToken is %q", tc.expected, tc.token, tc.peekedToken), func(t *testing.T) {
			got := isSubstitution(tc.token, tc.peekedToken)
			assertEquals(got, tc.expected, t)
		})
	}
}