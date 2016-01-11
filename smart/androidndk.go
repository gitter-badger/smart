package smart

import (
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "strings"
)

func init() {
        ndk := &_androidndk{ root:"", toolchainByAbi:make(map[string]string, 5) }
        registerToolset("android-ndk", ndk)

        if c, e := exec.LookPath("ndk-build"); e == nil {
                ndk.root = filepath.Dir(c)
        } else {
                if ndk.root = os.Getenv("ANDROIDNDK"); ndk.root == "" {
                        message("cant locate Android NDK: %v", e)
                }
        }

        if ndk.root == "" {
                return
        }

        toolchainsDir := filepath.Join(ndk.root, "toolchains")

        names, err := readDirNames(toolchainsDir)
        if err != nil {
                message("no toolchains in Android NDK `%v' (%v)", ndk.root, err)
                return
        }

        for _, name := range names {
                d := filepath.Join(toolchainsDir, name)
                if !ndk.addToolchain(d) {
                        //message("bad toolchain `%v'", d)
                }
        }
}

type _androidndk struct {
        _gcc
        root string
        toolchainByAbi map[string]string
        modules map[string]*module
}

func (ndk *_androidndk) parseFile(fn string, vars map[string]string) (ctx *context, err error) {
        if ctx , err = newContext(fn); err != nil {
                //message("error: %v", err)
                return
        }

        if vars != nil {
                for n, v := range vars {
                        ctx.set(n, v)
                }
        }

        defer func() {
                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                message("%v: %v", ctx.l.location(), se)
                        } else {
                                panic(e)
                        }
                }
        }()

        if err = ctx.parse(); err != nil {
                return
        }

        return
}

func (ndk *_androidndk) addToolchain(d string) bool {
        toolchain := filepath.Base(d)
        if toolchain == "" {
                message("error: toolchain: %v", d)
                return false
        }

        fn := filepath.Join(d, "config.mk")
        ctx, err := ndk.parseFile(fn, nil)
        if err != nil {
                //message("error: toolchain: %v", err)
                return false
        }

        if s := strings.TrimSpace(ctx.call("TOOLCHAIN_ABIS")); s != "" {
                abis := strings.Split(s, " ")
                for _, abi := range abis {
                        //message("%v: %v, %v", fn, toolchain, abi)
                        ndk.toolchainByAbi[abi] = toolchain
                }
                return true
        }

        return false
}

func (ndk *_androidndk) toolchainDir(abi string) string {
        osname := ""

        switch runtime.GOOS {
        case "linux":
                osname = "linux-x86"
                osname = "linux-x86_64"
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

func (ndk *_androidndk) configModule(ctx *context, args []string, vars map[string]string) bool {
        if !ndk._gcc.configModule(ctx, args, vars) {
                return false
        }

        var m = ctx.module
        if _, ok := ndk.modules[ctx.module.name]; ok {
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

        bin := filepath.Join(ndk.toolchainDir(abi), "bin")
        switch filepath.Base(ld.path) {
        case "ld": ld.path = filepath.Join(bin, "arm-linux-androideabi-ld")
        case "ar": ld.path = filepath.Join(bin, "arm-linux-androideabi-ar")
        }

        ld.ia32 = isIA32Command(ld.path)

        arch := "arch-"
        switch {
        case strings.HasSuffix(abi, "armeabi"): arch += "arm"
        case abi == "x86": arch += "x86"
        }

        includes := filepath.Join(ndk.root, "platforms", platform, arch, "usr/include")
        libdirs := filepath.Join(ndk.root, "platforms", platform, arch, "usr/lib")

        var v *variable
        loc := ctx.l.location()
        v = ctx.set("this.abi", abi); v.loc = *loc
        v = ctx.set("this.platform", platform); v.loc = *loc
        v = ctx.set("this.includes", includes); v.loc = *loc
        v = ctx.set("this.libdirs", libdirs); v.loc = *loc
        return true
}

func (ndk *_androidndk) createActions(ctx *context, args []string) bool {
        if !ndk._gcc.createActions(ctx, args) {
                return false
        }

        var m = ctx.module

        platform := strings.TrimSpace(ctx.call("this.platform"))
        if platform == "" {
                errorf(0, "unkown platform for `%v'", m.name)
        }

        bin := filepath.Join(ndk.toolchainDir(ctx.call("this.abi")), "bin")
        binAs := filepath.Join(bin, "arm-linux-androideabi-as")
        binGcc := filepath.Join(bin, "arm-linux-androideabi-gcc")
        binGxx := filepath.Join(bin, "arm-linux-androideabi-g++")

        var setCommands func(a *action)
        setCommands = func(a *action) {
                for _, pre := range a.prequisites {
                        if pre.command == nil { continue }
                        if c, ok := pre.command.(*gccCommand); !ok {
                                message("%v: wrong command `%v'", ctx.l.location(), pre.command)
                                continue
                        } else {
                                switch filepath.Base(c.path) {
                                case "as":  c.path = binAs
                                case "gcc": c.path = binGcc
                                case "g++": c.path = binGxx
                                default: errorf(0, "unknown command %v", c.path)
                                }

                                c.ia32 = isIA32Command(c.path)
                        }
                        setCommands(pre)
                }
        }
        setCommands(m.action)
        return true
}

func (ndk *_androidndk) loadModule(fn, ndksrc, subdir string) (ok bool) {
        ctx, err := ndk.parseFile(fn, map[string]string{
                "my-dir": filepath.Join(ndksrc, subdir),
        })

        if err != nil {
                errorf(0, "failed to load module resident in $ANDROIDNDK/sources/%v", subdir)
        }

        message("ndk: %v, %v", subdir, ctx.call("LOCAL_PATH"))
        return false
}

func (ndk *_androidndk) loadModules() (ok bool) {
        return true

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

func (ndk *_androidndk) useModule(ctx *context, m *module) bool {
        if !(m.toolset == nil && m.kind == "") {
                //errorf(0, "no toolset for `%v'", ctx.module.name)
                return false
        }

        if ndk.modules == nil && !ndk.loadModules() {
                errorf(0, "failed to load modules resident in $ANDROIDNDK/sources")
        }

        message("use: %v by %v", m.name, ctx.module.name)

        return false
}
