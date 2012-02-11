package smart

import (
        "fmt"
        "strings"
)

var internals = map[string]func(args []string) string {
        "info": internalInfo,
        "module": internalModule,
        "build": internalBuild,
        "use": internalUse,
}

func internalInfo(args []string) string {
        fmt.Printf("%v\n", strings.Join(args, " "))
        return ""
}

func internalModule(args []string) string {
        fmt.Printf("TODO: module: %v\n", args)
        return ""
}

func internalBuild(args []string) string {
        fmt.Printf("TODO: build: %v\n", args)
        return ""
}

func internalUse(args []string) string {
        fmt.Printf("TODO: use: %v\n", args)
        return ""
}
