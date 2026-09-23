package hocon

import (
	"errors"
	"fmt"
)

// ErrPathNotFound is the error returned by the Get*E methods when there is no value
// at the given path, it can be checked with errors.Is
var ErrPathNotFound = errors.New("path not found")

func pathNotFoundError(path string) error {
	return fmt.Errorf("%w: %q", ErrPathNotFound, path)
}

func cannotParseError(value Value, targetType string) error {
	return fmt.Errorf("cannot parse value: %s to %s!", value, targetType)
}

// ParseError represents an error occurred while parsing a resource or string to a hocon configuration
type ParseError struct {
	errType string
	message string
	line    int
	column  int
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("%s at: %d:%d, %s", p.errType, p.line, p.column, p.message)
}

func parseError(errType, message string, line, column int) *ParseError {
	return &ParseError{errType: errType, message: message, line: line, column: column}
}

func leadingPeriodError(line, column int) *ParseError {
	return parseError("leading period '.'", `(use quoted "" empty string if you want an empty element)`, line, column)
}

func trailingPeriodError(line, column int) *ParseError {
	return parseError("trailing period '.'", `(use quoted "" empty string if you want an empty element)`, line, column)
}

func adjacentPeriodsError(line, column int) *ParseError {
	return parseError("two adjacent periods '.'", `(use quoted "" empty string if you want an empty element)`, line, column)
}

func invalidSubstitutionError(message string, line, column int) *ParseError {
	return parseError("invalid substitution!", message, line, column)
}

func invalidArrayError(message string, line, column int) *ParseError {
	return parseError("invalid config array!", message, line, column)
}

func invalidObjectError(message string, line, column int) *ParseError {
	return parseError("invalid config object!", message, line, column)
}

func invalidKeyError(key string, line, column int) *ParseError {
	return parseError("invalid key!", fmt.Sprintf("%q is a forbidden character in keys", key), line, column)
}

func invalidKeyValueSeparatorError(key, token string, line, column int) *ParseError {
	got := fmt.Sprintf("%q", token)
	if token == "" {
		got = "end of file"
	}

	return parseError("invalid key-value separator!", fmt.Sprintf("expected ':', '=', '{' or '+=' after the key %q, got: %s", key, got), line, column)
}

func invalidValueError(message string, line, column int) *ParseError {
	return parseError("invalid value!", message, line, column)
}

func invalidQuotedStringError(token string, line, column int) *ParseError {
	return parseError("invalid quoted string!", fmt.Sprintf("%s is not a valid JSON string, check its escape sequences", token), line, column)
}

func unclosedMultiLineStringError() *ParseError {
	return parseError("unclosed multi-line string!", "", 0, 0)
}

func missingCommaError(line, column int) *ParseError {
	return parseError("missing comma!", `values should have comma or ASCII newline ('\n') between them`, line, column)
}

func adjacentCommasError(line, column int) *ParseError {
	return parseError("two adjacent commas", "adjacent commas in arrays and objects are invalid!", line, column)
}

func leadingCommaError(line, column int) *ParseError {
	return parseError("leading comma", "leading comma in arrays and objects are invalid!", line, column)
}

func invalidConcatenationError() *ParseError {
	return parseError("invalid concatenation!", "arrays and objects cannot be concatenated with other types", 0, 0)
}
