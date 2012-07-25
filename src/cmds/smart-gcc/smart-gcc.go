package main

import (
        "../../pkg/smart/gcc"
        "os"
)

func main() {
        args := os.Args
        gcc.CommandLine(args)
}
