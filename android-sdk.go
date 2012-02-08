package smart

import (
        "os"
)

func init() {
        registerToolset("android-sdk", &_androidsdk{})
}

type _androidsdk struct {
}

func (sdk *_androidsdk) processFile(dname string, fi os.FileInfo) {
}
