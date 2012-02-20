package smart

import (
        "bufio"
        "fmt"
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "strings"
)

func init() {
        ndk := &_androidndk{ root:"", toolchainByAbi:make(map[string]string, 5) }
        registerToolset("android-ndk", ndk)

        /****/ if c, e := exec.LookPath("ndk-build"); e == nil {
                ndk.root = filepath.Dir(c)
        } else {
                if ndk.root = os.Getenv("ANDROIDNDK"); ndk.root == "" {
                        fmt.Printf("can't locate Android NDK: %v\n", e)
                }
        }

        if ndk.root == "" {
                return
        }

        toolchainsDir := filepath.Join(ndk.root, "toolchains")

        fd, err := os.Open(toolchainsDir)
        if err != nil {
                fmt.Printf("no toolchains in Android NDK `%v'\n", ndk.root)
                return
        }

        defer fd.Close()

        names, err := fd.Readdirnames(5)
        if err != nil {
                fmt.Printf("no toolchains in Android NDK `%v' (%v)\n", ndk.root, err)
                return
        }

        for _, name := range names {
                d := filepath.Join(toolchainsDir, name)
                if !ndk.addToolchain(d) {
                        fmt.Printf("bad toolchain `%v'\n", d)
                }
        }
}

type _androidndk struct {
        _gcc
        root string
        toolchainByAbi map[string]string
        modules map[string]*module
}

func (ndk *_androidndk) parseFile(fn string, vars map[string]string) (p *parser, err error) {
        f , err := os.Open(fn)
        if err != nil {
                return
        }

        p = &parser{
                file: fn,
                in: bufio.NewReader(f),
                variables: make(map[string]*variable, 128),
        }

        if vars != nil {
                for n, v := range vars {
                        p.setVariable(n, v)
                }
        }

        defer func() {
                f.Close()

                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("%v: %v\n", p.location(), se)
                        } else {
                                panic(e)
                        }
                }
        }()

        if err = p.parse(); err != nil {
                return
        }

        return
}

func (ndk *_androidndk) addToolchain(d string) bool {
        toolchain := filepath.Base(d)
        if toolchain == "" {
                fmt.Printf("error: toolchain: %v\n", d)
                return false
        }

        fn := filepath.Join(d, "config.mk")
        p, err := ndk.parseFile(fn, nil)
        if err != nil {
                fmt.Printf("error: toolchain: %v\n", err)
                return false
        }

        if s := strings.TrimSpace(p.call("TOOLCHAIN_ABIS")); s != "" {
                abis := strings.Split(s, " ")
                for _, abi := range abis {
                        //fmt.Printf("%v: %v, %v\n", fn, toolchain, abi)
                        ndk.toolchainByAbi[abi] = toolchain
                }
                return true
        }

        return false
}

func (ndk *_androidndk) toolchain(abi string) string {
        osname := ""

        switch runtime.GOOS {
        case "linux": osname = "linux-x86"
        default:
                print("TODO: choose Android NDK toolchain for `"+runtime.GOOS+"'\n")
        }

        if osname == "" || abi == "" {
                return ""
        }

        //toolchain := "arm-linux-androideabi-4.4.3"
        if toolchain, ok := ndk.toolchainByAbi[abi]; ok {
                return filepath.Join(ndk.root, "toolchains", toolchain, "prebuilt", osname)
        }
        return ""
}

func (ndk *_androidndk) setupModule(p *parser, args []string, vars map[string]string) bool {
        if !ndk._gcc.setupModule(p, args, vars) {
                return false
        }

        var m = p.module
        if _, ok := ndk.modules[p.module.name]; ok {
                //errorf(0, "module `%v' already defined in $ANDROIDNDK/sources", m.name)
        }

        var ld *gccCommand
        if c, ok := m.action.command.(*gccCommand); !ok {
                errorf(0, "not a gcc command")
        } else {
                ld = c
        }

        var abi, platform string
        if s, ok := vars["ABI"]; ok { abi = s } else { abi = "armeabi" }
        if s, ok := vars["PLATFORM"]; ok { platform = s } else { platform = "android-9" }

        bin := filepath.Join(ndk.toolchain(abi), "bin")
        switch ld.name {
        case "ld": ld.path = filepath.Join(bin, "arm-linux-androideabi-ld")
        case "ar": ld.path = filepath.Join(bin, "arm-linux-androideabi-ar")
        }

        ld.name, ld.ia32 = filepath.Base(ld.path), true

        arch := "arch-"
        switch {
        case strings.HasSuffix(abi, "armeabi"): arch += "arm"
        case abi == "x86": arch += "x86"
        }

        includes := filepath.Join(ndk.root, "platforms", platform, arch, "usr/include")
        libdirs := filepath.Join(ndk.root, "platforms", platform, arch, "usr/lib")

        var v *variable
        loc := location{ file:&(p.file), lineno:p.lineno-1, colno:p.prevColno+1 }
        v = p.setVariable("this.abi", abi); v.loc = loc
        v = p.setVariable("this.platform", platform); v.loc = loc
        v = p.setVariable("this.includes", includes); v.loc = loc
        v = p.setVariable("this.libdirs", libdirs); v.loc = loc
        return true
}

func (ndk *_androidndk) buildModule(p *parser, args []string) bool {
        if !ndk._gcc.buildModule(p, args) {
                return false
        }

        var m = p.module

        platform := strings.TrimSpace(p.call("this.platform"))
        if platform == "" {
                errorf(0, "unkown platform for `%v'", m.name)
        }

        bin := filepath.Join(ndk.toolchain(p.call("this.abi")), "bin")
        binAs := filepath.Join(bin, "arm-linux-androideabi-as")
        binGcc := filepath.Join(bin, "arm-linux-androideabi-gcc")
        binGxx := filepath.Join(bin, "arm-linux-androideabi-g++")

        var setCommands func(a *action)
        setCommands = func(a *action) {
                for _, pre := range a.prequisites {
                        if pre.command == nil { continue }
                        if c, ok := pre.command.(*gccCommand); !ok {
                                fmt.Printf("%v: wrong command `%v'\n", p.location(), pre.command)
                                continue
                        } else {
                                switch c.name {
                                case "as":  c.path = binAs
                                case "gcc": c.path = binGcc
                                case "g++": c.path = binGxx
                                default: errorf(0, "unknown command %v (%v)", c.name, c.path)
                                }

                                c.name, c.ia32 = filepath.Base(c.path), true
                        }
                        setCommands(pre)
                }
        }
        setCommands(m.action)
        return true
}

func (ndk *_androidndk) loadModule(fn, ndksrc, subdir string) (ok bool) {
        p, err := ndk.parseFile(fn, map[string]string{
                "my-dir": filepath.Join(ndksrc, subdir),
        })
        if err != nil {
                errorf(0, "failed to load module resident in $ANDROIDNDK/sources/%v", subdir)
        }

        fmt.Printf("ndk: %v, %v\n", subdir, p.call("LOCAL_PATH"))
        return false
}

func (ndk *_androidndk) loadModules() (ok bool) {
        ndksrc := filepath.Join(ndk.root, "sources")
        err := traverse(ndksrc, func(fn string, fi os.FileInfo) bool {
                if !fi.IsDir() && fi.Name() == "Android.mk" {
                        if ok = ndk.loadModule(fn, ndksrc, filepath.Dir(fn[len(ndksrc)+1:])); !ok {
                                //return false
                        }
                }
                return true
        })
        if err != nil {
                return false
        }
        return ok
}

func (ndk *_androidndk) useModule(p *parser, m *module) bool {
        if !(m.toolset == nil && m.kind == "") {
                //errorf(0, "no toolset for `%v'", p.module.name)
                return false
        }

        if ndk.modules == nil && !ndk.loadModules() {
                errorf(0, "failed to load modules resident in $ANDROIDNDK/sources")
        }

        fmt.Printf("use: %v by %v\n", m.name, p.module.name)

        return false
}
