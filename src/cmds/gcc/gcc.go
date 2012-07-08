package main

import (
        "../../pkg/smart"
        "../../pkg/smart/gcc"
        "os"
        "fmt"
)

func main() {
        tool := gcc.New()

        if e := smart.Build(tool); e != nil {
                fmt.Fprintf(os.Stderr, "error: %v", e)
        }

        os.Exit(0)
}
