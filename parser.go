package hocon

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

var s scanner.Scanner

func Parse(input string) *Config {
	s.Init(strings.NewReader(input))
	s.Scan()
	if s.TokenText() == "[" {
		configArray := extractConfigArray()
		fmt.Println("gk =========> configArray:", configArray)
		return &Config{root:configArray}
	}

	configObject := extractConfigObject()
	fmt.Println("gk =========> configObject:", configObject)
	return &Config{root:configObject}
}

func extractConfigObject() *ConfigObject {
	root := map[string]ConfigValue{}
	parenthesisBalanced := true

	for tok := s.Peek(); tok != scanner.EOF; tok = s.Peek()  {
		if s.TokenText() == "{" {
			parenthesisBalanced = false
			s.Scan() // skip if current text is '{'
		}

		if s.TokenText() == "," {
			s.Scan() // skip ','
		}

		key := s.TokenText()
		s.Scan()
		text := s.TokenText()

		if text == "." {
			s.Scan() // skip '.'
			configObject := extractConfigObject()
			if !parenthesisBalanced && s.TokenText() == "}" {
				parenthesisBalanced = true
				s.Scan()
			}
			return NewConfigObject(map[string]ConfigValue{key:configObject})
		}

		if text == "=" || text == ":" { // skip '=' or ':'
			s.Scan()
		}
		configValue := extractConfigValue()
		root[key] = configValue

		if !parenthesisBalanced && s.Peek() == '}' {  // skip '}'
			parenthesisBalanced = true
			s.Scan()
			s.Scan()
		}

		if parenthesisBalanced {
			return NewConfigObject(root)
		}
	}
	return NewConfigObject(root)
}

func extractConfigArray() *ConfigArray {
	var values []ConfigValue
	for tok := s.Peek(); tok != ']' && tok != scanner.EOF; tok = s.Peek() {
		s.Scan()
		if s.TokenText() == "[" {
			s.Scan() // skip // '['
		}
		configValue := extractConfigValue()
		if configValue != nil {
			values = append(values, configValue)
		}
		if s.Peek() == ',' {
			s.Scan() // skip comma
		}
	}
	if s.Peek() == ']' {  // skip ']'
		s.Scan()
		s.Scan()
	}
	return NewConfigArray(values)
}

func extractConfigValue() ConfigValue {
	// TODO gk: int, float32, bool cases parse two times
	token := s.TokenText()
	switch {
	case isTokenString(token):
		return NewConfigString(strings.ReplaceAll(token, "\"", ""))
	case isTokenObject(token):
		return extractConfigObject()
	case isTokenArray(token):
		return extractConfigArray()
	case isTokenInt(token):
		value, _ := strconv.Atoi(token)
		return NewConfigInt(value)
	case isTokenFloat32(token):
		value, _ := strconv.ParseFloat(token, 32)
		return NewConfigFloat32(float32(value))
	case isTokenBoolean(token):
		value, _ := strconv.ParseBool(token)
		return NewConfigBoolean(value)
	case isTokenBooleanString(token):
		return NewConfigBooleanFromString(token)
	}
	return nil
}

func isTokenString(token string) bool {
	return strings.HasPrefix(token, `"`)
}

func isTokenObject(token string) bool {
	return strings.HasPrefix(token, "{")
}

func isTokenArray(token string) bool {
	return strings.HasPrefix(token, "[")
}

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
