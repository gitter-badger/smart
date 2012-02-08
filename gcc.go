package smart

import (
        "os"
)

func init() {
        registerToolset("gcc", &_gcc{})
}

type _gcc struct {
}

func (gcc *_gcc) processFile(dname string, fi os.FileInfo) {
        print("TODO: "+dname+"\n")
}
