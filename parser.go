package hocon

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"time"
)

const (
	equalsToken      = "="
	commaToken       = ","
	colonToken       = ":"
	dotToken         = "."
	objectStartToken = "{"
	objectEndToken   = "}"
	arrayStartToken  = "["
	arrayEndToken    = "]"
	plusEqualsToken  = "+="
	includeToken     = "include"
	commentToken	 = "#"
)

var forbiddenCharacters = map[string]bool{
	"$": true, `"`: true, objectStartToken: true, objectEndToken: true, arrayStartToken: true, arrayEndToken: true,
	colonToken: true, equalsToken: true, commaToken: true, "+": true, commentToken: true, "`": true, "^": true, "?": true,
	"!": true, "@": true, "*": true, "&": true, `\`: true, "(": true, ")": true,
}

type Parser struct {
	scanner *scanner.Scanner
}

func newParser(src io.Reader) *Parser {
	s := new(scanner.Scanner)
	s.Init(src)
	s.Error = func(*scanner.Scanner, string) {} // do not print errors to stderr
	return &Parser{scanner:s}
}

func ParseString(input string) (*Config, error) {
	parser := newParser(strings.NewReader(input))
	return parser.parse()
}

func ParseResource(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not parse resource: %w", err)
	}
	return newParser(file).parse()
}

func (p *Parser) parse() (*Config, error) {
	p.scanner.Scan()
	if p.scanner.TokenText() == arrayStartToken {
		array, err := p.extractArray()
		if err != nil {
			return nil, err
		}
		return &Config{root: array}, nil
	}

	object, err := p.extractObject()
	if err != nil {
		return nil, err
	}
	if token := p.scanner.TokenText(); token != "" {
		return nil, invalidObjectError("invalid token " + token, p.scanner.Line, p.scanner.Column)
	}
	err = resolveSubstitutions(object)
	if err != nil {
		return nil, err
	}
	return &Config{root: object}, nil
}

func resolveSubstitutions(root Object, valueOptional ...Value) error {
	var value Value
	if valueOptional == nil {
		value = root
	} else {
		value = valueOptional[0]
	}

	switch v := value.(type) {
	case Array:
		for i, value := range v {
			err := processSubstitution(root, value, func(foundValue Value) { v[i] = foundValue })
			if err != nil {
				return err
			}
		}
	case Object:
		for key, value := range v {
			err := processSubstitution(root, value, func(foundValue Value) { v[key] = foundValue })
			if err != nil {
				return err
			}
		}
	default:
		return invalidValueError("substitutions are only allowed in field values and array elements", 0, 0)
	}
	return nil
}

func processSubstitution(root Object, value Value, resolveFunc func(value Value)) error {
	if value.Type() == SubstitutionType {
		substitution := value.(*Substitution)
		if foundValue := root.find(substitution.path); foundValue != nil {
			resolveFunc(foundValue)
		} else if env, ok := os.LookupEnv(substitution.path); ok {
			resolveFunc(String(env))
		} else if !substitution.optional {
			return errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		}
	} else if valueType := value.Type(); valueType == ObjectType || valueType == ArrayType {
		return resolveSubstitutions(root, value)
	}
	return nil
}

func (p *Parser) extractObject() (Object, error) {
	root := Object{}
	parenthesisBalanced := true

	if p.scanner.TokenText() == objectStartToken {
		parenthesisBalanced = false
		p.scanner.Scan()
		if !parenthesisBalanced && p.scanner.TokenText() == objectEndToken {
			parenthesisBalanced = true
			p.scanner.Scan()
			return root, nil
		}
	}
	for tok := p.scanner.Peek(); tok != scanner.EOF; tok = p.scanner.Peek() {
		if p.scanner.TokenText() == commentToken {
			p.consumeComment()
		}

		if p.scanner.TokenText() == includeToken {
			p.scanner.Scan()
			includedObject, err := p.parseIncludedResource()
			if err != nil {
				return nil, err
			}
			mergeObjects(root, includedObject)
			p.scanner.Scan()
		}

		key := p.scanner.TokenText()
		if forbiddenCharacters[key] {
			return nil, invalidKeyError(key, p.scanner.Line, p.scanner.Column)
		}
		if key == dotToken {
			return nil, leadingPeriodError(p.scanner.Line, p.scanner.Column)
		}
		p.scanner.Scan()
		text := p.scanner.TokenText()

		if text == dotToken || text == objectStartToken {
			if text == dotToken {
				p.scanner.Scan() // skip "."
				if p.scanner.TokenText() == dotToken {
					return nil, adjacentPeriodsError(p.scanner.Line, p.scanner.Column)
				}
				if isSeparator(p.scanner.TokenText(), p.scanner.Peek()) {
					return nil, trailingPeriodError(p.scanner.Line, p.scanner.Column - 1)
				}
			}
			object, err := p.extractObject()
			if err != nil {
				return nil, err
			}
			root[key] = object
		}

		switch text {
		case equalsToken, colonToken:
			currentRune := p.scanner.Scan()
			value, err := p.extractValue(currentRune)
			if err != nil {
				return nil, err
			}

			if object, ok := value.(Object); ok {
				if existingObject, ok := root[key].(Object); ok {
					mergeObjects(existingObject, object)
					value = existingObject
				}
			}
			root[key] = value
		case "+" :
			if p.scanner.Peek() == '=' {
				p.scanner.Scan()
				currentRune := p.scanner.Scan()
				err := p.parsePlusEqualsValue(root, key, currentRune)
				if err != nil {
					return nil, err
				}
			}
		}

		if p.scanner.TokenText() == commaToken {
			p.scanner.Scan() // skip ","
		}

		if !parenthesisBalanced && p.scanner.TokenText() == objectEndToken {
			parenthesisBalanced = true
			p.scanner.Scan()
			break
		}
	}

	if !parenthesisBalanced {
		return nil, invalidObjectError("parenthesis do not match", p.scanner.Line, p.scanner.Column)
	}
	return root, nil
}

func mergeObjects(existing Object, new Object) {
	for key, value := range new {
		existingValue, ok := existing[key]
		if ok && existingValue.Type() == ObjectType && value.Type() == ObjectType {
			existingObj := existingValue.(Object)
			mergeObjects(existingObj, value.(Object))
			value = existingObj
		}
		existing[key] = value
	}
}

func (p *Parser) parsePlusEqualsValue(existingObject Object, key string, currentRune rune) error {
	existingValue, ok := existingObject[key]
	if !ok {
		value, err := p.extractValue(currentRune)
		if err != nil {
			return err
		}
		existingObject[key] = Array{value}
	} else {
		if existingValue.Type() != ArrayType {
			return invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", existingValue.String(), key), p.scanner.Line, p.scanner.Pos().Column)
		}
		value, err := p.extractValue(currentRune)
		if err != nil {
			return err
		}
		existingObject[key] = append(existingValue.(Array), value)
	}
	return nil
}

func (p *Parser) validateIncludeValue() (*IncludeToken, error) {
	var required bool
	token := p.scanner.TokenText()
	if token == "required" {
		required = true
		p.scanner.Scan()
		if p.scanner.TokenText() != "(" {
			return nil, invalidValueError("missing opening parenthesis", p.scanner.Line, p.scanner.Column)
		}
		p.scanner.Scan()
		token = p.scanner.TokenText()
	}
	if token == "file" || token == "classpath" {
		p.scanner.Scan()
		if p.scanner.TokenText() != "(" {
			return nil, invalidValueError("missing opening parenthesis", p.scanner.Line, p.scanner.Column)
		}
		p.scanner.Scan()
		path := p.scanner.TokenText()
		p.scanner.Scan()
		if p.scanner.TokenText() != ")" {
			return nil, invalidValueError("missing closing parenthesis", p.scanner.Line, p.scanner.Column)
		}
		token = path
	}

	if required {
		p.scanner.Scan()
		if p.scanner.TokenText() != ")" {
			return nil, invalidValueError("missing closing parenthesis", p.scanner.Line, p.scanner.Column)
		}
	}

	tokenLength := len(token)
	if !strings.HasPrefix(token, `"`) || !strings.HasSuffix(token, `"`) || tokenLength < 2 {
		return nil, invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", p.scanner.Line, p.scanner.Column)
	}
	return &IncludeToken{path: token[1 : tokenLength-1], required: required}, nil // remove double quotes
}

func (p *Parser) parseIncludedResource() (includeObject Object, err error) {
	includeToken, err := p.validateIncludeValue()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(includeToken.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !includeToken.required {
			return Object{}, nil
		}
		return nil, fmt.Errorf("could not parse resource: %w", err)
	}
	includeParser := newParser(file)
	defer func() {
		if closingErr := file.Close(); closingErr != nil {
			err = closingErr
		}
	}()

	includeParser.scanner.Scan()
	if includeParser.scanner.TokenText() == arrayStartToken {
		return nil, invalidValueError("included file cannot contain an array as the root value", p.scanner.Line, p.scanner.Column)
	}

	return includeParser.extractObject()
}

func (p *Parser) extractArray() (Array, error) {
	if firstToken := p.scanner.TokenText(); firstToken != arrayStartToken {
		return nil, invalidArrayError(fmt.Sprintf("%q is not an array start token", firstToken), p.scanner.Line, p.scanner.Column)
	}
	var array Array
	parenthesisBalanced := false
	currentRune := p.scanner.Scan()
	if p.scanner.TokenText() == arrayEndToken { // empty array
		currentRune = p.scanner.Scan()
		return array, nil
	}
	for tok := p.scanner.Peek() ; tok != scanner.EOF; tok = p.scanner.Peek() {
		value, err := p.extractValue(currentRune)
		if err != nil {
			return nil, err
		}
		array = append(array, value)
		if p.scanner.TokenText() == commaToken {
			currentRune = p.scanner.Scan() // skip comma
		}

		if !parenthesisBalanced && p.scanner.TokenText() == arrayEndToken {
			parenthesisBalanced = true
			currentRune = p.scanner.Scan()
			break
		}
	}
	if !parenthesisBalanced {
		return nil, invalidArrayError("parenthesis do not match", p.scanner.Line, p.scanner.Column)
	}
	return array, nil
}

func (p *Parser) extractValue(currentRune rune) (Value, error) {
	token := p.scanner.TokenText()
	if token == commentToken {
		currentRune = p.consumeComment()
		token = p.scanner.TokenText()
	}
	switch currentRune {
	case scanner.Int:
		value, err := strconv.Atoi(token)
		if err != nil {
			return nil, err
		}
		durationUnit := p.extractDurationUnit()
		if durationUnit != 0 {
			p.scanner.Scan()
			return Duration(time.Duration(value) * durationUnit), nil
		}
		return Int(value), nil
	case scanner.Float:
		value, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, err
		}
		durationUnit := p.extractDurationUnit()
		if durationUnit != 0 {
			p.scanner.Scan()
			return Duration(time.Duration(value) * durationUnit), nil
		}
		return Float64(value), nil
	case scanner.String:
		if isMultiLineString(token, p.scanner.Peek()) {
			return p.extractMultiLineString()
		}
		p.scanner.Scan()
		return String(strings.ReplaceAll(token, `"`, "")), nil
	case scanner.Ident:
		switch {
		case token == string(null):
			p.scanner.Scan()
			return null, nil
		case isBooleanString(token):
			p.scanner.Scan()
			return NewBooleanFromString(token), nil
		case isUnquotedString(token):
			p.scanner.Scan()
			return String(token), nil
		}
	default:
		switch {
		case token == objectStartToken:
			return p.extractObject()
		case token == arrayStartToken:
			return p.extractArray()
		case isSubstitution(token, p.scanner.Peek()):
			return p.extractSubstitution()
		}
	}
	return nil, invalidValueError(fmt.Sprintf("unknown value: %q", token), p.scanner.Line, p.scanner.Column)
}

func (p *Parser) extractDurationUnit() time.Duration {
	nextCharacter := p.scanner.Peek()
	p.scanner.Scan()
	if nextCharacter != '\n' && p.scanner.Line == p.scanner.Pos().Line {
		switch p.scanner.TokenText() {
		case "ns", "nano", "nanos", "nanosecond", "nanoseconds":
			return time.Nanosecond
		case "us", "micro", "micros", "microsecond", "microseconds":
			return time.Microsecond
		case "ms", "milli", "millis", "millisecond", "milliseconds":
			return time.Millisecond
		case "s", "second", "seconds":
			return time.Second
		case "m", "minute", "minutes":
			return time.Minute
		case "h", "hour", "hours":
			return time.Hour
		case "d", "day", "days":
			return time.Hour * 24
		}
	}
	return time.Duration(0)
}

func (p *Parser) extractSubstitution() (*Substitution, error) {
	p.scanner.Scan() // skip "$"
	p.scanner.Scan() // skip "{"
	optional := false
	if p.scanner.TokenText() == "?" {
		optional = true
		p.scanner.Scan()
	}
	token := p.scanner.TokenText()
	if token == objectEndToken {
		return nil, invalidSubstitutionError("path expression cannot be empty", p.scanner.Line, p.scanner.Column)
	}
	if token == dotToken {
		return nil, leadingPeriodError(p.scanner.Line, p.scanner.Column)
	}

	var pathBuilder strings.Builder
	parenthesisBalanced := false
	var previousToken string
	for tok := p.scanner.Peek(); tok != scanner.EOF; p.scanner.Peek() {
		if token == commentToken {
			return nil, invalidSubstitutionError("comments are not allowed inside substitutions", p.scanner.Line, p.scanner.Column)
		}
		pathBuilder.WriteString(token)
		p.scanner.Scan()
		token = p.scanner.TokenText()

		if previousToken == dotToken && token == dotToken {
			return nil, adjacentPeriodsError(p.scanner.Line, p.scanner.Column)
		}

		if token == objectEndToken {
			if previousToken == dotToken {
				return nil, trailingPeriodError(p.scanner.Line, p.scanner.Column - 1)
			}
			parenthesisBalanced = true
			p.scanner.Scan()
			break
		}

		if forbiddenCharacters[token] {
			return nil, invalidKeyError(token, p.scanner.Line, p.scanner.Column)
		}

		previousToken = token
	}

	if !parenthesisBalanced {
		return nil, invalidSubstitutionError("missing closing parenthesis", p.scanner.Line, p.scanner.Column)
	}

	return &Substitution{path: pathBuilder.String(), optional:optional}, nil
}

func (p *Parser) consumeComment() rune {
	for token := p.scanner.Peek(); token != '\n' && token != scanner.EOF; token = p.scanner.Peek() {
		p.scanner.Scan()
	}
	currentRune := p.scanner.Scan()
	return currentRune
}

func (p *Parser) extractMultiLineString() (String, error) {
	p.scanner.Next()
	adjacentQuoteCount := 0
	var multiLineBuilder strings.Builder
	for next := p.scanner.Next(); next != scanner.EOF; next = p.scanner.Next() {
		multiLineBuilder.WriteRune(next)
		if next == '"' {
			adjacentQuoteCount++
		} else {
			adjacentQuoteCount = 0
		}
		if adjacentQuoteCount >= 3 && p.scanner.Peek() != '"' {
			break
		}
	}
	if adjacentQuoteCount >= 3 {
		return String(multiLineBuilder.String()[:multiLineBuilder.Len()-3]), nil
	}
	return "", unclosedMultiLineStringError()
}

func isBooleanString(token string) bool {
	return token == "true" || token == "yes" || token == "on" || token == "false" || token == "no" || token == "off"
}

func isSubstitution(token string, peekedToken rune) bool {
	return token == "$" && peekedToken == '{'
}

func isSeparator(token string, peekedToken rune) bool {
	return token == equalsToken || token == colonToken || (token == "+" && peekedToken == '=')
}

func isUnquotedString(token string) bool {
	for forbiddenChar, _ := range forbiddenCharacters {
		if strings.Contains(token, forbiddenChar) {
			return false
		}
	}
	return true
}

func isMultiLineString(token string, peekedToken rune) bool {
	return token == `""` && peekedToken == '"'
}

type IncludeToken struct {
	path     string
	required bool
}