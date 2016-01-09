# Build smartly (in Go)

[![GoDoc](https://godoc.org/github.com/duzy/smart/smart?status.svg)](http://godoc.org/github.com/duzy/smart/smart)

```makefile
$(module foo_static, gcc, static)

this.sources := foo.c
this.export.libdirs := out/foo_static
this.export.libs := foo_static

$(build)
```

Why
===

Build faster the simple way!
