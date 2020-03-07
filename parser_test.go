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
		assertDeepEqual(t, got, &Config{Array{Int(1), Int(2), Int(3)}})
	})
}

func TestParse(t *testing.T) {
	t.Run("try to parse as config array if the input starts with '[' and return the error from extractArray if any", func(t *testing.T) {
		parser := newParser(strings.NewReader("[5}"))
		expectedError := invalidArrayError("parenthesis do not match", 1, 3)
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
		parser := newParser(strings.NewReader("{a:5]"))
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
}

func TestExtractObject(t *testing.T) {
	t.Run("extract empty object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{}"))
		parser.scanner.Scan() // move scanner to the first token for the test case
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{})
	})

	t.Run("extract object with the root braces omitted", func(t *testing.T) {
		parser := newParser(strings.NewReader("a=1"))
		parser.scanner.Scan()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1)})
	})

	t.Run("extract simple object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=1}"))
		parser.scanner.Scan()
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1)})
	})

	t.Run("return the error if any error occurs in parseIncludedResource method", func(t *testing.T) {
		parser := newParser(strings.NewReader(`{include "testdata/array.conf"}`))
		parser.scanner.Scan()
		expectedErr := invalidValueError("included file cannot contain an array as the root value", 1, 10)
		got, err := parser.extractObject()
		assertError(t, err, expectedErr)
		assertNil(t, got)
	})

	t.Run("merge the included object with the existing", func(t *testing.T) {
		parser := newParser(strings.NewReader(`b:2 include "testdata/a.conf"`))
		parser.scanner.Scan()
		expected := Object{"a": Int(1), "b": Int(2)}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	for forbiddenChar, _ := range forbiddenCharacters {
		t.Run(fmt.Sprintf("return error if the key contains the forbidden character: %q", forbiddenChar), func(t *testing.T) {
			if forbiddenChar != "`" && forbiddenChar != `"` && forbiddenChar != "}" && forbiddenChar != "#" { // TODO gk: add test cases for '`' and '"' characters
				parser := newParser(strings.NewReader(fmt.Sprintf("{%s:1}", forbiddenChar)))
				parser.scanner.Scan()
				expectedError := invalidKeyError(forbiddenChar, 1, 2)
				got, err := parser.extractObject()
				assertError(t, err, expectedError)
				assertNil(t, got)
			}
		})
	}

	t.Run("return a leadingPeriodError if the key starts with a period '.'", func(t *testing.T) {
		parser := newParser(strings.NewReader("{.a:1}"))
		parser.scanner.Scan()
		expectedError := leadingPeriodError(1, 2)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return a adjacentPeriodsError if the key contains two adjacent periods", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a..b:1}"))
		parser.scanner.Scan()
		expectedError := adjacentPeriodsError(1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return a trailingPeriodError if the ends with a period", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.:1}"))
		parser.scanner.Scan()
		expectedError := trailingPeriodError(1, 3)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs while extracting the sub-object (with object start token after key)", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a{.b:1}}"))
		parser.scanner.Scan()
		expectedError := leadingPeriodError(1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs while extracting the sub-object (with path expression as key)", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a.b.:1}"))
		parser.scanner.Scan()
		expectedError := trailingPeriodError(1, 5)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return the error if any error occurs in extractValue method after equals separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=b}"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "b"), 1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return merged object if the current value (after equals separator) is object and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a={b:1},a={c:2}}"))
		parser.scanner.Scan()
		expected := Object{"a": Object{"b": Int(1), "c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after equals separator) is object and there is an existing non-object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a=1,a={c:2}}"))
		parser.scanner.Scan()
		expected := Object{"a": Object{"c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after equals separator) is not object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a={b:1},a=2}"))
		parser.scanner.Scan()
		expected := Object{"a": Int(2)}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the error if any error occurs in extractValue method after colon separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:b}"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "b"), 1, 4)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return merged object if the current value (after colon separator) is object and there is an existing object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:{b:1},a:{c:2}}"))
		parser.scanner.Scan()
		expected := Object{"a": Object{"b": Int(1), "c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after colon separator) is object and there is an existing non-object with the same key", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,a:{c:2}}"))
		parser.scanner.Scan()
		expected := Object{"a": Object{"c": Int(2)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("override the existing value if the current value (after colon separator) is not object", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:{b:1},a:2}"))
		parser.scanner.Scan()
		expected := Object{"a": Int(2)}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return the error if any error occurs in parsePlusEquals method", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1,a+=2}"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", "1", "a"), 1, 10)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract object with the += separator", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a+=1}"))
		parser.scanner.Scan()
		expected := Object{"a": Array{Int(1)}}
		got, err := parser.extractObject()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return error if '=' does not exist after '+'", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a+1}"))
		parser.scanner.Scan()
		expectedError := invalidKeyError("+", 1, 3)
		got, err := parser.extractObject()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return invalidObjectError if parenthesis do not match", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1"))
		parser.scanner.Scan()
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

	t.Run("merge config objects recursively if both parameters contain the same key as of type Object", func(t *testing.T) {
		existing := Object{"b": Object{"e": Int(5)}}
		new := Object{"b": Object{"f": Int(7)}, "c": Int(3)}
		expected := Object{"b": Object{"e": Int(5), "f": Int(7)}, "c": Int(3)}
		mergeObjects(existing, new)
		assertDeepEqual(t, existing, expected)
	})

	t.Run("merge config objects recursively, config from the second parameter should override the first one if any of them are not of type Object", func(t *testing.T) {
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

	t.Run("return error for non-existing substitution path inside an array", func(t *testing.T) {
		subArray := Array{&Substitution{"a", true}}
		object := Object{"a": Int(5), "b": subArray}
		err := resolveSubstitutions(object, subArray)
		assertNoError(t, err)
	})

	t.Run("return array if subConfig is not an object or array", func(t *testing.T) {
		subInt := Int(42)
		object := Object{"a": Int(5), "b": subInt}
		err := resolveSubstitutions(object, subInt)
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
		existingItems := Object{}
		expected := Object{"a": Array{Int(42)}}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, existingItems, expected)
	})

	t.Run("return the error received from extractValue method if any, if the existingItems map does not contain a value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a += [42"))
		var currentRune int32
		for parser.scanner.TokenText() != "[" {
			currentRune = parser.scanner.Scan()
		}
		err := parser.parsePlusEqualsValue(Object{}, "a", currentRune)
		expectedError := invalidArrayError("parenthesis do not match", 1, 7)
		assertError(t, err, expectedError)
	})

	t.Run("return an error if the existingItems map contains non-array value with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: 1, a += 42"))
		var currentRune int32
		for parser.scanner.TokenText() != "42" { // move the scanner to the position for the test case
			currentRune = parser.scanner.Scan()
		}
		existingItems := Object{"a": Int(1)}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		expectedError := invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", "1", "a"), 1, 14)
		assertError(t, err, expectedError)
	})

	t.Run("return the error received from extractValue method if any, if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += {42"))
		var currentRune int32
		for parser.scanner.TokenText() != "{" {
			currentRune = parser.scanner.Scan()
		}
		existingItems := Object{"a": Array{Int(5)}}
		err := parser.parsePlusEqualsValue(existingItems, "a", currentRune)
		expectedError := invalidObjectError("parenthesis do not match", 1, 15)
		assertError(t, err, expectedError)
	})

	t.Run("append the value if the existingItems map contains an array with the given key", func(t *testing.T) {
		parser := newParser(strings.NewReader("a: [5], a += 42"))
		var currentRune int32
		for parser.scanner.TokenText() != "42" {
			currentRune = parser.scanner.Scan()
		}
		existingItems := Object{"a": Array{Int(5)}}
		expected := Object{"a": Array{Int(5), Int(42)}}
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
		object, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, object)
	})

	t.Run("return an empty object if the file does not exist and the include token is not required", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "nonExistFile.conf"`))
		for ; parser.scanner.TokenText() != `"nonExistFile.conf"`; parser.scanner.Scan() {}
		got, err := parser.parseIncludedResource()
		assertNil(t, err)
		assertDeepEqual(t, got, Object{})
	})

	t.Run("return an error if the file does not exist but the include token is required", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include required("nonExistFile.conf")`))
		for ; parser.scanner.TokenText() != "required"; parser.scanner.Scan() {}
		expectedError := fmt.Errorf("could not parse resource: %w", &os.PathError{Op: "open", Path: "nonExistFile.conf", Err: errors.New("no such file or directory")})
		object, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, object)
	})

	t.Run("return an error if the included file contains an array as the value", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/array.conf"`))
		for ; parser.scanner.TokenText() != `"testdata/array.conf"`; parser.scanner.Scan() {}
		expectedError := invalidValueError("included file cannot contain an array as the root value", 1, 9)
		object, err := parser.parseIncludedResource()
		assertError(t, err, expectedError)
		assertNil(t, object)
	})

	t.Run("parse the included resource and return the parsed object if there is no error", func(t *testing.T) {
		parser := newParser(strings.NewReader(`include "testdata/a.conf"`))
		for ; parser.scanner.TokenText() != `"testdata/a.conf"`; parser.scanner.Scan() {}
		got, err := parser.parseIncludedResource()
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"a": Int(1)})
	})
}

func TestExtractArray(t *testing.T) {
	t.Run("return invalidArray error if the first token is not '['", func(t *testing.T) {
		parser := newParser(strings.NewReader("{a:1}"))
		parser.scanner.Scan()
		expectedError := invalidArrayError(fmt.Sprintf("%q is not an array start token", "{"), 1, 1)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract the empty array", func(t *testing.T) {
		parser := newParser(strings.NewReader("[]"))
		parser.scanner.Scan()
		got, err := parser.extractArray()
		assertNoError(t, err)
		assertEquals(t, len(got), 0)
	})

	t.Run("return the error if any error occurs in extractValue method", func(t *testing.T) {
		parser := newParser(strings.NewReader("[a]"))
		parser.scanner.Scan()
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "a"), 1, 2)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("return invalidConfigArray if the closing parenthesis is missing", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1"))
		parser.scanner.Scan()
		expectedError := invalidArrayError("parenthesis do not match", 1, 2)
		got, err := parser.extractArray()
		assertError(t, err, expectedError)
		assertNil(t, got)
	})

	t.Run("extract the array", func(t *testing.T) {
		parser := newParser(strings.NewReader("[1, 2]"))
		parser.scanner.Scan()
		expected := Array{Int(1), Int(2)}
		got, err := parser.extractArray()
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})
}

func TestExtractConfigValue(t *testing.T) {
	t.Run("extract int value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "1"; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, Int(1))
	})

	t.Run("extract float32 value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:1.5"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "1.5"; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, Float32(1.5))
	})

	t.Run("extract string value", func(t *testing.T) {
		parser := newParser(strings.NewReader(`a:"b"`))
		var currentRune rune
		for ; parser.scanner.TokenText() != `"b"`; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, String("b"))
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
			var currentRune rune
			for ; parser.scanner.TokenText() != tc.input; currentRune = parser.scanner.Scan() {}
			got, err := parser.extractValue(currentRune)
			assertNoError(t, err)
			assertDeepEqual(t, got, tc.expected)
		})
	}

	t.Run("extract object value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:{b:1}"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "{"; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, Object{"b": Int(1)})
	})

	t.Run("extract array value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:[1]"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "["; currentRune = parser.scanner.Scan() {}
		got, err := parser.extractValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, Array{Int(1)})
	})

	t.Run("extract substitution value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:${b}"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "$"; currentRune = parser.scanner.Scan() {}
		expected := &Substitution{"b", false}
		got, err := parser.extractValue(currentRune)
		assertNoError(t, err)
		assertDeepEqual(t, got, expected)
	})

	t.Run("return error for an unknown value", func(t *testing.T) {
		parser := newParser(strings.NewReader("a:bb"))
		var currentRune rune
		for ; parser.scanner.TokenText() != "bb"; currentRune = parser.scanner.Scan() {}
		expectedError := invalidValueError(fmt.Sprintf("unknown value: %q", "bb"), 1, 3)
		got, err := parser.extractValue(currentRune)
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