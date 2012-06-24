package smart

import (
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
        top string
        target *Target
}

func (sdk *asdk) SetTop(d string) {
        sdk.top = d
}

func (sdk *asdk) Goals() (a []*Target) {
        if sdk.target != nil {
                a = []*Target{ sdk.target }
        }
        return
}

func (sdk *asdk) NewCollector(t *Target) Collector {
        return &asdkCollector{ sdk, t }
}

func (sdk *asdk) Generate(t *Target) error {
        //fmt.Printf("Generate: %v (goal:%v, inter:%v; file:%v, dir:%v)\n", t, t.IsGoal, t.IsIntermediate, t.IsFile, t.IsDir)

        // .dex+res --(pack)--> .apk.unsigned --(sign)--> .apk.signed --(align)--> .apk
        switch {
        case t.IsDir && t.Name == "out/classes": return sdk.compileJava(t)
        case t.IsDir && t.Name == "out/res": return sdk.compileResource(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".dex"): return sdk.dx(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".apk.unsigned"): return sdk.packUnsigned(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".apk.signed"): return sdk.sign(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".apk"): return sdk.align(t)
        case t.IsFile && strings.HasSuffix(t.Name, ".jar"): return sdk.packJar(t)
        }
        return nil
}

func (sdk *asdk) compileJava(t *Target) error {
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
        os.MkdirAll(t.Name, 0755) // make empty out/classes

        fmt.Printf("compile -o %v %v\n", t.Name, t.Depends)
        p := exec.Command("javac", args...)
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        return p.Run() //run("javac", args...)
}

func (sdk *asdk) compileResource(t *Target) error {
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

        if e := os.MkdirAll(t.Name, 0755); e != nil {
                return e
        }

        // Produces R.java under t.Name
        fmt.Printf("compile -o %v assets+resources\n", t)
        p := exec.Command("aapt", args...)
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        return p.Run() //return run("aapt", args...)
}

func (sdk *asdk) dx(t *Target) error {
        var classes *Target
        if len(t.Depends) == 1 {
                classes = t.Depends[0]
        } else {
                return NewErrorf("expect 1 depend: %v->%v\n", t, t.Depends)
        }

        var args []string

        switch runtime.GOOS {
        case "windows": args = append(args, "-JXms16M", "-JXmx1536M")
        }

        args = append(args, "--dex", "--output="+t.Name)
        args = append(args, classes.Name)

        fmt.Printf("dex -o %v %v\n", t, classes)
        p := exec.Command("dx", args...)
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        return p.Run() //run("dx", args...)
}

func (sdk *asdk) getKeystore() (keystore, keypass, storepass string) {
        var canditates []string

        defaultPass := "smart.android"

        readpass := func(s, sn string) string {
                b := readFile(filepath.Join(filepath.Dir(s), sn))
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
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        p.Dir = dir
        if e := p.Run(); e != nil {
                return e
        }

        p = exec.Command("zip", "-qd", name, "dummy")
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        p.Dir = dir
        if e := p.Run(); e != nil {
                os.Remove(name)
                return e
        }

        return nil
}

func (sdk *asdk) sign(t *Target) (e error) {
        if (1 != len(t.Depends)) {
                return NewErrorf("expect 1 depend: %v", t.Depends)
        }

        unsigned := t.Depends[0]

        keystore, keypass, storepass := sdk.getKeystore()
        if keystore == "" || keypass == "" || storepass == "" {
                //return NewErrorf("no available keystore")
        }

        if e = copyFile(unsigned.Name, t.Name); e != nil {
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
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr

        fmt.Printf("sign -o %v %v\n", t, unsigned)
        return p.Run()
}

func (sdk *asdk) align(t *Target) (e error) {
        if (1 != len(t.Depends)) {
                return NewErrorf("expect 1 depend: %v", t.Depends)
        }

        signed := t.Depends[0]

        zipalign := filepath.Join(asdkRoot, "tools", "zipalign")
        args := []string{ "-f", "4", signed.Name, t.Name, }
        return run32(zipalign, args...)
}

func (sdk *asdk) packUnsigned(t *Target) (e error) {
        if len(t.Depends) != 1 {
                return NewErrorf("expect 1 depends: %v->%v", t, t.Depends)
        }

        dex := t.Depends[0]
        dexDir := filepath.Dir(dex.Name)
        dexName := filepath.Base(dex.Name)

        apkName := filepath.Base(t.Name)

        if e = sdk.createEmptyPackage(t.Name); e != nil {
                return e
        }
        
        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        args := []string{ "package", "-u",
                "-F", t.Name, // e.g. "out/_.apk.unsigned"
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

func (sdk *asdk) packJar(t *Target) error {
        return nil
}

type asdkCollector struct {
        sdk *asdk
        t *Target
}

func (coll *asdkCollector) classes() (classes *Target) {
        classesName := "out/classes"
        if classes = T(classesName); classes == nil {
                // assert(len(coll.t.Depends[0]) == 1)
                // assert(len(coll.t.Depends[0].Depends[0]) == 1)
                dexName := classesName + ".dex"
                t := coll.t.Depends[0].Depends[0]
                dex := t.AddIntermediateFile(dexName, nil)
                classes = dex.AddIntermediateDir(classesName, nil)
        }
        return
}

func (coll *asdkCollector) extractPackageName(am string) (pkg string, tagline int) {
        tagline = -1
        forEachLine(am, func(lineno int, line []byte) bool {
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

func (coll *asdkCollector) AddDir(dir string) (t *Target) {
        //fmt.Printf("%v\n", dir)

        switch dir {
        case "out": return nil
        case "src":
                find(dir, `^.*?\.java$`, coll)
        case "res": fallthrough
        case "assets":
                resName := "out/res"
                res := T(resName)
                if res == nil {
                        res = NewDirIntermediate(resName)
                }

                t = res.AddDir(dir)

                // Add R.java target
                if pkg, ok := coll.t.Variables["package"]; ok {
                        pkg = strings.Replace(pkg, ".", string(filepath.Separator), -1)
                        rjava := filepath.Join(res.Name, pkg, "R.java")

                        classes := coll.classes()
                        r := classes.AddIntermediateFile(rjava, res)

                        if r == nil {
                                // TODO: error
                        }

                        //fmt.Printf("%v: %v\n", coll.t, coll.t.Depends)
                        //fmt.Printf("%v: %v\n", r, r.Depends)
                } else {
                        // TODO: error
                }
        }

        return
}

func (coll *asdkCollector) AddFile(dir, name string) (t *Target) {
        dname := filepath.Join(dir, name)
        //fmt.Printf("file: %v\n", dname)

        if coll.t == nil && dir == "" && name == "AndroidManifest.xml" {
                pkg, tagline := coll.extractPackageName(name)
                if 0 < tagline && pkg != "" {
                        coll.t = NewFileGoal(pkg + ".apk")
                        coll.t.Variables["package"] = pkg
                        signed := coll.t.AddIntermediateFile("out/_.apk.signed", nil)
                        unsigned := signed.AddIntermediateFile("out/_.apk.unsigned", nil)
                        if unsigned == nil { /* TODO: error */ }
                } else {
                        fmt.Printf("%v:%v: no package name", name, tagline)
                        return nil
                }

                if coll.sdk.target == nil {
                        coll.sdk.target = coll.t
                }

                return coll.t
        }

        if coll.t == nil {
                fmt.Printf("no target for %v\n", dname)
                return
        }

        if !strings.HasSuffix(name, ".java") {
                return
        }

        classes := coll.classes()
        t = classes.AddFile(dname)
        //fmt.Printf("%v:%v\n", coll.t, coll.t.Depends)

        return
}
