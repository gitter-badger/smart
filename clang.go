package smart

import (
        //"os"
)

func init() {
        registerToolset("clang", &_clang{})
}

type _clang struct {
}

func (clang *_clang) setupModule(p *parser, args []string) bool {
        return true
}

func (clang *_clang) buildModule(p *parser, args []string) bool {
        return false
}
