package smart

import (
        //"path/filepath"
)

func init() {
        clang := &toolset{ name:"clang" }

        registerToolset(clang)
}

