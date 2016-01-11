package smart

import (
        "fmt"
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
                        platform = "android-9"
                        script = "Android.mk"
                        stl = "system" // system|stlport_static|gnustl_static
                )

                // APP_BUILD_SCRIPT, APP_PLATFORM, APP_STL, APP_ABI, APP_OPTIM
                if s, ok := vars["BUILD_SCRIPT"]; ok { script = s }
                if s, ok := vars["PLATFORM"];     ok { platform = s }
                if s, ok := vars["STL"];          ok { stl = s }
                if s, ok := vars["ABI"];          ok { abi = s }
                if s, ok := vars["OPTIM"];        ok { optim = s }

                ctx.set("this.abi",      abi)
                ctx.set("this.optim",    optim)
                ctx.set("this.platform", platform)
                ctx.set("this.script",   script)
                ctx.set("this.stl",      stl)

                //message("ndk-build: config: name=%v, dir=%v, args=%v, vars=%v", ctx.module.name, ctx.module.dir, args, vars)
                return script != ""
        }
        return false
}

func (ndk *_ndkbuild) createActions(ctx *context, args []string) bool {
        message("ndk-build: createActions: %v", ctx.module.name)

        // ndk-build NDK_PROJECT_PATH=. \
        // APP_BUILD_SCRIPT=Android.mk
        // APP_PLATFORM=android-9
        // APP_STL=stlport_static|gnustl_static
        // APP_OPTIM=release|debug
        cmd := &_ndkbuildCmd{
                script:   filepath.Join(ctx.module.dir, ctx.call("this.script")),
                abis:     strings.Fields(ctx.call("this.abi")),
                abi:      ctx.call("this.abi"),
                platform: ctx.call("this.platform"),
                stl:      ctx.call("this.stl"),
                optim:    ctx.call("this.optim"),
        }
        ctx.module.action = &action{ command:cmd }

        for _, s := range cmd.abis {
                dump := cmd.dump(s)
                //fmt.Printf("dump: %v\n", dump)

                //s = filepath.Join("obj", dump.appName, s, dump.module)
                s = strings.TrimPrefix(dump.builtModule, "./")
                ctx.module.action.targets = append(ctx.module.action.targets, s)

                ctx.module.action.prequisites = append(ctx.module.action.prequisites, newAction(cmd.script, nil))
                for _, s := range strings.Fields(dump.srcFiles) {
                        s = filepath.Join(ctx.module.dir, s)
                        ctx.module.action.prequisites = append(ctx.module.action.prequisites, newAction(s, nil))
                }
        }

        return true
}

func (ndk *_ndkbuild) useModule(ctx *context, m *module) bool {
        return false
}

type _ndkbuildDump struct {
        appName, module, moduleClass, srcFiles, builtModule,
        targetOut, targetObjs, targetGdbSetup, targetGdbServer string
}

type _ndkbuildCmd struct {
        script, abi, platform, stl, optim string
        abis []string
}

func (n *_ndkbuildCmd) execute(targets []string, prequisites []string) bool {
        message("%v -> %v", prequisites, targets)

        c := &excmd{ path:"ndk-build" }

        //$(call import-module,third_party/googletest)
        //$(call import-module,native_app_glue)

        vars := []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_OUT=%s", "obj"),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
                fmt.Sprintf("APP_BUILD_SCRIPT=%s", n.script),
                fmt.Sprintf("APP_ABI=%s", n.abi),
                fmt.Sprintf("APP_PLATFORM=%s", n.platform),
                fmt.Sprintf("APP_STL=%s", n.stl),
                fmt.Sprintf("APP_OPTIM=%s", n.optim),
        }

        return c.run(fmt.Sprintf("%v", targets), vars...)
}

func (n *_ndkbuildCmd) dump(abi string) (res *_ndkbuildDump) {
        res = &_ndkbuildDump{}

        tf := n.createDummyDumpFile(); defer os.Remove(tf)

        c := &excmd{ path:"ndk-build" }

        vars := []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                //fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_OUT=%s", "obj"),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
                fmt.Sprintf("APP_BUILD_SCRIPT=%s", tf),
                fmt.Sprintf("APP_ABI=%s", abi),
                fmt.Sprintf("APP_PLATFORM=%s", n.platform),
                fmt.Sprintf("APP_STL=%s", n.stl),
                fmt.Sprintf("APP_OPTIM=%s", n.optim),
                "dummy-dump",
        }

        if c.run("DummyDump", vars...) {
                //fmt.Printf( "%v", c.stdout.String() )
                ctx := &context{
                        l: lex{ s:c.stdout.Bytes() },
                        variables: make(map[string]*variable, 16),
                }
                if e := ctx.parse(); e != nil {
                        errorf(0, "DummyDump: %v", e)
                }
                res.appName = ctx.call("NDK_APP_NAME")
                res.module = ctx.call("LOCAL_MODULE")
                res.moduleClass = ctx.call("LOCAL_MODULE_CLASS")
                res.srcFiles = ctx.call("LOCAL_SRC_FILES")
                res.builtModule = ctx.call("LOCAL_BUILT_MODULE")
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
