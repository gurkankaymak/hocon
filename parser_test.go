package hocon

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestParseResource(t *testing.T) {
	t.Run("return error if there is an error in the os.Open(path) method", func(t *testing.T) {
		got, err := ParseResource("nonExistPath")
		expectedError := fmt.Errorf("could not parse resource: open nonExistPath: no such file or directory")
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("parse and return a pointer to the config if there is no error", func(t *testing.T) {
		got, err := ParseResource("testdata/array.conf")
		assertNoError(t, err)
		assertConfigEquals(t, got, "[1,2,3]")
	})
}

func TestParse(t *testing.T) {
	t.Run("try to parse as config array if the input starts with '[' and return the error from extractConfigArray if any", func(t *testing.T) {
		parser := newParser(strings.NewReader("[5}"))
		expectedError := invalidConfigArrayError("parenthesis do not match", 1, 3)
		got, err := parser.parse()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("parse as config array if the input starts with '['", func(t *testing.T) {
		testConfig := "[5]"
		parser := newParser(strings.NewReader(testConfig))
		got, err := parser.parse()
		assertNoError(t, err)
		assertConfigEquals(t, got, testConfig)
	})

	t.Run("return the same error if any error occurs in the extractConfigObject method", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:5]"))
		expectedError := invalidConfigObjectError("parenthesis do not match", 1, 5)
		got, err := parser.parse()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return an invalidConfigObject error if the EOF is not reached after extractConfigObject method returns", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:{b:1}bb"))
		expectedError := invalidConfigObjectError("invalid token bb", 1, 8)
		got, err := parser.parse()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the same error if any error occurs in the resolveSubstitution method", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b}"))
		expectedError := fmt.Errorf("could not resolve substitution: ${b} to a value")
		got, err := parser.parse()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("parse as config object if the input does not start with '['", func(t *testing.T) {
		testConfig := "{a:42}"
		parser := newParser(strings.NewReader(testConfig))
		got, err := parser.parse()
		assertNoError(t, err)
		assertConfigEquals(t, got, testConfig)
	})

	// ###############################################################
	// ###############################################################
	t.Run("parse simple object", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a:"b"}`))
		config, err := parser.parse()
		assertNoError(t, err)
		assertConfigEquals(t, config, "{a:b}")
	})

	t.Run("parse simple array", func(t *testing.T) {
		parser := newParser(strings.NewReader(`["a", "b"]`))
		config, err := parser.parse()
		assertNoError(t, err)
		assertConfigEquals(t, config, "[a,b]")
	})

	t.Run("parse nested object", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a: {c: "d"}}`))
		config, err := parser.parse()
		assertNoError(t, err)
		assertConfigEquals(t, config, "{a:{c:d}}")
	})

	t.Run("parse with the omitted root braces", func(t *testing.T) {
		parser := newParser(strings.NewReader("a=1"))
		config, err := parser.parse()
		assertNoError(t, err)
		assertConfigEquals(t, config, "{a:1}")
	})

	t.Run("parse the path key", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a.b:"c"}`))
		config, err := parser.parse()
		assertNoError(t, err)
		assertConfigEquals(t, config, "{a:{b:c}}")
	})
}

func TestExtractConfigObject(t *testing.T) {
	t.Run("extract empty config object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{}"))
		parser.scanner.Scan() // move scanner to the first token for the test case
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertConfigEquals(t, got, "{}")
	})

	t.Run("extract config object with the root braces omitted", func(t *testing.T) {
		parser := newParser(strings.NewReader("a=1"))
		parser.scanner.Scan()
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertConfigEquals(t, got, "{a:1}")
	})

	t.Run("extract simple config object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=1}"))
		parser.scanner.Scan()
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertConfigEquals(t, got, "{a:1}")
	})

	t.Run("return the error if any error occurs in parseIncludedResource method", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{include "testdata/array.conf"}`))
		parser.scanner.Scan()
		expectedErr := invalidValueError("included file cannot contain an array as the root value", 1, 10)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedErr)
		assertNil(t, got)
	})

	t.Run("merge the included config object with the existing", func(t *testing.T) {
		parser := newParser(strings.NewReader(`b:2 include "testdata/a.conf"`))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{"a": ConfigInt(1), "b": ConfigInt(2)})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	for forbiddenChar, _ := range forbiddenCharacters {
		t.Run(fmt.Sprintf("return error if the key contains the forbidden character: %q", forbiddenChar), func(t *testing.T) {
			if forbiddenChar != "`" && forbiddenChar != `"` && forbiddenChar != "}" && forbiddenChar != "#" { // TODO gk: add test cases for '`' and '"' characters
				parser := newParser(strings.NewReader(fmt.Sprintf("{%s:1}", forbiddenChar)))
				parser.scanner.Scan()
				expectedError := invalidKeyError(forbiddenChar, 1, 2)
				got, err := parser.extractConfigObject()
				assertError(t, err, expectedError)
				assertNil(t, got)
			}
		})
	}

	t.Run("return a leadingPeriodError if the key starts with a period '.'", func(t *testing.T) {
		parser := newParser(strings.NewReader("{.a:1}"))
		parser.scanner.Scan()
		expectedError := leadingPeriodError(1, 2)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return a adjacentPeriodsError if the key contains two adjacent periods", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a..b:1}"))
		parser.scanner.Scan()
		expectedError := adjacentPeriodsError(1, 4)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return a trailingPeriodError if the ends with a period", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.:1}"))
		parser.scanner.Scan()
		expectedError := trailingPeriodError(1, 3)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs while extracting the sub-config object (with object start token after key)", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a{.b:1}}"))
		parser.scanner.Scan()
		expectedError := leadingPeriodError(1, 4)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs while extracting the sub-config object (with path expression as key)", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.b.:1}"))
		parser.scanner.Scan()
		expectedError := trailingPeriodError(1, 5)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs in extractConfigValue method after equals separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=b}"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("unknown config value: %q", "b"), 1, 4)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return merged object if the current value (after equals separator) is object and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a={b:1},a={c:2}}"))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigObject(map[string]ConfigValue{
				"b": ConfigInt(1),
				"c": ConfigInt(2),
			}),
		})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after equals separator) is object and there is an existing non-object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=1,a={c:2}}"))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigObject(map[string]ConfigValue{
				"c": ConfigInt(2),
			}),
		})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after equals separator) is not object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a={b:1},a=2}"))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{"a": ConfigInt(2)})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the error if any error occurs in extractConfigValue method after colon separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:b}"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("unknown config value: %q", "b"), 1, 4)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return merged object if the current value (after colon separator) is object and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:{b:1},a:{c:2}}"))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigObject(map[string]ConfigValue{
				"b": ConfigInt(1),
				"c": ConfigInt(2),
			}),
		})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after colon separator) is object and there is an existing non-object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,a:{c:2}}"))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{
			"a": NewConfigObject(map[string]ConfigValue{
				"c": ConfigInt(2),
			}),
		})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after colon separator) is not object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:{b:1},a:2}"))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{"a": ConfigInt(2)})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the error if any error occurs in parsePlusEquals method", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,a+=2}"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", "1", "a"), 1, 10)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract config object with the += separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a+=1}"))
		parser.scanner.Scan()
		expected := NewConfigObject(map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{ConfigInt(1)})})
		got, err := parser.extractConfigObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return error if '=' does not exist after '+'", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a+1}"))
		parser.scanner.Scan()
		expectedError := invalidKeyError("+", 1, 3)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return invalidConfigObject error if parenthesis do not match", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1"))
		parser.scanner.Scan()
		expectedError := invalidConfigObjectError("parenthesis do not match", 1, 5)
		got, err := parser.extractConfigObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})
}

func TestMergeConfigObjects(t *testing.T) {
	t.Run("merge config objects", func(t *testing.T) {
		existingItems := map[string]ConfigValue{"b": ConfigInt(5)}
		configObject := NewConfigObject(map[string]ConfigValue{"c": ConfigInt(3)})
		expected := map[string]ConfigValue{"b": ConfigInt(5), "c": ConfigInt(3)}
		mergeConfigObjects(existingItems, configObject)
		assertDeepEqual(t, existingItems, expected)
	})

	t.Run("merge config objects recursively if both parameters contain the same key as of type *ConfigObject", func(t *testing.T) {
		existingItems := map[string]ConfigValue{"b": NewConfigObject(map[string]ConfigValue{"e": ConfigInt(5)})} // {b:{e:5}}
		configObject := NewConfigObject(map[string]ConfigValue{ // {b:{f:7}, c:3}
			"b": NewConfigObject(map[string]ConfigValue{"f": ConfigInt(7)}),
			"c": ConfigInt(3),
		})
		expected := map[string]ConfigValue{ // {b:{e:5, f:7}, c:3}
			"b": NewConfigObject(map[string]ConfigValue{
				"e": ConfigInt(5),
				"f": ConfigInt(7),
			}),
			"c": ConfigInt(3),
		}
		mergeConfigObjects(existingItems, configObject)
		assertDeepEqual(t, existingItems, expected)
	})

	t.Run("merge config objects recursively, config from the second parameter should override the first one if any of them are not of type *ConfigObject", func(t *testing.T) {
		existingItems := map[string]ConfigValue{ // {b:{e:5}, c:3}
			"b": NewConfigObject(map[string]ConfigValue{"e": ConfigInt(5)}),
			"c": ConfigInt(3),
		}
		configObject := NewConfigObject(map[string]ConfigValue{"b": ConfigInt(7)})
		expected := map[string]ConfigValue{"b": ConfigInt(7), "c": ConfigInt(3)} // {b:7, c:3}
		mergeConfigObjects(existingItems, configObject)
		assertDeepEqual(t, existingItems, expected)
	})
}

func TestResolveSubstitutions(t *testing.T) {
	t.Run("resolve valid substitution at the root level", func(t *testing.T) {
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": &Substitution{"a", false},
		})
		err := resolveSubstitutions(configObject)
		assertNoError(t, err)
	})

	t.Run("return an error for non-existing substitution path", func(t *testing.T) {
		substitution := &Substitution{"c", false}
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": substitution,
		})
		err := resolveSubstitutions(configObject)
		expectedError := errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		assertError(t, err, expectedError)
	})

	t.Run("ignore the optional substitution if it's path does not exist", func(t *testing.T) {
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": &Substitution{"c", true},
		})
		err := resolveSubstitutions(configObject)
		assertNoError(t, err)
	})

	t.Run("resolve valid substitution at the non-root level", func(t *testing.T) {
		subConfigObject := NewConfigObject(map[string]ConfigValue{"c": &Substitution{"a", false}})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": subConfigObject,
		})
		err := resolveSubstitutions(configObject, subConfigObject)
		assertNoError(t, err)
	})

	t.Run("resolve valid substitution inside an array", func(t *testing.T) {
		subConfigArray := NewConfigArray([]ConfigValue{&Substitution{"a", false}})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": subConfigArray,
		})
		err := resolveSubstitutions(configObject, subConfigArray)
		assertNoError(t, err)
	})

	t.Run("return error for non-existing substitution path inside an array", func(t *testing.T) {
		substitution := &Substitution{"c", false}
		subConfigArray := NewConfigArray([]ConfigValue{substitution})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": subConfigArray,
		})
		err := resolveSubstitutions(configObject, subConfigArray)
		expectedError := errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		assertError(t, err, expectedError)
	})

	t.Run("return error for non-existing substitution path inside an array", func(t *testing.T) {
		subConfigArray := NewConfigArray([]ConfigValue{&Substitution{"a", true}})
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": subConfigArray,
		})
		err := resolveSubstitutions(configObject, subConfigArray)
		assertNoError(t, err)
	})

	t.Run("return array if subConfig is not an object or array", func(t *testing.T) {
		subConfigInt := ConfigInt(42)
		configObject := NewConfigObject(map[string]ConfigValue{
			"a": ConfigInt(5),
			"b": subConfigInt,
		})
		err := resolveSubstitutions(configObject, subConfigInt)
		expectedError := invalidValueError("substitutions are only allowed in field values and array elements", 0, 0)
		assertError(t, err, expectedError)
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
		expected := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{ConfigInt(42)})}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, existingItems, expected)
	})

	t.Run("return the error received from extractConfigValue method if any, if the existingItems map does not contain a value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a += [42"))
		var currentRune int32
		for parser.scanner.TokenText() != "[" {
			currentRune = parser.scanner.Scan()
		}
		err := parser.parsePlusEqualsValue(map[string]ConfigValue{}, "a", currentRune)
		expectedError := invalidConfigArrayError("parenthesis do not match", 1, 7)
		assertError(t, err, expectedError)
	})

	t.Run("return an error if the existingItems map contains non-array value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: 1, a += 42"))
		var currentRune int32
		for parser.scanner.TokenText() != "42" { // move the scanner to the position for the test case
			currentRune = parser.scanner.Scan()
		}
		existingItems := map[string]ConfigValue{"a": ConfigInt(1)}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		expectedError := invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", "1", "a"), 1, 14)
		assertError(t, err, expectedError)
	})

	t.Run("return the error received from extractConfigValue method if any, if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += {42"))
		var currentRune int32
		for parser.scanner.TokenText() != "{" {
			currentRune = parser.scanner.Scan()
		}
		existingItems := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{ConfigInt(5)})}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		expectedError := invalidConfigObjectError("parenthesis do not match", 1, 15)
		assertError(t, err, expectedError)
	})

	t.Run("append the value if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += 42"))
		var currentRune int32
		for parser.scanner.TokenText() != "42" {
			currentRune = parser.scanner.Scan()
		}
		existingItems := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{ConfigInt(5)})}
		expected := map[string]ConfigValue{"a": NewConfigArray([]ConfigValue{ConfigInt(5), ConfigInt(42)})}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, existingItems, expected)
	})
}

func TestValidateIncludeValue(t *testing.T) {
	t.Run("return error if the include value starts with 'file' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include file[abc.conf]"))
		for ; parser.scanner.TokenText() != "file"; parser.scanner.Scan() {}
		expectedError := invalidValueError("missing opening parenthesis", 1, 13)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'file' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include file(abc.conf"))
		for ; parser.scanner.TokenText() != "file"; parser.scanner.Scan() {}
		expectedError := invalidValueError("missing closing parenthesis", 1, 17)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'classpath' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include classpath[abc.conf]"))
		for ; parser.scanner.TokenText() != "classpath"; parser.scanner.Scan() {}
		expectedError := invalidValueError("missing opening parenthesis", 1, 18)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'classpath' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include classpath(abc.conf"))
		for ; parser.scanner.TokenText() != "classpath"; parser.scanner.Scan() {}
		expectedError := invalidValueError("missing closing parenthesis", 1, 22)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value does not start with double quotes", func(t *testing.T) {
		parser := newParser(strings.NewReader("include abc.conf"))
		for ; parser.scanner.TokenText() != "abc"; parser.scanner.Scan() {}
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value does not end with double quotes", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "abc.conf`))
		for ; parser.scanner.TokenText() != `"abc.conf`; parser.scanner.Scan() {}
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value is just a double quotes", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "`))
		for ; parser.scanner.TokenText() != `"`; parser.scanner.Scan() {}
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})
	
	t.Run("return the path with quotes removed", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "abc.conf"`))
		for ; parser.scanner.TokenText() != `"abc.conf"`; parser.scanner.Scan() {}
		expected := &IncludeToken{path:"abc.conf", required:false}
		got, err := parser.validateIncludeValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the include token containing the path in file(...) with quotes removed and required as 'false'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include file("abc.conf")`))
		for ; parser.scanner.TokenText() != "file"; parser.scanner.Scan() {}
		got, err := parser.validateIncludeValue()
		expected := &IncludeToken{path:"abc.conf", required:false}
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the include token containing the path in classpath(...) with quotes removed and required as 'false'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include classpath("abc.conf")`))
		for ; parser.scanner.TokenText() != "classpath"; parser.scanner.Scan() {}
		expected := &IncludeToken{path:"abc.conf", required:false}
		got, err := parser.validateIncludeValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return error if the include value starts with 'required' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include required[abc.conf]"))
		for ; parser.scanner.TokenText() != "required"; parser.scanner.Scan() {}
		expectedError := invalidValueError("missing opening parenthesis", 1, 17)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'required' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include required(abc.conf"))
		for ; parser.scanner.TokenText() != "required"; parser.scanner.Scan() {}
		expectedError := invalidValueError("missing closing parenthesis", 1, 21)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the include token containing the path in required(file(...)) with quotes removed, and required as 'true'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include required(file("abc.conf"))`))
		for ; parser.scanner.TokenText() != "required"; parser.scanner.Scan() {}
		got, err := parser.validateIncludeValue()
		expected := &IncludeToken{path:"abc.conf", required:true}
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the include token containing the path in required(classpath(...)) with quotes removed and required as 'true'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include required(classpath("abc.conf"))`))
		for ; parser.scanner.TokenText() != "required"; parser.scanner.Scan() {}
		expected := &IncludeToken{path:"abc.conf", required:true}
		got, err := parser.validateIncludeValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})
}

func TestParseIncludedResource(t *testing.T) {
	t.Run("return the error from the validateIncludeValue method if it returns an error", func(t *testing.T) {
		parser := newParser(strings.NewReader("include abc.conf"))
		for ; parser.scanner.TokenText() != "abc"; parser.scanner.Scan() {}
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		configObject, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, configObject)
	})

	t.Run("return an empty object if the file does not exist and the include token is not required", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "nonExistFile.conf"`))
		for ; parser.scanner.TokenText() != `"nonExistFile.conf"`; parser.scanner.Scan() {}
		configObject, err := parser.parseIncludedResource()
		assertNil(t, err)
		assertConfigEquals(t, configObject, "{}")
	})

	t.Run("return an error if the file does not exist but the include token is required", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include required("nonExistFile.conf")`))
		for ; parser.scanner.TokenText() != "required"; parser.scanner.Scan() {}
		expectedError := fmt.Errorf("could not parse resource: %w", &os.PathError{Op: "open", Path: "nonExistFile.conf", Err: errors.New("no such file or directory")})
		configObject, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, configObject)
	})

	t.Run("return an error if the included file contains an array as the value", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/array.conf"`))
		for ; parser.scanner.TokenText() != `"testdata/array.conf"`; parser.scanner.Scan() {}
		expectedError := invalidValueError("included file cannot contain an array as the root value", 1, 9)
		configObject, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, configObject)
	})

	t.Run("parse the included resource and return the parsed object if there is no error", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/a.conf"`))
		for ; parser.scanner.TokenText() != `"testdata/a.conf"`; parser.scanner.Scan() {}
		configObject, err := parser.parseIncludedResource()
		assertNoError(t, err)
		assertConfigEquals(t, configObject, "{a:1}")
	})
}

func TestExtractConfigArray(t *testing.T) {
	t.Run("return invalidConfigArray error if the first token is not '['", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1}"))
		parser.scanner.Scan()
		expectedError := invalidConfigArrayError(fmt.Sprintf("%q is not an array start token", "{"), 1, 1)
		got, err := parser.extractConfigArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract the empty config array", func(t *testing.T) {
		parser := newParser(strings.NewReader("[]"))
		parser.scanner.Scan()
		got, err := parser.extractConfigArray()
		assertNoError(t, err)
		assertEquals(t, len(got.values), 0)
	})

	t.Run("return the error if any error occurs in extractConfigValue method", func(t *testing.T) {
		parser := newParser(strings.NewReader("[a]"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("unknown config value: %q", "a"), 1, 2)
		got, err := parser.extractConfigArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return invalidConfigArray if the closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1"))
		parser.scanner.Scan()
		expectedError := invalidConfigArrayError("parenthesis do not match", 1, 2)
		got, err := parser.extractConfigArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract the array", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1, 2]"))
		parser.scanner.Scan()
		expected := NewConfigArray([]ConfigValue{ConfigInt(1), ConfigInt(2)})
		got, err := parser.extractConfigArray()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})
}

func TestExtractConfigValue(t *testing.T) {
	t.Run("extract int value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "1"; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractConfigValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, ConfigInt(1))
	})

	t.Run("extract float32 value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1.5"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "1.5"; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractConfigValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, ConfigFloat32(1.5))
	})

	t.Run("extract string value", func(t *testing.T) {
		parser := newParser(strings.NewReader(`a:"b"`))
		var currentRune rune
		for ; parser.scanner.TokenText() != `"b"`; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractConfigValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, ConfigString("b"))
	})

	var booleanTestCases = []struct {
		input    string
		expected ConfigBoolean
	}{
		{"true", ConfigBoolean(true)},
		{"yes", ConfigBoolean(true)},
		{"on", ConfigBoolean(true)},
		{"false", ConfigBoolean(false)},
		{"no", ConfigBoolean(false)},
		{"off", ConfigBoolean(false)},
	}

	for _, tc := range booleanTestCases {
		t.Run(fmt.Sprintf("extract boolean value: %q", tc.input), func(t *testing.T) {
			parser := newParser(strings.NewReader(fmt.Sprintf("a:%s", tc.input)))
			var currentRune rune
			for ; parser.scanner.TokenText() != tc.input; currentRune = parser.scanner.Scan() {}
			got, err := parser.extractConfigValue(currentRune)
			assertNoError(t, err)
			assertDeepEqual(t, got, tc.expected)
		})
	}

	t.Run("extract config object value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:{b:1}"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "{"; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractConfigValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, NewConfigObject(map[string]ConfigValue{"b": ConfigInt(1)}))
	})

	t.Run("extract config array value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:[1]"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "["; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractConfigValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, NewConfigArray([]ConfigValue{ConfigInt(1)}))
	})

	t.Run("extract substitution value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b}"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "$"; currentRune = parser.scanner.Scan() {}
		expected := &Substitution{"b", false}
		got, err := parser.extractConfigValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return error for an unknown config value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:bb"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "bb"; currentRune = parser.scanner.Scan() {}
		expectedError := invalidValueError(fmt.Sprintf("unknown config value: %q", "bb"), 1, 3)
		got, err := parser.extractConfigValue(currentRune)
		assertError(t, err, expectedError)
		assertNil(t, got)
	})
}

func TestExtractSubstitution(t *testing.T) {
	t.Run("return invalidSubstitutionError if the path expression is empty", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := invalidSubstitutionError("path expression cannot be empty", 1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return leadingPeriodError if the path expression starts with a period '.' ", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${.a}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := leadingPeriodError(1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return adjacentPeriodsError if the substitution path contains two adjacent periods", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b..c}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := adjacentPeriodsError(1,7)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return invalidSubstitutionError if the closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := invalidSubstitutionError("missing closing parenthesis", 1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return trailingPeriodError if the path expression starts with a period '.' ", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${a.}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expectedError := trailingPeriodError(1, 6)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("parse and return a pointer to the substitution", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b.c}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expected := &Substitution{path: "b.c", optional: false}
		substitution, err := parser.extractSubstitution()
		assertNoError(t, err)
		assertDeepEqual(t, substitution, expected)
	})

	t.Run("parse and return a pointer to the optional substitution", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${?b.c}"))
		for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
		expected := &Substitution{path: "b.c", optional: true}
		substitution, err := parser.extractSubstitution()
		assertNoError(t, err)
		assertDeepEqual(t, substitution, expected)
	})

	for forbiddenChar, _ := range forbiddenCharacters {
		t.Run(fmt.Sprintf("return error for the forbidden character: %q", forbiddenChar), func(t *testing.T) {
			if forbiddenChar != "`" && forbiddenChar != `"` && forbiddenChar != "}" && forbiddenChar != "#" { // TODO gk: add test cases for '`' and '"' characters
				parser := newParser(strings.NewReader(fmt.Sprintf("a:${b%s}", forbiddenChar)))
				for ; parser.scanner.TokenText() != "$"; parser.scanner.Scan() {}
				expectedError := invalidKeyError(forbiddenChar, 1, 6)
				substitution, err := parser.extractSubstitution()
				assertError(t, err, expectedError)
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
			assertEquals(t, got, tc.expected)
		})
	}
}