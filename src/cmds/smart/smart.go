package main

import (
        "path/filepath"
        "flag"
        "fmt"
        "os"
        "os/exec"
)

var root = "bin"

func main() {
        root = filepath.Dir(os.Args[0])

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
                fmt.Fprintf(os.Stderr, "sub command required\n")
                os.Exit(-1)
        }

        name := "smart-"+args[0]
        args  = args[1:]

        p := exec.Command(filepath.Join(root, name), args...)
        p.Stdin, p.Stdout, p.Stderr = os.Stdin, os.Stdout, os.Stderr

        if e := p.Run(); e != nil {
                fmt.Fprintf(os.Stderr, "%v\n", e)
                os.Exit(-1)
        }
}
