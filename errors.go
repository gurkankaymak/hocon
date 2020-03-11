package hocon

import "fmt"

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

func invalidValueError(message string, line, column int) *ParseError {
	return parseError("invalid value!", message, line, column)
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
