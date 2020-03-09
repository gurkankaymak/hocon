package hocon

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Type int

const (
	ObjectType Type = iota
	StringType
	ArrayType
	NumberType
	BooleanType
	NullType
	SubstitutionType
)

type Config struct {
	root Value
}

func (c *Config) String() string { return c.root.String() }

func (c *Config) GetObject(path string) Object {
	value := c.find(path)
	if value == nil {
		return nil
	}
	return value.(Object)
}

func (c *Config) GetArray(path string) Array {
	value := c.find(path)
	if value == nil {
		return nil
	}
	return value.(Array)
}

func (c *Config) GetString(path string) string {
	value := c.find(path)
	if value == nil {
		return ""
	}
	return value.String()
}

func (c *Config) GetInt(path string) int {
	value := c.find(path)
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

func (c *Config) GetFloat32(path string) float32 {
	value := c.find(path)
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

func (c *Config) GetFloat64(path string) float64 {
	value := c.find(path)
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

func (c *Config) GetBoolean(path string) bool {
	value := c.find(path)
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

func (c *Config) find(path string) Value {
	if c.root.Type() != ObjectType {
		return nil
	}
	return c.root.(Object).find(path)

}

type Value interface {
	Type() Type
	String() string
}

type String string

func (s String) Type() Type     { return StringType }
func (s String) String() string { return string(s) }

type Object map[string]Value

func (o Object) Type() Type { return ObjectType }

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

type Array []Value

func (a Array) Type() Type { return ArrayType }

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

type Int int

func (i Int) Type() Type     { return NumberType }
func (i Int) String() string { return strconv.Itoa(int(i)) }

type Float32 float32

func (f Float32) Type() Type     { return NumberType }
func (f Float32) String() string { return strconv.FormatFloat(float64(f), 'e', -1, 32) }

type Float64 float64

func (f Float64) Type() Type     { return NumberType }
func (f Float64) String() string { return strconv.FormatFloat(float64(f), 'e', -1, 64) }

type Boolean bool

func NewBooleanFromString(value string) Boolean {
	switch value {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	default:
		panic(fmt.Sprintf("cannot parse value: %s to boolean!", value))
	}
}

func (b Boolean) Type() Type     { return BooleanType }
func (b Boolean) String() string { return strconv.FormatBool(bool(b)) }

type Substitution struct {
	path     string
	optional bool
}

func (s *Substitution) Type() Type { return SubstitutionType }

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

func (n Null) Type() Type     { return NullType }
func (n Null) String() string { return string(null) }

type Duration time.Duration

func (d Duration) Type() Type     { return StringType }
func (d Duration) String() string { return time.Duration(d).String() }