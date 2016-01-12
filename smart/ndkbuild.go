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
)

func init() {
        ndk := &_ndkbuild{}
        registerToolset("ndk-build", ndk)

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

        message("ndk-build: %v", ndk.root)
}

type _ndkbuild struct {
        root string
}

func (ndk *_ndkbuild) configModule(ctx *context, args []string, vars map[string]string) bool {
        if ctx.module != nil {
                var (
                        abi = "armeabi" // all
                        optim = "debug" // release|debug
                        platform = "android-9" // minimal is "android-8"
                        script = "obj/boot.mk"
                        stl = "system" // system|stlport_static|gnustl_static
                )

                // APP_BUILD_SCRIPT, APP_PLATFORM, APP_STL, APP_ABI, APP_OPTIM
                if s, ok := vars["BUILD_SCRIPT"]; ok { script = s; ctx.set("this.is_custom_script", "yes") }
                if s, ok := vars["PLATFORM"];     ok { platform = s }
                if s, ok := vars["STL"];          ok { stl = s }
                if s, ok := vars["ABI"];          ok { abi = s }
                if s, ok := vars["OPTIM"];        ok { optim = s }

                ctx.set("this.abi",      strings.TrimSpace(abi))
                ctx.set("this.optim",    strings.TrimSpace(optim))
                ctx.set("this.platform", strings.TrimSpace(platform))
                ctx.set("this.script",   strings.TrimSpace(script))
                ctx.set("this.stl",      strings.TrimSpace(stl))

                //message("ndk-build: config: name=%v, dir=%v, args=%v, vars=%v", ctx.module.name, ctx.module.dir, args, vars)
                return script != ""
        }
        return false
}

func (ndk *_ndkbuild) createActions(ctx *context, args []string) bool {
        //message("ndk-build: createActions: %v", ctx.module.name)

        targets, prequisites := make(map[string]int, 4), make(map[string]int, 16)

        cmd := &_ndkbuildCmd{
                abis:     strings.Fields(ctx.call("this.abi")),
                abi:      ctx.call("this.abi"),
                script:   ctx.call("this.script"),
                platform: ctx.call("this.platform"),
                stl:      ctx.call("this.stl"),
                optim:    ctx.call("this.optim"),
        }
        if !filepath.IsAbs(cmd.script) {
                cmd.script = filepath.Join(ctx.module.dir, ctx.call("this.script"))
        }

        ctx.module.action = &action{ command:cmd }
        ctx.module.action.prequisites = append(ctx.module.action.prequisites, newAction(cmd.script, nil))
        prequisites[cmd.script]++

        var scripts []string
        if ctx.call("this.is_custom_script") == "yes" {
                scripts = append(scripts, cmd.script)
        } else {
                pa := ctx.module.action.prequisites[0]
                pa.command = &_ndkbuildGenBuildScript{}

                var e error
                if scripts, e = findFiles(ctx.module.dir, `Android\.mk$`); e == nil {
                        for _, s := range scripts {
                                if filepath.Base(s) != "Android.mk" { continue }
                                pa.prequisites = append(pa.prequisites, newAction(s, nil))
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
                ctx.module.action.targets = append(ctx.module.action.targets, s)
        }
        for s, _ := range prequisites {
                ctx.module.action.prequisites = append(ctx.module.action.prequisites,
                        newAction(s, nil))
        }

        return true
}

func (ndk *_ndkbuild) useModule(ctx *context, m *module) bool {
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

func (g *_ndkbuildGenBuildScript) execute(targets []string, prequisites []string) bool {
        //message("ndkbuild: GenBuildScript: %v: %v", targets, prequisites)

        buf := new(bytes.Buffer)
        fmt.Fprintf(buf, "# %v\n", targets)
        for _, p := range prequisites {
                fmt.Fprintf(buf, "include %s\n", p)
        }

        for _, t := range targets {
                if e := os.MkdirAll(filepath.Dir(t), os.FileMode(0755)); e != nil {
                        message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }

                f, e := os.Create(t)
                if e != nil {
                        message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
                defer f.Close()
                if _, e := f.Write(buf.Bytes()); e != nil {
                        message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
        }
        return true
}

func (g *_ndkbuildGenDumpScript) execute(targets []string, prequisites []string) bool {
        message("ndkbuild: GenDumpScript: %v: %v", targets, prequisites)

        buf := new(bytes.Buffer)
        fmt.Fprintf(buf, "# %v\n", targets)
        for _, p := range prequisites {
                fmt.Fprintf(buf, "include %s\n", p)
        }

        for _, t := range targets {
                f, e := os.Create(t)
                if e != nil {
                        message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
                defer f.Close()
                if _, e := f.Write(buf.Bytes()); e != nil {
                        message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
        }
        return true
}

func (n *_ndkbuildCmd) execute(targets []string, prequisites []string) bool {
        //message("ndkbuild: %v: %v", targets, prequisites)
        c := &excmd{ path:"ndk-build" }
        vars := n.getBuildVars(n.abi, n.script)
        return c.run(fmt.Sprintf("%v", targets), vars...)
}

func (n *_ndkbuildCmd) getBuildVars(abi string, script string) []string {
        return []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                //fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_OUT=%s", "obj"),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
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

        c := &excmd{ path:"ndk-build" }
        if c.run("" /*fmt.Sprintf("%s (%s)", filepath.Base(tf), abi)*/, vars...) {
                //fmt.Printf("%v", c.stdout.String())
                ctx := &context{
                        l: lex{ s:c.stdout.Bytes() },
                        variables: make(map[string]*variable, 16),
                }
                if e := ctx.parse(); e != nil {
                        errorf(0, "DummyDump: %v", e)
                }
                res.ndkRoot = ctx.call("NDK_ROOT")
                res.targetOut = ctx.call("TARGET_OUT")
                res.targetObjs = ctx.call("TARGET_OBJS")
                res.targetGdbSetup = ctx.call("TARGET_GDB_SETUP")
                res.targetGdbServer = ctx.call("TARGET_GDB_SERVER")
                res.modules = strings.Fields(ctx.call("MODULES"))
                for _, s := range res.modules {
                        m := &_ndkbuildModuleInfo{}
                        m.name = ctx.call(s+".NAME")
                        m.filename = ctx.call(s+".FILENAME")
                        m.path = ctx.call(s+".PATH")
                        m.sources = strings.Fields(ctx.call(s+".SOURCES"))
                        m.script = ctx.call(s+".SCRIPT")
                        m.objsDir = ctx.call(s+".OBJS_DIR")
                        m.built = ctx.call(s+".BUILT")
                        m.installed = ctx.call(s+".INSTALLED")
                        m.class = ctx.call(s+".CLASS")
                        res.m[s] = *m
                }
        }
        return
}

func (n *_ndkbuildCmd) dumpSingle(abi string) (res *_ndkbuildDump) {
        res = &_ndkbuildDump{}

        tf := n.createDummyDumpFile(); defer os.Remove(tf)

        c := &excmd{ path:"ndk-build" }

        vars := []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                //fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_OUT=%s", "obj"),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
                //fmt.Sprintf("NDK_APP_NAME=%s", "."),
                fmt.Sprintf("APP_BUILD_SCRIPT=%s", tf),
                fmt.Sprintf("APP_ABI=%s", abi),
                fmt.Sprintf("APP_PLATFORM=%s", n.platform),
                fmt.Sprintf("APP_STL=%s", n.stl),
                fmt.Sprintf("APP_OPTIM=%s", n.optim),
                "dummy-dump",
        }

        if c.run("", vars...) {
                //fmt.Printf( "%v", c.stdout.String() )
                ctx := &context{
                        l: lex{ s:c.stdout.Bytes() },
                        variables: make(map[string]*variable, 16),
                }
                if e := ctx.parse(); e != nil {
                        errorf(0, "DummyDump: %v", e)
                }
                //res.appName = ctx.call("NDK_APP_NAME")
                //res.module = ctx.call("LOCAL_MODULE")
                //res.moduleClass = ctx.call("LOCAL_MODULE_CLASS")
                //res.srcFiles = ctx.call("LOCAL_SRC_FILES")
                //res.builtModule = ctx.call("LOCAL_BUILT_MODULE")
                res.targetOut = ctx.call("TARGET_OUT")
                res.targetObjs = ctx.call("TARGET_OBJS")
                res.targetGdbSetup = ctx.call("TARGET_GDB_SETUP")
                res.targetGdbServer = ctx.call("TARGET_GDB_SERVER")
                //fmt.Printf( "%v\n", ctx.variables )
                //fmt.Printf( "%v\n", ctx.call("LOCAL_OBJECTS") )
        }

        return
}

func (n *_ndkbuildCmd) createDummyDumpFile() string {
        tf, e := ioutil.TempFile("/tmp", "smart-dummy-dump-")
        if e != nil {
                errorf(0, "TempFile: %v", e)
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
                errorf(0, "TempFile: %v", e)
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
echo "$s.NAME :=$(__ndk_modules.$s.MODULE)" &&\
echo "$s.FILENAME :=$(__ndk_modules.$s.MODULE_FILENAME)" &&\
echo "$s.PATH :=$(__ndk_modules.$s.PATH)" &&\
echo "$s.SOURCES :=$(__ndk_modules.$s.SRC_FILES)" &&\
echo "$s.SCRIPT :=$(__ndk_modules.$s.MAKEFILE)" &&\
echo "$s.OBJS_DIR :=$(__ndk_modules.$s.OBJS_DIR)" &&\
echo "$s.BUILT :=$(__ndk_modules.$s.BUILT_MODULE)" &&\
echo "$s.INSTALLED :=$(__ndk_modules.$s.INSTALLED)" &&\
echo "$s.CLASS :=$(__ndk_modules.$s.MODULE_CLASS)" &&\
) true
	@echo
`, filepath.Base(tf.Name()))
        return tf.Name()
}
