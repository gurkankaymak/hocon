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

func (c *Config) GetConfigObject(path string) ConfigObject {
	configValue := c.find(path)
	if configValue == nil {
		return nil
	}
	return configValue.(ConfigObject)
}

func (c *Config) GetConfigArray(path string) ConfigArray {
	configValue := c.find(path)
	if configValue == nil {
		return nil
	}
	return configValue.(ConfigArray)
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
	case ConfigInt:
		return int(configValue)
	case ConfigString:
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
	case ConfigFloat32:
		return float32(configValue)
	case ConfigString:
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
	case ConfigBoolean:
		return bool(configValue)
	case ConfigString:
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
	return c.root.(ConfigObject).find(path)

}

type ConfigValue interface {
	ValueType() ValueType
	String() string
}

type ConfigString string

func (c ConfigString) ValueType() ValueType { return ValueTypeString }
func (c ConfigString) String() string       { return string(c) }

type ConfigObject map[string]ConfigValue

func (c ConfigObject) ValueType() ValueType       { return ValueTypeObject }
func (c ConfigObject) Get(key string) ConfigValue { return c[key] }

func (c ConfigObject) String() string {
	var builder strings.Builder

	itemsSize := len(c)
	i := 1
	builder.WriteString(objectStartToken)
	for key, value := range c {
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

func (c ConfigObject) find(path string) ConfigValue {
	keys := strings.Split(path, dotToken)
	size := len(keys)
	lastKey := keys[size-1]
	keysWithoutLast := keys[:size-1]
	configObject := c
	for _, key := range keysWithoutLast {
		value := configObject.Get(key)
		if value == nil {
			return nil
		}
		configObject = value.(ConfigObject)
	}
	return configObject.Get(lastKey)
}

type ConfigArray []ConfigValue

func (c ConfigArray) ValueType() ValueType { return ValueTypeArray }

func (c ConfigArray) String() string {
	if len(c) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString(arrayStartToken)
	builder.WriteString(c[0].String())
	for _, configValue := range c[1:] {
		builder.WriteString(commaToken)
		builder.WriteString(configValue.String())
	}
	builder.WriteString(arrayEndToken)

	return builder.String()
}

func (c *ConfigArray) Append(value ConfigValue) {
	*c = append(*c, value)
}

type ConfigInt int

func (c ConfigInt) ValueType() ValueType { return ValueTypeNumber }
func (c ConfigInt) String() string { return strconv.Itoa(int(c)) }

type ConfigFloat32 float32

func (c ConfigFloat32) ValueType() ValueType { return ValueTypeNumber }

func (c ConfigFloat32) String() string {
	return strconv.FormatFloat(float64(c), 'e', -1, 32)
}

type ConfigBoolean bool

func NewConfigBooleanFromString(value string) ConfigBoolean {
	switch value {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	default:
		panic("cannot parse value: " + value + " to boolean!")
	}
}

func (c ConfigBoolean) ValueType() ValueType { return ValueTypeBoolean }
func (c ConfigBoolean) String() string       { return strconv.FormatBool(bool(c)) }

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

type ConfigNull string
const null ConfigNull = "null"

func (c ConfigNull) ValueType() ValueType { return ValueTypeNull }
func (c ConfigNull) String() string       { return string(null) }

type ConfigDuration time.Duration

func (d ConfigDuration) ValueType() ValueType { return ValueTypeString }
func (d ConfigDuration) String() string       { return time.Duration(d).String() }