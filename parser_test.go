package hocon

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseString(t *testing.T) {
	t.Run("parse the string and return a pointer to the Config", func(t *testing.T) {
		got, err := ParseString("{a:1}")
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": Int(1)}})
	})

	t.Run("return the error if any error occurs in the parse() method", func(t *testing.T) {
		got, err := ParseString("{.a:1}")
		assertError(t, err, leadingPeriodError(1, 2))
		assertNil(t, got)
	})
}

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
		assertDeepEqual(t, got, &Config{Array{Int(1), Int(2), Int(3)}})
	})
}

func TestParse(t *testing.T) {
	t.Run("try to parse as config array if the input starts with '[' and return the error from extractArray if any", func(t *testing.T) {
		parser := newParser(strings.NewReader("[5"))
		expectedError := invalidArrayError("parenthesis do not match", 1, 2)
		got, err := parser.parse()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("parse as config array if the input starts with '['", func(t *testing.T) {
		parser := newParser(strings.NewReader("[5]"))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Array{Int(5)}})
	})

	t.Run("return the same error if any error occurs in the extractObject method", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:5"))
		expectedError := invalidObjectError("parenthesis do not match", 1, 5)
		got, err := parser.parse()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return an invalidObjectError if the EOF is not reached after extractObject method returns", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:{b:1}bb"))
		expectedError := invalidObjectError("invalid token bb", 1, 8)
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

	t.Run("parse as object if the input does not start with '['", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:42}"))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": Int(42)}})
	})

	// ###############################################################
	// ###############################################################
	t.Run("parse simple object", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a:"b"}`))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": String("b")}})
	})

	t.Run("parse simple array", func(t *testing.T) {
		parser := newParser(strings.NewReader(`["a", "b"]`))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Array{String("a"), String("b")}})
	})

	t.Run("parse nested object", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a: {c: "d"}}`))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": Object{"c": String("d")}}})
	})

	t.Run("parse with the omitted root braces", func(t *testing.T) {
		parser := newParser(strings.NewReader("a=1"))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": Int(1)}})
	})

	t.Run("parse the path key", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a.b:"c"}`))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": Object{"b": String("c")}}})
	})

	t.Run("parse the path key that contains a hyphen", func(t *testing.T) {
		parser := newParser(strings.NewReader(`a.b-1: "c"`))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": Object{"b-1": String("c")}}})
	})

	t.Run("parse the nested object with a key containing a hyphen", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{a: {b-1: "c"}}`))
		got, err := parser.parse()
		assertNoError(t, err)
		assertDeepEqual(t, got, &Config{Object{"a": Object{"b-1": String("c")}}})
	})
}

func TestExtractObject(t *testing.T) {
	t.Run("extract empty object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{}"))
		parser.advance() // move scanner to the first token for the test case
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{})
	})

	t.Run("extract object with the root braces omitted", func(t *testing.T) {
		parser := newParser(strings.NewReader("a=1"))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1)})
	})

	t.Run("extract simple object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=1}"))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1)})
	})

	t.Run("extract nested object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.b:1,c:2}"))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Object{"b": Int(1)}, "c": Int(2)})
	})

	t.Run("skip the comments inside objects", func(t *testing.T) {
		config := `{
			# this is a comment
			# this is also a comment
			a: 1
		}
		`
		parser := newParser(strings.NewReader(config))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1)})
	})

	t.Run("return the error if any error occurs in parseIncludedResource method", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{include "testdata/array.conf"}`))
		parser.advance()
		expectedErr := invalidValueError("included file cannot contain an array as the root value", 1, 10)
		got, err := parser.extractObject()
		assertError(t, err, expectedErr)
		assertNil(t, got)
	})

	t.Run("merge the included object with the existing", func(t *testing.T) {
		parser := newParser(strings.NewReader(`b:2, include "testdata/a.conf"`))
		parser.advance()
		expected := Object{"a": Int(1), "b": Int(2)}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("parse correctly if the last line is a comment", func(t *testing.T) {
		config := `{
			a: 1
			# this is a comment
		}
		`
		parser := newParser(strings.NewReader(config))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1)})
	})

	for forbiddenChar := range forbiddenCharacters {
		t.Run(fmt.Sprintf("return error if the key contains the forbidden character: %q", forbiddenChar), func(t *testing.T) {
			if forbiddenChar != "`" && forbiddenChar != `"` && forbiddenChar != "}" && forbiddenChar != "#" {
				parser := newParser(strings.NewReader(fmt.Sprintf("{%s:1}", forbiddenChar)))
				parser.advance()
				expectedError := invalidKeyError(forbiddenChar, 1, 2)
				got, err := parser.extractObject()
				assertError(t, err, expectedError)
				assertNil(t, got)
			}
		})
	}

	t.Run("return a leadingPeriodError if the key starts with a period '.'", func(t *testing.T) {
		parser := newParser(strings.NewReader("{.a:1}"))
		parser.advance()
		expectedError := leadingPeriodError(1, 2)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return a adjacentPeriodsError if the key contains two adjacent periods", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a..b:1}"))
		parser.advance()
		expectedError := adjacentPeriodsError(1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return a trailingPeriodError if the ends with a period", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.:1}"))
		parser.advance()
		expectedError := trailingPeriodError(1, 3)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs while extracting the sub-object (with object start token after key)", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a{.b:1}}"))
		parser.advance()
		expectedError := leadingPeriodError(1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs while extracting the sub-object (with path expression as key)", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.b.:1}"))
		parser.advance()
		expectedError := trailingPeriodError(1, 5)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs in extractValue method after equals separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=&}"))
		parser.advance()
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "&"), 1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return merged object if the current value (after equals separator) is object and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a={b:1},a={c:2}}"))
		parser.advance()
		expected := Object{"a": Object{"b": Int(1), "c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after equals separator) is object and there is an existing non-object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=1,a={c:2}}"))
		parser.advance()
		expected := Object{"a": Object{"c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after equals separator) is not object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a={b:1},a=2}"))
		parser.advance()
		expected := Object{"a": Int(2)}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the error if any error occurs in extractValue method after colon separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:&}"))
		parser.advance()
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "&"), 1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return merged object if the current value (after colon separator) is object and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:{b:1},a:{c:2}}"))
		parser.advance()
		expected := Object{"a": Object{"b": Int(1), "c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return object containing a concatenation if the current value (after colon separator) is substitution and there is an existing substitution with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,b:2,c:${a},c:${b}}"))
		parser.advance()
		expected := Object{
			"a": Int(1),
			"b": Int(2),
			"c": concatenation{&Substitution{path: "a", optional: false}, &Substitution{path: "b", optional: false}},
		}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return object containing a concatenation if the current value (after colon separator) is substitution and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{b:2,c:{a:1},c:${b}}"))
		parser.advance()
		expected := Object{
			"b": Int(2),
			"c": concatenation{Object{"a": Int(1)}, &Substitution{path: "b", optional: false}},
		}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return object containing a concatenation if the current value (after colon separator) is substitution and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,c:${a},c:{b:2}}"))
		parser.advance()
		expected := Object{
			"a": Int(1),
			"c": concatenation{&Substitution{path: "a", optional: false}, Object{"b": Int(2)}},
		}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return valueWithAlternative object if the current value (after colon separator) is substitution and the existing value is neither a substitution nor object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,a:${?b}}"))
		parser.advance()
		expected := Object{
			"a": &valueWithAlternative{
				value:       Int(1),
				alternative: &Substitution{path: "b", optional: true},
			},
		}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after colon separator) is object and there is an existing non-object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,a:{c:2}}"))
		parser.advance()
		expected := Object{"a": Object{"c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after colon separator) is not object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:{b:1},a:2}"))
		parser.advance()
		expected := Object{"a": Int(2)}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return merged object if the current value (without separator) is object and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a{b:1},a{c:2}}"))
		parser.advance()
		expected := Object{"a": Object{"b": Int(1), "c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return overwritten object if a key is repeated three times, and the first occurrence is not an object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=1,a{b:1},a{c:2}}"))
		parser.advance()
		expected := Object{"a": Object{"b": Int(1), "c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return overwritten object if a key is repeated three times, and the second occurrence is not an object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a{b:1},a=1,a{c:2}}"))
		parser.advance()
		expected := Object{"a": Object{"c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return overwritten object if a key is repeated three times, and the last occurrence is not an object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a{b:1},a{c:2},a=1}"))
		parser.advance()
		expected := Object{"a": Int(1)}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the error if any error occurs in parsePlusEquals method", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,a+=2}"))
		parser.advance()
		expectedError := invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", "1", "a"), 1, 10)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract object with the += separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a+=1}"))
		parser.advance()
		expected := Object{"a": Array{Int(1)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return error if '=' does not exist after '+'", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a+1}"))
		parser.advance()
		expectedError := invalidKeyError("+", 1, 3)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract only the sub-object and return if the isSubObject is given 'true'", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.b:1,c:2}"))
		advanceScanner(t, parser, "b")
		got, err := parser.extractObject(true)
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"b": Int(1)})
	})

	t.Run("return the error if any error occurs while concatenating", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:b ${"))
		parser.advance()
		got, err := parser.extractObject()
		assertError(t, err, invalidSubstitutionError("missing closing parenthesis", 1, 7))
		assertNil(t, got)
	})

	t.Run("should break the concatenation loop if the checkAndConcatenate method returns false", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:[1] bb, c:d"))
		parser.advance()
		got, err := parser.extractObject()
		assertError(t, err, missingCommaError(1, 7))
		assertNil(t, got)
	})

	t.Run("concatenate multiple values if they are concatenable and in the same line", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:bb cc dd"))
		parser.advance()
		expected := Object{"a": concatenation{String("bb"), String(" "), String("cc"), String(" "), String("dd")}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertEquals(t, got.String(), expected.String())
	})

	t.Run("should parse properly if the line ends with a comment", func(t *testing.T) {
		parser := newParser(strings.NewReader(`name: value #this is a comment`))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"name": String("value")})
	})

	t.Run("should parse properly if the comment contains a `'` character (which results golang scanner to append `\n` to the latest token instead of a separate token)", func(t *testing.T) {
		config := `
		# it's a comment
		name: value
		`
		parser := newParser(strings.NewReader(config))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"name": String("value")})
	})

	t.Run("return missingCommaError if there is no comma or ASCII newline between the object elements", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1 b:2}"))
		parser.advance()
		expectedError := missingCommaError(1, 6)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("skip comma between the object elements", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,b:2}"))
		parser.advance()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1), "b": Int(2)})
	})

	t.Run("return adjacentCommasError if there are two adjacent commas between the elements of the object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,,b:2}"))
		parser.advance()
		expectedError := adjacentCommasError(1, 6)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return invalidObjectError if parenthesis do not match", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1"))
		parser.advance()
		expectedError := invalidObjectError("parenthesis do not match", 1, 5)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})
}

func TestMergeObjects(t *testing.T) {
	t.Run("merge objects", func(t *testing.T) {
		existing := Object{"b": Int(5)}
		new := Object{"c": Int(3)}
		expected := Object{"b": Int(5), "c": Int(3)}
		mergeObjects(existing, new)
		assertDeepEqual(t, existing, expected)
	})

	t.Run("merge objects recursively if both parameters contain the same key as of type Object", func(t *testing.T) {
		existing := Object{"b": Object{"e": Int(5)}}
		new := Object{"b": Object{"f": Int(7)}, "c": Int(3)}
		expected := Object{"b": Object{"e": Int(5), "f": Int(7)}, "c": Int(3)}
		mergeObjects(existing, new)
		assertDeepEqual(t, existing, expected)
	})

	t.Run("merge objects recursively, value from the second parameter should override the first one if any of them are not of type Object", func(t *testing.T) {
		existing := Object{"b": Object{"e": Int(5)}, "c": Int(3)}
		new := Object{"b": Int(7)}
		expected := Object{"b": Int(7), "c": Int(3)}
		mergeObjects(existing, new)
		assertDeepEqual(t, existing, expected)
	})
}

func TestResolveSubstitutions(t *testing.T) {
	t.Run("resolve valid substitution at the root level", func(t *testing.T) {
		object := Object{"a": Int(5), "b": &Substitution{"a", false}}
		err := resolveSubstitutions(object)
		assertNoError(t, err)
	})

	t.Run("resolve to the environment variable if substitution path does not exist and an environment variable is set with the substitution path", func(t *testing.T) {
		testEnv := "TEST_ENV"
		substitution := &Substitution{testEnv, false}
		object := Object{"a": Int(5), "b": substitution}
		err := os.Setenv(testEnv, "test")
		assertNoError(t, err)
		err = resolveSubstitutions(object)
		assertNoError(t, err)
		err = os.Unsetenv(testEnv)
		assertNoError(t, err)
	})

	t.Run("resolve to the environment variable if substitution path does not exist and environment variable is set and default value was provided", func(t *testing.T) {
		testEnv := "TEST_ENV"
		testEnvValue := "test"
		envSubstitution := &Substitution{path: testEnv, optional: false}
		staticWithEnv := &valueWithAlternative{value: String("static"), alternative: envSubstitution}
		object := Object{"a": staticWithEnv}
		err := os.Setenv(testEnv, testEnvValue)
		assertNoError(t, err)
		expected := String(testEnvValue)
		err = resolveSubstitutions(object)
		assertNoError(t, err)
		err = os.Unsetenv(testEnv)
		assertNoError(t, err)

		if expected != object["a"] {
			t.Errorf("expected value: %s from environment variable: %s, got: %s", expected, testEnv, object["a"])
		}
	})

	t.Run("resolve to the static value if substitution path does not exist and environment variable is not set and default value was not provided", func(t *testing.T) {
		defaultValue := String("default")
		envSubstitution := &Substitution{path: "TEST_ENV", optional: true}
		staticWithEnv := &valueWithAlternative{value: defaultValue, alternative: envSubstitution}
		object := Object{"a": staticWithEnv}
		err := resolveSubstitutions(object)
		assertNoError(t, err)

		if defaultValue != object["a"] {
			t.Errorf("expected default value: %s, got: %s", defaultValue, object["a"])
		}
	})

	t.Run("return an error if cannot find required substitution and default value was provided", func(t *testing.T) {
		defaultValue := String("default")
		envSubstitution := &Substitution{path: "TEST_ENV", optional: false}
		staticWithEnv := &valueWithAlternative{value: defaultValue, alternative: envSubstitution}
		object := Object{"a": staticWithEnv}
		err := resolveSubstitutions(object)

		expectedErr := errors.New("could not resolve substitution: ${TEST_ENV} to a value")
		assertError(t, err, expectedErr)
	})

	t.Run("return an error for non-existing substitution path", func(t *testing.T) {
		substitution := &Substitution{"c", false}
		object := Object{"a": Int(5), "b": substitution}
		err := resolveSubstitutions(object)
		expectedError := errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		assertError(t, err, expectedError)
	})

	t.Run("ignore the optional substitution if it's path does not exist", func(t *testing.T) {
		object := Object{"a": Int(5), "b": &Substitution{"c", true}}
		err := resolveSubstitutions(object)
		assertNoError(t, err)
	})

	t.Run("resolve valid substitution at the non-root level", func(t *testing.T) {
		subObject := Object{"c": &Substitution{"a", false}}
		object := Object{"a": Int(5), "b": subObject}
		err := resolveSubstitutions(object, subObject)
		assertNoError(t, err)
	})

	t.Run("return invalid concatenation error if the concatenation contains an object and a different type", func(t *testing.T) {
		substitution := &Substitution{"a", false}
		object := Object{"a": Int(5), "b": concatenation{Object{"aa": Int(1)}, substitution}}
		err := resolveSubstitutions(object)
		assertError(t, err, invalidConcatenationError())
	})

	t.Run("resolve the substitution in concatenation and merge the objects if the concatenation's every element is object", func(t *testing.T) {
		substitution := &Substitution{"a", false}
		object := Object{"bb": Int(1)}
		root := Object{"a": Object{"aa": Int(5)}, "b": concatenation{object, substitution}}
		expected := Object{"aa": Int(5), "bb": Int(1)}
		err := resolveSubstitutions(root)
		got := root.find("b")
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("resolve valid substitution inside an array", func(t *testing.T) {
		subArray := Array{&Substitution{"a", false}}
		object := Object{"a": Int(5), "b": subArray}
		err := resolveSubstitutions(object, subArray)
		assertNoError(t, err)
	})

	t.Run("return error for non-existing substitution path inside an array", func(t *testing.T) {
		substitution := &Substitution{"c", false}
		subArray := Array{substitution}
		object := Object{"a": Int(5), "b": subArray}
		err := resolveSubstitutions(object, subArray)
		expectedError := errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		assertError(t, err, expectedError)
	})

	t.Run("ignore the optional substitution inside an array if it's path does not exist", func(t *testing.T) {
		subArray := Array{&Substitution{"a", true}}
		object := Object{"a": Int(5), "b": subArray}
		err := resolveSubstitutions(object, subArray)
		assertNoError(t, err)
	})

	t.Run("resolve valid substitution inside a concatenation", func(t *testing.T) {
		concatenation := concatenation{&Substitution{"a", false}}
		object := Object{"a": Int(5), "b": concatenation}
		err := resolveSubstitutions(object, concatenation)
		assertNoError(t, err)
	})

	t.Run("return error for non-existing substitution path inside an concatenation", func(t *testing.T) {
		substitution := &Substitution{"c", false}
		concatenation := concatenation{substitution}
		object := Object{"a": Int(5), "b": concatenation}
		err := resolveSubstitutions(object, concatenation)
		expectedError := errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		assertError(t, err, expectedError)
	})

	t.Run("ignore the optional substitution inside an concatenation if it's path does not exist", func(t *testing.T) {
		concatenation := concatenation{&Substitution{"a", true}}
		object := Object{"a": Int(5), "b": concatenation}
		err := resolveSubstitutions(object, concatenation)
		assertNoError(t, err)
	})

	t.Run("return error if subConfig is not an object, array or concatenation", func(t *testing.T) {
		subInt := Int(42)
		object := Object{"a": Int(5), "b": subInt}
		err := resolveSubstitutions(object, subInt)
		expectedError := invalidValueError("substitutions are only allowed in field values and array elements", 0, 0)
		assertError(t, err, expectedError)
	})

	t.Run("extract valueWithAlternative value with string type", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: stringValue, a:${?b}"))
		expected := Object{"a": &valueWithAlternative{
			value:       String("stringValue"),
			alternative: &Substitution{path: "b", optional: true},
		}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("extract valueWithAlternative value with number type", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: 1, a:${?b}"))
		expected := Object{"a": &valueWithAlternative{
			value:       Int(1),
			alternative: &Substitution{path: "b", optional: true},
		}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("extract valueWithAlternative value with duration type", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: 1s, a:${?b}"))
		expected := Object{"a": &valueWithAlternative{
			value:       Duration(time.Second),
			alternative: &Substitution{path: "b", optional: true},
		}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("extract valueWithAlternative value with boolean type", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: true, a:${?b}"))
		expected := Object{"a": &valueWithAlternative{
			value:       Boolean(true),
			alternative: &Substitution{path: "b", optional: true},
		}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("extract valueWithAlternative value and overwrite alternatives", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: static, a:${?b}"))
		expected := Object{
			"a": &valueWithAlternative{value: String("static"), alternative: &Substitution{path: "b", optional: true}},
		}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})
}

func TestParsePlusEqualsValue(t *testing.T) {
	t.Run("create an array that contains the value if the existingItems map does not contain a value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a += 42"))
		advanceScanner(t, parser, "42")
		existingItems := Object{}
		expected := Object{"a": Array{Int(42)}}
		err := parser.parsePlusEqualsValue(existingItems, "a")
		assertNoError(t, err)
		assertDeepEqual(t, existingItems, expected)
	})

	t.Run("return the error received from extractValue method if any, if the existingItems map does not contain a value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a += [42"))
		advanceScanner(t, parser, "[")
		err := parser.parsePlusEqualsValue(Object{}, "a")
		expectedError := invalidArrayError("parenthesis do not match", 1, 7)
		assertError(t, err, expectedError)
	})

	t.Run("return an error if the existingItems map contains non-array value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: 1, a += 42"))
		advanceScanner(t, parser, "42")
		existingItems := Object{"a": Int(1)}
		err := parser.parsePlusEqualsValue(existingItems, "a")
		expectedError := invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", "1", "a"), 1, 14)
		assertError(t, err, expectedError)
	})

	t.Run("return the error received from extractValue method if any, if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += {42"))
		advanceScanner(t, parser, "{")
		existingItems := Object{"a": Array{Int(5)}}
		err := parser.parsePlusEqualsValue(existingItems, "a")
		expectedError := invalidObjectError("parenthesis do not match", 1, 15)
		assertError(t, err, expectedError)
	})

	t.Run("append the value if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += 42"))
		advanceScanner(t, parser, "42")
		existingItems := Object{"a": Array{Int(5)}}
		expected := Object{"a": Array{Int(5), Int(42)}}
		err := parser.parsePlusEqualsValue(existingItems, "a")
		assertNoError(t, err)
		assertDeepEqual(t, existingItems, expected)
	})
}

func TestValidateIncludeValue(t *testing.T) {
	t.Run("return error if the include value starts with 'file' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include file[abc.conf]"))
		advanceScanner(t, parser, "file")
		expectedError := invalidValueError("missing opening parenthesis", 1, 13)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'file' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include file(abc.conf"))
		advanceScanner(t, parser, "file")
		expectedError := invalidValueError("missing closing parenthesis", 1, 17)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'classpath' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include classpath[abc.conf]"))
		advanceScanner(t, parser, "classpath")
		expectedError := invalidValueError("missing opening parenthesis", 1, 18)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'classpath' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include classpath(abc.conf"))
		advanceScanner(t, parser, "classpath")
		expectedError := invalidValueError("missing closing parenthesis", 1, 22)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value does not start with double quotes", func(t *testing.T) {
		parser := newParser(strings.NewReader("include abc.conf"))
		advanceScanner(t, parser, "abc")
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value does not end with double quotes", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "abc.conf`))
		advanceScanner(t, parser, `"abc.conf`)
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value is just a double quotes", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "`))
		advanceScanner(t, parser, `"`)
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the path with quotes removed", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "abc.conf"`))
		advanceScanner(t, parser, `"abc.conf"`)
		expected := &include{path: "abc.conf", required: false}
		got, err := parser.validateIncludeValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the include token containing the path in file(...) with quotes removed and required as 'false'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include file("abc.conf")`))
		advanceScanner(t, parser, "file")
		got, err := parser.validateIncludeValue()
		expected := &include{path: "abc.conf", required: false}
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the include token containing the path in classpath(...) with quotes removed and required as 'false'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include classpath("abc.conf")`))
		advanceScanner(t, parser, "classpath")
		expected := &include{path: "abc.conf", required: false}
		got, err := parser.validateIncludeValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return error if the include value starts with 'required' but opening parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include required[abc.conf]"))
		advanceScanner(t, parser, "required")
		expectedError := invalidValueError("missing opening parenthesis", 1, 17)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return error if the include value starts with 'required' but closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("include required(abc.conf"))
		advanceScanner(t, parser, "required")
		expectedError := invalidValueError("missing closing parenthesis", 1, 21)
		got, err := parser.validateIncludeValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the include token containing the path in required(file(...)) with quotes removed, and required as 'true'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include required(file("abc.conf"))`))
		advanceScanner(t, parser, "required")
		got, err := parser.validateIncludeValue()
		expected := &include{path: "abc.conf", required: true}
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the include token containing the path in required(classpath(...)) with quotes removed and required as 'true'", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include required(classpath("abc.conf"))`))
		advanceScanner(t, parser, "required")
		expected := &include{path: "abc.conf", required: true}
		got, err := parser.validateIncludeValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})
}

func TestParseIncludedResource(t *testing.T) {
	t.Run("return the error from the validateIncludeValue method if it returns an error", func(t *testing.T) {
		parser := newParser(strings.NewReader("include abc.conf"))
		advanceScanner(t, parser, "abc")
		expectedError := invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", 1, 9)
		object, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, object)
	})

	t.Run("return an empty object if the file does not exist and the include token is not required", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "nonExistFile.conf"`))
		advanceScanner(t, parser, `"nonExistFile.conf"`)
		got, err := parser.parseIncludedResource()
		assertNil(t, err)
		assertDeepEqual(t, got, Object{})
	})

	t.Run("return an error if the file does not exist but the include token is required", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include required("nonExistFile.conf")`))
		advanceScanner(t, parser, "required")
		expectedError := fmt.Errorf("could not parse resource: %w", &os.PathError{Op: "open", Path: "nonExistFile.conf", Err: errors.New("no such file or directory")})
		object, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, object)
	})

	t.Run("return an error if the included file contains an array as the value", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/array.conf"`))
		advanceScanner(t, parser, `"testdata/array.conf"`)
		expectedError := invalidValueError("included file cannot contain an array as the root value", 1, 9)
		object, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, object)
	})

	t.Run("parse the included resource and return the parsed object if there is no error", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/x.conf"`))
		advanceScanner(t, parser, `"testdata/x.conf"`)
		got, err := parser.parseIncludedResource()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1), "x": Int(7), "y": String("foo")})
	})
}

func TestExtractArray(t *testing.T) {
	t.Run("return invalidArray error if the first token is not '['", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1}"))
		parser.advance()
		expectedError := invalidArrayError(fmt.Sprintf("%q is not an array start token", "{"), 1, 1)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return leadingCommaError if the array starts with a comma", func(t *testing.T) {
		parser := newParser(strings.NewReader("[,1]"))
		parser.advance()
		expectedError := leadingCommaError(1, 2)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract the empty array", func(t *testing.T) {
		parser := newParser(strings.NewReader("[]"))
		parser.advance()
		got, err := parser.extractArray()
		assertNoError(t, err)
		assertEquals(t, len(got), 0)
	})

	t.Run("return the error if any error occurs in extractValue method", func(t *testing.T) {
		parser := newParser(strings.NewReader("[&a]"))
		parser.advance()
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "&"), 1, 2)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return invalidArrayError if the closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1"))
		parser.advance()
		expectedError := invalidArrayError("parenthesis do not match", 1, 2)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return missingCommaError if there is no comma or ASCII newline between the array elements", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1 2]"))
		parser.advance()
		expectedError := missingCommaError(1, 4)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return adjacentCommasError if there are two adjacent commas between the elements of the array", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1,,2]"))
		parser.advance()
		expectedError := adjacentCommasError(1, 4)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract the array without an error even if the array ends with a comma", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1,]"))
		parser.advance()
		got, err := parser.extractArray()
		assertNoError(t, err)
		assertDeepEqual(t, got, Array{Int(1)})
	})

	t.Run("extract the array without an error if if elements are separated with ASCII newline", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1\n2]"))
		parser.advance()
		got, err := parser.extractArray()
		assertNoError(t, err)
		assertDeepEqual(t, got, Array{Int(1), Int(2)})
	})

	t.Run("extract the array", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1, 2]"))
		parser.advance()
		expected := Array{Int(1), Int(2)}
		got, err := parser.extractArray()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})
}

func TestExtractValue(t *testing.T) {
	t.Run("skip the comment at the beginning of the value", func(t *testing.T) {
		config := `
			a: # this is a comment
			1`
		parser := newParser(strings.NewReader(config))
		advanceScanner(t, parser, "#")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, Int(1))
	})

	t.Run("extract int duration", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1 second"))
		advanceScanner(t, parser, "1")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, Duration(time.Second))
	})

	t.Run("extract int value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1"))
		advanceScanner(t, parser, "1")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, Int(1))
	})

	t.Run("extract float value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1.5 seconds"))
		advanceScanner(t, parser, "1.5")
		got, err := parser.extractValue()
		assertNoError(t, err)
		expected := 1.5
		assertEquals(t, got, Duration(time.Duration(expected)*time.Second))
	})

	t.Run("extract float value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1.5"))
		advanceScanner(t, parser, "1.5")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, Float64(1.5))
	})

	t.Run("extract multi-line string", func(t *testing.T) {
		config := `a: """
			this is a
			multi-line string
		"""`
		parser := newParser(strings.NewReader(config))
		advanceScanner(t, parser, `""`)
		expected := String(`
			this is a
			multi-line string
		`)
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, expected)
	})

	t.Run("extract string value", func(t *testing.T) {
		parser := newParser(strings.NewReader(`a:"b"`))
		advanceScanner(t, parser, `"b"`)
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, String("b"))
	})

	t.Run("extract null value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:null"))
		advanceScanner(t, parser, "null")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, null)
	})

	var booleanTestCases = []struct {
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

	for _, tc := range booleanTestCases {
		t.Run(fmt.Sprintf("extract boolean value: %q", tc.input), func(t *testing.T) {
			parser := newParser(strings.NewReader(fmt.Sprintf("a:%s", tc.input)))
			advanceScanner(t, parser, tc.input)
			got, err := parser.extractValue()
			assertNoError(t, err)
			assertEquals(t, got, tc.expected)
		})
	}

	t.Run("extract unquoted string value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:bbb"))
		advanceScanner(t, parser, "bbb")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertEquals(t, got, String("bbb"))
	})

	t.Run("extract object value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:{b:1}"))
		advanceScanner(t, parser, "{")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"b": Int(1)})
	})

	t.Run("extract array value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:[1]"))
		advanceScanner(t, parser, "[")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, Array{Int(1)})
	})

	t.Run("extract substitution value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b}"))
		advanceScanner(t, parser, "$")
		expected := &Substitution{"b", false}
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("extract unquoted string value if the value is non-alphanumeric and non-forbidden character", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:bbb.ccc"))
		advanceScanner(t, parser, ".")
		got, err := parser.extractValue()
		assertNoError(t, err)
		assertDeepEqual(t, got, String("."))
	})

	t.Run("return error for an unknown value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:&"))
		advanceScanner(t, parser, "&")
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "&"), 1, 3)
		got, err := parser.extractValue()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})
}

func TestExtractDurationUnit(t *testing.T) {
	var durationTestCases = []struct {
		input    string
		expected time.Duration
	}{
		{"ns", time.Nanosecond},
		{"nano", time.Nanosecond},
		{"nanos", time.Nanosecond},
		{"nanosecond", time.Nanosecond},
		{"nanoseconds", time.Nanosecond},
		{"us", time.Microsecond},
		{"micro", time.Microsecond},
		{"micros", time.Microsecond},
		{"microsecond", time.Microsecond},
		{"microseconds", time.Microsecond},
		{"ms", time.Millisecond},
		{"milli", time.Millisecond},
		{"millis", time.Millisecond},
		{"millisecond", time.Millisecond},
		{"milliseconds", time.Millisecond},
		{"s", time.Second},
		{"second", time.Second},
		{"seconds", time.Second},
		{"m", time.Minute},
		{"minute", time.Minute},
		{"minutes", time.Minute},
		{"h", time.Hour},
		{"hour", time.Hour},
		{"hours", time.Hour},
		{"d", time.Hour * 24},
		{"day", time.Hour * 24},
		{"days", time.Hour * 24},
		{"nonDurationUnit", time.Duration(0)},
	}

	for _, tc := range durationTestCases {
		t.Run(fmt.Sprintf("extract duration unit: %s", tc.input), func(t *testing.T) {
			parser := newParser(strings.NewReader(fmt.Sprintf("a:1 %s", tc.input)))
			advanceScanner(t, parser, "1")
			got := parser.extractDurationUnit()
			assertEquals(t, got, tc.expected)
		})
	}
}

func TestExtractSubstitution(t *testing.T) {
	t.Run("return invalidSubstitutionError if the path expression is empty", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${}"))
		advanceScanner(t, parser, "$")
		expectedError := invalidSubstitutionError("path expression cannot be empty", 1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return leadingPeriodError if the path expression starts with a period '.' ", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${.a}"))
		advanceScanner(t, parser, "$")
		expectedError := leadingPeriodError(1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("...", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${#comment}"))
		advanceScanner(t, parser, "$")
		expectedError := invalidSubstitutionError("comments are not allowed inside substitutions", 1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return adjacentPeriodsError if the substitution path contains two adjacent periods", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b..c}"))
		advanceScanner(t, parser, "$")
		expectedError := adjacentPeriodsError(1, 7)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return invalidSubstitutionError if the closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b"))
		advanceScanner(t, parser, "$")
		expectedError := invalidSubstitutionError("missing closing parenthesis", 1, 5)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("return trailingPeriodError if the path expression starts with a period '.' ", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${a.}"))
		advanceScanner(t, parser, "$")
		expectedError := trailingPeriodError(1, 6)
		substitution, err := parser.extractSubstitution()
		assertError(t, err, expectedError)
		assertNil(t, substitution)
	})

	t.Run("parse and return a pointer to the substitution", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b.c}"))
		advanceScanner(t, parser, "$")
		expected := &Substitution{path: "b.c", optional: false}
		substitution, err := parser.extractSubstitution()
		assertNoError(t, err)
		assertDeepEqual(t, substitution, expected)
	})

	t.Run("parse and return a pointer to the optional substitution", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${?b.c}"))
		advanceScanner(t, parser, "$")
		expected := &Substitution{path: "b.c", optional: true}
		substitution, err := parser.extractSubstitution()
		assertNoError(t, err)
		assertDeepEqual(t, substitution, expected)
	})

	for forbiddenChar := range forbiddenCharacters {
		t.Run(fmt.Sprintf("return error for the forbidden character: %q", forbiddenChar), func(t *testing.T) {
			if forbiddenChar != "`" && forbiddenChar != `"` && forbiddenChar != "}" && forbiddenChar != "#" {
				parser := newParser(strings.NewReader(fmt.Sprintf("a:${b%s}", forbiddenChar)))
				advanceScanner(t, parser, "$")
				expectedError := invalidKeyError(forbiddenChar, 1, 6)
				substitution, err := parser.extractSubstitution()
				assertError(t, err, expectedError)
				assertNil(t, substitution)
			}
		})
	}
}

func TestExtractMultiLineString(t *testing.T) {
	t.Run("extract multi-line string", func(t *testing.T) {
		parser := newParser(strings.NewReader(`a:"""abc"""`))
		advanceScanner(t, parser, `""`)
		got, err := parser.extractMultiLineString()
		assertNoError(t, err)
		assertEquals(t, got, String("abc"))
	})

	t.Run("extract multi-line string with the quotes inside", func(t *testing.T) {
		parser := newParser(strings.NewReader(`a:"""abc"def"""`))
		advanceScanner(t, parser, `""`)
		got, err := parser.extractMultiLineString()
		assertNoError(t, err)
		assertEquals(t, got, String(`abc"def`))
	})

	t.Run("extract multi-line string with ending more than three quotes, extra quotes treated as part of the string", func(t *testing.T) {
		parser := newParser(strings.NewReader(`a:"""abc"""""`))
		advanceScanner(t, parser, `""`)
		got, err := parser.extractMultiLineString()
		assertNoError(t, err)
		assertEquals(t, got, String(`abc""`))
	})

	t.Run("return the unclosedMultiLineStringError if the multi line string is not closed", func(t *testing.T) {
		parser := newParser(strings.NewReader(`"""abc"`))
		advanceScanner(t, parser, `""`)
		got, err := parser.extractMultiLineString()
		assertError(t, err, unclosedMultiLineStringError())
		assertEquals(t, got, String(""))
	})
}

func TestIsSubstitution(t *testing.T) {
	var testCases = []struct {
		token       string
		peekedToken rune
		expected    bool
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

func TestIsUnquotedString(t *testing.T) {
	for forbiddenChar := range forbiddenCharacters {
		t.Run(fmt.Sprintf("return false if the token contains the forbidden character: %q", forbiddenChar), func(t *testing.T) {
			if forbiddenChar != "`" && forbiddenChar != `"` && forbiddenChar != "}" && forbiddenChar != "#" {
				got := isUnquotedString(fmt.Sprintf("aa%sbb", forbiddenChar))
				assertEquals(t, got, false)
			}
		})
	}

	t.Run("return true if the token does not contain any forbidden character", func(t *testing.T) {
		got := isUnquotedString("aaa")
		assertEquals(t, got, true)
	})
}

func TestIsMultiLineString(t *testing.T) {
	var testCases = []struct {
		token       string
		peekedToken rune
		expected    bool
	}{
		{`""`, '"', true},
		{"a", '"', false},
		{`""`, 'a', false},
		{"a", 'b', false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("return %v if the token is %q and the peekedToken is %q", tc.expected, tc.token, tc.peekedToken), func(t *testing.T) {
			got := isMultiLineString(tc.token, tc.peekedToken)
			assertEquals(t, got, tc.expected)
		})
	}
}

func TestCheckAndConcatenate(t *testing.T) {
	t.Run("return false if there isn't any value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:aa bb"))
		advanceScanner(t, parser, "bb")
		got, err := parser.checkAndConcatenate(Object{"a": String("aa")}, "c")
		assertNoError(t, err)
		assertEquals(t, got, false)
	})

	t.Run("return false if the value with the given is not concatenable", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1 bb"))
		advanceScanner(t, parser, "bb")
		got, err := parser.checkAndConcatenate(Object{"a": Int(1)}, "a")
		assertNoError(t, err)
		assertEquals(t, got, false)
	})

	t.Run("return false if the current token is not concatenable", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:abc 5"))
		advanceScanner(t, parser, "5")
		got, err := parser.checkAndConcatenate(Object{"a": String("abc")}, "5")
		assertNoError(t, err)
		assertEquals(t, got, false)
	})

	t.Run("return the error if any error occurs in the extractValue method", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:abc ${"))
		advanceScanner(t, parser, "$")
		object := Object{"a": String("abc")}
		got, err := parser.checkAndConcatenate(object, "a")
		assertError(t, err, invalidSubstitutionError("missing closing parenthesis", 1, 9))
		assertEquals(t, got, false)
	})

	t.Run("concatenate the value to the previous value if the previous one is a concatenation", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:aa bb cc"))
		advanceScanner(t, parser, "cc")
		whitespace := parser.lastConsumedWhitespaces
		object := Object{"a": concatenation{String("aa"), String(whitespace), String("bb")}}
		got, err := parser.checkAndConcatenate(object, "a")
		assertNoError(t, err)
		assertEquals(t, got, true)
		expected := Object{"a": concatenation{String("aa"), String(whitespace), String("bb"), String(whitespace), String("cc")}}
		assertDeepEqual(t, object, expected)
	})

	t.Run("create a concatenation with the value and the previous value if the previous one is not a concatenation", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:aa bb"))
		advanceScanner(t, parser, "bb")
		object := Object{"a": String("aa")}
		got, err := parser.checkAndConcatenate(object, "a")
		assertNoError(t, err)
		assertEquals(t, got, true)
		expected := Object{"a": concatenation{String("aa"), String(" "), String("bb")}}
		assertEquals(t, object.String(), expected.String())
	})
}
