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

type toolset struct { BasicToolset }

func (clang *toolset) CreateActions(ctx *Context) bool {
        return false
}
