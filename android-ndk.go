package smart

import (
        "os"
)

func init() {
        registerToolset("android-ndk", &_androidndk{})
}

type _androidndk struct {
}

func (ndk *_androidndk) processFile(dname string, fi os.FileInfo) {
}
