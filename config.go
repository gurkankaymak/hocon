package hocon

import (
	"strconv"
	"strings"
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

func (c *Config) String() string {
	return c.root.String()
}

func (c *Config) GetConfigObject(path string) *ConfigObject {
	configValue := c.find(path)
	if configValue == nil {
		return nil
	}
	return configValue.(*ConfigObject)
}

func (c *Config) GetConfigArray(path string) *ConfigArray {
	configValue := c.find(path)
	if configValue == nil {
		return nil
	}
	return configValue.(*ConfigArray)
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
	case *ConfigInt:
		return configValue.value
	case *ConfigString:
		intValue, err := strconv.Atoi(configValue.value)
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
	case *ConfigFloat32:
		return configValue.value
	case *ConfigString:
		floatValue, err := strconv.ParseFloat(configValue.value, 32)
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
	case *ConfigBoolean:
		return configValue.value
	case *ConfigString:
		switch configValue.value {
		case "true", "yes", "on":
			return true
		case "false", "no", "off":
			return false
		default:
			panic("cannot parse value: " + configValue.value + " to boolean!")
		}
	default:
		panic("cannot parse value: " + configValue.String() + " to boolean!")
	}
}

func (c *Config) find(path string) ConfigValue {
	if c.root.ValueType() != ValueTypeObject {
		return nil
	}
	return c.root.(*ConfigObject).find(path)

}

type ConfigValue interface {
	ValueType() ValueType
	String() string
}

type ConfigString struct {
	value string
}

func NewConfigString(value string) *ConfigString {
	return &ConfigString{value: value}
}

func (c *ConfigString) ValueType() ValueType {
	return ValueTypeString
}

func (c *ConfigString) String() string {
	return c.value
}

type ConfigObject struct {
	items map[string]ConfigValue
}

func NewConfigObject(items map[string]ConfigValue) *ConfigObject {
	return &ConfigObject{items: items}
}

func (c *ConfigObject) ValueType() ValueType {
	return ValueTypeObject
}

func (c *ConfigObject) Get(key string) ConfigValue {
	return c.items[key]
}

func (c *ConfigObject) String() string {
	var builder strings.Builder

	itemsSize := len(c.items)
	i := 1
	builder.WriteString(objectStartToken)
	for key, value := range c.items {
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

func (c *ConfigObject) find(path string) ConfigValue {
	keys := strings.Split(path, dotToken)
	size := len(keys)
	lastKey := keys[size-1]
	keysWithoutLast := keys[:size-1]
	configObject := c
	for _, key := range keysWithoutLast {
		value, ok := configObject.items[key]
		if !ok {
			return nil
		}
		configObject = value.(*ConfigObject)
	}
	return configObject.items[lastKey]
}

type ConfigArray struct {
	values []ConfigValue
}

func NewConfigArray(values []ConfigValue) *ConfigArray {
	return &ConfigArray{values: values}
}

func (c *ConfigArray) ValueType() ValueType {
	return ValueTypeArray
}

func (c *ConfigArray) String() string {
	if len(c.values) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString(arrayStartToken)
	builder.WriteString(c.values[0].String())
	for _, configValue := range c.values[1:] {
		builder.WriteString(commaToken)
		builder.WriteString(configValue.String())
	}
	builder.WriteString(arrayEndToken)

	return builder.String()
}

func (c *ConfigArray) Append(value ConfigValue) {
	c.values = append(c.values, value)
}

type ConfigNumber interface {
	ValueType() ValueType
	String() string
}

type ConfigInt struct {
	value int
}

func NewConfigInt(value int) *ConfigInt {
	return &ConfigInt{value: value}
}

func (c *ConfigInt) ValueType() ValueType {
	return ValueTypeNumber
}

func (c *ConfigInt) String() string {
	return strconv.Itoa(c.value)
}

type ConfigFloat32 struct {
	value float32
}

func NewConfigFloat32(value float32) *ConfigFloat32 {
	return &ConfigFloat32{value: value}
}

func (c *ConfigFloat32) ValueType() ValueType {
	return ValueTypeNumber
}

func (c *ConfigFloat32) String() string {
	return strconv.FormatFloat(float64(c.value), 'e', -1, 32)
}

type ConfigBoolean struct {
	value bool
}

func NewConfigBoolean(value bool) *ConfigBoolean {
	return &ConfigBoolean{value: value}
}

func NewConfigBooleanFromString(value string) *ConfigBoolean {
	switch value {
	case "true", "yes", "on":
		return &ConfigBoolean{value: true}
	case "false", "no", "off":
		return &ConfigBoolean{value: false}
	default:
		panic("cannot parse value: " + value + " to boolean!")
	}
}

func (c *ConfigBoolean) ValueType() ValueType {
	return ValueTypeBoolean
}

func (c *ConfigBoolean) String() string {
	return strconv.FormatBool(c.value)
}

type Substitution struct {
	path     string
	optional bool
}

func (s *Substitution) ValueType() ValueType {
	return ValueTypeSubstitution
}

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
