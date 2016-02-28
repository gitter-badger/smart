# Build smartly (in Go)

[![GoDoc](https://godoc.org/github.com/duzy/smart/build?status.svg)](http://godoc.org/github.com/duzy/smart/build)

## Overview

This `smart` utility is made to ease the build process of software development. It's inspired by [GNU make]() (having almost the same syntax as makefile, but not guaranteed to be compatible [GNU make]()). It's written in [Go]().

## Quick Example

```makefile
$(module foo, gcc, static)

me.sources := foo.c
me.export.libdirs := out/foo
me.export.libs := foo

$(commit)
```

Why
===

Build faster the simple way! (By comparing to autogen, autoconf, automake and makefile.)
