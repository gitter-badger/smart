package main

import (
        "path/filepath"
        "flag"
        "fmt"
        "os"
        "os/exec"
)

var root = "bin"
var cmdLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
var flagOutput = cmdLine.String("o", "", "output string")

func main() {
        args := os.Args[1:]
        subCmdIdx := -1

        for i, arg := range args {
                switch {
                case i == 0: continue
                case arg[0] != '-':
                        cmdLine.Parse(args[0:i])
                        subCmdIdx, args = 1+i, args[i:]
                }
        }

        if subCmdIdx < 1 || len(args) < 1 {
                fmt.Fprintf(os.Stderr, "sub command required (TODO: show usage)")
                os.Exit(-1)
        }

        name := args[0]
        args = args[1:]

        p := exec.Command(filepath.Join(root, name), args...)
        p.Stdin, p.Stdout, p.Stderr = os.Stdin, os.Stdout, os.Stderr

        p.Run()
        
        os.Exit(0)
}
