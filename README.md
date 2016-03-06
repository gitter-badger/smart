# Smart (Drafting)

**Smart** is a [Semi-Functional Scripting Language]() designed for building hierachical tasks easily.
It's written in [Go](http://golang.org) programming language.

[![GoDoc](https://godoc.org/github.com/duzy/smart/build?status.svg)](http://godoc.org/github.com/duzy/smart/build)

## Overview

The language is inspired by [GNU make]() (having almost the same syntax as makefile), it could do the same
job as [GNU make]() but not the same as it. **Smart** is supposed to be used more in a programming manner of a
[semi-functional]() paradigm.

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
