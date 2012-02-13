package smart

import (
        //"os"
)

func init() {
        registerToolset("android-ndk", &_androidndk{})
}

type _androidndk struct {
}

func (ndk *_androidndk) setupModule(p *parser, args []string) bool {
        return true
}

func (ndk *_androidndk) buildModule(p *parser, args []string) bool {
        return false
}
