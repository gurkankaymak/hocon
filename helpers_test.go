package hocon

import (
	"fmt"
	"reflect"
	"testing"
)

func assertEquals(got, expected interface{}, t *testing.T) {
	t.Helper()
	if got != expected {
		fail(got, expected, t)
	}
}

func assertPanic(fn func(), t *testing.T, expectedMessage ...string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, but did not get any!")
		}
		switch recovered := r.(type) {
		case string:
			if len(expectedMessage) > 0 && recovered != expectedMessage[0] {
				wrongPanic(recovered, expectedMessage[0], t)
			}
		case error:
			if messageGot := recovered.Error(); len(expectedMessage) > 0 && messageGot != expectedMessage[0] {
				wrongPanic(messageGot, expectedMessage[0], t)
			}
		}
	}()
	fn()
}

func assertConfigEquals(got fmt.Stringer, expected string, t *testing.T) {
	t.Helper()
	if got.String() != expected {
		fail(got, expected, t)
	}
}

func assertNoError(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatalf("not expected an error, got err: %q", err)
	}
}

// TODO gk: instead of comparing error strings compare the errors, fix this after the custom errors are defined
func assertError(err error, t *testing.T, expected ...error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error but did not get one")
	} else if len(expected) > 0 && expected[0].Error() != err.Error() {
		t.Fatalf("wrong error received! expected: %q, got: %q", expected[0], err)
	}
}

func assertDeepEqual(got, expected interface{}, t *testing.T) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		fail(got, expected, t)
	}
}

func assertNil(t *testing.T, i interface{}) {
	t.Helper()
	if !isNil(i) {
		fail(i, nil, t)
	}
}

func fail(got, expected interface{}, t *testing.T) {
	t.Helper()
	t.Errorf("expected: %q, got: %q", expected, got)
}

func wrongPanic(got, expected string, t *testing.T) {
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
