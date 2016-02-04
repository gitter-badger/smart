# Build smartly (in Go)

[![GoDoc](https://godoc.org/github.com/duzy/smart/build?status.svg)](http://godoc.org/github.com/duzy/smart/build)

## Overview

This utility named `smart` is made to ease the build process of software development.
It's inspired by [GNU make][] (having almost the same syntax as makefile) and written
in [Go][].

## Example

```makefile
$(module foo, gcc, static)

me.sources := foo.c
me.export.libdirs := out/foo
me.export.libs := foo

$(build)
```

Why
===

Build faster the simple way! (By comparing to autogen, autoconf, automake.)
