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

func assertDeepEqual(got, expected interface{}, t *testing.T) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		fail(got, expected, t)
	}
}

func fail(got, expected interface{}, t *testing.T) {
	t.Helper()
	t.Errorf("expected: %q, got: %q", expected, got)
}

func wrongPanic(got, expected string, t *testing.T) {
	t.Errorf("wrong panic received! expected: %q, got: %q", expected, got)
}
