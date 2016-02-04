//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"os"
        . "github.com/duzy/smart/build"
)

func init() {
        RegisterToolset("clang", &_clang{})
}

type _clang struct {
}

func (clang *_clang) ConfigModule(ctx *Context, m *Module, args []string, vars map[string]string) bool {
        return true
}

func (clang *_clang) CreateActions(ctx *Context, m *Module, args []string) bool {
        return false
}

func (clang *_clang) UseModule(ctx *Context, m, o *Module) bool {
        return false
}
