//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"os"
        //"fmt"
        //"strings"
        //"path/filepath"
        . "github.com/duzy/smart/build"
)

func init() {
        AppendInit(`# Execute Shell Command
$(template shell)
start:!: $(me.depends) ; $(me.command) $(me.args)
$(commit)
`)
}
