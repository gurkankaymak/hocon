package hocon

import (
	"strconv"
	"strings"
	"time"
)

type ValueType int

const (
	ValueTypeObject ValueType = iota
	ValueTypeString
	ValueTypeArray
	ValueTypeNumber
	ValueTypeBoolean
	ValueTypeNull
	ValueTypeSubstitution
)

type Config struct {
	root ConfigValue
}

func (c *Config) String() string { return c.root.String() }

func (c *Config) GetObject(path string) Object {
	configValue := c.find(path)
	if configValue == nil {
		return nil
	}
	return configValue.(Object)
}

func (c *Config) GetArray(path string) Array {
	configValue := c.find(path)
	if configValue == nil {
		return nil
	}
	return configValue.(Array)
}

func (c *Config) GetString(path string) string {
	configValue := c.find(path)
	if configValue == nil {
		return ""
	}
	return configValue.String()
}

func (c *Config) GetInt(path string) int {
	value := c.find(path)
	if value == nil {
		return 0
	}
	switch configValue := value.(type) {
	case Int:
		return int(configValue)
	case String:
		intValue, err := strconv.Atoi(string(configValue))
		if err != nil {
			panic(err)
		}
		return intValue
	default:
		panic("cannot parse value: " + configValue.String() + " to int!")
	}
}

func (c *Config) GetFloat32(path string) float32 {
	value := c.find(path)
	if value == nil {
		return float32(0.0)
	}
	switch configValue := value.(type) {
	case Float32:
		return float32(configValue)
	case String:
		floatValue, err := strconv.ParseFloat(string(configValue), 32)
		if err != nil {
			panic(err)
		}
		return float32(floatValue)
	default:
		panic("cannot parse value: " + configValue.String() + " to float32!")
	}
}

func (c *Config) GetBoolean(path string) bool {
	value := c.find(path)
	if value == nil {
		return false
	}
	switch configValue := value.(type) {
	case Boolean:
		return bool(configValue)
	case String:
		switch configValue {
		case "true", "yes", "on":
			return true
		case "false", "no", "off":
			return false
		default:
			panic("cannot parse value: " + configValue + " to boolean!")
		}
	default:
		panic("cannot parse value: " + configValue.String() + " to boolean!")
	}
}

func (c *Config) find(path string) ConfigValue {
	if c.root.ValueType() != ValueTypeObject {
		return nil
	}
	return c.root.(Object).find(path)

}

type ConfigValue interface {
	ValueType() ValueType
	String() string
}

type String string

func (s String) ValueType() ValueType { return ValueTypeString }
func (s String) String() string       { return string(s) }

type Object map[string]ConfigValue

func (o Object) ValueType() ValueType { return ValueTypeObject }

func (o Object) String() string {
	var builder strings.Builder

	itemsSize := len(o)
	i := 1
	builder.WriteString(objectStartToken)
	for key, value := range o {
		builder.WriteString(key)
		builder.WriteString(colonToken)
		builder.WriteString(value.String())
		if i < itemsSize {
			builder.WriteString(", ")
		}
		i++
	}
	builder.WriteString(objectEndToken)

	return builder.String()
}

func (o Object) find(path string) ConfigValue {
	keys := strings.Split(path, dotToken)
	size := len(keys)
	lastKey := keys[size-1]
	keysWithoutLast := keys[:size-1]
	configObject := o
	for _, key := range keysWithoutLast {
		value, ok := configObject[key]
		if !ok {
			return nil
		}
		configObject = value.(Object)
	}
	return configObject[lastKey]
}

type Array []ConfigValue

func (a Array) ValueType() ValueType { return ValueTypeArray }

func (a Array) String() string {
	if len(a) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString(arrayStartToken)
	builder.WriteString(a[0].String())
	for _, configValue := range a[1:] {
		builder.WriteString(commaToken)
		builder.WriteString(configValue.String())
	}
	builder.WriteString(arrayEndToken)

	return builder.String()
}

type Int int

func (i Int) ValueType() ValueType { return ValueTypeNumber }
func (i Int) String() string       { return strconv.Itoa(int(i)) }

type Float32 float32

func (f Float32) ValueType() ValueType { return ValueTypeNumber }

func (f Float32) String() string {
	return strconv.FormatFloat(float64(f), 'e', -1, 32)
}

type Boolean bool

func NewBooleanFromString(value string) Boolean {
	switch value {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	default:
		panic("cannot parse value: " + value + " to boolean!")
	}
}

func (b Boolean) ValueType() ValueType { return ValueTypeBoolean }
func (b Boolean) String() string       { return strconv.FormatBool(bool(b)) }

type Substitution struct {
	path     string
	optional bool
}

func (s *Substitution) ValueType() ValueType { return ValueTypeSubstitution }

func (s *Substitution) String() string {
	var builder strings.Builder
	builder.WriteString("${")
	if s.optional {
		builder.WriteString("?")
	}
	builder.WriteString(s.path)
	builder.WriteString("}")
	return builder.String()
}

type Null string
const null Null = "null"

func (n Null) ValueType() ValueType { return ValueTypeNull }
func (n Null) String() string       { return string(null) }

type Duration time.Duration

func (d Duration) ValueType() ValueType { return ValueTypeString }
func (d Duration) String() string       { return time.Duration(d).String() }