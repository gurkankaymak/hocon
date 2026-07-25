package hocon

import (
	"errors"
	"fmt"
	"regexp"
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
	valueWithAlternativeType
)

// Config stores the root of the configuration tree
// and provides an API to retrieve configuration values with the path expressions
type Config struct {
	root Value
}

// String method returns the string representation of the Config object
func (c *Config) String() string { return c.root.String() }

// GetRoot method returns the root value of the configuration
func (c *Config) GetRoot() Value {
	return c.root
}

// GetObject method finds the value at the given path and returns it as an Object
// returns nil if the value is not found, panics if the value is not an object
func (c *Config) GetObject(path string) Object {
	object, err := c.GetObjectE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return object
}

// GetObjectE method finds the value at the given path and returns it as an Object
// returns an error if the value is not found or is not an object
func (c *Config) GetObjectE(path string) (Object, error) {
	value := c.Get(path)
	if value == nil {
		return nil, pathNotFoundError(path)
	}

	object, ok := value.(Object)
	if !ok {
		return nil, cannotParseError(value, "Object")
	}

	return object, nil
}

// GetConfig method finds the value at the given path and returns it as a Config
// returns nil if the value is not found, panics if the value is not an object
func (c *Config) GetConfig(path string) *Config {
	config, err := c.GetConfigE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return config
}

// GetConfigE method finds the value at the given path and returns it as a Config
// returns an error if the value is not found or is not an object
func (c *Config) GetConfigE(path string) (*Config, error) {
	object, err := c.GetObjectE(path)
	if err != nil {
		return nil, err
	}

	return object.ToConfig(), nil
}

// GetStringMap method finds the value at the given path and returns it as a map[string]Value
// returns nil if the value is not found, panics if the value is not an object
func (c *Config) GetStringMap(path string) map[string]Value {
	return c.GetObject(path)
}

// GetStringMapE method finds the value at the given path and returns it as a map[string]Value
// returns an error if the value is not found or is not an object
func (c *Config) GetStringMapE(path string) (map[string]Value, error) {
	return c.GetObjectE(path)
}

// GetStringMapString method finds the value at the given path and returns it as a map[string]string
// returns nil if the value is not found, panics if the value is not an object
func (c *Config) GetStringMapString(path string) map[string]string {
	stringMap, err := c.GetStringMapStringE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return stringMap
}

// GetStringMapStringE method finds the value at the given path and returns it as a map[string]string
// returns an error if the value is not found or is not an object
func (c *Config) GetStringMapStringE(path string) (map[string]string, error) {
	object, err := c.GetObjectE(path)
	if err != nil {
		return nil, err
	}

	var m = make(map[string]string, len(object))
	for k, v := range object {
		m[k] = rawString(v)
	}

	return m, nil
}

// GetArray method finds the value at the given path and returns it as an Array
// returns nil if the value is not found, panics if the value is not an array
func (c *Config) GetArray(path string) Array {
	array, err := c.GetArrayE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return array
}

// GetArrayE method finds the value at the given path and returns it as an Array
// returns an error if the value is not found or is not an array
func (c *Config) GetArrayE(path string) (Array, error) {
	value := c.Get(path)
	if value == nil {
		return nil, pathNotFoundError(path)
	}

	array, ok := value.(Array)
	if !ok {
		return nil, cannotParseError(value, "Array")
	}

	return array, nil
}

// GetIntSlice method finds the value at the given path and returns it as []int
// returns nil if the value is not found, panics if the value is not an array of integers
func (c *Config) GetIntSlice(path string) []int {
	slice, err := c.GetIntSliceE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return slice
}

// GetIntSliceE method finds the value at the given path and returns it as []int
// returns an error if the value is not found or is not an array of integers
func (c *Config) GetIntSliceE(path string) ([]int, error) {
	arr, err := c.GetArrayE(path)
	if err != nil {
		return nil, err
	}

	slice := make([]int, 0, len(arr))

	for _, v := range arr {
		intValue, ok := v.(Int)
		if !ok {
			return nil, cannotParseError(v, "int")
		}

		slice = append(slice, int(intValue))
	}

	return slice, nil
}

// GetStringSlice method finds the value at the given path and returns it as []string
// returns nil if the value is not found, panics if the value is not an array
func (c *Config) GetStringSlice(path string) []string {
	slice, err := c.GetStringSliceE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return slice
}

// GetStringSliceE method finds the value at the given path and returns it as []string
// returns an error if the value is not found or is not an array
func (c *Config) GetStringSliceE(path string) ([]string, error) {
	arr, err := c.GetArrayE(path)
	if err != nil {
		return nil, err
	}

	slice := make([]string, 0, len(arr))

	for _, v := range arr {
		slice = append(slice, rawString(v))
	}

	return slice, nil
}

// GetString method finds the value at the given path and returns its raw string content
// (without the hocon quoting), returns empty string if the value is not found
func (c *Config) GetString(path string) string {
	str, _ := c.GetStringE(path)
	return str
}

// GetStringE method finds the value at the given path and returns its raw string content
// (without the hocon quoting), returns an error if the value is not found
func (c *Config) GetStringE(path string) (string, error) {
	value := c.Get(path)
	if value == nil {
		return "", pathNotFoundError(path)
	}

	return rawString(value), nil
}

// GetInt method finds the value at the given path and returns it as an int
// returns zero if the value is not found, panics if the value cannot be converted to an int
func (c *Config) GetInt(path string) int {
	intValue, err := c.GetIntE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return intValue
}

// GetIntE method finds the value at the given path and returns it as an int
// returns an error if the value is not found or cannot be converted to an int
func (c *Config) GetIntE(path string) (int, error) {
	value := c.Get(path)
	if value == nil {
		return 0, pathNotFoundError(path)
	}

	switch val := value.(type) {
	case Int:
		return int(val), nil
	case String:
		intValue, err := strconv.Atoi(string(val))
		if err != nil {
			return 0, err
		}

		return intValue, nil
	default:
		return 0, cannotParseError(val, "int")
	}
}

// GetFloat32 method finds the value at the given path and returns it as a float32
// returns float32(0.0) if the value is not found, panics if the value cannot be converted to a float32
func (c *Config) GetFloat32(path string) float32 {
	floatValue, err := c.GetFloat32E(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return floatValue
}

// GetFloat32E method finds the value at the given path and returns it as a float32
// returns an error if the value is not found or cannot be converted to a float32
func (c *Config) GetFloat32E(path string) (float32, error) {
	value := c.Get(path)
	if value == nil {
		return 0, pathNotFoundError(path)
	}

	switch val := value.(type) {
	case Float32:
		return float32(val), nil
	case Float64:
		return float32(val), nil
	case String:
		floatValue, err := strconv.ParseFloat(string(val), 32)
		if err != nil {
			return 0, err
		}

		return float32(floatValue), nil
	default:
		return 0, cannotParseError(val, "float32")
	}
}

// GetFloat64 method finds the value at the given path and returns it as a float64
// returns 0.0 if the value is not found, panics if the value cannot be converted to a float64
func (c *Config) GetFloat64(path string) float64 {
	floatValue, err := c.GetFloat64E(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return floatValue
}

// GetFloat64E method finds the value at the given path and returns it as a float64
// returns an error if the value is not found or cannot be converted to a float64
func (c *Config) GetFloat64E(path string) (float64, error) {
	value := c.Get(path)
	if value == nil {
		return 0, pathNotFoundError(path)
	}

	switch val := value.(type) {
	case Float64:
		return float64(val), nil
	case Float32:
		return float64(val), nil
	case String:
		floatValue, err := strconv.ParseFloat(string(val), 64)
		if err != nil {
			return 0, err
		}

		return floatValue, nil
	default:
		return 0, cannotParseError(val, "float64")
	}
}

// GetBoolean method finds the value at the given path and returns it as a bool
// returns false if the value is not found, panics if the value cannot be converted to a bool
func (c *Config) GetBoolean(path string) bool {
	booleanValue, err := c.GetBooleanE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return booleanValue
}

// GetBooleanE method finds the value at the given path and returns it as a bool
// returns an error if the value is not found or cannot be converted to a bool
func (c *Config) GetBooleanE(path string) (bool, error) {
	value := c.Get(path)
	if value == nil {
		return false, pathNotFoundError(path)
	}

	switch val := value.(type) {
	case Boolean:
		return bool(val), nil
	case String:
		switch val {
		case "true", "yes", "on":
			return true, nil
		case "false", "no", "off":
			return false, nil
		default:
			return false, cannotParseError(val, "boolean")
		}
	default:
		return false, cannotParseError(val, "boolean")
	}
}

// GetDuration method finds the value at the given path and returns it as a time.Duration
// returns 0 if the value is not found, panics if the value is not a duration
func (c *Config) GetDuration(path string) time.Duration {
	durationValue, err := c.GetDurationE(path)
	if err != nil && !errors.Is(err, ErrPathNotFound) {
		panic(err)
	}

	return durationValue
}

// GetDurationE method finds the value at the given path and returns it as a time.Duration
// returns an error if the value is not found or is not a duration
func (c *Config) GetDurationE(path string) (time.Duration, error) {
	value := c.Get(path)
	if value == nil {
		return 0, pathNotFoundError(path)
	}

	duration, ok := value.(Duration)
	if !ok {
		return 0, cannotParseError(value, "Duration")
	}

	return time.Duration(duration), nil
}

// Get method finds the value at the given path and returns it without casting to any type
// returns nil if the value is not found
func (c *Config) Get(path string) Value {
	if c.root.Type() != ObjectType {
		return nil
	}

	return c.root.(Object).find(path)
}

// HasPath method returns true if a value exists at the given path, false otherwise,
// it can be used to validate the configuration before accessing the values
func (c *Config) HasPath(path string) bool {
	return c.Get(path) != nil
}

// WithFallback method returns a new *Config (or the current config, if the given fallback doesn't get used)
// 1. merges the values of the current and fallback *Configs, if the root of both of them are of type Object
// for the same keys current values overrides the fallback values
// 2. if any of the *Configs has non-object root then returns the current *Config ignoring the fallback parameter
func (c *Config) WithFallback(fallback *Config) *Config {
	if current, ok := c.root.(Object); ok {
		if fallbackObject, ok := fallback.root.(Object); ok {
			resultConfig := fallbackObject.copy()
			mergeObjects(resultConfig, current.copy())

			return resultConfig.ToConfig()
		}
	}

	return c
}

// Resolve method resolves the substitutions in the configuration tree in place, as the
// hocon spec defines: substitutions are looked up in the configuration itself, then in the
// environment variables, unresolved optional substitutions (${?path}) are omitted and
// string value concatenations are flattened. It returns the config itself on success and
// an error if a required substitution cannot be resolved to any value.
//
// The Parse* functions already return resolved configs, Resolve is meant to be used with
// the ParseStringUnresolved and ParseResourceUnresolved functions, e.g. to resolve
// substitutions with the values of a fallback config:
//
//	config, err := hocon.ParseStringUnresolved(mainConfig)
//	...
//	config, err = config.WithFallback(fallbackConfig).Resolve()
func (c *Config) Resolve() (*Config, error) {
	if root, ok := c.root.(Object); ok {
		if err := resolveSubstitutions(root); err != nil {
			return nil, err
		}
	}

	return c, nil
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
func (s String) Type() Type { return StringType }

func (s String) String() string {
	str := strings.Trim(string(s), `"`)
	if str == "" {
		return `""`
	}
	compile := regexp.MustCompile("[ !\\\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~]+")

	if compile.MatchString(str) {
		return fmt.Sprintf(`"%s"`, str)
	}
	return str
}

func (s String) isConcatenable() bool { return true }

// rawString returns the content of the given value as a raw string, without the
// quoting that the String method applies when rendering a value as hocon
func rawString(value Value) string {
	if str, ok := value.(String); ok {
		return string(str)
	}

	return value.String()
}

// valueWithAlternative represents a value with Substitution which might override the original value
type valueWithAlternative struct {
	value       Value
	alternative *Substitution
}

func (s *valueWithAlternative) Type() Type { return valueWithAlternativeType }

func (s *valueWithAlternative) String() string {
	return fmt.Sprintf("(%s | %s)", s.value, s.alternative.String())
}

func (s *valueWithAlternative) isConcatenable() bool { return false }

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

// ToConfig method converts object to *Config
func (o Object) ToConfig() *Config {
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

		subObject, ok := value.(Object)
		if !ok {
			return nil
		}

		object = subObject
	}

	return object[lastKey]
}

func (o Object) copy() Object {
	result := Object{}

	for k, v := range o {
		result[k] = copyValue(v)
	}

	return result
}

// copyValue deep-copies the values that the substitution resolution modifies in place
// (objects, arrays, concatenations and values with alternatives), so that resolving a
// config does not modify the configs it was created from; the other values are immutable
// and are returned as they are
func copyValue(value Value) Value {
	switch v := value.(type) {
	case Object:
		return v.copy()
	case Array:
		result := make(Array, len(v))

		for i, element := range v {
			result[i] = copyValue(element)
		}

		return result
	case concatenation:
		result := make(concatenation, len(v))

		for i, element := range v {
			result[i] = copyValue(element)
		}

		return result
	case *valueWithAlternative:
		return &valueWithAlternative{value: copyValue(v.value), alternative: v.alternative}
	default:
		return value
	}
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
func (i Int) isConcatenable() bool { return true }

// Float32 represents a Float32 value
type Float32 float32

// Type Number
func (f Float32) Type() Type           { return NumberType }
func (f Float32) String() string       { return strconv.FormatFloat(float64(f), 'g', -1, 32) }
func (f Float32) isConcatenable() bool { return true }

// Float64 represents a Float64 value
type Float64 float64

// Type Number
func (f Float64) Type() Type           { return NumberType }
func (f Float64) String() string       { return strconv.FormatFloat(float64(f), 'g', -1, 64) }
func (f Float64) isConcatenable() bool { return true }

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

// Type Duration
func (d Duration) Type() Type           { return StringType }
func (d Duration) String() string       { return time.Duration(d).String() }
func (d Duration) isConcatenable() bool { return false }

type concatenation Array

func (c concatenation) Type() Type           { return ConcatenationType }
func (c concatenation) isConcatenable() bool { return true }
func (c concatenation) containsObject() bool {
	for _, value := range c {
		if value != nil && value.Type() == ObjectType {
			return true
		}
	}

	return false
}

func (c concatenation) containsArray() bool {
	for _, value := range c {
		if value != nil && value.Type() == ArrayType {
			return true
		}
	}

	return false
}
func (c concatenation) String() string {
	var builder strings.Builder

	for _, value := range c {
		builder.WriteString(value.String())
	}

	return builder.String()
}
