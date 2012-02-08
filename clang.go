package smart

import (
        "os"
)

func init() {
        registerToolset("clang", &_clang{})
}

type _clang struct {
}

func (clang *_clang) processFile(dname string, fi os.FileInfo) {
}
