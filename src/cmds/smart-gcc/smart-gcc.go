package main

import (
        "../../pkg/smart/gcc"
        "os"
)

func main() {
        args := os.Args[1:]
        gcc.CommandLine(args)
}
