package hocon

import "fmt"

type ParseError struct {
	line   int
	column int
}

type LeadingPeriodError struct {
	*ParseError
}

func (e *LeadingPeriodError) Error() string {
	return fmt.Sprintf(`leading period '.' at: %d:%d (use quoted "" empty string if you want an empty element)`, e.line, e.column)
}

func leadingPeriodError(line, column int) *LeadingPeriodError {
	return &LeadingPeriodError{ParseError: &ParseError{line: line, column: column}}
}

type TrailingPeriodError struct {
	*ParseError
}

func (t *TrailingPeriodError) Error() string {
	return fmt.Sprintf(`trailing period '.' at: %d:%d (use quoted "" empty string if you want an empty element)`, t.line, t.column)
}

func trailingPeriodError(line, column int) *TrailingPeriodError {
	return &TrailingPeriodError{ParseError: &ParseError{line: line, column: column}}
}

type AdjacentPeriodsError struct {
	*ParseError
}

func (a *AdjacentPeriodsError) Error() string {
	return fmt.Sprintf(`two adjacent periods '.' at: %d:%d (use quoted "" empty string if you want an empty element)`, a.line, a.column)
}

func adjacentPeriodsError(line, column int) *AdjacentPeriodsError {
	return &AdjacentPeriodsError{ParseError: &ParseError{line: line, column: column}}
}

type InvalidSubstitutionError struct {
	*ParseError
	message string
}

func (i *InvalidSubstitutionError) Error() string {
	return fmt.Sprintf("invalid substitution at: %d:%d, %s", i.line, i.column, i.message)
}

func invalidSubstitutionError(message string, line, column int) *InvalidSubstitutionError {
	return &InvalidSubstitutionError{ParseError: &ParseError{line: line, column: column}, message: message}
}

type InvalidConfigArray struct {
	*ParseError
	message string
}

func (i *InvalidConfigArray) Error() string {
	return fmt.Sprintf("invalid config array at: %d:%d, %s", i.line, i.column, i.message)
}

func invalidConfigArray(message string, line, column int) *InvalidConfigArray {
	return &InvalidConfigArray{ParseError: &ParseError{line: line, column: column}, message: message}
}

type InvalidConfigObject struct {
	*ParseError
	message string
}

func (i *InvalidConfigObject) Error() string {
	return fmt.Sprintf("invalid config object at: %d:%d, %s", i.line, i.column, i.message)
}

func invalidConfigObject(message string, line, column int) *InvalidConfigObject {
	return &InvalidConfigObject{ParseError: &ParseError{line: line, column: column}, message: message}
}
