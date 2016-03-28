# Smart (Drafting)
01234567890123456789012345678901234567890123456789012345678901234567890123456789
**Smart** is a [Semi-Functional Scripting Language]() designed for performing hierachical tasks easily.
It's written in [Go](http://golang.org) programming language.

[![GoDoc](https://godoc.org/github.com/duzy/smart/build?status.svg)](http://godoc.org/github.com/duzy/smart/build)

## Overview

The language is inspired by [GNU make]() (having almost the same syntax as makefile, but no directives),
it could do the same job as [GNU make]() but not the same as it. **Smart** is supposed to be used more in
a programming manner of a [semi-functional]() paradigm.

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

Get tasks of complex dependency done the easy way!
