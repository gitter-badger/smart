//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"os"
)

func init() {
        registerToolset("clang", &_clang{})
}

type _clang struct {
}

func (clang *_clang) configModule(ctx *context, args []string, vars map[string]string) bool {
        return true
}

func (clang *_clang) createActions(ctx *context, args []string) bool {
        return false
}

func (clang *_clang) useModule(ctx *context, m *module) bool {
        return false
}
