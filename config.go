package hocon

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Type of an hocon Value
type Type int

// Type constants
const (
	ObjectType Type = iota
	StringType
	ArrayType
	NumberType
	BooleanType
	NullType
	SubstitutionType
	ConcatenationType
)

// Config stores the root of the configuration tree and provides an API to retrieve configuration values with the path expressions
type Config struct {
	root Value
}

// String method returns the string representation of the Config object
func (c *Config) String() string { return c.root.String() }

// GetObject method finds the value at the given path and returns it as an Object, returns nil if the value is not found
func (c *Config) GetObject(path string) Object {
	value := c.Get(path)
	if value == nil {
		return nil
	}
	return value.(Object)
}

// GetStringMap method finds the value at the given path and returns it as a map[string]Value, returns nil if the value is not found
func (c *Config) GetStringMap(path string) map[string]Value {
	return c.GetObject(path)
}

// GetStringMapString method finds the value at the given path and returns it as a map[string]string, returns nil if the value is not found
func (c *Config) GetStringMapString(path string) map[string]string {
	value := c.Get(path)
	if value == nil {
		return nil
	}
	object := value.(Object)
	var m = make(map[string]string, len(object))
	for k, v := range object {
		m[k] = v.String()
	}
	return m
}

// GetArray method finds the value at the given path and returns it as an Array, returns nil if the value is not found
func (c *Config) GetArray(path string) Array {
	value := c.Get(path)
	if value == nil {
		return nil
	}
	return value.(Array)
}

// GetIntSlice method finds the value at the given path and returns it as []int, returns nil if the value is not found
func (c *Config) GetIntSlice(path string) []int {
	value := c.Get(path)
	if value == nil {
		return nil
	}
	var slice []int
	for _, v := range value.(Array) {
		slice = append(slice, int(v.(Int)))
	}
	return slice
}

// GetStringSlice method finds the value at the given path and returns it as []string, returns nil if the value is not found
func (c *Config) GetStringSlice(path string) []string {
	value := c.Get(path)
	if value == nil {
		return nil
	}
	var slice []string
	for _, v := range value.(Array) {
		slice = append(slice, v.String())
	}
	return slice
}

// GetString method finds the value at the given path and returns it as a String, returns empty string if the value is not found
func (c *Config) GetString(path string) string {
	value := c.Get(path)
	if value == nil {
		return ""
	}
	return value.String()
}

// GetInt method finds the value at the given path and returns it as an Int, returns zero if the value is not found
func (c *Config) GetInt(path string) int {
	value := c.Get(path)
	if value == nil {
		return 0
	}
	switch val := value.(type) {
	case Int:
		return int(val)
	case String:
		intValue, err := strconv.Atoi(string(val))
		if err != nil {
			panic(err)
		}
		return intValue
	default:
		panic("cannot parse value: " + val.String() + " to int!")
	}
}

// GetFloat32 method finds the value at the given path and returns it as a Float32, returns float32(0.0) if the value is not found
func (c *Config) GetFloat32(path string) float32 {
	value := c.Get(path)
	if value == nil {
		return float32(0.0)
	}
	switch val := value.(type) {
	case Float32:
		return float32(val)
	case Float64:
		return float32(val)
	case String:
		floatValue, err := strconv.ParseFloat(string(val), 32)
		if err != nil {
			panic(err)
		}
		return float32(floatValue)
	default:
		panic("cannot parse value: " + val.String() + " to float32!")
	}
}

// GetFloat64 method finds the value at the given path and returns it as a Float64, returns 0.0 if the value is not found
func (c *Config) GetFloat64(path string) float64 {
	value := c.Get(path)
	if value == nil {
		return 0.0
	}
	switch val := value.(type) {
	case Float64:
		return float64(val)
	case Float32:
		return float64(val)
	case String:
		floatValue, err := strconv.ParseFloat(string(val), 64)
		if err != nil {
			panic(err)
		}
		return floatValue
	default:
		panic("cannot parse value: " + val.String() + "to float64!")
	}
}

// GetBoolean method finds the value at the given path and returns it as a Boolean, returns false if the value is not found
func (c *Config) GetBoolean(path string) bool {
	value := c.Get(path)
	if value == nil {
		return false
	}
	switch val := value.(type) {
	case Boolean:
		return bool(val)
	case String:
		switch val {
		case "true", "yes", "on":
			return true
		case "false", "no", "off":
			return false
		default:
			panic("cannot parse value: " + val + " to boolean!")
		}
	default:
		panic("cannot parse value: " + val.String() + " to boolean!")
	}
}

// GetDuration method finds the value at the given path and returns it as a time.Duration, returns 0 if the value is not found
func (c *Config) GetDuration(path string) time.Duration {
	value := c.Get(path)
	if value == nil {
		return 0
	}
	return time.Duration(value.(Duration))
}

// Get method finds the value at the given path and returns it without casting to any type, returns nil if the value is not found
func (c *Config) Get(path string) Value {
	if c.root.Type() != ObjectType {
		return nil
	}
	return c.root.(Object).find(path)

}

// Value interface represents a value in the configuration tree, all the value types implements this interface
type Value interface {
	Type() Type
	String() string
	isConcatenable() bool
}

// String represents a string value
type String string

// Type String
func (s String) Type() Type           { return StringType }
func (s String) String() string       { return strings.ReplaceAll(string(s), `"`, "") }
func (s String) isConcatenable() bool { return true }

// Object represents an object node in the configuration tree
type Object map[string]Value

// Type Object
func (o Object) Type() Type           { return ObjectType }
func (o Object) isConcatenable() bool { return false }

// String method returns the string representation of the Object
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

// toConfig method converts object to *Config
func (o Object) toConfig() *Config {
	return &Config{o}
}

func (o Object) find(path string) Value {
	keys := strings.Split(path, dotToken)
	size := len(keys)
	lastKey := keys[size-1]
	keysWithoutLast := keys[:size-1]
	object := o
	for _, key := range keysWithoutLast {
		value, ok := object[key]
		if !ok {
			return nil
		}
		object = value.(Object)
	}
	return object[lastKey]
}

// Array represents an array node in the configuration tree
type Array []Value

// Type Array
func (a Array) Type() Type           { return ArrayType }
func (a Array) isConcatenable() bool { return false }

// String method returns the string representation of the Array
func (a Array) String() string {
	if len(a) == 0 {
		return "[]"
	}
	var builder strings.Builder
	builder.WriteString(arrayStartToken)
	builder.WriteString(a[0].String())
	for _, value := range a[1:] {
		builder.WriteString(commaToken)
		builder.WriteString(value.String())
	}
	builder.WriteString(arrayEndToken)
	return builder.String()
}

// Int represents an Integer value
type Int int

// Type Number
func (i Int) Type() Type           { return NumberType }
func (i Int) String() string       { return strconv.Itoa(int(i)) }
func (i Int) isConcatenable() bool { return false }

// Float32 represents a Float32 value
type Float32 float32

// Type Number
func (f Float32) Type() Type           { return NumberType }
func (f Float32) String() string       { return strconv.FormatFloat(float64(f), 'e', -1, 32) }
func (f Float32) isConcatenable() bool { return false }

// Float64 represents a Float64 value
type Float64 float64

// Type Number
func (f Float64) Type() Type           { return NumberType }
func (f Float64) String() string       { return strconv.FormatFloat(float64(f), 'e', -1, 64) }
func (f Float64) isConcatenable() bool { return false }

// Boolean represents bool value
type Boolean bool

func newBooleanFromString(value string) Boolean {
	switch value {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	default:
		panic(fmt.Sprintf("cannot parse value: %s to Boolean!", value))
	}
}

// Type Boolean
func (b Boolean) Type() Type           { return BooleanType }
func (b Boolean) String() string       { return strconv.FormatBool(bool(b)) }
func (b Boolean) isConcatenable() bool { return true }

// Substitution refers to another value in the configuration tree
type Substitution struct {
	path     string
	optional bool
}

// Type Substitution
func (s *Substitution) Type() Type           { return SubstitutionType }
func (s *Substitution) isConcatenable() bool { return true }

// String method returns the string representation of the Substitution
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

// Null represents a null value
type Null string

const null Null = "null"

// Type Null
func (n Null) Type() Type           { return NullType }
func (n Null) String() string       { return string(null) }
func (n Null) isConcatenable() bool { return true }

// Duration represents a duration value
type Duration time.Duration

// Type String
func (d Duration) Type() Type           { return StringType }
func (d Duration) String() string       { return time.Duration(d).String() }
func (d Duration) isConcatenable() bool { return false }

type concatenation Array

func (c concatenation) Type() Type           { return ConcatenationType }
func (c concatenation) isConcatenable() bool { return true }
func (c concatenation) String() string {
	var builder strings.Builder
	for _, value := range c {
		builder.WriteString(value.String())
	}
	return builder.String()
}
