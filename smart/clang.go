package smart

import (
        //"os"
)

func init() {
        registerToolset("clang", &_clang{})
}

type _clang struct {
}

func (clang *_clang) setupModule(p *context, args []string, vars map[string]string) bool {
        return true
}

func (clang *_clang) buildModule(p *context, args []string) bool {
        return false
}

func (clang *_clang) useModule(p *context, m *module) bool {
        return false
}
