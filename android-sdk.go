package smart

import (
        //"bytes"
        "fmt"
        "os"
        "path/filepath"
        "runtime"
        "strings"

        "os/exec"
)

var androidsdk = "/android-sdk-linux_x86"
var androidPlatform = "android-10"
var androidsdkSlientSome = false

func init() {
        registerToolset("android-sdk", &_androidsdk{})

        /****/ if c, e := exec.LookPath("android"); e == nil {
                androidsdk = filepath.Dir(filepath.Dir(c))
        } else if c, e := exec.LookPath("aapt"); e == nil {
                androidsdk = filepath.Dir(filepath.Dir(c))
        } else {
                if androidsdk = os.Getenv("ANDROIDSDK"); androidsdk == "" {
                        fmt.Printf("error: %v\n", e)
                }
        }
}

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

        switch m.kind {
        case "apk": m.action = newAction(m.name+".apk", nil)
        case "jar": m.action = newAction(m.name+".jar", nil)
        default: p.stepLineBack(); panic(p.newError(0, "unknown module type `%v'", m.kind))
        }

        d := filepath.Dir(p.file)
        out := filepath.Join("out", m.name)

        var a *action
        var hasRes, hasAssets, hasSrc bool
        if fi, err := os.Stat(filepath.Join(d, "src")); err == nil && fi.IsDir() { hasSrc = true }
        if fi, err := os.Stat(filepath.Join(d, "res")); err == nil && fi.IsDir() { hasRes = true }
        if fi, err := os.Stat(filepath.Join(d, "assets")); err == nil && fi.IsDir() { hasAssets = true }
        if hasRes || hasAssets {
                c := &androidsdkGenR{ out:out, d:d }
                if hasRes { c.res = filepath.Join(d, "res") }
                if hasAssets { c.assets = filepath.Join(d, "assets") }
                a = newAction("R.java", c)
        }

        if hasSrc {
                var ps []*action
                if a != nil { ps = append(ps, a) }
                if sources := p.getModuleSources(); 0 < len(sources) {
                        var classpath []string
                        for _, u := range m.using {
                                if v, ok := u.variables["this.export.jar"]; ok {
                                        //fmt.Printf("use: `%v' by `%v', %v\n", u.name, m.name, v.value)
                                        classpath = append(classpath, v.value)
                                } else {
                                        //fmt.Printf("use: `%v' by `%v', %v\n", u.name, m.name)
                                }
                        }

                        for _, src := range sources { ps = append(ps, newAction(src, nil)) }
                        c := &androidsdkGenClasses{ out:out, d:d, sourcepath:filepath.Join(d, "src"), classpath:classpath, }
                        a = newAction("*.class", c, ps...)
                }
        }

        if a != nil {
                m.action.prequisites = append(m.action.prequisites, a)
        }

        switch m.kind {
        case "apk":
                c := &androidsdkGenApk{ out:out, d:d }
                if hasRes { c.res = filepath.Join(d, "res") }
                if hasAssets { c.assets = filepath.Join(d, "assets") }
                m.action.command = c
        case "jar":
                c := &androidsdkGenJar{ out:out, d:d }
                if hasRes { c.res = filepath.Join(d, "res") }
                if hasAssets { c.assets = filepath.Join(d, "assets") }
                p.setVariable("this.export.jar", filepath.Join(out, "library.jar"))
                m.action.command = c
        }
        return true
}

type androidsdkGenR struct{
        out, d, res, assets string
        r string // "r" holds the R.java file path
}
func (ic *androidsdkGenR) target() string { return ic.r }
func (ic *androidsdkGenR) needsUpdate() bool { return ic.r == "" }
func (ic *androidsdkGenR) execute(target string, prequisites []string) bool {
        ic.r = ""

        args := []string{
                "package", "-m",
                "-J", filepath.Join(ic.out, "res"),
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar"),
        }

        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        // TODO: -P -G

        c := &execCommand{
        name: "aapt", slient: androidsdkSlientSome,
        mkdir: filepath.Join(ic.out, "res"),
        path: filepath.Join(androidsdk, "platform-tools", "aapt"),
        }
        if !c.run("resources", args...) {
                return false
        }

        if ic.r = findFile(filepath.Join(ic.out, "res"), `R\.java$`); ic.r != "" {
                return true
        }

        return false
}

type androidsdkGenClasses struct{
        out, d, sourcepath string
        classpath, classes []string // holds the *.class file
}
func (ic *androidsdkGenClasses) target() string { return strings.Join(ic.classes, " ") }
func (ic *androidsdkGenClasses) needsUpdate() bool { return len(ic.classes) == 0 }
func (ic *androidsdkGenClasses) execute(target string, prequisites []string) bool {
        classpath := filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar")
        if 0 < len(ic.classpath) {
                classpath += ":" + strings.Join(ic.classpath, ":")
        }

        args := []string {
                "-d", filepath.Join(ic.out, "classes"),
                "-sourcepath", ic.sourcepath,
                "-cp", classpath,
        }

        args = append(args, prequisites...)
        c := &execCommand{ name:"javac", mkdir:filepath.Join(ic.out, "classes"), }
        if !c.run("classes", args...) {
                return false
        }

        var e error
        ic.classes, e = findFiles(filepath.Join(ic.out, "classes"), `\.class$`, -1)
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
        out, d, res, assets, apk string
}
func (ic *androidsdkGenApk) target() string { return ic.apk }
func (ic *androidsdkGenApk) needsUpdate() bool { return ic.apk == "" }
func (ic *androidsdkGenApk) execute(target string, prequisites []string) bool {
        outclasses := filepath.Join(ic.out, "classes")

        args := []string {}
        if runtime.GOOS != "windows" { args = append(args, "-JXms16M", "-JXmx1536M") }
        args = append(args, "--dex", "--output=classes.dex")

        countClasses := 0
        for _, s := range prequisites {
                if s == "" { continue }
                if strings.HasPrefix(s, outclasses) {
                        args = append(args, s[len(outclasses)+1:])
                } else {
                        args = append(args, s)
                }
                countClasses += 1
        }
        if countClasses == 0 {
                fmt.Printf("error: no classes: %v\n", countClasses)
                return false
        }

        c := &execCommand{ name:"dx", dir:outclasses, slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "dx"), }
        if !c.run("classes.dex", args...) {
                //fmt.Printf("error: %v\n", e)
                return false
        }

        if e := os.Rename(filepath.Join(ic.out, "classes/classes.dex"), filepath.Join(ic.out, "classes.dex")); e != nil {
                fmt.Printf("error: %v\n", e)
                return false
        }

        if !androidsdkCreateEmptyPackage(filepath.Join(ic.out, "unsigned.apk")) {
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

        args = []string{ "add", "-k", filepath.Join(ic.out, "unsigned.apk"), filepath.Join(ic.out, "classes.dex") }
        if !c.run("package dex file", args...) {
                //fmt.Printf("error: %v\n", e)
                return false
        }

        fmt.Printf("TODO: package JNI files\n")

        if e := copyFile(filepath.Join(ic.out, "unsigned.apk"), filepath.Join(ic.out, "signed.apk")); e != nil {
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
        args = append(args, filepath.Join(ic.out, "signed.apk"), "cert")

        fmt.Printf("smart: android signing `%v'...\n", target)
        c = &execCommand{ name:"jarsigner", slient:true/*androidsdkSlientSome*/, }
        if !c.run("sign package", args...) {
                //fmt.Printf("error: %v\n", e)
                os.Remove(filepath.Join(ic.out, "signed.apk"))
                return false
        }

        ic.apk = filepath.Join(ic.out, target)
        if e := os.Rename(filepath.Join(ic.out, "signed.apk"), ic.apk); e != nil {
                fmt.Printf("error: %v\n", e)
                ic.apk = ""
                return false
        }

        return true
}

type androidsdkGenJar struct {
        out, d, res, assets, jar string
}
func (ic *androidsdkGenJar) target() string { return ic.jar }
func (ic *androidsdkGenJar) needsUpdate() bool { return ic.jar == "" }
func (ic *androidsdkGenJar) execute(target string, prequisites []string) bool {
        libname := filepath.Join(ic.out, "library.jar")
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
        args = append(args, libname, "-C", filepath.Join(ic.out, "classes"), ".")
        c = &execCommand{ name:"jar", slient:androidsdkSlientSome, }
        if !c.run("PackageClasses", args...) {
                return false
        }

        fmt.Printf("TODO: %v, %v\n", target, prequisites)
        return false
}
