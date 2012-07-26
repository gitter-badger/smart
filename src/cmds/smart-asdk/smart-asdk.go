package main

import (
        "../../pkg/smart/asdk"
        "os"
)

func main() {
        args := os.Args[1:]
        asdk.CommandLine(args)
}
