//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"bytes"
        "archive/zip"
        "fmt"
        "os"
        "path/filepath"
        "runtime"
        "strings"
        "io/ioutil"
        "io"
        "os/exec"
        . "github.com/duzy/smart/build"
)

var androidsdk = "/android-sdk-linux_x86"
var androidsdkBuildToolVersion = "23.0.2"
var androidsdkDefaultPlatform = "android-23" // "android-10"
var androidsdkSourceCompatibility = "1.7" // gradle: sourceCompatibility = 1.7
var androidsdkTargetCompatibility = "1.7" // gradle: targetCompatibility = 1.7

func init() {
        RegisterToolset("android-sdk", &_androidsdk{})

        if c, e := exec.LookPath("android"); e == nil {
                androidsdk = filepath.Dir(filepath.Dir(c))
        } else if c, e := exec.LookPath("aapt"); e == nil {
                androidsdk = filepath.Dir(filepath.Dir(c))
        } else {
                if androidsdk = os.Getenv("ANDROIDSDK"); androidsdk == "" {
                        fmt.Printf("can't locate Android SDK: %v\n", e)
                }
        }
}

func androidsdkGetBuildTool(name string) string {
        version := androidsdkBuildToolVersion
        return filepath.Join(androidsdk, "build-tools", version, name)
}

func androidsdkGetPlatformTool(name string) string {
        return filepath.Join(androidsdk, "platform-tools", name)
}

func androidsdkGetTool(name string) string {
        return filepath.Join(androidsdk, "tools", name)
}

type _androidsdk struct {
}

func (sdk *_androidsdk) getResourceFiles(ds ...string) (as []*Action) {
        for _, d := range ds {
                Traverse(d, func(fn string, fi os.FileInfo) bool {
                        if !strings.HasSuffix(fn, "~") && !fi.IsDir() {
                                as = append(as, NewAction(fn, nil))
                        }
                        return true
                })
        }
        return
}

func (sdk *_androidsdk) ConfigModule(ctx *Context, m *Module, args []string, vars map[string]string) bool {
        d := filepath.Dir(ctx.CurrentScope())
        sources, err := FindFiles(filepath.Join(d, "src"), `\.java$`)
        for i := range sources {
                if strings.HasPrefix(sources[i], d) {
                        sources[i] = sources[i][len(d)+1:]
                }
        }

        //fmt.Printf("sources: (%v) %v\n", d, sources)

        if err != nil {
                Errorf(0, fmt.Sprintf("can't find Java sources in `%v'", d))
        }

        var platform string
        if s, ok := vars["PLATFORM"]; ok { platform = s } else { platform = androidsdkDefaultPlatform }

        /*
        var v *define
        loc := ctx.l.location()
        v = ctx.set("me.platform", platform);                  v.loc = *loc
        v = ctx.set("me.sources", strings.Join(sources, " ")); v.loc = *loc */
        ctx.Set("me.platform", platform)
        ctx.Set("me.sources", strings.Join(sources, " "))
        return true
}

func (sdk *_androidsdk) CreateActions(ctx *Context, m *Module, args []string) bool {
        platform := strings.TrimSpace(ctx.Call("me.platform"))
        if platform == "" { Errorf(0, "unkown platform for `%v'", m.Name) }

        //fmt.Printf("platform: %v\n", platform)

        gen := &androidGen{ platform:platform, out:filepath.Join("out", m.Name), d:filepath.Dir(ctx.CurrentScope()) }

        var a *Action
        var prequisites []*Action
        var staticLibs []string
        var hasRes, hasAssets, hasSrc bool
        if fi, err := os.Stat(filepath.Join(gen.d, "src")); err == nil && fi.IsDir() { hasSrc = true }
        if fi, err := os.Stat(filepath.Join(gen.d, "res")); err == nil && fi.IsDir() { hasRes = true }
        if fi, err := os.Stat(filepath.Join(gen.d, "assets")); err == nil && fi.IsDir() { hasAssets = true }
        if hasRes { gen.res = filepath.Join(gen.d, "res") }
        if hasAssets { gen.assets = filepath.Join(gen.d, "assets") }
        if hasRes || hasAssets {
                c := &androidGenR{ androidGen:gen }
                a = NewAction("R.java", c, sdk.getResourceFiles(gen.res, gen.assets)...)
        }
        if hasSrc {
                var ps []*Action
                if a != nil { ps = append(ps, a) }
                if sources := m.GetSources(ctx); 0 < len(sources) {
                        var classpath []string
                        for _, u := range m.Using {
                                if u.Kind != "jar" { Errorf(0, "can't use module of type `%v'", u.Kind) }
                                if v := strings.TrimSpace(ctx.Call("me.export.jar")); v != "" {
                                        classpath = append(classpath, v)
                                }
                                /*
                                if v := strings.TrimSpace(ctx.call("me.export.libs.static")); v != "" {
                                        classpath = append(classpath, v)
                                } */
                        }

                        staticLibs = append(staticLibs, strings.Split(strings.TrimSpace(ctx.Call("me.libs.static")), " ")...)
                        classpath = append(classpath, strings.Split(strings.TrimSpace(ctx.Call("me.classpath")), " ")...)
                        classpath = append(classpath, staticLibs...)

                        for _, src := range sources { ps = append(ps, NewAction(src, nil)) }
                        c := &androidGenClasses{
                                androidGen:gen,
                                sourcepath:filepath.Join(gen.d, "src"),
                                classpath:classpath,
                        }
                        a = NewAction("*.class", c, ps...)
                }
        }

        if a != nil {
                prequisites = append(prequisites, a)
        }

        switch m.Kind {
        case "apk":
                c := &androidGenApk{androidGenTar{ androidGen:gen, target:filepath.Join(gen.out, m.Name+".apk"), staticlibs:staticLibs, }}
                m.Action = NewAction(m.Name+".apk", c, prequisites...)
        case "jar":
                c := &androidGenJar{androidGenTar{ androidGen:gen, target:filepath.Join(gen.out, m.Name+".jar"), staticlibs:staticLibs, }}
                ctx.Set("me.export.jar", c.target)
                m.Action = NewAction(m.Name+".jar", c, prequisites...)
        default:
                Errorf(0, "unknown module type `%v'", m.Kind)
        }
        return true
}

func (sdk *_androidsdk) UseModule(ctx *Context, m, o *Module) bool {
        return false
}

type androidGen struct{
        platform, out, d, res, assets string
}
type androidGenR struct{
        *androidGen
        r string // "r" holds the R.java file path
        outdates int
}
func (ic *androidGenR) targets(prequisites []*Action) (targets []string, needsUpdate bool) {
        if ic.r != "" {
                targets = []string{ ic.r }
                needsUpdate = 0 < ic.outdates
                return
        }

        targets, outdates, _ := ComputeInterTargets(filepath.Join(ic.out, "res"), `R\.java$`, prequisites)
        if 0 < len(targets) { ic.r = targets[0] }

        needsUpdate = ic.r == "" || 0 < outdates
        return
}
func (ic *androidGenR) Execute(targets []string, prequisites []string) bool {
        ic.r = ""

        outRes := filepath.Join(ic.out, "res")
        os.RemoveAll(outRes)

        args := []string{
                "package", "-m",
                "-J", filepath.Join(ic.out, "res"),
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", ic.platform, "android.jar"),
        }

        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        // TODO: -P -G

        c := NewExcmd(androidsdkGetBuildTool("aapt"))
        c.SetIA32(IsIA32Command(c.GetPath()))
        c.SetMkdir(outRes)
        if GetFlagV() {
                if ic.res != "" { Verbose("resources `%v'", ic.res) }
                if ic.assets != "" { Verbose("assets `%v'", ic.assets) }
        }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if !c.Run("resources", args...) {
                Errorf(0, "resources: %v", outRes)
        }

        if ic.r = FindFile(outRes, `R\.java$`); ic.r != "" {
                return true
        }

        Errorf(0, "resources: R.java not found")
        return false
}

type androidGenClasses struct{
        *androidGen
        sourcepath string
        classpath []string // holds the *.class file
        outdates int
}
func (ic *androidGenClasses) targets(prequisites []*Action) (targets []string, needsUpdate bool) {
        targets, outdates, _ := ComputeInterTargets(filepath.Join(ic.out, "classes"), `\.class$`, prequisites)
        needsUpdate = len(targets) == 0 || 0 < outdates
        //fmt.Printf("classes: %v\n", targets);
        return
}
func (ic *androidGenClasses) Execute(targets []string, prequisites []string) bool {
        classpath := filepath.Join(androidsdk, "platforms", ic.platform, "android.jar")
        if 0 < len(ic.classpath) {
                classpath += ":" + strings.Join(ic.classpath, ":")
        }

        outClasses := filepath.Join(ic.out, "classes")
        os.RemoveAll(outClasses)

        args := []string {
                "-d", filepath.Join(ic.out, "classes"),
                "-sourcepath", ic.sourcepath,
                "-cp", classpath,
        }

        args = append(args, "-source", androidsdkSourceCompatibility)
        args = append(args, "-target", androidsdkTargetCompatibility)
        args = append(args, prequisites...)
        c := NewExcmd("javac")
        c.SetMkdir(outClasses)
        if !c.Run("classes", args...) {
                Errorf(0, "classes: %v", outClasses)
                return false
        }

        return true
}

// androidsdkCreateDummyPackage is deprecated
func androidsdkCreateDummyPackage(name string) bool {
        if f, e := os.Create(filepath.Join(filepath.Dir(name), "dummy")); e == nil {
                f.Close()
        } else {
                return false
        }

        c := NewExcmd("jar")
        c.SetDir(filepath.Dir(name))
        if !c.Run("DummyPackage", "cf", filepath.Base(name), "dummy") {
                return false
        }

        if e := os.Remove(filepath.Join(filepath.Dir(name), "dummy")); e != nil {
                Errorf(0, "remove: %v (%v)\n", "dummy", e)
        }

        c = NewExcmd("zip")
        c.SetDir(filepath.Dir(name))
        if !c.Run("DummyPackage", "-qd", filepath.Base(name), "dummy") {
                return false
        }

        return true
}

type androidGenTar struct {
        *androidGen
        target string
        staticlibs []string
}
type androidGenApk struct { androidGenTar }
type androidGenJar struct { androidGenTar }

func (ic *androidGenTar) targets(prequisites []*Action) (targets []string, needsUpdate bool) {
        if ic.target == "" {
                Errorf(0, "unknown APK name")
        }
        afi, _ := os.Stat(ic.target)
        newerCount := 0
        Traverse(filepath.Join(ic.out, "classes"), func(fn string, fi os.FileInfo) bool {
                if afi == nil { return false }
                if strings.HasSuffix(fn, ".class") && !fi.IsDir() {
                        if afi.ModTime().Before(fi.ModTime()) {
                                newerCount ++
                        }
                }
                return true
        })
        targets = []string{ ic.target }
        needsUpdate = afi == nil || 0 < newerCount
        return
}

func (ic *androidGenTar) createPackage(packFilename string, files ...string) {
        var (
                f *os.File
                err error
        )

        f, err = os.Create(packFilename)
        if err != nil {
                Errorf(0, "can't find keystore for sigining APK")
        }

        zw := zip.NewWriter(f)

        defer func() {
                if err := zw.Close(); err != nil {
                        Errorf(0, "%v (%s)", err, f.Name())
                }

                f.Close()
        }()

        libsDir := filepath.Join(ic.d, "libs")
        if fi, err := os.Stat(libsDir); err != nil || !fi.IsDir() {
                return
        }

        err = filepath.Walk(libsDir, func(path string, fi os.FileInfo, err error) error {
                if err != nil { return err }
                if fi.IsDir() { return nil } // path == libsDir

                h, err := zip.FileInfoHeader(fi)
                if err != nil { return err }

                h.Method = zip.Deflate
                h.Name = filepath.Join("lib", strings.TrimPrefix(
                        strings.TrimPrefix(path, libsDir), "/"))

                z, err := zw.CreateHeader(h)
                if err != nil { return err }
                if fi.IsDir() { return nil }

                Message("package: %v(%v)", f.Name(), h.Name)

                l, err := os.Open(path)
                if err != nil { return err }
                defer l.Close()

                _, err = io.Copy(z, l)
                return err
        })

        if err != nil {
                Errorf(0, "libs: %v", err)
        }
}

func (ic *androidGenTar) packageAddFiles(packFilename string, files ...string) {
        var (
                f *os.File
                err error
        )

        f, err = os.OpenFile(packFilename, os.O_RDWR, 0)
        if err != nil {
                Errorf(0, "can't find keystore for sigining APK")
        }

        zw := zip.NewWriter(f)

        defer func() {
                if err := zw.Close(); err != nil {
                        Errorf(0, "%v (%s)", err, f.Name())
                }

                f.Close()
        }()
        
        for _, s := range files {
                fi, err := os.Stat(s)
                if err != nil { Errorf(0, "%v", err) }

                h, err := zip.FileInfoHeader(fi)
                if err != nil { Errorf(0, "package: add: %v", err) }

                // h.Name ???
                h.Method = zip.Deflate

                Message("package: %v(%v)", f.Name(), h.Name)

                z, err := zw.CreateHeader(h)
                if err != nil { Errorf(0, "package: add: %v", err) }

                l, err := os.Open(s)
                if err != nil { Errorf(0, "package: add: %v", err) }
                defer l.Close()

                _, err = io.Copy(z, l)
                if err != nil { Errorf(0, "package: add: %v", err) }
        }
}

func androidsdkGetKeystore() (keystore, keypass, storepass string) {
        var canditates []string

        defaultPass := "smart.android"

        readpass := func(s, sn string) string {
                if f, e := os.Open(filepath.Join(filepath.Dir(s), sn)); e == nil {
                        defer f.Close()
                        if b, e := ioutil.ReadAll(f); e == nil {
                                return strings.TrimSpace(string(b))
                        }
                }
                return defaultPass
        }

        find := func() (s1, s2, s3 string) {
                for _, s := range canditates {
                        s1, s2, s3 = "", "", ""
                        if fi, e := os.Stat(s); e == nil && !fi.IsDir() {
                                s1, s2, s3 = s, readpass(s, "keypass"), readpass(s, "storepass")
                                return
                        }
                }
                return
        }

        if wd, e := os.Getwd(); e == nil {
                for {
                        //fmt.Printf("wd: %v\n", wd)
                        canditates = []string{
                                filepath.Join(wd, ".androidsdk", "keystore"),
                        }
                        if s1, s2, s3 := find(); s1 != "" { keystore, keypass, storepass = s1, s2, s3; return }
                        if s := filepath.Dir(wd); wd == s || s == "" { break } else { wd = s }
                }
        }

        if s, e := exec.LookPath("smart"); e == nil {
                canditates = []string{
                        filepath.Join(filepath.Dir(s), "data", "androidsdk", "keystore"),
                }
                if s1, s2, s3 := find(); s1 != "" { keystore, keypass, storepass = s1, s2, s3; return }
        }

        return "", "", ""
}

func androidsdkExtractClasses(outclasses, lib string, cs []string) (classes []string) {
        f, err := os.Open(lib)
        if err != nil { Errorf(0, "open: %v (%v)", lib, err) }
        defer f.Close()

        wd, err := os.Getwd()
        if err != nil {
                Errorf(0, "getwd: %v", err)
                return
        }

        if e := os.Chdir(outclasses); e != nil {
                Errorf(0, "chdir: %v", e)
                return
        }

        defer func() {
                if e := os.Chdir(wd); e != nil { Errorf(0, "chdir: %v", e) }
        }()

        c := NewExcmd("jar")
        c.SetStdin(f)
        args := append([]string{ "-x" }, cs...)
        if !c.Run("classes", args...) { Errorf(0, "static %v\n", lib) }

        for _, s := range cs {
                if fi, er := os.Stat(s); er != nil || fi == nil {
                        Errorf(0, "class `%v' not extracted (%v)", s, lib); return
                }
                classes = append(classes, s)
        }

        return
}

func androidsdkExtractStaticLibsClasses(outclasses string, libs []string) (classes []string) {
        for _, lib := range libs {
                if lib = strings.TrimSpace(lib); lib == "" { continue }

                c := NewExcmd("jar")
                args := []string{ "-tf", lib }
                if !c.Run("classes", args...) { Errorf(0, "static %v\n", lib) }

                var cs []string
                for _, s := range strings.Split(c.GetStdout().String(), "\n") {
                        if strings.HasSuffix(s, ".class") { cs = append(cs, s) }
                }

                //fmt.Printf("jar: %v: %v\n", lib, cs)

                classes = append(classes, androidsdkExtractClasses(outclasses, lib, cs)...)
        }
        //fmt.Printf("embeded-classes: %v\n", classes)
        return
}

func (ic *androidGenApk) Execute(targets []string, prequisites []string) bool {
        outclasses := filepath.Join(ic.out, "classes")

        // extract classes from static libraries (me.libs.static)

        embclasses := androidsdkExtractStaticLibsClasses(outclasses, ic.staticlibs)
        //fmt.Printf("staticlibs: %v\n", embclasses)

        // make classes.dex

        args := []string {}
        if runtime.GOOS != "windows" { args = append(args, "-JXms16M", "-JXmx1536M") }
        args = append(args, "--dex", "--output=../classes.dex")

        countClasses := 0
        for _, s := range prequisites {
                if s == "" { continue }
                if strings.HasPrefix(s, outclasses) {
                        args = append(args, s[len(outclasses)+1:])
                } else {
                        args = append(args, s)
                }
                countClasses ++
        }
        if countClasses == 0 { Errorf(0, "no classes for `%v'", targets) }

        args = append(args, embclasses...) // add classes from static libraries

        c := NewExcmd(androidsdkGetBuildTool("dx"))
        c.SetDir(outclasses)
        if !c.Run("classes", args...) { Errorf(0, "dex: %v\n", "classes.dex") }

        // Create new package unsigned.apk
        unsignedApk := filepath.Join(ic.out, "unsigned.apk")
        ic.createPackage(unsignedApk, filepath.Join(ic.out, "classes.dex")) // androidsdkCreateDummyPackage

        // package resources into unsigned.apk

        c = NewExcmd(androidsdkGetBuildTool("aapt"))
        c.SetIA32(IsIA32Command(c.GetPath()))

        args = []string{ "package", "-u", "-F", unsignedApk,
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", ic.platform, "android.jar"),
        }
        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if GetFlagV() { Verbose("resources -> %v", targets) }
        if !c.Run("resources", args...) {
                Errorf(0, "package: %v(%v)", unsignedApk, "assets,res,...")
        }

        //ic.packageAddFiles(unsignedApk, filepath.Join(ic.out, "classes.dex"))
        args = []string{ "add", "-k", unsignedApk, filepath.Join(ic.out, "classes.dex") }
        if !c.Run("classes", args...) {
                Errorf(0, "package: %v(%v)", unsignedApk, "classes")
        }

        keystore, keypass, storepass := androidsdkGetKeystore()
        if keystore == ""       { Errorf(0, "keystore is empty") }
        if keypass == ""        { Errorf(0, "keypass is empty") }
        if storepass == ""      { Errorf(0, "storepass is empty") }

        if GetFlagV() { Verbose("signing -> %v (%v)", targets, keystore) }
        if e := CopyFile(filepath.Join(ic.out, "unsigned.apk"), filepath.Join(ic.out, "signed.apk")); e != nil {
                return false
        }

        // signing unsigned.apk into signed.apk

        args = []string{
                "-keystore", keystore,
                "-keypass", keypass,
                "-storepass", storepass,
                filepath.Join(ic.out, "signed.apk"), "cert",
        }

        c = NewExcmd("jarsigner")
        if !c.Run("signing", args...) { os.Remove(filepath.Join(ic.out, "signed.apk")); return false }

        // zipalign signed.apk into aligned.apk then rename aligned.apk into final target

        if GetFlagV() { Verbose("aligning -> %v", targets) }

        c = NewExcmd(androidsdkGetBuildTool("zipalign"))
        c.SetIA32(IsIA32Command(c.GetPath()))

        args = []string{ "4", filepath.Join(ic.out, "signed.apk"), filepath.Join(ic.out, "aligned.apk"), }
        if !c.Run("aligning", args...) { os.Remove(filepath.Join(ic.out, "aligned.apk")); return false }

        if e := os.Rename(filepath.Join(ic.out, "aligned.apk"), ic.target); e != nil { Errorf(0, "rename: %v", ic.target) }
        return true
}

func (ic *androidGenJar) Execute(targets []string, prequisites []string) bool {
        // package native libs
        libname := filepath.Join(ic.out, "library.jar")
        ic.createPackage(libname) // androidsdkCreateDummyPackage

        c := NewExcmd(androidsdkGetBuildTool("aapt"))
        c.SetIA32(IsIA32Command(c.GetPath()))

        args := []string{ "package", "-u",
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", ic.platform, "android.jar"),
        }
        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if !c.Run("resources", args...) {
                Errorf(0, "package: %v(%v)", libname, "assets,res,...")
        }

        if GetFlagV() { Verbose("classes -> %v", targets) }

        manifest := ""
        args = []string{}
        if manifest != "" {
                args = append(args, "-ufm")
        } else {
                args = append(args, "-uf")
        }
        args = append(args, libname, "-C", filepath.Join(ic.out, "classes"), ".")
        c = NewExcmd("jar")
        if !c.Run("classes", args...) {
                Errorf(0, "package: %v(%v)", libname, "classes")
        }

        if e := os.Rename(libname, ic.target); e != nil {
                Errorf(0, "rename: %v", ic.target)
        }

        return true
}
