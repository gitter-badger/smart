//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "fmt"
        "bytes"
        "os"
        "os/exec"
        "io/ioutil"
        "path/filepath"
        //"runtime"
        "strings"
        . "github.com/duzy/smart/build"
)

func init() {
        ndk := &_ndkbuild{}
        RegisterToolset("ndk-build", ndk)

        if c, e := exec.LookPath("ndk-build"); e == nil {
                ndk.root = filepath.Dir(c)
        } else {
                if ndk.root = os.Getenv("ANDROIDNDK"); ndk.root == "" {
                        Message("cant locate Android NDK: %v", e)
                }
        }

        if ndk.root == "" {
                return
        }

        Message("ndk-build: %v", ndk.root)
}

type _ndkbuild struct {
        root string
}

func (ndk *_ndkbuild) ConfigModule(ctx *Context, m *Module, args []string, vars map[string]string) bool {
        if m != nil {
                var (
                        abi = "armeabi" // all
                        optim = "debug" // release|debug
                        platform = "android-9" // minimal is "android-8"
                        script = "out/boot.mk"
                        stl = "system" // system|stlport_static|gnustl_static
                )

                // APP_BUILD_SCRIPT, APP_PLATFORM, APP_STL, APP_ABI, APP_OPTIM
                if s, ok := vars["BUILD_SCRIPT"]; ok { script = s; ctx.Set("me.is_custom_script", "yes") }
                if s, ok := vars["PLATFORM"];     ok { platform = s }
                if s, ok := vars["STL"];          ok { stl = s }
                if s, ok := vars["ABI"];          ok { abi = s }
                if s, ok := vars["OPTIM"];        ok { optim = s }

                ctx.Set("me.abi",      strings.TrimSpace(abi))
                ctx.Set("me.optim",    strings.TrimSpace(optim))
                ctx.Set("me.platform", strings.TrimSpace(platform))
                ctx.Set("me.script",   strings.TrimSpace(script))
                ctx.Set("me.stl",      strings.TrimSpace(stl))

                Message("ndk-build: config: name=%v, dir=%v, args=%v, vars=%v", m.Name, m.Dir, args, vars)
                return script != ""
        }
        return false
}

func (ndk *_ndkbuild) CreateActions(ctx *Context, m *Module, args []string) bool {
        //Message("ndk-build: createActions: %v", m.name)

        targets, prequisites := make(map[string]int, 4), make(map[string]int, 16)

        cmd := &_ndkbuildCmd{
                abis:     strings.Fields(ctx.Call("me.abi")),
                abi:      ctx.Call("me.abi"),
                script:   ctx.Call("me.script"),
                platform: ctx.Call("me.platform"),
                stl:      ctx.Call("me.stl"),
                optim:    ctx.Call("me.optim"),
        }
        if !filepath.IsAbs(cmd.script) {
                cmd.script = filepath.Join(m.Dir, ctx.Call("me.script"))
        }

        m.Action = new(Action)
        m.Action.Command = cmd
        m.Action.Prequisites = append(m.Action.Prequisites, NewAction(cmd.script, nil))
        prequisites[cmd.script]++

        var scripts []string
        if ctx.Call("me.is_custom_script") == "yes" {
                scripts = append(scripts, cmd.script)
        } else {
                pa := m.Action.Prequisites[0]
                pa.Command = &_ndkbuildGenBuildScript{}

                var e error
                if scripts, e = FindFiles(m.Dir, `Android\.mk$`); e == nil {
                        for _, s := range scripts {
                                if filepath.Base(s) != "Android.mk" { continue }
                                pa.Prequisites = append(pa.Prequisites, NewAction(s, nil))
                        }
                }
        }

        for _, s := range cmd.abis {
                dump := cmd.dumpAll(s, scripts) //; fmt.Printf("dump: %v\n", dump)
                for _, p := range dump.m {
                        // skip standard modules, e.g. stdc++
                        if strings.HasPrefix(p.path, dump.ndkRoot) { continue }

                        //fmt.Printf("target: %v: %v\n", s, p.built)
                        //fmt.Printf("target: %v: %v\n", s, p.installed)

                        targets[strings.TrimPrefix(p.built, "./")]++
                        prequisites[strings.TrimPrefix(p.script, "./")]++
                        for _, s := range p.sources {
                                prequisites[filepath.Join(p.path, s)]++
                        }
                }
        }

        //fmt.Printf("targets: %v\n", targets)

        for s, _ := range targets {
                m.Action.Targets = append(m.Action.Targets, s)
        }
        for s, _ := range prequisites {
                m.Action.Prequisites = append(m.Action.Prequisites, NewAction(s, nil))
        }

        return true
}

func (ndk *_ndkbuild) UseModule(ctx *Context, m, o *Module) bool {
        return false
}

type _ndkbuildGenBuildScript struct {}
type _ndkbuildGenDumpScript struct {}
type _ndkbuildModuleInfo struct {
        name, filename, path, script, objsDir, built, installed, class string
        sources []string
}
type _ndkbuildDump struct {
        ndkRoot, targetOut, targetObjs, targetGdbSetup, targetGdbServer string
        modules []string
        m map[string]_ndkbuildModuleInfo
}

type _ndkbuildCmd struct {
        script, abi, platform, stl, optim string
        abis []string
}

func (g *_ndkbuildGenBuildScript) Execute(targets []string, prequisites []string) bool {
        //Message("ndkbuild: GenBuildScript: %v: %v", targets, prequisites)

        buf := new(bytes.Buffer)
        fmt.Fprintf(buf, "# %v\n", targets)
        for _, p := range prequisites {
                fmt.Fprintf(buf, "include %s\n", p)
        }

        for _, t := range targets {
                if e := os.MkdirAll(filepath.Dir(t), os.FileMode(0755)); e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }

                f, e := os.Create(t)
                if e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
                defer f.Close()
                if _, e := f.Write(buf.Bytes()); e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
        }
        return true
}

func (g *_ndkbuildGenDumpScript) Execute(targets []string, prequisites []string) bool {
        Message("ndkbuild: GenDumpScript: %v: %v", targets, prequisites)

        buf := new(bytes.Buffer)
        fmt.Fprintf(buf, "# %v\n", targets)
        for _, p := range prequisites {
                fmt.Fprintf(buf, "include %s\n", p)
        }

        for _, t := range targets {
                f, e := os.Create(t)
                if e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
                defer f.Close()
                if _, e := f.Write(buf.Bytes()); e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
        }
        return true
}

func (n *_ndkbuildCmd) Execute(targets []string, prequisites []string) bool {
        //Message("ndkbuild: %v: %v", targets, prequisites)
        c := NewExcmd("ndk-build")
        vars := n.getBuildVars(n.abi, n.script)
        return c.Run(fmt.Sprintf("%v", targets), vars...)
}

func (n *_ndkbuildCmd) getBuildVars(abi string, script string) []string {
        return []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                //fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
                fmt.Sprintf("NDK_OUT=%s", "out"),
                fmt.Sprintf("APP_BUILD_SCRIPT=%s", script),
                fmt.Sprintf("APP_ABI=%s", abi),
                fmt.Sprintf("APP_PLATFORM=%s", n.platform),
                fmt.Sprintf("APP_STL=%s", n.stl),
                fmt.Sprintf("APP_OPTIM=%s", n.optim),
        }
}

func (n *_ndkbuildCmd) dumpAll(abi string, scripts []string) (res *_ndkbuildDump) {
        res = &_ndkbuildDump{ m:make(map[string]_ndkbuildModuleInfo) }

        tf := n.createSmartDumpFile(scripts); defer os.Remove(tf)

        vars := n.getBuildVars(abi, tf)
        vars = append(vars, filepath.Base(tf))

        c := NewExcmd("ndk-build")
        if c.Run("" /*fmt.Sprintf("%s (%s)", filepath.Base(tf), abi)*/, vars...) {
                //fmt.Printf("%v", c.GetStdout().String())
                ctx, e := NewContext("DummyDump", c.GetStdout().Bytes(), nil)
                if e != nil { Fatal("DummyDump: %v", e) }
                res.ndkRoot = ctx.Call("NDK_ROOT")
                res.targetOut = ctx.Call("TARGET_OUT")
                res.targetObjs = ctx.Call("TARGET_OBJS")
                res.targetGdbSetup = ctx.Call("TARGET_GDB_SETUP")
                res.targetGdbServer = ctx.Call("TARGET_GDB_SERVER")
                res.modules = strings.Fields(ctx.Call("MODULES"))
                for _, s := range res.modules {
                        m := &_ndkbuildModuleInfo{}
                        m.name = ctx.Call(s+".NAME")
                        m.filename = ctx.Call(s+".FILENAME")
                        m.path = ctx.Call(s+".PATH")
                        m.sources = strings.Fields(ctx.Call(s+".SOURCES"))
                        m.script = ctx.Call(s+".SCRIPT")
                        m.objsDir = ctx.Call(s+".OBJS_DIR")
                        m.built = ctx.Call(s+".BUILT")
                        m.installed = ctx.Call(s+".INSTALLED")
                        m.class = ctx.Call(s+".CLASS")
                        res.m[s] = *m
                }
        }
        return
}

func (n *_ndkbuildCmd) dumpSingle(abi string) (res *_ndkbuildDump) {
        res = &_ndkbuildDump{}

        tf := n.createDummyDumpFile(); defer os.Remove(tf)

        c := NewExcmd("ndk-build")

        vars := []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                //fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_APP_NAME=%s", "."),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
                fmt.Sprintf("NDK_OUT=%s", "out"),
                fmt.Sprintf("APP_BUILD_SCRIPT=%s", tf),
                fmt.Sprintf("APP_ABI=%s", abi),
                fmt.Sprintf("APP_PLATFORM=%s", n.platform),
                fmt.Sprintf("APP_STL=%s", n.stl),
                fmt.Sprintf("APP_OPTIM=%s", n.optim),
                "dummy-dump",
        }

        if c.Run("", vars...) {
                //fmt.Printf( "%v", c.GetStdout().String() )
                ctx, e := NewContext("DummyDump", c.GetStdout().Bytes(), nil)
                if e != nil { Fatal("DummyDump: %v", e) }
                //res.appName = ctx.Call("NDK_APP_NAME")
                //res.module = ctx.Call("LOCAL_MODULE")
                //res.moduleClass = ctx.Call("LOCAL_MODULE_CLASS")
                //res.srcFiles = ctx.Call("LOCAL_SRC_FILES")
                //res.builtModule = ctx.Call("LOCAL_BUILT_MODULE")
                res.targetOut = ctx.Call("TARGET_OUT")
                res.targetObjs = ctx.Call("TARGET_OBJS")
                res.targetGdbSetup = ctx.Call("TARGET_GDB_SETUP")
                res.targetGdbServer = ctx.Call("TARGET_GDB_SERVER")
                //fmt.Printf( "%v\n", ctx.variables )
                //fmt.Printf( "%v\n", ctx.Call("LOCAL_OBJECTS") )
        }

        return
}

func (n *_ndkbuildCmd) createDummyDumpFile() string {
        tf, e := ioutil.TempFile("/tmp", "smart-dummy-dump-")
        if e != nil {
                Fatal("TempFile: %v", e)
        }

        defer tf.Close()

        //$(call import-module,third_party/googletest)
        //$(call import-module,native_app_glue)

        fmt.Fprintf(tf, `# dummy
include %s
dummy-dump: DUMMY_LOCAL_MODULE := $(LOCAL_MODULE)
dummy-dump: DUMMY_LOCAL_MODULE_CLASS := $(LOCAL_MODULE_CLASS)
dummy-dump: DUMMY_LOCAL_SRC_FILES := $(LOCAL_SRC_FILES)
dummy-dump: DUMMY_LOCAL_OBJECTS := $(LOCAL_OBJECTS)
dummy-dump: DUMMY_LOCAL_RS_OBJECTS := $(LOCAL_RS_OBJECTS)
dummy-dump: DUMMY_LOCAL_BUILT_MODULE := $(LOCAL_BUILT_MODULE)
dummy-dump: DUMMY_LOCAL_INSTALLED := $(LOCAL_INSTALLED)
dummy-dump: DUMMY_TARGET_OUT := $(TARGET_OUT)
dummy-dump: DUMMY_TARGET_OBJS := $(TARGET_OBJS)
dummy-dump: DUMMY_TARGET_GDB_SETUP := $(TARGET_GDB_SETUP)
dummy-dump: DUMMY_TARGET_GDB_SERVER := $(TARGET_GDBSERVER)
dummy-dump: DUMMY_TARGET_SONAME_EXTENSION := $(TARGET_SONAME_EXTENSION)
dummy-dump: DUMMY_NDK_APP_NAME := $(NDK_APP_NAME)
dummy-dump:
	@echo "LOCAL_MODULE :=$(DUMMY_LOCAL_MODULE)"
	@echo "LOCAL_MODULE_CLASS :=$(DUMMY_LOCAL_MODULE_CLASS)"
	@echo "LOCAL_SRC_FILES :=$(DUMMY_LOCAL_SRC_FILES)"
	@echo "LOCAL_OBJECTS :=$(DUMMY_LOCAL_OBJECTS)"
	@echo "LOCAL_RS_OBJECTS :=$(DUMMY_LOCAL_RS_OBJECTS)"
	@echo "LOCAL_BUILT_MODULE :=$(DUMMY_LOCAL_BUILT_MODULE)"
	@echo "LOCAL_INSTALLED :=$(DUMMY_LOCAL_INSTALLED)"
	@echo "TARGET_OUT :=$(DUMMY_TARGET_OUT)"
	@echo "TARGET_LIBS :=$(DUMMY_TARGET_LIBS)"
	@echo "TARGET_OBJS :=$(DUMMY_TARGET_OBJS)"
	@echo "TARGET_GDB_SETUP :=$(DUMMY_TARGET_GDB_SETUP)"
	@echo "TARGET_GDB_SERVER :=$(DUMMY_TARGET_GDB_SERVER)"
	@echo "TARGET_SONAME_EXTENSION :=$(DUMMY_TARGET_SONAME_EXTENSION)"
	@echo "NDK_APP_NAME :=$(DUMMY_NDK_APP_NAME)"
	@echo
`, n.script)
        return tf.Name()
}

func (n *_ndkbuildCmd) createSmartDumpFile(scripts []string) string {
        tf, e := ioutil.TempFile("/tmp", "smart-dummy-dump-")
        if e != nil {
                Fatal("TempFile: %v", e)
        }

        defer tf.Close()

        for _, s := range scripts {
                fmt.Fprintf(tf, "include %s\n", s)
        }
        fmt.Fprintf(tf, `# $(info $(modules-dump-database))
~ := %s
$~: DUMMY_TARGET_OUT := $(TARGET_OUT)
$~: DUMMY_TARGET_OBJS := $(TARGET_OBJS)
$~: DUMMY_TARGET_GDB_SETUP := $(TARGET_GDB_SETUP)
$~: DUMMY_TARGET_GDB_SERVER := $(TARGET_GDBSERVER)
$~: DUMMY_MODULES := $(modules-get-list)
$~:
	@echo "NDK_ROOT :=$(NDK_ROOT)"
	@echo "TARGET_OUT :=$(DUMMY_TARGET_OUT)"
	@echo "TARGET_OBJS :=$(DUMMY_TARGET_OBJS)"
	@echo "TARGET_GDB_SETUP :=$(DUMMY_TARGET_GDB_SETUP)"
	@echo "TARGET_GDB_SERVER :=$(DUMMY_TARGET_GDB_SERVER)"
	@echo "MODULES :=$(DUMMY_MODULES)"
	@$(foreach s,$(DUMMY_MODULES),\
echo "$s.NAME := $(__ndk_modules.$s.MODULE)" &&\
echo "$s.FILENAME := $(__ndk_modules.$s.MODULE_FILENAME)" &&\
echo "$s.PATH := $(__ndk_modules.$s.PATH)" &&\
echo "$s.SOURCES := $(__ndk_modules.$s.SRC_FILES)" &&\
echo "$s.SCRIPT := $(__ndk_modules.$s.MAKEFILE)" &&\
echo "$s.OBJS_DIR := $(__ndk_modules.$s.OBJS_DIR)" &&\
echo "$s.BUILT := $(__ndk_modules.$s.BUILT_MODULE)" &&\
echo "$s.INSTALLED := $(__ndk_modules.$s.INSTALLED)" &&\
echo "$s.CLASS := $(__ndk_modules.$s.MODULE_CLASS)" &&\
) true
	@echo
`, filepath.Base(tf.Name()))
        return tf.Name()
}
