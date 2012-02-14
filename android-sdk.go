package smart

import (
        //"bytes"
        "fmt"
        "os"
        "path/filepath"
        "runtime"
        "strings"
)

func init() {
        registerToolset("android-sdk", &_androidsdk{})
}

var androidsdk = "/home/duzy/open/android-sdk-linux_x86"
var androidPlatform = "android-10"

type _androidsdk struct {
}

func (sdk *_androidsdk) setupModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); panic(p.newError(0, "no module"))
        }

        d := filepath.Dir(p.file)
        sources, err := findFiles(filepath.Join(d, "src"), `\.java$`, -1)
        for i, _ := range sources { sources[i] = sources[i][len(d)+1:] }

        if err != nil {
                p.stepLineBack(); panic(p.newError(0, fmt.Sprintf("can't find Java sources in `%v'", d)))
        }

        v := p.setVariable("this.sources", strings.Join(sources, " "))
        v.loc = location{file:v.loc.file, lineno:p.lineno-1, colno:p.prevColno+1 }
        return true
}

func (sdk *_androidsdk) buildModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); panic(p.newError(0, "no module"))
        }

        m.action = newAction(m.name+".apk", nil)

        d := filepath.Dir(p.file)

        var a *action
        var hasRes, hasAssets, hasSrc bool
        if fi, err := os.Stat(filepath.Join(d, "src")); err == nil && fi.IsDir() { hasSrc = true }
        if fi, err := os.Stat(filepath.Join(d, "res")); err == nil && fi.IsDir() { hasRes = true }
        if fi, err := os.Stat(filepath.Join(d, "assets")); err == nil && fi.IsDir() { hasAssets = true }
        if hasRes || hasAssets {
                c := &androidsdkGenR{ d:d }
                if hasRes { c.res = filepath.Join(d, "res") }
                if hasAssets { c.assets = filepath.Join(d, "assets") }
                a = newAction("R.java", c)
        }

        if hasSrc {
                var ps []*action
                if a != nil { ps = append(ps, a) }
                if sources := p.getModuleSources(); 0 < len(sources) {
                        for _, src := range sources { ps = append(ps, newAction(src, nil)) }
                        c := &androidsdkGenClasses{ d:d, src:filepath.Join(d, "src"), }
                        a = newAction("*.class", c, ps...)
                }
        }

        if a != nil {
                m.action.prequisites = append(m.action.prequisites, a)
        }

        switch m.kind {
        case "apk":
                c := &androidsdkGenApk{ d:d }
                if hasRes { c.res = filepath.Join(d, "res") }
                if hasAssets { c.assets = filepath.Join(d, "assets") }
                m.action.command = c
        case "jar":
                c := &androidsdkGenJar{ d:d }
                if hasRes { c.res = filepath.Join(d, "res") }
                if hasAssets { c.assets = filepath.Join(d, "assets") }
                m.action.command = c
        }
        return true
}

var androidsdkSlientSome = true

type androidsdkGenR struct{
        d, res, assets string
        r string // "r" holds the R.java file path
}
func (ic *androidsdkGenR) target() string { return ic.r }
func (ic *androidsdkGenR) needsUpdate() bool { return ic.r == "" }
func (ic *androidsdkGenR) execute(target string, prequisites []string) bool {
        ic.r = ""

        args := []string{
                "package", "-m",
                "-J", "out/res",
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar"),
        }

        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        // TODO: -P -G

        c := &execCommand{ name:"aapt", mkdir:"out/res", slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "aapt"), }
        if !c.run("resources", args...) {
                return false
        }

        if ic.r = findFile("out/res", `R\.java$`); ic.r != "" {
                return true
        }

        return false
}

type androidsdkGenClasses struct{
        d, src string
        classes []string // holds the *.class file
}
func (ic *androidsdkGenClasses) target() string {
        return strings.Join(ic.classes, " ")
}
func (ic *androidsdkGenClasses) needsUpdate() bool { return len(ic.classes) == 0 }
func (ic *androidsdkGenClasses) execute(target string, prequisites []string) bool {
        args := []string {
                "-d", "out/classes",
                "-sourcepath", ic.src,
                "-cp", filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar"),
        }

        args = append(args, prequisites...)
        c := &execCommand{ name:"javac", mkdir:"out/classes", }
        if !c.run("classes", args...) {
                return false
        }

        var e error
        ic.classes, e = findFiles("out/classes", `\.class$`, -1)
        if e != nil {
                fmt.Printf("error: %v\n", e)
                return false
        }

        return 0 < len(ic.classes)
}

func androidsdkCreateEmptyPackage(name string) bool {
        if f, e := os.Create(filepath.Join(filepath.Dir(name), "dummy")); e == nil {
                f.Close()
        } else {
                return false
        }

        c := &execCommand{ name:"jar", dir:filepath.Dir(name), slient:androidsdkSlientSome, }
        if !c.run("EmptyPackage", "cf", filepath.Base(name), "dummy") {
                return false
        }

        if e := os.Remove(filepath.Join(filepath.Dir(name), "dummy")); e != nil {
                fmt.Printf("error: %v\n", e)
        }

        c = &execCommand{ name:"zip", dir:filepath.Dir(name), slient:androidsdkSlientSome, }
        if !c.run("EmptyPackage", "-qd", filepath.Base(name), "dummy") {
                return false
        }

        return true
}

type androidsdkGenApk struct {
        d, res, assets, apk string
}
func (ic *androidsdkGenApk) target() string { return ic.apk }
func (ic *androidsdkGenApk) needsUpdate() bool { return ic.apk == "" }
func (ic *androidsdkGenApk) execute(target string, prequisites []string) bool {
        outclasses := "out/classes"

        args := []string {}
        if runtime.GOOS != "windows" { args = append(args, "-JXms16M", "-JXmx1536M") }
        args = append(args, "--dex", "--output=classes.dex")
        for _, s := range prequisites { args = append(args, s[len(outclasses)+1:]) }
        c := &execCommand{ name:"dx", dir:outclasses, slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "dx"), }
        if !c.run("classes.dex", args...) {
                //fmt.Printf("error: %v\n", e)
                return false
        }

        if e := os.Rename("out/classes/classes.dex", "out/classes.dex"); e != nil {
                fmt.Printf("error: %v\n", e)
                return false
        }

        if !androidsdkCreateEmptyPackage("out/unsigned.apk") {
                return false
        }

        c = &execCommand{ name:"aapt", slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "aapt"), }

        args = []string{ "package", "-u",
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar"),
        }
        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        if !c.run("package resources", args...) {
                //fmt.Printf("error: %v\n", e)
                return false
        }

        args = []string{ "add", "-k", "out/unsigned.apk", "out/classes.dex" }
        if !c.run("package dex file", args...) {
                //fmt.Printf("error: %v\n", e)
                return false
        }

        fmt.Printf("TODO: package JNI files\n")

        if e := copyFile("out/unsigned.apk", "out/signed.apk"); e != nil {
                return false
        }

        args = []string{}

        d := filepath.Dir(os.Args[0])
        keystore := filepath.Join(d, "data", "androidsdk", "keystore")
        //keypass := filepath.Join(d, "data", "androidsdk", "keypass")
        //storepass := filepath.Join(d, "data", "androidsdk", "storepass")
        //if fi, e := os.Stat(keystore); e == nil && !fi.IsDir() { args = append(args, "-keystore", keystore) }
        //if fi, e := os.Stat(keypass); e == nil && !fi.IsDir() { args = append(args, "-keypass", keypass) }
        //if fi, e := os.Stat(storepass); e == nil && !fi.IsDir() { args = append(args, "-storepass", storepass) }
        if fi, e := os.Stat(keystore); e == nil && !fi.IsDir() {
                args = append(args, "-keystore", keystore, "-keypass", "smart.android", "-storepass", "smart.android")
        }
        args = append(args, "out/signed.apk", "cert")

        fmt.Printf("smart: android signing `%v'...\n", target)
        c = &execCommand{ name:"jarsigner", slient:true/*androidsdkSlientSome*/, }
        if !c.run("sign package", args...) {
                //fmt.Printf("error: %v\n", e)
                os.Remove("out/signed.apk")
                return false
        }

        ic.apk = fmt.Sprintf("out/%v", target)
        if e := os.Rename("out/signed.apk", ic.apk); e != nil {
                fmt.Printf("error: %v\n", e)
                ic.apk = ""
                return false
        }

        return true
}

type androidsdkGenJar struct {
        d, res, assets, jar string
}
func (ic *androidsdkGenJar) target() string { return ic.jar }
func (ic *androidsdkGenJar) needsUpdate() bool { return ic.jar == "" }
func (ic *androidsdkGenJar) execute(target string, prequisites []string) bool {
        libname := "out/library.jar"
        if !androidsdkCreateEmptyPackage(libname) {
                os.Remove(libname)
                return false
        }

        c := &execCommand{ name:"aapt", slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "aapt"), }

        args := []string{ "package", "-u",
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar"),
        }
        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        if !c.run("package resources", args...) {
                //fmt.Printf("error: %v\n", e)
                return false
        }

        manifest := ""
        args = []string{}
        if manifest != "" {
                args = append(args, "-ufm")
        } else {
                args = append(args, "-uf")
        }
        args = append(args, libname, "-C", "out/classes", ".")
        c = &execCommand{ name:"jar", slient:androidsdkSlientSome, }
        if !c.run("PackageClasses", args...) {
                return false
        }

        fmt.Printf("TODO: %v, %v\n", target, prequisites)
        return false
}
