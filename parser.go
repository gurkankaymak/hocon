package hocon

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"
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
	includeToken     = "include"
	commentToken     = "#"
)

var forbiddenCharacters = map[string]bool{
	"$": true, `"`: true, objectStartToken: true, objectEndToken: true, arrayStartToken: true, arrayEndToken: true,
	colonToken: true, equalsToken: true, commaToken: true, "+": true, commentToken: true, "`": true, "^": true, "?": true,
	"!": true, "@": true, "*": true, "&": true, `\`: true, "(": true, ")": true,
}

type parser struct {
	scanner                 *scanner.Scanner
	currentRune             rune
	lastConsumedWhitespaces string // used in concatenation not to lose whitespaces between values
	filepath                string
	rootDir                 string // directory of the top-level parsed file, the classpath() includes are resolved against it
}

func newParser(src io.Reader) *parser {
	s := newScanner(src)
	currWd := "."

	return &parser{scanner: s, filepath: currWd, rootDir: currWd}
}

func newFileParser(src *os.File) *parser {
	s := newScanner(src)

	return &parser{scanner: s, filepath: src.Name(), rootDir: path.Dir(src.Name())}
}

func newScanner(src io.Reader) *scanner.Scanner {
	s := new(scanner.Scanner)
	s.Init(src)
	s.Whitespace ^= 1<<'\t' | 1<<' '                       // do not skip tabs and spaces
	s.Mode &^= scanner.ScanComments | scanner.SkipComments // do not treat the go comments ('//' and '/* */') as comments, hocon comments ('#' and '//') are handled by the parser
	s.Error = func(*scanner.Scanner, string) {}            // do not print errors to stderr
	s.IsIdentRune = func(ch rune, i int) bool {
		return ch == '_' || ch == '-' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
	}

	return s
}

// ParseString function parses the given hocon string, creates the configuration tree and
// returns a pointer to the Config, returns a ParseError if any error occurs while parsing
func ParseString(input string) (*Config, error) {
	parser := newParser(strings.NewReader(input))
	return parser.parse()
}

// ParseStringUnresolved parses the given hocon string like ParseString, but does not
// resolve the substitutions, so that the values of another config (e.g. a fallback config)
// can be used to resolve them later with the Resolve method:
//
//	config, err := hocon.ParseStringUnresolved(mainConfig)
//	...
//	config, err = config.WithFallback(fallbackConfig).Resolve()
func ParseStringUnresolved(input string) (*Config, error) {
	parser := newParser(strings.NewReader(input))
	return parser.parseUnresolved()
}

// ParseResource parses the resource at the given path, creates the configuration tree and
// returns a pointer to the Config, returns the error if any error occurs while parsing
func ParseResource(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not parse resource: %w", err)
	}

	return newFileParser(file).parse()
}

// ParseResourceUnresolved parses the resource at the given path like ParseResource,
// but does not resolve the substitutions, so that the values of another config
// (e.g. a fallback config) can be used to resolve them later with the Resolve method
func ParseResourceUnresolved(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not parse resource: %w", err)
	}

	return newFileParser(file).parseUnresolved()
}

func (p *parser) parse() (*Config, error) {
	config, err := p.parseUnresolved()
	if err != nil {
		return nil, err
	}

	return config.Resolve()
}

func (p *parser) parseUnresolved() (*Config, error) {
	p.advance()

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
		return nil, invalidObjectError("invalid token "+token, p.scanner.Line, p.scanner.Column)
	}

	return &Config{root: object}, nil
}

func (p *parser) advance() {
	p.currentRune = p.scanner.Scan()

	var builder strings.Builder

	for p.currentRune == '\t' || p.currentRune == ' ' {
		builder.WriteRune(p.currentRune)
		p.currentRune = p.scanner.Scan()
	}

	p.lastConsumedWhitespaces = builder.String()
}

func resolveSubstitutions(root Object, valueOptional ...Value) error {
	visitedPaths := make(map[string]bool)

	err := resolveAcyclicSubstitutions(root, visitedPaths, valueOptional...)
	if err != nil {
		return err
	}

	if valueOptional == nil {
		if _, err := normalize(root); err != nil {
			return err
		}
	}

	return nil
}

func resolveAcyclicSubstitutions(root Object, visitedPaths map[string]bool, valueOptional ...Value) error {
	var value Value
	if valueOptional == nil {
		value = root
	} else {
		value = valueOptional[0]
	}

	switch v := value.(type) {
	case Array:
		for i, value := range v {
			err := processSubstitution(root, value, visitedPaths, func(foundValue Value) { v[i] = foundValue })
			if err != nil {
				return err
			}
		}
	case concatenation:
		for i, value := range v {
			err := processSubstitution(root, value, visitedPaths, func(foundValue Value) { v[i] = foundValue })
			if err != nil {
				return err
			}
		}
	case Object:
		for key, value := range v {
			err := processSubstitution(root, value, visitedPaths, func(foundValue Value) { v[key] = foundValue })
			if err != nil {
				return err
			}
		}
	default:
		return invalidValueError("substitutions are only allowed in field values and array elements", 0, 0)
	}

	return nil
}

func processSubstitution(root Object, value Value, visitedPaths map[string]bool, resolveFunc func(value Value)) error {
	if valueType := value.Type(); valueType == SubstitutionType {
		processed, err := processSubstitutionType(root, value.(*Substitution), visitedPaths)
		if err != nil {
			return err
		}
		resolveFunc(processed)
		return nil
	} else if valueType == valueWithAlternativeType {
		withAlternative := value.(*valueWithAlternative)
		if withAlternative.alternative != nil {
			processed, err := processSubstitutionType(root, withAlternative.alternative, visitedPaths)
			if err != nil {
				return err
			}
			if processed != nil {
				resolveFunc(processed)
				return nil
			}
		}
		// the alternative could not be resolved, fall back to the original value,
		// which may itself be or contain a substitution
		resolveFunc(withAlternative.value)
		return processSubstitution(root, withAlternative.value, visitedPaths, resolveFunc)
	} else if valueType == ObjectType || valueType == ArrayType || valueType == ConcatenationType {
		return resolveAcyclicSubstitutions(root, visitedPaths, value)
	}

	return nil
}

func processSubstitutionType(root Object, substitution *Substitution, visitedPaths map[string]bool) (Value, error) {
	if _, ok := visitedPaths[substitution.path]; ok {
		return nil, errors.New("detected substitution cycle: " + substitution.String())
	}

	if foundValue := root.find(substitution.path); foundValue != nil {
		visitedPaths[substitution.path] = true

		if err := processSubstitution(root, foundValue, visitedPaths, func(v Value) { foundValue = v }); err != nil {
			return nil, err
		}

		delete(visitedPaths, substitution.path)

		if foundValue != nil {
			return foundValue, nil
		}
		// the found value is itself an unresolved optional substitution, treat the path as undefined and fall through
	}

	if env, ok := os.LookupEnv(substitution.path); ok {
		return String(env), nil
	}

	if !substitution.optional {
		return nil, errors.New("could not resolve substitution: " + substitution.String() + " to a value")
	}

	return nil, nil
}

// normalize removes the values of the unresolved optional substitutions
// (fields, array elements and concatenation parts whose value is an undefined ${?path})
// from the configuration tree, as the hocon spec requires them to be omitted, and
// resolves the remaining concatenations: objects are merged, arrays are appended
// and string values are flattened into single String values
func normalize(value Value) (Value, error) {
	switch v := value.(type) {
	case Object:
		for key, element := range v {
			resolved, err := normalize(element)
			if err != nil {
				return nil, err
			}

			if resolved == nil {
				delete(v, key)
			} else {
				v[key] = resolved
			}
		}

		return v, nil
	case Array:
		containsNil := false

		for i, element := range v {
			resolved, err := normalize(element)
			if err != nil {
				return nil, err
			}

			if v[i] = resolved; v[i] == nil {
				containsNil = true
			}
		}

		if !containsNil {
			return v, nil
		}

		result := make(Array, 0, len(v))

		for _, element := range v {
			if element != nil {
				result = append(result, element)
			}
		}

		return result, nil
	case concatenation:
		result := make(concatenation, 0, len(v))

		for _, element := range v {
			resolved, err := normalize(element)
			if err != nil {
				return nil, err
			}

			if resolved == nil || resolved == String("") { // empty strings do not contribute to a concatenation
				continue
			}

			result = append(result, resolved)
		}

		switch {
		case len(result) == 0:
			return nil, nil
		case len(result) == 1:
			return result[0], nil
		case result.containsObject():
			return mergeConcatenatedObjects(result)
		case result.containsArray():
			return concatenateArrays(result)
		default:
			return flattenStrings(result), nil
		}
	default:
		return value, nil
	}
}

// mergeConcatenatedObjects merges the objects of the given concatenation into a single
// object (for the same keys the values of the later objects override the earlier ones),
// the whitespaces between the concatenated values are ignored, any other value is invalid
func mergeConcatenatedObjects(concat concatenation) (Value, error) {
	merged := Object{}

	for _, value := range concat {
		if isWhitespaceString(value) {
			continue
		}

		object, ok := value.(Object)
		if !ok {
			return nil, invalidConcatenationError()
		}

		mergeObjects(merged, object)
	}

	return merged, nil
}

// concatenateArrays appends the arrays of the given concatenation into a single array,
// the whitespaces between the concatenated values are ignored, any other value is invalid
func concatenateArrays(concat concatenation) (Value, error) {
	result := Array{}

	for _, value := range concat {
		if isWhitespaceString(value) {
			continue
		}

		array, ok := value.(Array)
		if !ok {
			return nil, invalidConcatenationError()
		}

		result = append(result, array...)
	}

	return result, nil
}

func isWhitespaceString(value Value) bool {
	str, ok := value.(String)
	return ok && strings.TrimSpace(string(str)) == ""
}

// flattenStrings joins the parts of the given concatenation into a single String
// value, as the hocon spec defines the result of a string value concatenation to
// be a string; returns the concatenation as it is if any of its parts is not a
// simple value (e.g. an object or an array)
func flattenStrings(concat concatenation) Value {
	var builder strings.Builder

	for _, element := range concat {
		switch element.(type) {
		case String, Int, Float32, Float64, Boolean, Duration, Null:
			builder.WriteString(rawString(element))
		default:
			return concat
		}
	}

	return String(builder.String())
}

func (p *parser) extractObject(isSubObject ...bool) (Object, error) {
	object := Object{}
	parenthesisBalanced := true

	if p.scanner.TokenText() == objectStartToken {
		parenthesisBalanced = false

		p.advance()

		if !parenthesisBalanced && p.scanner.TokenText() == objectEndToken {
			parenthesisBalanced = true

			p.advance()

			return object, nil
		}
	}

	lastRow := 0

	// the second condition processes the last token before the end of the file, e.g. a trailing key without a value
	for tok := p.scanner.Peek(); tok != scanner.EOF || p.scanner.TokenText() != ""; tok = p.scanner.Peek() {
		if isComment(p.scanner.TokenText(), p.scanner.Peek()) {
			p.consumeComment()
			continue
		}

		if p.scanner.TokenText() == includeToken {
			p.advance()

			includedObject, err := p.parseIncludedResource()
			if err != nil {
				return nil, err
			}

			mergeObjects(object, includedObject)
			p.advance()
			continue
		}

		if !parenthesisBalanced && p.scanner.TokenText() == objectEndToken {
			parenthesisBalanced = true

			p.advance()

			break
		}

		key := p.scanner.TokenText()
		if !strings.HasPrefix(key, `"`) && key != dotToken {
			// glue the tokens that immediately follow, as the scanner splits keys with numeric path segments like ".2g" into multiple tokens
			for isAdjacentKeyRune(p.scanner.Peek()) {
				p.advance()
				key += p.scanner.TokenText()
			}
		}

		key = strings.Trim(key, `"`)
		if strings.HasPrefix(key, dotToken) && key != dotToken {
			key = strings.TrimPrefix(key, dotToken)
		}

		if forbiddenCharacters[key] {
			return nil, invalidKeyError(key, p.scanner.Line, p.scanner.Column)
		}

		if key == dotToken {
			return nil, leadingPeriodError(p.scanner.Line, p.scanner.Column)
		}

		p.advance()
		text := p.scanner.TokenText()

		startsWithDot := strings.HasPrefix(text, dotToken) && text != dotToken
		isNestedObjectPath := text == dotToken || text == objectStartToken || startsWithDot

		if isNestedObjectPath {
			if text == dotToken {
				p.advance() // skip "."

				if p.scanner.TokenText() == dotToken || strings.HasPrefix(p.scanner.TokenText(), dotToken) {
					return nil, adjacentPeriodsError(p.scanner.Line, p.scanner.Column)
				}

				if isSeparator(p.scanner.TokenText(), p.scanner.Peek()) {
					return nil, trailingPeriodError(p.scanner.Line, p.scanner.Column-1)
				}
			}

			lastRow = p.scanner.Line

			extractedObject, err := p.extractObject(true)
			if err != nil {
				return nil, err
			}

			if existingValue, ok := object[key]; ok {
				if existingValue.Type() == ObjectType {
					mergeObjects(existingValue.(Object), extractedObject)
					extractedObject = existingValue.(Object)
				}
			}

			object[key] = extractedObject
		}

		switch text {
		case equalsToken, colonToken:
			p.advance()
			lastRow = p.scanner.Line

			value, err := p.extractValue()
			if err != nil {
				return nil, err
			}

			if existingValue, ok := object[key]; ok {
				if existingValue.Type() == ObjectType && value.Type() == ObjectType {
					mergeObjects(existingValue.(Object), value.(Object))
					value = existingValue
				} else if (existingValue.Type() == SubstitutionType && value.Type() == SubstitutionType) ||
					(existingValue.Type() == ObjectType && value.Type() == SubstitutionType) ||
					(existingValue.Type() == SubstitutionType && value.Type() == ObjectType) {
					value = concatenation{existingValue, value}
				} else if existingValue.Type() == valueWithAlternativeType && value.Type() == SubstitutionType {
					value = &valueWithAlternative{value: existingValue, alternative: value.(*Substitution)}
				} else if value.Type() == SubstitutionType {
					value = &valueWithAlternative{value: existingValue, alternative: value.(*Substitution)}
				}
			}

			object[key] = value
		case "+":
			if p.scanner.Peek() != '=' {
				return nil, invalidKeyValueSeparatorError(key, text, p.scanner.Line, p.scanner.Column)
			}

			p.advance()
			p.advance()

			err := p.parsePlusEqualsValue(object, key)
			if err != nil {
				return nil, err
			}
		default:
			if !isNestedObjectPath { // a key must be followed by a separator or an object
				return nil, invalidKeyValueSeparatorError(key, text, p.scanner.Line, p.scanner.Column)
			}
		}

		for currentRow := p.scanner.Line; currentRow == lastRow && p.scanner.TokenText() != ""; currentRow = p.scanner.Line {
			concatenated, err := p.checkAndConcatenate(object, key)
			if err != nil {
				return nil, err
			}

			if !concatenated {
				break
			}
		}

		if parenthesisBalanced && len(isSubObject) > 0 && isSubObject[0] {
			return object, nil
		}

		for isComment(p.scanner.TokenText(), p.scanner.Peek()) {
			p.consumeComment()
		}

		if p.scanner.Line == lastRow &&
			p.scanner.TokenText() != commaToken &&
			p.scanner.TokenText() != objectEndToken &&
			p.scanner.TokenText() != "" {
			return nil, missingCommaError(p.scanner.Line, p.scanner.Column)
		}

		if p.scanner.TokenText() == commaToken {
			p.advance() // skip ","

			if p.scanner.TokenText() == commaToken {
				return nil, adjacentCommasError(p.scanner.Line, p.scanner.Column)
			}
		}

		if !parenthesisBalanced && p.scanner.TokenText() == objectEndToken {
			parenthesisBalanced = true

			p.advance()

			break
		}
	}

	if !parenthesisBalanced {
		return nil, invalidObjectError("parenthesis do not match", p.scanner.Line, p.scanner.Column)
	}

	return object, nil
}

func mergeObjects(existing Object, new Object) {
	for key, value := range new {
		existingValue, ok := existing[key]
		if ok && existingValue != nil && existingValue.Type() == ObjectType && value != nil &&
			value.Type() == ObjectType {
			existingObj := existingValue.(Object)
			mergeObjects(existingObj, value.(Object))
			value = existingObj
		}
		if value != nil {
			existing[key] = value
		}
	}
}

func (p *parser) parsePlusEqualsValue(existingObject Object, key string) error {
	existingValue, ok := existingObject[key]
	if !ok {
		value, err := p.extractValue()
		if err != nil {
			return err
		}

		existingObject[key] = Array{value}
	} else {
		if existingValue.Type() != ArrayType {
			return invalidValueError(fmt.Sprintf("value: %q of the key: %q is not an array", existingValue.String(), key), p.scanner.Line, p.scanner.Pos().Column)
		}
		value, err := p.extractValue()
		if err != nil {
			return err
		}
		existingObject[key] = append(existingValue.(Array), value)
	}

	return nil
}

func (p *parser) validateIncludeValue() (*include, error) {
	var required, classpath bool

	token := p.scanner.TokenText()
	if token == "required" {
		required = true

		p.advance()

		if p.scanner.TokenText() != "(" {
			return nil, invalidValueError("missing opening parenthesis", p.scanner.Line, p.scanner.Column)
		}

		p.advance()
		token = p.scanner.TokenText()
	}

	if token == "file" || token == "classpath" {
		classpath = token == "classpath"

		p.advance()

		if p.scanner.TokenText() != "(" {
			return nil, invalidValueError("missing opening parenthesis", p.scanner.Line, p.scanner.Column)
		}

		p.advance()
		path := p.scanner.TokenText()
		p.advance()

		if p.scanner.TokenText() != ")" {
			return nil, invalidValueError("missing closing parenthesis", p.scanner.Line, p.scanner.Column)
		}

		token = path
	}

	if required {
		p.advance()

		if p.scanner.TokenText() != ")" {
			return nil, invalidValueError("missing closing parenthesis", p.scanner.Line, p.scanner.Column)
		}
	}

	tokenLength := len(token)
	if !strings.HasPrefix(token, `"`) || !strings.HasSuffix(token, `"`) || tokenLength < 2 {
		return nil, invalidValueError("expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)'", p.scanner.Line, p.scanner.Column)
	}

	return &include{path: token[1 : tokenLength-1], classpath: classpath, required: required}, nil // remove double quotes
}

func (p *parser) parseIncludedResource() (Object, error) {
	includeToken, err := p.validateIncludeValue()
	if err != nil {
		return nil, err
	}

	baseDir := path.Dir(p.filepath)
	if includeToken.classpath {
		// the classpath() includes are resolved against the directory of the top-level parsed file
		// (as the go equivalent of the java classpath root) instead of the directory of the including file
		baseDir = p.rootDir
	}

	includePath := path.Join(baseDir, includeToken.path)

	includePaths := []string{includePath}
	if path.Ext(includePath) == "" {
		// an include without a file extension also includes the .json and .conf versions of the file, the values of the .conf version override the .json ones
		includePaths = append(includePaths, includePath+".json", includePath+".conf")
	}

	includedObject := Object{}
	found := false

	var notExistErr error

	for _, includePath := range includePaths {
		object, err := p.parseIncludedFile(includePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if notExistErr == nil {
					notExistErr = err
				}

				continue
			}

			return nil, err
		}

		found = true

		mergeObjects(includedObject, object)
	}

	if !found && includeToken.required {
		return nil, notExistErr
	}

	return includedObject, nil
}

// parseIncludedFile parses the file at the given path into an Object, inheriting the
// root directory of the current parser, the returned error wraps os.ErrNotExist if
// the file does not exist
func (p *parser) parseIncludedFile(includePath string) (includedObject Object, err error) {
	file, err := os.Open(includePath)
	if err != nil {
		return nil, fmt.Errorf("could not parse resource: %w", err)
	}

	defer func() {
		if closingErr := file.Close(); closingErr != nil {
			err = closingErr
		}
	}()

	includeParser := newFileParser(file)
	includeParser.rootDir = p.rootDir
	includeParser.advance()

	if includeParser.scanner.TokenText() == arrayStartToken {
		return nil, invalidValueError("included file cannot contain an array as the root value", p.scanner.Line, p.scanner.Column)
	}

	return includeParser.extractObject()
}

func (p *parser) checkAndConcatenate(object Object, key string) (bool, error) {
	if lastValue, ok := object[key]; ok && p.canConcatenate(lastValue, p.scanner.TokenText(), p.scanner.Peek()) {
		lastConsumedWhitespaces := p.lastConsumedWhitespaces

		value, err := p.extractValue()
		if err != nil {
			return false, err
		}

		if lastValue.Type() == ConcatenationType {
			object[key] = append(lastValue.(concatenation), String(lastConsumedWhitespaces), value)
		} else {
			object[key] = concatenation{lastValue, String(lastConsumedWhitespaces), value}
		}

		return true, nil
	}

	return false, nil
}

func (p *parser) checkConcatenation(lastValue Value) (Value, error) {
	if p.canConcatenate(lastValue, p.scanner.TokenText(), p.scanner.Peek()) {
		lastConsumedWhitespaces := p.lastConsumedWhitespaces

		value, err := p.extractValue()
		if err != nil {
			return nil, err
		}

		if lastValue.Type() == ConcatenationType {
			return append(lastValue.(concatenation), String(lastConsumedWhitespaces), value), nil
		} else {
			return concatenation{lastValue, String(lastConsumedWhitespaces), value}, nil
		}
	}

	return nil, nil
}

func (p *parser) extractArray() (Array, error) {
	if firstToken := p.scanner.TokenText(); firstToken != arrayStartToken {
		return nil, invalidArrayError(fmt.Sprintf("%q is not an array start token", firstToken), p.scanner.Line, p.scanner.Column)
	}

	p.advance()

	token := p.scanner.TokenText()
	if token == commaToken {
		return nil, leadingCommaError(p.scanner.Line, p.scanner.Column)
	}

	var array Array

	if token == arrayEndToken { // empty array
		p.advance()
		return array, nil
	}

	parenthesisBalanced := false
	lastRow := 0

	for tok := p.scanner.Peek(); tok != scanner.EOF; tok = p.scanner.Peek() {
		lastRow = p.scanner.Line

		value, err := p.extractValue()
		if err != nil {
			return nil, err
		}

		token = p.scanner.TokenText()
		if isComment(token, p.scanner.Peek()) {
			p.consumeComment()
			token = p.scanner.TokenText()
		}

		if p.scanner.Line == lastRow && token != commaToken && token != arrayEndToken {
			concatenatedValue, err := p.checkConcatenation(value)
			if err != nil {
				return nil, err
			}
			if concatenatedValue == nil {
				return nil, missingCommaError(p.scanner.Line, p.scanner.Column)
			} else {
				lastValue := concatenatedValue
				token = p.scanner.TokenText()
				for concatenatedValue != nil && token != commaToken && token != arrayEndToken {
					concatenatedValue, err = p.checkConcatenation(lastValue)
					if err != nil {
						return nil, err
					}
					if concatenatedValue != nil {
						lastValue = concatenatedValue
					} else {
						break
					}
					token = p.scanner.TokenText()
				}
				array = append(array, lastValue)
			}
		} else {
			array = append(array, value)
		}

		if p.scanner.TokenText() == commaToken {
			p.advance() // skip comma

			token = p.scanner.TokenText()

			if isComment(token, p.scanner.Peek()) {
				p.consumeComment()
				token = p.scanner.TokenText()
			}

			if token == commaToken {
				return nil, adjacentCommasError(p.scanner.Line, p.scanner.Column)
			}
		}

		if !parenthesisBalanced && token == arrayEndToken {
			parenthesisBalanced = true

			p.advance()

			break
		}
	}

	if !parenthesisBalanced {
		return nil, invalidArrayError("parenthesis do not match", p.scanner.Line, p.scanner.Column)
	}

	return array, nil
}

func (p *parser) extractValue() (Value, error) {
	token := p.scanner.TokenText()
	if isComment(token, p.scanner.Peek()) {
		p.consumeComment()
		token = p.scanner.TokenText()
	}

	switch p.currentRune {
	case scanner.Int:
		if glued := p.glueAdjacent(token); glued != token {
			p.advance()
			return numberLedValue(glued), nil
		}

		value, err := strconv.Atoi(token)
		if err != nil {
			return nil, err
		}

		durationUnit := p.extractDurationUnit()
		if durationUnit != 0 {
			p.advance()
			return Duration(time.Duration(value) * durationUnit), nil
		}

		return Int(value), nil
	case scanner.Float:
		if glued := p.glueAdjacent(token); glued != token {
			p.advance()
			return numberLedValue(glued), nil
		}

		value, err := strconv.ParseFloat(token, 64)
		if err != nil {
			if isUnquotedString(token) {
				p.advance()
				return String(token), nil
			} else {
				return nil, err
			}
		}

		durationUnit := p.extractDurationUnit()
		if durationUnit != 0 {
			p.advance()
			return Duration(time.Duration(value * float64(durationUnit))), nil
		}

		return Float64(value), nil
	case scanner.String:
		if isMultiLineString(token, p.scanner.Peek()) {
			return p.extractMultiLineString()
		}

		p.advance()

		return String(strings.Trim(token, `"`)), nil
	case scanner.Ident:
		token = p.glueAdjacent(token)

		switch {
		case token == string(null):
			p.advance()
			return null, nil
		case isBooleanString(token):
			p.advance()
			return newBooleanFromString(token), nil
		case isUnquotedString(token):
			p.advance()
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
		case isUnquotedString(token):
			p.advance()
			return String(token), nil
		}
	}

	return nil, invalidValueError(fmt.Sprintf("unknown value: %q", token), p.scanner.Line, p.scanner.Column)
}

func (p *parser) extractDurationUnit() time.Duration {
	nextCharacter := p.scanner.Peek()
	p.advance()

	if nextCharacter != '\n' && p.scanner.Line == p.scanner.Pos().Line {
		return durationUnitOf(p.scanner.TokenText())
	}

	return time.Duration(0)
}

func durationUnitOf(text string) time.Duration {
	switch text {
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

	return time.Duration(0)
}

// glueAdjacent glues the tokens that immediately follow the given token (without
// any whitespace in between) into a single unquoted string run, as the scanner
// splits values like "2.2.0" or "bar10.0" into multiple tokens
func (p *parser) glueAdjacent(token string) string {
	if !isAdjacentValueRune(p.scanner.Peek()) {
		return token
	}

	var builder strings.Builder

	builder.WriteString(token)

	for isAdjacentValueRune(p.scanner.Peek()) {
		p.advance()
		builder.WriteString(p.scanner.TokenText())
	}

	return builder.String()
}

// numberLedValue converts the given unquoted string run that starts with a number
// into an Int, Float64 or Duration if the whole text forms one, otherwise returns
// it as a String (e.g. version numbers like "2.2.0" or "1.2.3-SNAPSHOT")
func numberLedValue(text string) Value {
	if value, err := strconv.Atoi(text); err == nil {
		return Int(value)
	}

	if value, err := strconv.ParseFloat(text, 64); err == nil {
		return Float64(value)
	}

	if duration, ok := durationOf(text); ok {
		return duration
	}

	return String(text)
}

// durationOf converts texts like "10s" or "1.5hours" to a Duration
func durationOf(text string) (Duration, bool) {
	unitStart := len(text)
	for unitStart > 0 && unicode.IsLetter(rune(text[unitStart-1])) {
		unitStart--
	}

	if unitStart == 0 || unitStart == len(text) {
		return 0, false
	}

	unit := durationUnitOf(text[unitStart:])
	if unit == 0 {
		return 0, false
	}

	if value, err := strconv.Atoi(text[:unitStart]); err == nil {
		return Duration(time.Duration(value) * unit), true
	}

	if value, err := strconv.ParseFloat(text[:unitStart], 64); err == nil {
		return Duration(time.Duration(value * float64(unit))), true
	}

	return 0, false
}

func isAdjacentValueRune(ch rune) bool {
	return ch == '.' || isAdjacentKeyRune(ch)
}

func isAdjacentKeyRune(ch rune) bool {
	return ch == '-' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}

func (p *parser) extractSubstitution() (*Substitution, error) {
	p.advance() // skip "$"
	p.advance() // skip "{"

	optional := false
	if p.scanner.TokenText() == "?" {
		optional = true

		p.advance()
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
		if isComment(token, p.scanner.Peek()) {
			return nil, invalidSubstitutionError("comments are not allowed inside substitutions", p.scanner.Line, p.scanner.Column)
		}

		pathBuilder.WriteString(token)
		p.advance()
		token = p.scanner.TokenText()

		if previousToken == dotToken && token == dotToken {
			return nil, adjacentPeriodsError(p.scanner.Line, p.scanner.Column)
		}

		if token == objectEndToken {
			if previousToken == dotToken {
				return nil, trailingPeriodError(p.scanner.Line, p.scanner.Column-1)
			}

			parenthesisBalanced = true

			p.advance()

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

	return &Substitution{path: pathBuilder.String(), optional: optional}, nil
}

func (p *parser) consumeComment() {
	for token := p.scanner.Peek(); token != '\n' && token != scanner.EOF && !strings.HasSuffix(p.scanner.TokenText(), "\n"); token = p.scanner.Peek() {
		p.advance()
	}
	p.advance()
}

func (p *parser) extractMultiLineString() (String, error) {
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

func (p *parser) isTokenConcatenable(currentText string, peeked rune) bool {
	return !isComment(currentText, peeked) &&
		(isSubstitution(currentText, peeked) ||
			isUnquotedString(currentText) ||
			(p.currentRune == scanner.String && !isMultiLineString(currentText, peeked)))
}

type concatenationCategory int

const (
	concatenatesNothing concatenationCategory = iota
	concatenatesSimpleValues
	concatenatesObjects
	concatenatesArrays
	concatenatesAnything
)

// concatenationCategoryOf returns the kind of values the given value can be concatenated
// with, as the hocon spec defines: objects concatenate with objects, arrays with arrays and
// simple values with simple values; substitutions concatenate with anything, as their type
// is only known after the resolution
func concatenationCategoryOf(value Value) concatenationCategory {
	switch v := value.(type) {
	case Object:
		return concatenatesObjects
	case Array:
		return concatenatesArrays
	case *Substitution:
		return concatenatesAnything
	case concatenation:
		category := concatenatesAnything

		for _, element := range v {
			switch element := element.(type) {
			case Object:
				return concatenatesObjects
			case Array:
				return concatenatesArrays
			case *Substitution:
			case String:
				if strings.TrimSpace(string(element)) != "" { // ignore the whitespaces between the concatenated values
					category = concatenatesSimpleValues
				}
			default:
				category = concatenatesSimpleValues
			}
		}

		return category
	default:
		if value.isConcatenable() {
			return concatenatesSimpleValues
		}

		return concatenatesNothing
	}
}

// canConcatenate reports whether the given value can be concatenated with the value
// that starts at the current token
func (p *parser) canConcatenate(lastValue Value, token string, peeked rune) bool {
	switch concatenationCategoryOf(lastValue) {
	case concatenatesObjects:
		return token == objectStartToken || isSubstitution(token, peeked)
	case concatenatesArrays:
		return token == arrayStartToken || isSubstitution(token, peeked)
	case concatenatesSimpleValues:
		return p.isTokenConcatenable(token, peeked)
	case concatenatesAnything:
		return token == objectStartToken || token == arrayStartToken || p.isTokenConcatenable(token, peeked)
	default:
		return false
	}
}

func isBooleanString(token string) bool {
	return token == "true" || token == "yes" || token == "on" || token == "false" || token == "no" || token == "off"
}

func isSubstitution(token string, peekedToken rune) bool {
	return token == "$" && peekedToken == '{'
}

func isComment(token string, peekedToken rune) bool {
	return token == commentToken || (token == "/" && peekedToken == '/')
}

func isSeparator(token string, peekedToken rune) bool {
	return token == equalsToken || token == colonToken || (token == "+" && peekedToken == '=')
}

func isUnquotedString(token string) bool {
	for forbiddenChar := range forbiddenCharacters {
		if strings.Contains(token, forbiddenChar) {
			return false
		}
	}

	return true
}

func isMultiLineString(token string, peekedToken rune) bool {
	return token == `""` && peekedToken == '"'
}

type include struct {
	path      string
	classpath bool
	required  bool
}
