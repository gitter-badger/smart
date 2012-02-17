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
                        fmt.Printf("can't locate Android SDK: %v\n", e)
                }
        }
}

type _androidsdk struct {
}

func (sdk *_androidsdk) setupModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); errorf(0, "no module")
        }

        d := filepath.Dir(p.file)
        sources, err := findFiles(filepath.Join(d, "src"), `\.java$`, -1)
        for i, _ := range sources { sources[i] = sources[i][len(d)+1:] }

        if err != nil {
                p.stepLineBack(); errorf(0, fmt.Sprintf("can't find Java sources in `%v'", d))
        }

        v := p.setVariable("this.sources", strings.Join(sources, " "))
        v.loc = location{file:v.loc.file, lineno:p.lineno-1, colno:p.prevColno+1 }
        return true
}

func (sdk *_androidsdk) getResources(ds ...string) (as []*action) {
        for _, d := range ds {
                traverse(d, func(fn string, fi os.FileInfo) bool {
                        if !strings.HasSuffix(fn, "~") && !fi.IsDir() {
                                as = append(as, newAction(fn, nil))
                        }
                        return true
                })
        }
        return
}

func (sdk *_androidsdk) buildModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); errorf(0, "no module")
        }

        gen := &androidsdkGen{ out:filepath.Join("out", m.name), d:filepath.Dir(p.file) }

        var prequisites []*action
        var a *action
        var hasRes, hasAssets, hasSrc bool
        if fi, err := os.Stat(filepath.Join(gen.d, "src")); err == nil && fi.IsDir() { hasSrc = true }
        if fi, err := os.Stat(filepath.Join(gen.d, "res")); err == nil && fi.IsDir() { hasRes = true }
        if fi, err := os.Stat(filepath.Join(gen.d, "assets")); err == nil && fi.IsDir() { hasAssets = true }
        if hasRes { gen.res = filepath.Join(gen.d, "res") }
        if hasAssets { gen.assets = filepath.Join(gen.d, "assets") }
        if hasRes || hasAssets {
                c := &androidsdkGenR{ androidsdkGen:gen }
                a = newInAction("R.java", c, sdk.getResources(gen.res, gen.assets)...)
        }

        if hasSrc {
                var ps []*action
                if a != nil { ps = append(ps, a) }
                if sources := p.getModuleSources(); 0 < len(sources) {
                        var classpath []string
                        for _, u := range m.using {
                                if u.kind != "jar" {
                                        p.stepLineBack()
                                        errorf(0, "can't use module of type `%v'", u.kind)
                                }
                                if v, ok := u.variables["this.export.jar"]; ok {
                                        //fmt.Printf("use: `%v' by `%v', %v\n", u.name, m.name, v.value)
                                        classpath = append(classpath, v.value)
                                } else {
                                        //fmt.Printf("use: `%v' by `%v', %v\n", u.name, m.name)
                                }
                        }

                        for _, src := range sources { ps = append(ps, newAction(src, nil)) }
                        c := &androidsdkGenClasses{ androidsdkGen:gen, sourcepath:filepath.Join(gen.d, "src"), classpath:classpath, }
                        a = newInAction("*.class", c, ps...)
                }
        }

        if a != nil {
                prequisites = append(prequisites, a)
        }

        switch m.kind {
        case "apk":
                c := &androidsdkGenApk{androidsdkGenTar{ androidsdkGen:gen, target:filepath.Join(gen.out, m.name+".apk") }}
                m.action = newInAction(m.name+".apk", c, prequisites...)
        case "jar":
                c := &androidsdkGenJar{androidsdkGenTar{ androidsdkGen:gen, target:filepath.Join(gen.out, m.name+".jar") }}
                p.setVariable("this.export.jar", c.target)
                m.action = newInAction(m.name+".jar", c, prequisites...)
        default:
                p.stepLineBack(); errorf(0, "unknown module type `%v'", m.kind)
        }
        return true
}

type androidsdkGen struct{
        out, d, res, assets string
}
type androidsdkGenR struct{
        *androidsdkGen
        r string // "r" holds the R.java file path
        outdates int
}
func (ic *androidsdkGenR) targets(prequisites []*action) (targets []string, needsUpdate bool) {
        if ic.r != "" {
                targets = []string{ ic.r }
                needsUpdate = 0 < ic.outdates
                return
        }

        targets, outdates, _ := computeInterTargets(filepath.Join(ic.out, "res"), `R\.java$`, prequisites)
        if 0 < len(targets) { ic.r = targets[0] }

        needsUpdate = ic.r == "" || 0 < outdates
        return
}
func (ic *androidsdkGenR) execute(targets []string, prequisites []string) bool {
        ic.r = ""

        outRes := filepath.Join(ic.out, "res")
        os.RemoveAll(outRes)

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
        mkdir: outRes,
        path: filepath.Join(androidsdk, "platform-tools", "aapt"),
        }
        if *flag_v || *flag_V {
                if ic.res != "" { fmt.Printf("smart: resources `%v'...\n", ic.res) }
                if ic.assets != "" { fmt.Printf("smart: assets `%v'...\n", ic.assets) }
        }
        if !c.run("resources", args...) {
                errorf(0, "resources: %v", outRes)
        }

        if ic.r = findFile(outRes, `R\.java$`); ic.r != "" {
                return true
        } else {
                errorf(0, "resources: R.java not found")
        }

        return false
}

type androidsdkGenClasses struct{
        *androidsdkGen
        sourcepath string
        classpath []string // holds the *.class file
        outdates int
}
func (ic *androidsdkGenClasses) targets(prequisites []*action) (targets []string, needsUpdate bool) {
        targets, outdates, _ := computeInterTargets(filepath.Join(ic.out, "classes"), `\.class$`, prequisites)
        needsUpdate = len(targets) == 0 || 0 < outdates
        return
}
func (ic *androidsdkGenClasses) execute(targets []string, prequisites []string) bool {
        classpath := filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar")
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

        args = append(args, prequisites...)
        c := &execCommand{ name:"javac", mkdir:outClasses, }
        if !c.run("classes", args...) {
                errorf(0, "classes: %v", outClasses)
                return false
        }

        /*
        if classes, e := findFiles(outClasses, `\.class$`, -1); e == nil {
        } else {
                errorf(0, "classes: %v", e)
        }
        return 0 < len(ic.classes)
        */

        return true
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
                errorf(0, "remove: %v (%v)\n", "dummy", e)
        }

        c = &execCommand{ name:"zip", dir:filepath.Dir(name), slient:androidsdkSlientSome, }
        if !c.run("EmptyPackage", "-qd", filepath.Base(name), "dummy") {
                return false
        }

        return true
}

type androidsdkGenTar struct {
        *androidsdkGen
        target string
}
type androidsdkGenApk struct {
        androidsdkGenTar
}
type androidsdkGenJar struct {
        androidsdkGenTar
}

func (ic *androidsdkGenTar) targets(prequisites []*action) (targets []string, needsUpdate bool) {
        if ic.target == "" {
                errorf(0, "unknown APK name")
        }
        afi, _ := os.Stat(ic.target)
        newerCount := 0
        traverse(filepath.Join(ic.out, "classes"), func(fn string, fi os.FileInfo) bool {
                if afi == nil { return false }
                if strings.HasSuffix(fn, ".class") && !fi.IsDir() {
                        if afi.ModTime().Before(fi.ModTime()) {
                                newerCount += 1
                        }
                }
                return true
        })
        targets = []string{ ic.target }
        needsUpdate = afi == nil || 0 < newerCount
        return
}

func (ic *androidsdkGenApk) execute(targets []string, prequisites []string) bool {
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
                errorf(0, "no classes for `%v'", targets)
        }

        c := &execCommand{ name:"dx", dir:outclasses, slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "dx"), }
        if !c.run("classes.dex", args...) {
                errorf(0, "dex: %v\n", "classes.dex")
        }

        if e := os.Rename(filepath.Join(ic.out, "classes/classes.dex"), filepath.Join(ic.out, "classes.dex")); e != nil {
                errorf(0, "rename: %v (%v)\n", "classes.dex", e)
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
        if *flag_v || *flag_V {
                fmt.Printf("smart: resources in %v...\n", targets)
        }
        if !c.run("package resources", args...) {
                errorf(0, "pack classes: %v", targets)
        }

        args = []string{ "add", "-k", filepath.Join(ic.out, "unsigned.apk"), filepath.Join(ic.out, "classes.dex") }
        if *flag_v || *flag_V {
                fmt.Printf("smart: classes in %v...\n", targets)
        }
        if !c.run("package dex file", args...) {
                errorf(0, "pack classes: %v", targets)
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

        if *flag_v || *flag_V {
                fmt.Printf("smart: signing %v...\n", targets)
        }

        c = &execCommand{ name:"jarsigner", slient:true/*androidsdkSlientSome*/, }
        if !c.run("sign package", args...) {
                os.Remove(filepath.Join(ic.out, "signed.apk"))
                return false
        }

        if e := os.Rename(filepath.Join(ic.out, "signed.apk"), ic.target); e != nil {
                errorf(0, "rename: %v", ic.target)
        }

        return true
}

func (ic *androidsdkGenJar) execute(targets []string, prequisites []string) bool {
        libname := filepath.Join(ic.out, "library.jar")
        if !androidsdkCreateEmptyPackage(libname) {
                os.Remove(libname)
                errorf(0, "pack: %v", libname)
        }

        c := &execCommand{ name:"aapt", slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "aapt"), }

        args := []string{ "package", "-u",
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", androidPlatform, "android.jar"),
        }
        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        if !c.run("package resources", args...) {
                errorf(0, "pack resources: %v", libname)
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
                errorf(0, "pack classes: %v", libname)
        }

        if e := os.Rename(libname, ic.target); e != nil {
                errorf(0, "rename: %v", ic.target)
        }

        return true
}
