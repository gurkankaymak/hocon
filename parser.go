package hocon

import (
	"errors"
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
)

var s scanner.Scanner

func Parse(input string) (*Config, error) {
	s.Init(strings.NewReader(input))
	s.Scan()
	if s.TokenText() == arrayStartToken {
		configArray, err := extractConfigArray()
		if err != nil {
			return nil, err
		}
		return &Config{root:configArray}, nil
	}

	configObject, err := extractConfigObject()
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
		foundValue := root.find(substitution.path)
		if foundValue != nil {
			resolveFunc(foundValue)
		} else if !substitution.optional {
			return errors.New("could not resolve substitution: " + substitution.String() + " to a value")
		}
	} else if valueType := value.ValueType(); valueType == ValueTypeObject || valueType == ValueTypeArray {
		err := resolveSubstitutions(root, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func extractConfigObject() (*ConfigObject, error) {
	root := map[string]ConfigValue{}
	parenthesisBalanced := true

	if s.TokenText() == objectStartToken { // skip if current text is "{"
		parenthesisBalanced = false
		s.Scan()
	}
	for tok := s.Peek(); tok != scanner.EOF; tok = s.Peek() {
		if !parenthesisBalanced && s.TokenText() == objectEndToken {  // skip "}"
			parenthesisBalanced = true
			s.Scan()
			return NewConfigObject(root), nil
		}

		if s.TokenText() == commaToken {
			s.Scan() // skip ","
		}

		key := s.TokenText()
		if key == dotToken {
			return nil, errors.New(`invalid path, leading period '.' (use quoted "" empty string if you want an empty element)`)
		}
		s.Scan()
		text := s.TokenText()

		if text == dotToken {
			s.Scan() // skip "."
			if s.TokenText() == dotToken {
				return nil, errors.New(`invalid path, two adjacent periods '.' (use quoted "" empty string if you want an empty element)`)
			}
			if checkSeparator(s.TokenText()) != "" {
				return nil, errors.New(`invalid path, trailing period '.' (use quoted "" empty string if you want an empty element)`)
			}
			configObject, err := extractConfigObject()
			if err != nil {
				return nil, err
			}

			if !parenthesisBalanced && s.TokenText() == objectEndToken {
				parenthesisBalanced = true
				s.Scan()
			}

			if !parenthesisBalanced {
				return nil, errors.New("invalid config object, parenthesis does not match")
			}
			return NewConfigObject(map[string]ConfigValue{key:configObject}), nil
		}

		switch checkSeparator(text) {
		case equalsToken, colonToken:
			configValue, err := extractConfigValue()
			if err != nil {
				return nil, err
			}

			if configObject, ok := configValue.(*ConfigObject); ok {
				if existingConfigObject, ok := root[key].(*ConfigObject); ok {
					mergedObject := mergeConfigObjects(existingConfigObject, configObject)
					configValue = mergedObject
				}
			}
			root[key] = configValue
		case plusEqualsToken:
			existing, ok := root[key]
			if !ok {
				configValue, err := extractConfigValue()
				if err != nil {
					return nil, err
				}
				root[key] = NewConfigArray([]ConfigValue{configValue})
			} else {
				existingArray, ok := existing.(*ConfigArray)
				if !ok {
					return nil, errors.New("value of the key: " + key + " is not an array")
				}
				configValue, err := extractConfigValue()
				if err != nil {
					return nil, err
				}
				existingArray.Append(configValue)
			}
		}

		if !parenthesisBalanced && s.TokenText() == objectEndToken {  // skip "}"
			parenthesisBalanced = true
			s.Scan()
		}

		if parenthesisBalanced {
			return NewConfigObject(root), nil
		}
	}

	if !parenthesisBalanced {
		return nil, errors.New("invalid config object, parenthesis does not match")
	}
	return NewConfigObject(root), nil
}

func mergeConfigObjects(existing, new *ConfigObject) *ConfigObject {
	for key, value := range new.items {
		existingValue, ok := existing.items[key]
		if ok && existingValue.ValueType() == ValueTypeObject && value.ValueType() == ValueTypeObject {
			mergedObject := mergeConfigObjects(existingValue.(*ConfigObject), value.(*ConfigObject))
			value = mergedObject
		}
		existing.items[key] = value
	}
	return existing
}

func checkSeparator(token string) string {
	switch token {
	case equalsToken, colonToken:
		s.Scan()
		return equalsToken
	case "+":
		if s.Peek() == '=' {
			s.Scan()
			s.Scan()
			return plusEqualsToken
		}
		return ""
	default:
		return ""
	}
}

func extractConfigArray() (*ConfigArray, error) {
	var values []ConfigValue
	if s.TokenText() != arrayStartToken {
		return nil, errors.New("invalid config array")
	}
	parenthesisBalanced := false
	s.Scan() // skip "["
	if s.TokenText() == arrayEndToken { // empty array
		s.Scan()
		return NewConfigArray(values), nil
	}
	for tok := s.Peek() ; tok != scanner.EOF; tok = s.Peek() {
		configValue, err := extractConfigValue()
		if err != nil {
			return nil, err
		}
		if configValue != nil {
			values = append(values, configValue)
		}
		if s.TokenText() == commaToken {
			s.Scan() // skip comma
		}

		if !parenthesisBalanced && s.TokenText() == arrayEndToken {  // skip "]"
			parenthesisBalanced = true
			s.Scan()
			return NewConfigArray(values), nil
		}
	}
	if parenthesisBalanced {
		return NewConfigArray(values), nil
	}
	return nil, errors.New("invalid config array, parenthesis does not match")
}

func extractConfigValue() (ConfigValue, error) {
	// TODO gk: int, float32, bool cases parse two times
	token := s.TokenText()
	switch {
	case isTokenString(token):
		s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigString(strings.ReplaceAll(token, "\"", "")), nil
	case isConfigObject(token):
		return extractConfigObject()
	case isConfigArray(token):
		return extractConfigArray()
	case isTokenInt(token):
		value, _ := strconv.Atoi(token)
		s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigInt(value), nil
	case isTokenFloat32(token):
		value, _ := strconv.ParseFloat(token, 32)
		s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigFloat32(float32(value)), nil
	case isTokenBoolean(token):
		value, _ := strconv.ParseBool(token)
		s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigBoolean(value), nil
	case isTokenBooleanString(token):
		s.Scan() // advance the scanner to next token after extracting the value
		return NewConfigBooleanFromString(token), nil
	case isSubstitution(token):
		return extractSubstitution()
	}
	return nil, errors.New("unknown config value: " + token)
}

func extractSubstitution() (*Substitution, error) {
	s.Scan() // skip "$"
	s.Scan() // skip "{"
	optional := false
	if s.TokenText() == "?" {
		optional = true
		s.Scan()
	}
	var pathBuilder strings.Builder
	parenthesisBalanced := false
	var previousToken string
	for tok := s.Peek(); tok != scanner.EOF; s.Peek() {
		pathBuilder.WriteString(s.TokenText())
		s.Scan()
		text := s.TokenText()

		if previousToken == dotToken && text == dotToken {
			return nil, errors.New(`invalid substitution, two adjacent periods '.' (use quoted "" empty string if you want an empty element)`)
		}

		if text == "}" {
			parenthesisBalanced = true
			s.Scan()
			break
		}

		if text != dotToken && len(text) == 1 && !unicode.IsLetter(rune(text[0])) {
			break
		}

		previousToken = text
	}

	if !parenthesisBalanced {
		return nil, errors.New("invalid substitution")
	}

	substitutionPath := pathBuilder.String()
	if len(substitutionPath) == 0 {
		return nil, errors.New("invalid substitution, path expression cannot be empty")
	}
	if strings.HasPrefix(substitutionPath, dotToken) || strings.HasSuffix(substitutionPath, dotToken) {
		return nil, errors.New(`invalid substitution, leading or trailing period '.' (use quoted "" empty string if you want an empty element)`)
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

func isSubstitution(token string) bool {
	return token == "$" && s.Peek() == '{'
}
