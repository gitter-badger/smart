package smart

import (
        //"bytes"
        "fmt"
        "os"
        "path/filepath"
        "runtime"
        "strings"
        "io/ioutil"
        "os/exec"
)

var androidsdk = "/android-sdk-linux_x86"
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

func (sdk *_androidsdk) setupModule(p *context, args []string, vars map[string]string) bool {
        var m *module
        if m = p.module; m == nil {
                errorf(0, "no module")
        }

        d := filepath.Dir(p.l.file)
        sources, err := findFiles(filepath.Join(d, "src"), `\.java$`)
        for i := range sources {
                if strings.HasPrefix(sources[i], d) {
                        sources[i] = sources[i][len(d)+1:]
                }
        }

        //fmt.Printf("sources: (%v) %v\n", d, sources)

        if err != nil {
                errorf(0, fmt.Sprintf("can't find Java sources in `%v'", d))
        }

        var platform string
        if s, ok := vars["PLATFORM"]; ok { platform = s } else { platform = "android-10" }

        var v *variable
        loc := p.l.location()
        v = p.setVariable("this.platform", platform); v.loc = *loc
        v = p.setVariable("this.sources", strings.Join(sources, " ")); v.loc = *loc
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

func (sdk *_androidsdk) buildModule(p *context, args []string) bool {
        var m *module
        if m = p.module; m == nil { errorf(0, "no module") }

        platform := strings.TrimSpace(p.call("this.platform"))
        if platform == "" { errorf(0, "unkown platform for `%v'", m.name) }

        //fmt.Printf("platform: %v\n", platform)

        gen := &androidsdkGen{ platform:platform, out:filepath.Join("out", m.name), d:filepath.Dir(p.l.file) }

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

        var staticLibs []string
        if hasSrc {
                var ps []*action
                if a != nil { ps = append(ps, a) }
                if sources := p.getModuleSources(); 0 < len(sources) {
                        var classpath []string
                        for _, u := range m.using {
                                if u.kind != "jar" { errorf(0, "can't use module of type `%v'", u.kind) }
                                if v, ok := u.variables["this.export.jar"]; ok {
                                        //fmt.Printf("use: `%v' by `%v', %v\n", u.name, m.name, v.value)
                                        classpath = append(classpath, strings.TrimSpace(v.value))
                                }
                                /*
                                if v, ok := u.variables["this.export.libs.static"]; ok {
                                        staticLibs = append(staticLibs, strings.TrimSpace(v.value))
                                }*/
                        }

                        staticLibs = append(staticLibs, strings.Split(strings.TrimSpace(p.call("this.libs.static")), " ")...)
                        classpath = append(classpath, strings.Split(strings.TrimSpace(p.call("this.classpath")), " ")...)
                        classpath = append(classpath, staticLibs...)

                        for _, src := range sources { ps = append(ps, newAction(src, nil)) }
                        c := &androidsdkGenClasses{
                                androidsdkGen:gen,
                                sourcepath:filepath.Join(gen.d, "src"),
                                classpath:classpath,
                        }
                        a = newInAction("*.class", c, ps...)
                }
        }

        if a != nil {
                prequisites = append(prequisites, a)
        }

        switch m.kind {
        case "apk":
                c := &androidsdkGenApk{androidsdkGenTar{ androidsdkGen:gen, target:filepath.Join(gen.out, m.name+".apk"), staticlibs:staticLibs, }}
                m.action = newInAction(m.name+".apk", c, prequisites...)
        case "jar":
                c := &androidsdkGenJar{androidsdkGenTar{ androidsdkGen:gen, target:filepath.Join(gen.out, m.name+".jar"), staticlibs:staticLibs, }}
                p.setVariable("this.export.jar", c.target)
                m.action = newInAction(m.name+".jar", c, prequisites...)
        default:
                errorf(0, "unknown module type `%v'", m.kind)
        }
        return true
}

func (sdk *_androidsdk) useModule(p *context, m *module) bool {
        return false
}

type androidsdkGen struct{
        platform, out, d, res, assets string
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
                "-I", filepath.Join(androidsdk, "platforms", ic.platform, "android.jar"),
        }

        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        // TODO: -P -G

        c := &excmd{
                name: "aapt", slient: androidsdkSlientSome, mkdir: outRes,
                path: filepath.Join(androidsdk, "platform-tools", "aapt"),
        }
        if *flagV || *flagVV {
                if ic.res != "" { fmt.Printf("smart: resources `%v'...\n", ic.res) }
                if ic.assets != "" { fmt.Printf("smart: assets `%v'...\n", ic.assets) }
        }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if !c.run("resources", args...) {
                errorf(0, "resources: %v", outRes)
        }

        if ic.r = findFile(outRes, `R\.java$`); ic.r != "" {
                return true
        }

        errorf(0, "resources: R.java not found")
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
        //fmt.Printf("classes: %v\n", targets);
        return
}
func (ic *androidsdkGenClasses) execute(targets []string, prequisites []string) bool {
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

        args = append(args, prequisites...)
        c := &excmd{ name:"javac", mkdir:outClasses, }
        if !c.run("classes", args...) {
                errorf(0, "classes: %v", outClasses)
                return false
        }

        return true
}

func androidsdkCreateEmptyPackage(name string) bool {
        if f, e := os.Create(filepath.Join(filepath.Dir(name), "dummy")); e == nil {
                f.Close()
        } else {
                return false
        }

        c := &excmd{ name:"jar", dir:filepath.Dir(name), slient:true/*androidsdkSlientSome*/, }
        if !c.run("EmptyPackage", "cf", filepath.Base(name), "dummy") {
                return false
        }

        if e := os.Remove(filepath.Join(filepath.Dir(name), "dummy")); e != nil {
                errorf(0, "remove: %v (%v)\n", "dummy", e)
        }

        c = &excmd{ name:"zip", dir:filepath.Dir(name), slient:true/*androidsdkSlientSome*/, }
        if !c.run("EmptyPackage", "-qd", filepath.Base(name), "dummy") {
                return false
        }

        return true
}

type androidsdkGenTar struct {
        *androidsdkGen
        target string
        staticlibs []string
}
type androidsdkGenApk struct { androidsdkGenTar }
type androidsdkGenJar struct { androidsdkGenTar }

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
                                newerCount ++
                        }
                }
                return true
        })
        targets = []string{ ic.target }
        needsUpdate = afi == nil || 0 < newerCount
        return
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
        if err != nil { errorf(0, "open: %v (%v)", lib, err) }
        defer f.Close()

        wd, err := os.Getwd()
        if err != nil {
                errorf(0, "getwd: %v", err)
                return
        }

        if e := os.Chdir(outclasses); e != nil {
                errorf(0, "chdir: %v", e)
                return
        }

        defer func() {
                if e := os.Chdir(wd); e != nil { errorf(0, "chdir: %v", e) }
        }()

        c := &excmd{ name:"jar", slient:true/*androidsdkSlientSome*/, stdin:f, }
        args := append([]string{ "-x" }, cs...)
        if !c.run("classes.dex", args...) { errorf(0, "static %v\n", lib) }

        for _, s := range cs {
                if fi, er := os.Stat(s); er != nil || fi == nil {
                        errorf(0, "class `%v' not extracted (%v)", s, lib); return
                }
                classes = append(classes, s)
        }

        return
}

func androidsdkExtractStaticLibsClasses(outclasses string, libs []string) (classes []string) {
        for _, lib := range libs {
                if lib = strings.TrimSpace(lib); lib == "" { continue }

                c := &excmd{ name:"jar", slient:true/*androidsdkSlientSome*/, }
                args := []string{ "-tf", lib }
                if !c.run("classes.dex", args...) { errorf(0, "static %v\n", lib) }

                var cs []string
                for _, s := range strings.Split(c.stdout.String(), "\n") {
                        if strings.HasSuffix(s, ".class") { cs = append(cs, s) }
                }

                //fmt.Printf("jar: %v: %v\n", lib, cs)

                classes = append(classes, androidsdkExtractClasses(outclasses, lib, cs)...)
        }
        //fmt.Printf("embeded-classes: %v\n", classes)
        return
}

func (ic *androidsdkGenApk) execute(targets []string, prequisites []string) bool {
        outclasses := filepath.Join(ic.out, "classes")

        // extract classes from static libraries (this.libs.static)

        embclasses := androidsdkExtractStaticLibsClasses(outclasses, ic.staticlibs)
        //fmt.Printf("staticlibs: %v\n", embclasses)

        // make classes.dex

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
                countClasses ++
        }
        if countClasses == 0 { errorf(0, "no classes for `%v'", targets) }

        args = append(args, embclasses...) // add classes from static libraries

        //fmt.Printf("dex: %v\n", prequisites);
        //fmt.Printf("dex: %v\n", embclasses);

        if *flagV || *flagVV { fmt.Printf("smart: preparing classes.dex for %v...\n", targets) }

        c := &excmd{ name:"dx", dir:outclasses, slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "dx"), }
        if !c.run("classes.dex", args...) { errorf(0, "dex: %v\n", "classes.dex") }

        if e := os.Rename(filepath.Join(ic.out, "classes/classes.dex"), filepath.Join(ic.out, "classes.dex")); e != nil { errorf(0, "rename: %v (%v)\n", "classes.dex", e) }

        // generate empty unsigned.apk
        if !androidsdkCreateEmptyPackage(filepath.Join(ic.out, "unsigned.apk")) { return false }

        // package resources into unsigned.apk

        c = &excmd{
                name:"aapt", slient:androidsdkSlientSome,
                path:filepath.Join(androidsdk, "platform-tools", "aapt"),
        }

        args = []string{ "package", "-u",
                "-F", filepath.Join(ic.out, "unsigned.apk"),
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", ic.platform, "android.jar"),
        }
        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if *flagV || *flagVV { fmt.Printf("smart: pack resources for %v...\n", targets) }
        if !c.run("package resources", args...) { errorf(0, "pack classes: %v", targets) }

        // add classes.dex into unsigned.apk

        args = []string{ "add", "-k", filepath.Join(ic.out, "unsigned.apk"), filepath.Join(ic.out, "classes.dex") }
        if *flagV || *flagVV { fmt.Printf("smart: pack classes for %v...\n", targets) }
        if !c.run("package dex file", args...) { errorf(0, "pack classes: %v", targets) }

        fmt.Printf("TODO: package JNI files\n")

        keystore, keypass, storepass := androidsdkGetKeystore()
        if keystore == "" || keypass == "" || storepass == "" {
                errorf(0, "can't find keystore for sigining APK")
                //fmt.Printf("smart: no keystore for sigining %v\n", targets)
                //return true
        }

        if *flagV || *flagVV { fmt.Printf("smart: signing %v (%v)...\n", targets, keystore) }
        if e := copyFile(filepath.Join(ic.out, "unsigned.apk"), filepath.Join(ic.out, "signed.apk")); e != nil {
                return false
        }

        // signing unsigned.apk into signed.apk

        args = []string{
                "-keystore", keystore,
                "-keypass", keypass,
                "-storepass", storepass,
                filepath.Join(ic.out, "signed.apk"), "cert",
        }

        c = &excmd{ name:"jarsigner", slient:true/*androidsdkSlientSome*/, }
        if !c.run("sign package", args...) { os.Remove(filepath.Join(ic.out, "signed.apk")); return false }

        // zipalign signed.apk into aligned.apk then rename aligned.apk into final target

        if *flagV || *flagVV { fmt.Printf("smart: aligning %v...\n", targets) }
        c = &excmd{ name:"zipalign", ia32:true, slient:true/*androidsdkSlientSome*/,
                path:filepath.Join(androidsdk, "tools", "zipalign"),
        }
        args = []string{ "4", filepath.Join(ic.out, "signed.apk"), filepath.Join(ic.out, "aligned.apk"), }
        if !c.run("align package", args...) { os.Remove(filepath.Join(ic.out, "aligned.apk")); return false }

        if e := os.Rename(filepath.Join(ic.out, "aligned.apk"), ic.target); e != nil { errorf(0, "rename: %v", ic.target) }
        return true
}

func (ic *androidsdkGenJar) execute(targets []string, prequisites []string) bool {
        libname := filepath.Join(ic.out, "library.jar")
        if !androidsdkCreateEmptyPackage(libname) {
                os.Remove(libname)
                errorf(0, "pack: %v", libname)
        }

        c := &excmd{ name:"aapt", slient:androidsdkSlientSome, path: filepath.Join(androidsdk, "platform-tools", "aapt"), }

        args := []string{ "package", "-u",
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(androidsdk, "platforms", ic.platform, "android.jar"),
        }
        if ic.res != "" { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if !c.run("package resources", args...) {
                errorf(0, "pack resources: %v", libname)
        }

        if *flagV || *flagVV {
                fmt.Printf("smart: pack classes for %v...\n", targets)
        }

        manifest := ""
        args = []string{}
        if manifest != "" {
                args = append(args, "-ufm")
        } else {
                args = append(args, "-uf")
        }
        args = append(args, libname, "-C", filepath.Join(ic.out, "classes"), ".")
        c = &excmd{ name:"jar", slient:androidsdkSlientSome, }
        if !c.run("PackageClasses", args...) {
                errorf(0, "pack classes: %v", libname)
        }

        if e := os.Rename(libname, ic.target); e != nil {
                errorf(0, "rename: %v", ic.target)
        }

        return true
}
