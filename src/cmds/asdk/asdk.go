package main

import (
        "../../pkg/smart"
        "../../pkg/smart/asdk"
        "os"
        "fmt"
)

func main() {
        tool := asdk.New()

        if e := smart.Build(tool); e != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", e)
                os.Exit(-1)
        }

        os.Exit(0)
}
