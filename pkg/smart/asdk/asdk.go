package asdk

import (
        ".." // smart
        "bytes"
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
                smart.Fatal("Can't locate Android SDK.")
        }
}

type asdkProject struct {
        target, signed, unsigned, dex, classes, res *smart.Target
}

type asdk struct {
        top, out string
        proj *asdkProject
}

func New() (sdk *asdk) {
        sdk = &asdk{ out:"out" }
        return
}

func (sdk *asdk) SetTop(d string) {
        sdk.top = d
}

func (sdk *asdk) Goals() (a []*smart.Target) {
        if sdk.proj != nil && sdk.proj.target != nil {
                a = []*smart.Target{ sdk.proj.target }
        }
        return
}

func (sdk *asdk) NewCollector(t *smart.Target) smart.Collector {
        proj := &asdkProject{ target:t }
        coll := &asdkCollector{ sdk:sdk, proj:proj }

        if sdk.proj == nil {
                sdk.proj = proj
                coll.makeTargets("") // "" indicates top dir
        }

        return coll
}

func (sdk *asdk) Generate(t *smart.Target) error {
        //smart.Info("Generate: %v:%v...", t, t.Depends)

        isFile := func(s string) bool {
                return t.IsFile && strings.HasSuffix(t.Name, s)
        }

        isOutDir := func(s string) bool {
                if !t.IsDir { return false }
                separator := string(filepath.Separator)
                if !strings.HasPrefix(t.Name, sdk.out+separator) { return false }
                return strings.HasSuffix(t.Name, separator+s)
        }

        // .dex+res --(pack)--> .unsigned --(sign)--> .signed --(align)--> .apk
        switch {
        case isOutDir("classes"):       return sdk.compileJava(t)
        case isOutDir("res"):           return sdk.compileResource(t)
        case isFile(".dex"):            return sdk.dx(t)
        case isFile(".unsigned"):       return sdk.packUnsigned(t)
        case isFile(".signed"):         return sdk.sign(t)
        case isFile(".apk"):            return sdk.align(t)
        case isFile(".jar"):            return sdk.packJar(t)
        case isFile("R.java"):          return nil
        default: smart.Warn("ignored: %v", t)
        }

        return nil
}

func (sdk *asdk) compileResource(t *smart.Target) (e error) {
        //assert(t.IsDir)

        var top string
        if s, ok := t.Variables["top"]; ok { top = s }
        if top == "" {
                smart.Fatal("no top variable in %v", t)
        }

        args := []string{
                "package", "-m",
                "-J", t.Name, // should be out/res
                "-I", filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar"),
        }

        var sources []string
        if s := filepath.Join(top, "AndroidManifest.xml"); smart.IsFile(s) {
                args = append(args, "-M", s)
                sources = append(sources, s)
        }
        if s := filepath.Join(top, "res"); smart.IsDir(s) {
                args = append(args, "-S", s)
                sources = append(sources, s)
        }
        if s := filepath.Join(top, "assets"); smart.IsDir(s) {
                args = append(args, "-A", s)
                sources = append(sources, s)
        }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")

        // TODO: -P -G

        if e = os.MkdirAll(t.Name, 0755); e != nil {
                return
        }

        smart.Info("compile -o %v %v", t, strings.Join(sources, " "))

        // Produces R.java under t.Name
        p := smart.Command("aapt", args...)
        e = p.Run() //return run("aapt", args...)
        return
}

func (sdk *asdk) compileJava(t *smart.Target) (e error) {
        classpath := filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar")

        var sources []string
        for _, d := range t.Depends {
                ext := filepath.Ext(d.Name)
                switch {
                case ext == ".java": sources = append(sources, d.Name)
                case ext == ".jar": classpath += ":" + d.Name
                default: smart.Warn("ignored: %v", d)
                }
        }

        if 0 == len(sources) {
                e = smart.NewErrorf("no java sources for %v", t)
                return
        }

        os.RemoveAll(t.Name) // clean all in out/classes
        if e = os.MkdirAll(t.Name, 0755); e != nil { // make empty out/classes
                return
        }

        defer func() {
                if e != nil {
                        os.RemoveAll(t.Name)
                }
        }()

        args := []string {
                "-d", t.Name, // should be out/classes
                //"-sourcepath", filepath.Join(sdk.top, "src"),
                "-cp", classpath,
        }
        args = append(args, sources...)

        if true {
                smart.Info("compile -o %v %v", t.Name, strings.Join(sources, " "))
        } else {
                smart.Info("javac %v", strings.Join(args, " "))
        }

        p := smart.Command("javac", args...)
        e = p.Run() //run("javac", args...)
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

        smart.Info("dex -o %v %v", t, classes)
        p := smart.Command("dx", args...)
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

        smart.Info("pack -o %v <new>", pkg)
        p := smart.Command("jar", "cf", name, "dummy")
        p.Dir = dir
        if e := p.Run(); e != nil {
                return e
        }

        p = smart.Command("zip", "-qd", name, "dummy")
        p.Dir = dir
        if e := p.Run(); e != nil {
                os.Remove(name)
                return e
        }

        return nil
}

func (sdk *asdk) sign(t *smart.Target) (e error) {
        unsigned := sdk.proj.unsigned
        if unsigned == nil {
                smart.Fatal("no unsigned for %v", t)
        }

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

        p := smart.Command("jarsigner", args...)

        smart.Info("sign -o %v %v", t, unsigned)
        return p.Run()
}

func (sdk *asdk) align(t *smart.Target) (e error) {
        signed := sdk.proj.signed
        if signed == nil {
                smart.Fatal("no signed for %v", t)
        }

        smart.Info("align -o %v %v", t.Name, signed.Name)

        zipalign := filepath.Join(asdkRoot, "tools", "zipalign")
        args := []string{ zipalign, "-f", "4", signed.Name, t.Name, }
        p := smart.Command("linux32", args...)
        return p.Run() //run32(zipalign, args...)
}

func (sdk *asdk) packUnsigned(t *smart.Target) (e error) {
        var dex *smart.Target
        for _, d := range t.Depends {
                if d.IsFile && strings.HasSuffix(d.Name, ".dex") {
                        dex = d; break
                }
        }

        if dex == nil {
                return smart.NewErrorf("no dex for %v (%v)", t, t.Depends)
        }

        if e = sdk.createEmptyPackage(t.Name); e != nil {
                return e
        }
        
        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        var top string
        if s, ok := t.Variables["top"]; ok { top = s }
        if top == "" {
                smart.Fatal("no top variable in %v", t)
        }

        args := []string{ "package", "-u",
                "-F", t.Name, // e.g. "out/_.unsigned"
                "-I", filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar"),
        }

        var sources []string
        if s := filepath.Join(top, "AndroidManifest.xml"); smart.IsFile(s) {
                args = append(args, "-M", s)
                sources = append(sources, s)
        }
        if s := filepath.Join(top, "res"); smart.IsDir(s) {
                args = append(args, "-S", s)
                sources = append(sources, s)
        }
        if s := filepath.Join(top, "assets"); smart.IsDir(s) {
                args = append(args, "-A", s)
                sources = append(sources, s)
        }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")

        smart.Info("pack -o %v %v", t, strings.Join(sources, " "))
        p := smart.Command("aapt", args...)
        p.Stdout = nil
        if e = p.Run(); e != nil {
                return
        }

        smart.Info("pack -o %v %v\n", t, dex)

        dexName := filepath.Base(dex.Name)
        apkName := filepath.Base(t.Name)
        p = smart.Command("aapt", "add", "-k", apkName, dexName)
        p.Stdout, p.Dir = nil, filepath.Dir(dex.Name)
        e = p.Run()
        return
}

func (sdk *asdk) packJar(t *smart.Target) (e error) {
        var top string
        if s, ok := t.Variables["top"]; ok { top = s }
        if top == "" {
                smart.Fatal("no top variable in %v", t)
        }

        var classes *smart.Target
        sep := string(filepath.Separator)
        for _, d := range t.Depends {
                if strings.HasPrefix(d.Name, "out"+sep) && strings.HasSuffix(d.Name, sep+"classes") {
                        classes = d; break
                }
        }

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

        am := filepath.Join(top, "AndroidManifest.xml")
        if smart.IsFile(am) {
                args := []string{ "package", "-u", "-M", am,
                        "-I", filepath.Join(asdkRoot, "platforms", asdkPlatform, "android.jar"),
                }

                sources := []string{ am }
                if s := filepath.Join(top, "res"); smart.IsDir(s) {
                        args = append(args, "-S", s)
                        sources = append(sources, s)
                }
                if s := filepath.Join(top, "assets"); smart.IsDir(s) {
                        args = append(args, "-A", s)
                        sources = append(sources, s)
                }
                //args = append(args, "--min-sdk-version", "7")
                //args = append(args, "--target-sdk-version", "7")

                smart.Info("pack -o %v %v", t, strings.Join(sources, " "))
                p := smart.Command("aapt", args...)
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

        smart.Info("pack -o %v %v", t, classes)

        args = append(args, t.Name, "-C", classes.Name, ".")
        p := smart.Command("jar", args...)
        if e = p.Run(); e != nil {
                return
        }

        return
}

type asdkCollector struct {
        sdk *asdk
        proj *asdkProject
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

/*
func (coll *asdkCollector) getPackageName() (pkg string) {
        if s, ok := coll.proj.target.Variables["package"]; ok {
                pkg = s
        } else if s, ok = coll.proj.target.Variables["top"]; ok {
                s, tagline := coll.extractPackageName(am)
                coll.proj.target.Variables["package"] = s
                pkg = s
        } else {
                smart.Fatal("no top variable for %v", coll.proj.target)
        }
        return
}
*/

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
        p := smart.Command("jar", args...)
        p.Stdin = f
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
                p := smart.Command("jar", args...)
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

func (coll *asdkCollector) makeTargets(dir string) bool {
        tt := ".apk"
        if strings.HasSuffix(dir, ".jar") {
                tt = ".jar"
        }

        top := dir
        if top == "" {
                top = coll.sdk.top
        }

        base := filepath.Base(dir) // it could be "" for top level
        out := filepath.Join(coll.sdk.out, base)
        outClasses := filepath.Join(out, "classes")

        am := filepath.Join(top, "AndroidManifest.xml")
        pkg, tagline := coll.extractPackageName(am)
        if 0 < tagline && pkg != "" {
                if coll.proj.target == nil {
                        coll.proj.target = smart.NewFileGoal(pkg + tt)
                }
                coll.proj.target.Variables["package"] = pkg
        } else {
                if !strings.HasSuffix(dir, ".jar") {
                        smart.Fatal("not .jar directory %v", dir)
                }
                if coll.proj.target == nil {
                        coll.proj.target = smart.NewFileGoal(base + tt)
                }
                delete(coll.proj.target.Variables, "package")
        }

        coll.proj.target.Type = tt
        coll.proj.target.Variables["top"] = top

        switch tt {
        case ".apk":
                coll.proj.signed = coll.proj.target.AddIntermediateFile(filepath.Join(out, "_.signed"), nil)
                coll.proj.unsigned = coll.proj.signed.AddIntermediateFile(filepath.Join(out, "_.unsigned"), nil)
                coll.proj.unsigned.Variables["top"] = top
                coll.proj.dex = coll.proj.unsigned.AddIntermediateFile(outClasses + ".dex", nil)
                coll.proj.dex.Type = ".dex"
                coll.proj.classes = coll.proj.dex.AddIntermediateDir(outClasses, nil)

        case ".jar":
                coll.proj.classes = coll.proj.target.AddIntermediateDir(outClasses, nil)

        default:
                smart.Fatal("unknown type: %v", dir)
        }

        coll.proj.classes.Type = "classes"
        coll.proj.classes.Variables["top"] = top

        //smart.Info("target: %v (%v)", coll.proj.target, coll.proj.target.Depends)

        if coll.proj.target == nil {
                smart.Fatal("no target for '%v'", dir)
        }

        return coll.proj.target != nil
}

func (coll *asdkCollector) addResDir(dir string) (t *smart.Target) {
        if coll.proj.res == nil {
                outRes := filepath.Join(filepath.Dir(coll.proj.classes.Name), "res")
                coll.proj.res = smart.NewDirIntermediate(outRes)
                coll.proj.res.Variables["top"] = filepath.Dir(dir)
        }
        
        t = coll.proj.res.AddDir(dir)
        
        // Add R.java target
        if pkg, ok := coll.proj.target.Variables["package"]; ok {
                pkg = strings.Replace(pkg, ".", string(filepath.Separator), -1)
                rjava := filepath.Join(coll.proj.res.Name, pkg, "R.java")
                
                if coll.proj.classes == nil {
                        smart.Fatal("no classes for %v", rjava)
                }

                r := coll.proj.classes.AddIntermediateFile(rjava, coll.proj.res)
                if r == nil {
                        smart.Fatal("inter: %v:%v", rjava, coll.proj.res)
                }
                //fmt.Printf("%v: %v\n", r, r.Depends)
        } else {
                smart.Fatal("no package name")
        }
        return
}

func (coll *asdkCollector) addJarDir(dir string) (t *smart.Target) {
        if coll.proj.classes == nil {
                smart.Fatal("no class output for %v", dir)
        }
        
        name := filepath.Join(coll.sdk.out, dir, "_.jar")
        t = coll.proj.classes.AddIntermediateFile(name, dir)

        /*
        if s, ok := coll.proj.unsigned.Variables["classpath"]; !ok {
                coll.proj.classes.Variables["classpath"] = name
        } else {
                coll.proj.classes.Variables["classpath"] = s + ":" + name
        }
        */

        jarColl := coll.sdk.NewCollector(t).(*asdkCollector)
        jarColl.makeTargets(dir)
        smart.Scan(jarColl, coll.sdk.top, dir)
        return
}

func (coll *asdkCollector) AddDir(dir string) (t *smart.Target) {
        base := filepath.Base(dir)

        switch {
        case dir == "out": return nil

        case base == "src":
                smart.Find(dir, `^.*?\.java$`, coll)

        case base == "res": fallthrough
        case base == "assets":
                return coll.addResDir(dir)

        case strings.HasSuffix(dir, ".jar"):
                return coll.addJarDir(dir)

        default:
                smart.Warn("ignored: %v", dir)
        }

        return
}

func (coll *asdkCollector) AddFile(dir, name string) (t *smart.Target) {
        dname := filepath.Join(dir, name)

        if coll.proj.target == nil {
                smart.Fatal("no target for %v", dname)
        }

        switch {
        case name == "AndroidManifest.xml":
                t =  coll.proj.target.AddFile(dname)

        case strings.HasSuffix(name, ".java"):
                if coll.proj.classes == nil {
                        smart.Warn("ignored: %v (classes=%v)", dname, coll.proj.classes)
                } else {
                        t = coll.proj.classes.AddFile(dname)
                }

        default:
                smart.Warn("ignored: %v", dname)
        }

        return
}
