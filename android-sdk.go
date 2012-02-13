package smart

import (
        "os"
)

func init() {
        registerToolset("android-sdk", &_androidsdk{})
}

type _androidsdk struct {
}

func (sdk *_androidsdk) buildModule(p *parser, args []string) bool {
        return false
}

func (sdk *_androidsdk) processFile(dname string, fi os.FileInfo) {
}

func (sdk *_androidsdk) updateAll() {
        
}

func (sdk *_androidsdk) cleanAll() {
        
}
