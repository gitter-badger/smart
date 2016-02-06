//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"os"
        . "github.com/duzy/smart/build"
)

func init() {
        RegisterToolset("clang", &toolset{})
}

type toolset struct {
}

func (clang *toolset) ConfigModule(ctx *Context, args []string, vars map[string]string) bool {
        return true
}

func (clang *toolset) CreateActions(ctx *Context) bool {
        return false
}

func (clang *toolset) UseModule(ctx *Context, m, o *Module) bool {
        return false
}
