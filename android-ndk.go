package smart

import (
        "fmt"
        "os"
        "os/exec"
        "path/filepath"
        //"strings"
)

var androidndk = "/android-ndk-r7"

func init() {
        registerToolset("android-ndk", &_androidndk{})

        /****/ if c, e := exec.LookPath("ndk-build"); e == nil {
                androidndk = filepath.Dir(c)
        } else {
                if androidndk = os.Getenv("ANDROIDNDK"); androidndk == "" {
                        fmt.Printf("can't locate Android NDK: %v\n", e)
                }
        }
}

type _androidndk struct {
        _gcc
}

func (ndk *_androidndk) toolchainBin() string {
        toolchain := "arm-linux-androideabi-4.4.3"
        return filepath.Join(androidndk, "toolchains", toolchain, "prebuilt/linux-x86/bin")
}

func (ndk *_androidndk) setupModule(p *parser, args []string) bool {
        if !ndk._gcc.setupModule(p, args) {
                return false
        }

        var m = p.module
        var ld *gccCommand
        if c, ok := m.action.command.(*gccCommand); !ok {
                errorf(0, "not a gcc command")
        } else {
                ld = c
        }

        bin := ndk.toolchainBin()
        switch ld.name {
        case "ld":
                ld.path = filepath.Join(bin, "arm-linux-androideabi-ld")
        case "ar":
                ld.path = filepath.Join(bin, "arm-linux-androideabi-ar")
        }
        return true
}

func (ndk *_androidndk) buildModule(p *parser, args []string) bool {
        if !ndk._gcc.buildModule(p, args) {
                return false
        }

        var m = p.module
        var ld *gccCommand
        if l, ok := m.action.command.(*gccCommand); !ok {
                p.stepLineBack(); errorf(0, "internal: wrong module command")
        } else {
                ld = l
        }

        bin := ndk.toolchainBin()
        switch ld.name {
        case "gcc":
                ld.path = filepath.Join(bin, "arm-linux-androideabi-gcc")
        case "g++":
                ld.path = filepath.Join(bin, "arm-linux-androideabi-g++")
        }
        return true
}
