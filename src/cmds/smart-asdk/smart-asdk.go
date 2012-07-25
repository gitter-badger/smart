package main

import (
        "../../pkg/smart/asdk"
        "os"
)

func main() {
	args := os.Args
        asdk.CommandLine(args)
}
