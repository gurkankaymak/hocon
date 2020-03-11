package hocon

import (
	"reflect"
	"testing"
	"text/scanner"
)

func assertEquals(t *testing.T, got, expected interface{}) {
	t.Helper()
	if got != expected {
		fail(t, got, expected)
	}
}

func assertPanic(t *testing.T, fn func(), expectedMessage ...string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, but did not get any!")
		}
		switch recovered := r.(type) {
		case string:
			if len(expectedMessage) > 0 && recovered != expectedMessage[0] {
				wrongPanic(t, recovered, expectedMessage[0])
			}
		case error:
			if messageGot := recovered.Error(); len(expectedMessage) > 0 && messageGot != expectedMessage[0] {
				wrongPanic(t, messageGot, expectedMessage[0])
			}
		}
	}()
	fn()
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("not expected an error, got err: %q", err)
	}
}

func assertError(t *testing.T, err, expected error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error but did not get one")
	} else if expected.Error() != err.Error() {
		t.Fatalf("wrong error received! expected: %q, got: %q", expected, err)
	}
}

func assertDeepEqual(t *testing.T, got, expected interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		fail(t, got, expected)
	}
}

func assertNil(t *testing.T, i interface{}) {
	t.Helper()
	if !isNil(i) {
		fail(t, i, nil)
	}
}

func advanceScanner(t *testing.T, parser *parser, target string) rune {
	t.Helper()
	var currentRune rune
	for parser.scanner.TokenText() != target {
		currentRune = parser.scanner.Scan()
		if currentRune == scanner.EOF {
			panic("EOF is reached for the test: " + t.Name())
		}
	}
	return currentRune
}

func fail(t *testing.T, got, expected interface{}) {
	t.Helper()
	t.Errorf("expected: %q, got: %q", expected, got)
}

func wrongPanic(t *testing.T, got, expected string) {
	t.Helper()
	t.Errorf("wrong panic received! expected: %q, got: %q", expected, got)
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch value := reflect.ValueOf(i); value.Kind() {
	case reflect.Ptr, reflect.Chan, reflect.Func, reflect.Map, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return value.IsNil()
	}
	return false
}
