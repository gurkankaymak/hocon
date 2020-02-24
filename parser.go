package hocon

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
)

//type TokenType string

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
)

var forbiddenCharacters = map[string]bool{
	"$": true, `"`: true, objectStartToken: true, objectEndToken: true, arrayStartToken: true, arrayEndToken: true,
	colonToken: true, equalsToken: true, commaToken: true, "+": true, "#": true, "`": true, "^": true, "?": true,
	"!": true, "@": true, "*": true, "&": true, `\`: true, "(": true, ")": true,
}

type Parser struct {
	s *scanner.Scanner
}

func newParser(src io.Reader) *Parser {
	return &Parser{new(scanner.Scanner).Init(src)}
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
	p.s.Scan()
	if p.s.TokenText() == arrayStartToken {
		configArray, err := p.extractConfigArray()
		if err != nil {
			return nil, err
		}
		return &Config{root:configArray}, nil
	}

	configObject, err := p.extractConfigObject()
	if err != nil {
		return nil, err
	}
	err = resolveSubstitutions(configObject)
	if err != nil {
		return nil, err
	}
	return &Config{root:configObject}, nil
}

func resolveSubstitutions(root *ConfigObject, configValueOptional ...ConfigValue) error {
	var configValue ConfigValue
	if configValueOptional == nil {
		configValue = root
	} else {
		configValue = configValueOptional[0]
	}

	switch v := configValue.(type) {
	case *ConfigArray:
		for i, value := range v.values {
			err := processSubstitution(root, value, func(foundValue ConfigValue) { v.values[i] = foundValue })
			if err != nil {
				return err
			}
		}
	case *ConfigObject:
		for key, value := range v.items {
			err := processSubstitution(root, value, func(foundValue ConfigValue) { v.items[key] = foundValue })
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("invalid type for substitution, substitutions are only allowed in field values and array elements")
	}
	return nil
}

func processSubstitution(root *ConfigObject, value ConfigValue, resolveFunc func(configValue ConfigValue)) error {
	if value.ValueType() == ValueTypeSubstitution {
		substitution := value.(*Substitution)
		if foundValue := root.find(substitution.path); foundValue != nil {
			resolveFunc(foundValue)
		} else if !substitution.optional {
			return errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		}
	} else if valueType := value.ValueType(); valueType == ValueTypeObject || valueType == ValueTypeArray {
		return resolveSubstitutions(root, value)
	}
	return nil
}

func (p *Parser) extractConfigObject() (*ConfigObject, error) {
	root := map[string]ConfigValue{}
	parenthesisBalanced := true

	if p.s.TokenText() == objectStartToken { // skip if current text is "{"
		parenthesisBalanced = false
		p.s.Scan()
		if !parenthesisBalanced && p.s.TokenText() == objectEndToken {
			parenthesisBalanced = true
			p.s.Scan()
			return NewConfigObject(root), nil
		}
	}
	for tok := p.s.Peek(); tok != scanner.EOF; tok = p.s.Peek() {
		if p.s.TokenText() == includeToken {
			p.s.Scan()
			includePath, err := p.validateIncludeValue(p.s.TokenText())
			if err != nil {
				return nil, err
			}
			includedConfigObject, err := parseIncludedResource(includePath)
			if err != nil {
				return nil, err
			}
			mergeConfigObjects(root, includedConfigObject)
			p.s.Scan()
		}

		key := p.s.TokenText()
		if forbiddenCharacters[key] {
			return nil, errors.New("invalid key! '" + key + "' is a forbidden character in keys")
		}
		if key == dotToken {
			return nil, leadingPeriodError(p.s.Position.Line, p.s.Position.Column)
		}
		p.s.Scan()
		text := p.s.TokenText()

		if text == dotToken || text == objectStartToken {
			if text == dotToken {
				p.s.Scan() // skip "."
				if p.s.TokenText() == dotToken {
					return nil, adjacentPeriodsError(p.s.Position.Line, p.s.Position.Column)
				}
				if isSeparator(p.s.TokenText(), p.s.Peek()) {
					return nil, trailingPeriodError(p.s.Position.Line, p.s.Position.Column)
				}
			}
			configObject, err := p.extractConfigObject()
			if err != nil {
				return nil, err
			}

			if !parenthesisBalanced && p.s.TokenText() == objectEndToken {
				parenthesisBalanced = true
				p.s.Scan()
			}

			if !parenthesisBalanced {
				return nil, invalidConfigObject("parenthesis do not match", p.s.Position.Line, p.s.Position.Column)
			}
			return NewConfigObject(map[string]ConfigValue{key:configObject}), nil
		}

		switch text {
		case equalsToken, colonToken:
			p.s.Scan()
			configValue, err := p.extractConfigValue()
			if err != nil {
				return nil, err
			}

			if configObject, ok := configValue.(*ConfigObject); ok {
				if existingConfigObject, ok := root[key].(*ConfigObject); ok {
					mergeConfigObjects(existingConfigObject.items, configObject)
					configValue = existingConfigObject
				}
			}
			root[key] = configValue
		case "+" :
			if p.s.Peek() == '=' {
				p.s.Scan()
				p.s.Scan()
				err := p.parsePlusEqualsValue(root, key)
				if err != nil {
					return nil, err
				}
			}
		}

		if p.s.TokenText() == commaToken {
			p.s.Scan() // skip ","
		}

		if !parenthesisBalanced && p.s.TokenText() == objectEndToken {
			parenthesisBalanced = true
			p.s.Scan()
			return NewConfigObject(root), nil
		}
	}

	if !parenthesisBalanced {
		return nil, invalidConfigObject("parenthesis do not match", p.s.Position.Line, p.s.Position.Column)
	}
	return NewConfigObject(root), nil
}

func mergeConfigObjects(existingItems map[string]ConfigValue, new *ConfigObject) {
	for key, value := range new.items {
		existingValue, ok := existingItems[key]
		if ok && existingValue.ValueType() == ValueTypeObject && value.ValueType() == ValueTypeObject {
			existingObj := existingValue.(*ConfigObject)
			mergeConfigObjects(existingObj.items, value.(*ConfigObject))
			value = existingObj
		}
		existingItems[key] = value
	}
}

func (p *Parser) parsePlusEqualsValue(existingItems map[string]ConfigValue, key string) error {
	existing, ok := existingItems[key]
	if !ok {
		configValue, err := p.extractConfigValue()
		if err != nil {
			return err
		}
		existingItems[key] = NewConfigArray([]ConfigValue{configValue})
	} else {
		existingArray, ok := existing.(*ConfigArray)
		if !ok {
			return errors.New("value of the key: " + key + " is not an array")
		}
		configValue, err := p.extractConfigValue()
		if err != nil {
			return err
		}
		existingArray.Append(configValue)
	}
	return nil
}

func (p *Parser) validateIncludeValue(includeValue string) (string, error) {
	if includeValue == "file" || includeValue == "classpath" {
		p.s.Scan()
		if p.s.TokenText() != "(" {
			return "", errors.New("invalid include value! missing opening parenthesis")
		}
		p.s.Scan()
		path := p.s.TokenText()
		p.s.Scan()
		if p.s.TokenText() != ")" {
			return "", errors.New("invalid include value! missing closing parenthesis")
		}
		includeValue = path
	}
	return includeValue, nil
}

func parseIncludedResource(path string) (*ConfigObject, error) {
	if !strings.HasPrefix(path, `"`) {
		return nil, errors.New(`invalid include value! expected quoted string, optionally wrapped in 'file(...)' or 'classpath(...)' `)
	}
	file, err := os.Open(strings.ReplaceAll(path, `"`, ""))
	if errors.Is(err, os.ErrNotExist) {
		return NewConfigObject(map[string]ConfigValue{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not parse resource: %w", err)
	}
	parser := newParser(file)
	defer file.Close()
	parser.s.Scan()

	if parser.s.TokenText() == arrayStartToken {
		return nil, errors.New("invalid included file! included file cannot contain an array as the root value")
	}

	return parser.extractConfigObject()
}

func (p *Parser) extractConfigArray() (*ConfigArray, error) {
	var values []ConfigValue
	if p.s.TokenText() != arrayStartToken {
		return nil, invalidConfigArray("not an array start token", p.s.Position.Line, p.s.Position.Column)
	}
	parenthesisBalanced := false
	p.s.Scan() // skip "["
	if p.s.TokenText() == arrayEndToken { // empty array
		p.s.Scan()
		return NewConfigArray(values), nil
	}
	for tok := p.s.Peek() ; tok != scanner.EOF; tok = p.s.Peek() {
		configValue, err := p.extractConfigValue()
		if err != nil {
			return nil, err
		}
		if configValue != nil {
			values = append(values, configValue)
		}
		if p.s.TokenText() == commaToken {
			p.s.Scan() // skip comma
		}

		if !parenthesisBalanced && p.s.TokenText() == arrayEndToken {  // skip "]"
			parenthesisBalanced = true
			p.s.Scan()
			return NewConfigArray(values), nil
		}
	}
	if parenthesisBalanced {
		return NewConfigArray(values), nil
	}
	return nil, invalidConfigArray("parenthesis does not match", p.s.Position.Line, p.s.Position.Column)
}

func (p *Parser) extractConfigValue() (ConfigValue, error) {
	// TODO gk: int, float32, bool cases parse two times
	token := p.s.TokenText()
	switch {
	case isTokenString(token):
		p.s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigString(strings.ReplaceAll(token, "\"", "")), nil
	case isConfigObject(token):
		return p.extractConfigObject()
	case isConfigArray(token):
		return p.extractConfigArray()
	case isTokenInt(token):
		value, _ := strconv.Atoi(token)
		p.s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigInt(value), nil
	case isTokenFloat32(token):
		value, _ := strconv.ParseFloat(token, 32)
		p.s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigFloat32(float32(value)), nil
	case isTokenBoolean(token):
		value, _ := strconv.ParseBool(token)
		p.s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigBoolean(value), nil
	case isTokenBooleanString(token):
		p.s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigBooleanFromString(token), nil
	case isSubstitution(token, p.s.Peek()):
		return p.extractSubstitution()
	}
	return nil, errors.New("unknown config value: " + token)
}

func (p *Parser) extractSubstitution() (*Substitution, error) {
	p.s.Scan() // skip "$"
	p.s.Scan() // skip "{"
	optional := false
	if p.s.TokenText() == "?" {
		optional = true
		p.s.Scan()
	}
	var pathBuilder strings.Builder
	parenthesisBalanced := false
	var previousToken string
	for tok := p.s.Peek(); tok != scanner.EOF; p.s.Peek() {
		pathBuilder.WriteString(p.s.TokenText())
		p.s.Scan()
		text := p.s.TokenText()

		if previousToken == dotToken && text == dotToken {
			return nil, adjacentPeriodsError(p.s.Position.Line, p.s.Position.Column)
		}

		if text == "}" {
			parenthesisBalanced = true
			p.s.Scan()
			break
		}

		if text != dotToken && len(text) == 1 && !unicode.IsLetter(rune(text[0])) {
			break
		}

		previousToken = text
	}

	if !parenthesisBalanced {
		return nil, invalidSubstitutionError("missing closing parenthesis", p.s.Position.Line, p.s.Position.Column)
	}

	substitutionPath := pathBuilder.String()
	if len(substitutionPath) == 0 {
		return nil, invalidSubstitutionError("path expression cannot be empty", p.s.Position.Line, p.s.Position.Column)
	}
	if strings.HasPrefix(substitutionPath, dotToken) {
		return nil, leadingPeriodError(p.s.Position.Line, p.s.Position.Column)
	}
	if strings.HasSuffix(substitutionPath, dotToken) {
		return nil, trailingPeriodError(p.s.Position.Line, p.s.Position.Column)
	}
	return &Substitution{path: substitutionPath, optional:optional}, nil
}

func isTokenString(token string) bool  { return strings.HasPrefix(token, `"`) }
func isConfigObject(token string) bool { return token == objectStartToken }
func isConfigArray(token string) bool  { return token == arrayStartToken }

func isTokenInt(token string) bool {
	_, err := strconv.Atoi(token)
	return err == nil
}

func isTokenFloat32(token string) bool {
	_, err := strconv.ParseFloat(token, 32)
	return err == nil
}

func isTokenBoolean(token string) bool {
	_, err := strconv.ParseBool(token)
	return err == nil
}

func isTokenBooleanString(token string) bool {
	return token == "true" || token == "yes" || token == "on" || token == "false" || token == "no" || token == "off"
}

func isSubstitution(token string, peekedToken rune) bool {
	return token == "$" && peekedToken == '{'
}

func isSeparator(token string, peekedToken rune) bool {
	return token == equalsToken || token == colonToken || (token == "+" && peekedToken == '=')
}