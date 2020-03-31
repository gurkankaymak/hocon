# HOCON (Human-Optimized Config Object Notation)

[![Go Report Card](https://goreportcard.com/badge/github.com/gurkankaymak/hocon)](https://goreportcard.com/report/github.com/gurkankaymak/hocon)
[![codecov](https://codecov.io/gh/gurkankaymak/hocon/branch/master/graph/badge.svg)](https://codecov.io/gh/gurkankaymak/hocon)
[![Build Status](https://travis-ci.org/gurkankaymak/hocon.svg?branch=master)](https://travis-ci.org/gurkankaymak/hocon)
[![GoDoc](https://godoc.org/github.com/gurkankaymak/hocon?status.svg)](https://godoc.org/github.com/gurkankaymak/hocon)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

Configuration library for working with the Lightbend's HOCON format. HOCON is a human-friendly JSON superset

### Features of HOCON

  - Comments, with `#` or `//`
  - Allow omitting the `{}` around a root object
  - Allow `=` as a synonym for `:`
  - Allow omitting the `=` or `:` before a `{` so
    `foo { a : 42 }`
  - Allow omitting commas as long as there's a newline
  - Allow trailing commas after last element in objects and arrays
  - Allow unquoted strings for keys and values
  - Unquoted keys can use dot-notation for nested objects,
    `foo.bar=42` means `foo { bar : 42 }`
  - Duplicate keys are allowed; later values override earlier,
    except for object-valued keys where the two objects are merged
    recursively
  - `include` feature merges root object in another file into
    current object, so `foo { include "bar.json" }` merges keys in
    `bar.json` into the object `foo`
  - substitutions `foo : ${a.b}` sets key `foo` to the same value
    as the `b` field in the `a` object
  - substitutions concatenate into unquoted strings, `foo : the quick ${colors.fox} jumped`
  - substitutions fall back to environment variables if they don't
    resolve in the config itself, so `${HOME}` would work as you
    expect.
  - substitutions normally cause an error if unresolved, but
    there is a syntax `${?a.b}` to permit them to be missing.
  - `+=` syntax to append elements to arrays, `path += "/bin"`
  - multi-line strings with triple quotes as in Python or Scala
  
  see the documentation for more details about the HOCON https://github.com/lightbend/config/blob/master/HOCON.md

## Installation
```go get -u github.com/gurkankaymak/hocon```

## Usage
```go
package main

import (
    "fmt"
    "log"
    "github.com/gurkankaymak/hocon"
)

func main() {
    hoconString := `
    booleans {
      trueVal: true
      trueValAgain: ${booleans.trueVal}
      trueWithYes: yes
      falseWithNo: no
    }
    // this is a comment
    #  this is also a comment
    numbers {
      intVal: 3
      floatVal: 1.0
    }
    strings {
      a: "a"
      b: "b"
      c: "c"
    }
    arrays {
      empty: []
      ofInt: [1, 2, 3]
      ofString: [${strings.a}, ${strings.b}, ${strings.c}]
      ofDuration: [1 second, 2h, 3 days]
    }
    durations {
      second: 1s
      halfSecond: 0.5 second
      minutes: 5 minutes
      hours: 2hours
      day: 1d
    }
    objects {
      valueObject {
        mandatoryValue: "mandatoryValue"
        arrayValue: ${arrays.ofInt}
        nullValue: null
      }
    }`

    conf, err := hocon.ParseResource(hoconString)
    if err != nil {
        log.Fatal("error while parsing configuration: ", err)
    }
    objectValue := conf.GetObject("objects.valueObject")
    arrayValue := conf.GetArray("arrays.ofInt")
    stringValue := conf.GetString("strings.a")
    intValue := conf.GetInt("numbers.intVal")
    floatValue := conf.GetFloat64("numbers.floatVal")
    durationValue := conf.GetDuration("durations.second")
    fmt.Println("objectValue:", objectValue) // {mandatoryValue:mandatoryValue, arrayValue:[1,2,3], nullValue:null}
    fmt.Println("arrayValue:", arrayValue) // [1,2,3]
    fmt.Println("stringValue:", stringValue) // a
    fmt.Println("intValue:", intValue) // 3
    fmt.Println("floatValue:", floatValue) // 1.0
    fmt.Println("durationValue:", durationValue) // 1s
    fmt.Println("all configuration:", conf)
}
```