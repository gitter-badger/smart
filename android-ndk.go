package smart

import (
        //"path/filepath"
)

func init() {
        androidndk := &toolset{ name:"android-ndk" }

        registerToolset(androidndk)
}

