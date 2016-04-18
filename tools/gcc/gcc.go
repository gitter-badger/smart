//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"path/filepath"
        "fmt"
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(
        HooksMap{
                "gcc": HookTable{
                        "objects": hookObjects,
                },
        }, `# Build GCC Projects
template gcc

$(me.name): $(gcc:objects $(me.sources))
	@echo "todo: $(me.name) ($(me.workdir))"



post
commit
`)

func hookObjects(ctx *Context, args Items) (res Items) {
        fmt.Printf("objects: %v\n", args)
        /*
        cmd := exec.Command("sh", "-c", args.Expand(ctx))
        if cmd != nil {
                stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
                cmd.Stdout, cmd.Stderr = stdout, stderr
                if err := cmd.Run(); err != nil {
                        // TODO: report errors
                        return
                }
                res = append(res, StringItem(stdout.String()))
        } else {
                // TODO: report errors
        } */
        return
}
