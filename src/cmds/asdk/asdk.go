package main

import (
        "../../pkg/smart"
        "../../pkg/smart/asdk"
        "os"
        "fmt"
        "flag"
)

var commands = map[string] func(args []string) error {
        "build": build,
        "install": install,
        "create": create,
}

func build(args []string) (e error) {
        tool := asdk.New()
        e = smart.Build(tool)
        return
}

func install(args []string) (e error) {
        e = asdk.Install(args)
        return
}

func create(args []string) (e error) {
        e = asdk.Create(args)
        return
}

func main() {
        flag.Parse()

        args := flag.Args()

        if len(args) < 1 {
                fmt.Fprintf(os.Stderr, "asdk: no arguments\n")
                os.Exit(-1)
        }
        
        cmd := args[0]
        args = args[1:]

        if proc, ok := commands[cmd]; ok && proc != nil {
                if e := proc(args); e != nil {
                        fmt.Fprintf(os.Stderr, "asdk: %v\n", e)
                        os.Exit(-1)
                }
        } else {
                fmt.Fprintf(os.Stderr, "asdk: '%v' not supported\n", cmd)
                os.Exit(-1)
        }
}
