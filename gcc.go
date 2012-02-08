package smart

import (
        //"path/filepath"
)

func init() {
        gcc := &toolset{ name:"gcc" }

        registerToolset(gcc)
}

