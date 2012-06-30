package smart_asdk

import (
        ".."
        "bytes"
        "fmt"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "runtime"
        "strings"
)

var asdkRoot = "/android-sdk-linux_x86"
var asdkPlatform = "android-10"
var _regAsdkAMTag = regexp.MustCompile(`^\s*<\s*manifest\s+`)
var _regAsdkPkgAttr = regexp.MustCompile(`\s+(package\s*=\s*"([^"]*)"\s*)`)

func init() {
        /****/ if c, e := exec.LookPath("android"); e == nil {
                asdkRoot = filepath.Dir(filepath.Dir(c))
        } else if c, e := exec.LookPath("aapt"); e == nil {
                asdkRoot = filepath.Dir(filepath.Dir(c))
        } else {
                asdkRoot = os.Getenv("ANDROIDSDK")
        }

        if  asdkRoot == "" {
                fmt.Printf("Can't locate Android SDK.\n")
        }
}

type asdk struct {
        top, out, outRes, outClasses string
        target, signed, unsigned, dex, classes *smart.Target
}

func (sdk *asdk) SetTop(d string) {
        sdk.top = d
        sdk.out = "out"
        sdk.outRes = filepath.Join(sdk.out, "res")
        sdk.outClasses = filepath.Join(sdk.out, "classes")
}

func (sdk *asdk) Goals() (a []*smart.Target) {
        if sdk.target != nil {
                a = []*smart.Target{ sdk.target }
        }
        return
}

func (sdk *asdk) NewCollector(t *smart.Target) smart.Collector {
        return &asdkCollector{ sdk:sdk, target:t }
}

func (sdk *asdk) Generate(t *smart.Target) error {
        //fmt.Printf("Generate: %v (goal:%v, inter:%v; file:%v, dir:%v)\n", t, t.IsGoal, t.IsIntermediate, t.IsFile, t.IsDir)

        // .dex+res --(pack)--> .unsigned --(sign)--> .signed --(align)--> .apk
        switch {
        case t.IsDir && t.Name == sdk.outClasses: return sdk.compileJava(t)
        case t.IsDir && t.Name == sdk.outRes: return sdk.compileResource(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".dex"): return sdk.dx(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".unsigned"): return sdk.packUnsigned(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".signed"): return sdk.sign(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".apk"): return sdk.align(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".jar"): return sdk.packJar(t)
        }
        //fmt.Printf("ignored: %v\n", t)
        return nil
}

func (sdk *asdk) compileJava(t *smart.Target) (e error) {
        //assert(t.IsDir)

        classpath := filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar")
        //if 0 < len(sdk.classpath) {
        //        classpath += ":" + strings.Join(sdk.classpath, ":")
        //}

        args := []string {
                "-d", t.Name, // should be out/classes
                //"-sourcepath", filepath.Join(sdk.top, "src"),
                "-cp", classpath,
        }

        for _, d := range t.Depends {
                args = append(args, d.Name)
        }

        os.RemoveAll(t.Name) // clean all in out/classes
        if e = os.MkdirAll(t.Name, 0755); e != nil { // make empty out/classes
                return
        }

        fmt.Printf("compile -o %v %v\n", t.Name, t.Depends)
        p := exec.Command("javac", args...)
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        e = p.Run() //run("javac", args...)
        return
}

func (sdk *asdk) compileResource(t *smart.Target) (e error) {
        //assert(t.IsDir)

        args := []string{
                "package", "-m",
                "-J", t.Name, // should be out/res
                "-M", filepath.Join("", "AndroidManifest.xml"),
                "-I", filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar"),
        }
        args = append(args, "-S", filepath.Join("res"))
        //args = append(args, "-A", filepath.Join("assets"))

        // TODO: -P -G

        if e = os.MkdirAll(t.Name, 0755); e != nil {
                return
        }

        // Produces R.java under t.Name
        fmt.Printf("compile -o %v assets+resources\n", t)
        p := exec.Command("aapt", args...)
        p.Stdout, p.Stderr = os.Stdout, os.Stderr
        e = p.Run() //return run("aapt", args...)
        return
}

func (sdk *asdk) dx(t *smart.Target) error {
        var classes *smart.Target
        if len(t.Depends) == 1 {
                classes = t.Depends[0]
        } else {
                return smart.NewErrorf("expect 1 depend: %v->%v\n", t, t.Depends)
        }

        var args []string

        switch runtime.GOOS {
        case "windows": args = append(args, "-JXms16M", "-JXmx1536M")
        }

        args = append(args, "--dex", "--output="+t.Name)
        args = append(args, classes.Name)

        fmt.Printf("dex -o %v %v\n", t, classes)
        p := exec.Command("dx", args...)
        p.Stdout, p.Stderr = os.Stdout, os.Stderr
        return p.Run() //run("dx", args...)
}

func (sdk *asdk) getKeystore() (keystore, keypass, storepass string) {
        var canditates []string

        defaultPass := "smart.android"

        readpass := func(s, sn string) string {
                b := smart.ReadFile(filepath.Join(filepath.Dir(s), sn))
                if b == nil {
                        return defaultPass
                }
                return strings.TrimSpace(string(b))
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

func (sdk *asdk) createEmptyPackage(pkg string) error {
        dir := filepath.Dir(pkg)
        name := filepath.Base(pkg)
        dummy := filepath.Join(dir, "dummy")

        if f, e := os.Create(dummy); e != nil {
                return e
        } else {
                f.Close()
        }

        defer os.Remove(dummy)

        fmt.Printf("pack -o %v <new>\n", pkg)
        p := exec.Command("jar", "cf", name, "dummy")
        p.Stdout, p.Stderr = os.Stdout, os.Stderr
        p.Dir = dir
        if e := p.Run(); e != nil {
                return e
        }

        p = exec.Command("zip", "-qd", name, "dummy")
        p.Stdout, p.Stderr = os.Stdout, os.Stderr
        p.Dir = dir
        if e := p.Run(); e != nil {
                os.Remove(name)
                return e
        }

        return nil
}

func (sdk *asdk) sign(t *smart.Target) (e error) {
        if (1 != len(t.Depends)) {
                return smart.NewErrorf("expect 1 depend: %v", t.Depends)
        }

        unsigned := t.Depends[0]

        keystore, keypass, storepass := sdk.getKeystore()
        if keystore == "" || keypass == "" || storepass == "" {
                //return smart.NewErrorf("no available keystore")
        }

        if e = smart.CopyFile(unsigned.Name, t.Name); e != nil {
                return
        }

        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        args := []string{
                "-keystore", keystore,
                "-keypass", keypass,
                "-storepass", storepass,
                t.Name, "cert",
        }

        p := exec.Command("jarsigner", args...)
        p.Stdout, p.Stderr = os.Stdout, os.Stderr

        fmt.Printf("sign -o %v %v\n", t, unsigned)
        return p.Run()
}

func (sdk *asdk) align(t *smart.Target) (e error) {
        if (1 != len(t.Depends)) {
                return smart.NewErrorf("expect 1 depend: %v", t.Depends)
        }

        signed := t.Depends[0]

        fmt.Printf("align -o %v %v\n", t.Name, signed.Name)

        zipalign := filepath.Join(asdkRoot, "tools", "zipalign")
        args := []string{ zipalign, "-f", "4", signed.Name, t.Name, }
        p := exec.Command("linux32", args...)
        p.Stdout, p.Stderr = os.Stdout, os.Stderr
        return p.Run() //run32(zipalign, args...)
}

func (sdk *asdk) packUnsigned(t *smart.Target) (e error) {
        if len(t.Depends) != 1 {
                return smart.NewErrorf("expect 1 depends: %v->%v", t, t.Depends)
        }

        if e = sdk.createEmptyPackage(t.Name); e != nil {
                return e
        }
        
        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        dex := t.Depends[0]
        dexDir := filepath.Dir(dex.Name)
        dexName := filepath.Base(dex.Name)
        apkName := filepath.Base(t.Name)

        args := []string{ "package", "-u",
                "-F", t.Name, // e.g. "out/_.unsigned"
                "-M", filepath.Join(""/*sdk.top*/, "AndroidManifest.xml"),
                "-I", filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar"),
        }
        args = append(args, "-S", filepath.Join("res"))
        //args = append(args, "-A", filepath.Join("assets"))
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")

        fmt.Printf("pack -o %v assets+resources\n", t)
        p := exec.Command("aapt", args...)
        p.Stderr = os.Stderr
        if e = p.Run(); e != nil {
                return
        }

        fmt.Printf("pack -o %v %v\n", t, dex)
        p = exec.Command("aapt", "add", "-k", apkName, dexName)
        p.Stderr = os.Stderr
        p.Dir = dexDir
        if e = p.Run(); e != nil {
                return
        }

        return
}

func (sdk *asdk) packJar(t *smart.Target) (e error) {
        fmt.Printf("TODO: packJar: %v\n", t)

        os.RemoveAll(t.Name) // clean all in out/classes
        if e = os.MkdirAll(filepath.Dir(t.Name), 0755); e != nil { // make empty out dir
                return e
        }
        if e = sdk.createEmptyPackage(t.Name); e != nil {
                return e
        }
        
        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        am := filepath.Join("foo.jar", "AndroidManifest.xml")
        if fi, fe := os.Stat(am); fe == nil && fi.Mode()&os.ModeType == 0 {
                args := []string{ "package", "-u", "-M", am,
                        "-I", filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar"),
                }
                args = append(args, "-S", filepath.Join("res"))
                //args = append(args, "-A", filepath.Join("assets"))
                //args = append(args, "--min-sdk-version", "7")
                //args = append(args, "--target-sdk-version", "7")

                p := exec.Command("aapt", args...)
                p.Stdout, p.Stderr = os.Stdout, os.Stderr
                if e = p.Run(); e != nil {
                        return
                }
        }

        var args []string
        var manifest string
        if manifest != "" {
                args = []string{ "-ufm" }
        } else {
                args = []string{ "-uf" }
        }
        args = append(args, t.Name, "-C", sdk.outClasses, ".")
        p := exec.Command("jar", args...)
        p.Stdout, p.Stderr = os.Stdout, os.Stderr
        if e = p.Run(); e != nil {
                return
        }

        return
}

type asdkCollector struct {
        sdk *asdk
        target, signed, unsigned, dex, classes *smart.Target
}

func (coll *asdkCollector) extractPackageName(am string) (pkg string, tagline int) {
        tagline = -1
        smart.ForEachLine(am, func(lineno int, line []byte) bool {
                if _regAsdkAMTag.Match(line) {
                        tagline = lineno
                        return true
                }

                if 0 < tagline {
                        if a := _regAsdkPkgAttr.FindStringSubmatch(string(line)); a != nil {
                                //fmt.Printf("%v:%v: %v\n", am, lineno, a[2])
                                pkg = a[2]
                                return false
                        }
                }

                //fmt.Printf("%v:%v: %v\n", am, lineno, string(line))
                return true
        })
        return
}

func (coll *asdkCollector) extractClasses(outclasses, lib string, cs []string) (classes []string) {
        f, err := os.Open(lib)
        if err != nil {
                // TODO: error
                return
        }
        defer f.Close()

        var wd string
        if s, e := os.Getwd(); e != nil {
                // TODO: error
                return
        } else {
                wd = s
        }
        if e := os.Chdir(outclasses); e != nil {
                // TODO: error
                return
        }
        defer func() {
                if e := os.Chdir(wd); e != nil {
                        // TODO: error
                }
        }()

        args := append([]string{ "-x" }, cs...)
        p := exec.Command("jar", args...)
        p.Stdin, p.Stdout, p.Stderr = f, os.Stdout, os.Stderr
        if e := p.Run(); e != nil {
                // TODO: error
                return
        }

        for _, s := range cs {
                if fi, er := os.Stat(s); er != nil || fi == nil {
                        // TODO: error
                        return
                }
                classes = append(classes, s)
        }

        return
}

func (coll *asdkCollector) extractStaticLibsClasses(outclasses string, libs []string) (classes []string) {
        for _, lib := range libs {
                if lib = strings.TrimSpace(lib); lib == "" { continue }

                out := bytes.NewBuffer(nil)

                args := []string{ "-tf", lib }
                p := exec.Command("jar", args...)
                p.Stdout, p.Stderr = out, os.Stderr
                if e := p.Run(); e != nil {
                        // TODO: error
                        return
                }

                var cs []string
                for _, s := range strings.Split(out.String(), "\n") {
                        if strings.HasSuffix(s, ".class") {
                                cs = append(cs, s)
                        }
                }

                //fmt.Printf("jar: %v: %v\n", lib, cs)

                classes = append(classes, coll.extractClasses(outclasses, lib, cs)...)
        }
        //fmt.Printf("embeded-classes: %v\n", classes)
        return
}

func (coll *asdkCollector) AddDir(dir string) (t *smart.Target) {
        //fmt.Printf("%v\n", dir)

        switch dir {
        case "out": return nil
        case "src":
                smart.Find(dir, `^.*?\.java$`, coll)
        case "res": fallthrough
        case "assets":
                res := smart.T(coll.sdk.outRes)
                if res == nil {
                        res = smart.NewDirIntermediate(coll.sdk.outRes)
                }

                t = res.AddDir(dir)

                // Add R.java target
                if pkg, ok := coll.target.Variables["package"]; ok {
                        pkg = strings.Replace(pkg, ".", string(filepath.Separator), -1)
                        rjava := filepath.Join(res.Name, pkg, "R.java")

                        if coll.classes != nil {
                                r := coll.classes.AddIntermediateFile(rjava, res)
                                if r == nil {
                                        // TODO: error
                                }
                                //fmt.Printf("%v: %v\n", r, r.Depends)
                        }
                } else {
                        // TODO: error
                }

        default:
                if strings.HasSuffix(dir, ".jar") {
                        unsigned := coll.sdk.unsigned
                        if unsigned == nil {
                                fmt.Printf("no unsigned: %v\n", coll.sdk.target)
                                return
                        }

                        name := filepath.Join(coll.sdk.out, dir[0:len(dir)-4], dir)
                        t = unsigned.AddIntermediateFile(name, dir)
                        //fmt.Printf("AddDir: %v -> %v (%v:%v)\n", dir, t, unsigned, unsigned.Depends)

                        if t != nil {
                                smart.Scan(coll.sdk.NewCollector(t), coll.sdk.top, dir)
                        }
                }
        }

        return
}

func (coll *asdkCollector) AddFile(dir, name string) (t *smart.Target) {
        dname := filepath.Join(dir, name)
        //fmt.Printf("file: %v\n", dname)

        if coll.target == nil && dir == "" && name == "AndroidManifest.xml" {
                pkg, tagline := coll.extractPackageName(name)
                if 0 < tagline && pkg != "" {
                        coll.target = smart.NewFileGoal(pkg + ".apk")
                        coll.target.Variables["package"] = pkg
                        coll.signed = coll.target.AddIntermediateFile(filepath.Join(coll.sdk.out, "_.signed"), nil)
                        coll.unsigned = coll.signed.AddIntermediateFile(filepath.Join(coll.sdk.out, "_.unsigned"), nil)
                        coll.dex = coll.unsigned.AddIntermediateFile(coll.sdk.outClasses + ".dex", nil)
                        coll.classes = coll.dex.AddIntermediateDir(coll.sdk.outClasses, nil)
                } else {
                        fmt.Printf("%v:%v: no package name", name, tagline)
                        return nil
                }

                if coll.sdk.target == nil {
                        coll.sdk.target = coll.target
                        coll.sdk.signed = coll.signed
                        coll.sdk.unsigned = coll.unsigned
                        coll.sdk.dex = coll.dex
                        coll.sdk.classes = coll.classes
                }

                return coll.target
        }

        if coll.target == nil {
                fmt.Printf("no target for %v\n", dname)
                return
        }

        if !strings.HasSuffix(name, ".java") {
                return
        }

        if coll.classes != nil {
                t = coll.classes.AddFile(dname)
                //fmt.Printf("%v:%v\n", coll.target, coll.target.Depends)
        }

        return
}
