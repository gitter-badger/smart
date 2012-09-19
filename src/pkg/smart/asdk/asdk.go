package asdk

import (
        ".." // smart
	"fmt"
        "bufio"
        "bytes"
        "flag"
        "io"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "runtime"
        "strings"
	//"xml"
)

// ref: http://developer.android.com/guide/topics/manifest/uses-sdk-element.html
var apiLevels = [][]string{
	16: { "JELLY_BEAN",             "4.1", "4.1.1" },
	15: { "ICE_CREAM_SANDWICH_MR1", "4.0.3", "4.0.4" },
 	14: { "ICE_CREAM_SANDWICH",     "4.0", "4.0.1", "4.0.2" },
	13: { "HONEYCOMB_MR2",          "3.2" },
	12: { "HONEYCOMB_MR1",          "3.1.x" },
	11: { "HONEYCOMB",              "3.0.x" },
        10: { "GINGERBREAD_MR1",        "2.3.3", "2.3.4",},
        9:  { "GINGERBREAD",            "2.3", "2.3.1", "2.3.2", },
 	8:  { "FROYO",                  "2.2.x", },
 	7:  { "ECLAIR_MR1",             "2.1.x", },
 	6:  { "ECLAIR_0_1",             "2.0.1", },
 	5:  { "ECLAIR",                 "2.0", },
 	4:  { "DONUT",                  "1.6", },
 	3:  { "CUPCAKE",                "1.5", },
 	2:  { "BASE_1_1",               "1.1", },
  	1:  { "BASE",                   "1.0", },
        0:  nil,
}

var asdkRoot = "/android-sdk-linux_x86"
var asdkPlatform = "android-10"
var _regAsdkAMTag = regexp.MustCompile(`^\s*<\s*manifest\s*`)
var _regAsdkPkgAttr = regexp.MustCompile(`\s+(package\s*=\s*"([^"]*)"\s*)`)

const asdkExtractStaticLibs = false

func init() {
        /****/ if c, e := exec.LookPath("android"); e == nil {
                asdkRoot = filepath.Dir(filepath.Dir(c))
        } else if c, e := exec.LookPath("aapt"); e == nil {
                asdkRoot = filepath.Dir(filepath.Dir(c))
        } else {
                asdkRoot = os.Getenv("ANDROIDSDK")
        }

        if  asdkRoot == "" {
                smart.Fatal("Can't find any Android SDK installaion.")
        }
}

func asdkGetPlatformTool(name string) string {
        return filepath.Join(asdkRoot, "platform-tools", name)
}

func asdkGetTool(name string) string {
        return filepath.Join(asdkRoot, "tools", name)
}

func asdkGetPlatformFile(name string) string {
        return filepath.Join(asdkRoot, "platforms", asdkPlatform, name)
}

type asdkProject struct {
	*smart.Project
        top, out, pkg string
	proguard string
        target, signed, unsigned, dex string
	outClasses, outLibs, outRes string
}

func (proj *asdkProject) init(out, top string) {
	proj.top = top;

	isJar := strings.HasSuffix(proj.top, ".jar")
        tt := ".apk"
        if isJar {
                tt = ".jar"
        }

        base := filepath.Base(proj.top) // it could be "" for top level
        out = filepath.Join(out, base)
        outClasses := filepath.Join(out, "classes")

	proj.out = out

        am := filepath.Join(proj.top, "AndroidManifest.xml")
        pkg, tagline, pkgline := proj.extractPackageName(am)
        if 0 < tagline && 0 < pkgline && pkg != "" {
		if proj.target == "" {
			proj.target = pkg + tt
		}
		proj.pkg = pkg
        } else if isJar {
		if proj.target == "" {
			proj.target = base + tt
		}
        } else {
                smart.Fatal("asdk: no package name in %v", filepath.Join(proj.top, "AndroidManifest.xml"))
	}

        if proj.target == "" {
                smart.Fatal("no target for '%v'", proj.top)
        }

	t := smart.T(proj.target)
        t.Class = smart.FinalFile
        t.Type = tt
        t.SetVar("top", proj.top)

	var classes *smart.Target
        switch tt {
        case ".apk":
		proj.outClasses = outClasses
		proj.dex = proj.outClasses + ".dex"
		proj.signed = filepath.Join(out, "_.signed")
		proj.unsigned = filepath.Join(out, "_.unsigned")

                signed := t.Dep(proj.signed, smart.IntermediateFile)
		unsigned := signed.Dep(proj.unsigned, smart.IntermediateFile)
                unsigned.SetVar("top", proj.top)

                dex := unsigned.Dep(proj.dex, smart.IntermediateFile)
                dex.Type = ".dex"
                classes = dex.Dep(proj.outClasses, smart.IntermediateDir)

        case ".jar":
		proj.outClasses = outClasses
                classes = t.Dep(proj.outClasses, smart.IntermediateDir)

        default:
                smart.Fatal("unknown type: %v", proj.top)
        }

        classes.Type = "classes"
        classes.SetVar("top", proj.top)

        //smart.Info("target: %v (%v)", proj.target, proj.target.Dependees)
}

func (proj *asdkProject) extractPackageName(am string) (pkg string, tagline, pkgline int) {
        tagline, pkgline = -1, -1
        smart.ForEachLine(am, func(lineno int, line []byte) bool {
                if tagline < 0 && _regAsdkAMTag.Match(line) {
                        tagline = lineno
                        //return true
                }

                //smart.Info("%v:%v: (%d) %v", am, lineno, tagline, string(line))
                if 0 < tagline {
			a := _regAsdkPkgAttr.FindStringSubmatch(string(line))
                        //smart.Info("%v:%v: %v", am, lineno, a)
                        if a != nil {
                                pkg, pkgline = a[2], lineno
                                return false
                        }
                }

                //fmt.Printf("%v:%v: %v\n", am, lineno, string(line))
                return true
        })
        return
}

type asdk struct {
        top, out string
        proj *asdkProject // the root project
        staticLibs []string
}

func New() (sdk *asdk) {
        sdk = &asdk{ out:"out" }
        return
}

func (sdk *asdk) SetTop(dir string) {
	sdk.top = dir;
	//sdk.proj = new(asdkProject)
	//sdk.proj.init(sdk.out, dir);
}

func (sdk *asdk) Goals() (a []*smart.Target) {
	t := smart.T(sdk.proj.target)
        if sdk.proj != nil && t != nil {
                a = []*smart.Target{ t }
        }
        return
}

func (sdk *asdk) NewCollector(t *smart.Target) smart.Collector {
        coll := &asdkCollector{ sdk:sdk, proj:new(asdkProject) }

	if t != nil {
		coll.proj.target = t.Name
	}

	if sdk.proj == nil {
		sdk.proj = coll.proj
		sdk.proj.init(sdk.out, sdk.top);
	}
        return coll
}

func (sdk *asdk) Generate(t *smart.Target) error {
        //smart.Info("Generate: %v:%v...", t, t.Dependees)

        outlib := filepath.Join(sdk.proj.out, "lib")
        outsrc := filepath.Join(sdk.proj.out, "src")
        separator := string(filepath.Separator)

        isFile := func(s string) bool {
                return t.IsFile() && strings.HasSuffix(t.Name, s)
        }

        isOutDir := func(s string) bool {
                //if !t.IsDir() { return false }
                if !strings.HasPrefix(t.Name, sdk.out+separator) { return false }
                return strings.HasSuffix(t.Name, separator+s)
        }

        isLib := func() bool {
                return t.IsFile() && strings.HasPrefix(t.Name, outlib)
        }

	isJavaFromAidl := func() bool {
		if !t.IsFile() { return false }
		if !t.IsIntermediate() { return false }
		if !strings.HasPrefix(t.Name, outsrc) { return false }
                if !strings.HasSuffix(t.Name, ".java") { return false }
		if len(t.Dependees) <= 0 { return false }
		d0 := t.Dependees[0] // must have and .aidl dependee
		if !d0.IsFile() { return false }
		return strings.HasSuffix(d0.Name, ".aidl")
	}

        // .dex+res --(pack)--> .unsigned --(sign)--> .signed --(align)--> .apk
        switch {
        case isLib():                   return sdk.copyJNISharedLib(t)
        case isOutDir("classes"):       return sdk.compileJava(t)
        case isOutDir("res"):           return sdk.compileResource(t)
	case isJavaFromAidl():          return sdk.compileAidl(t)
        case isFile(".dex"):            return sdk.dx(t)
        case isFile(".unsigned"):       return sdk.packUnsigned(t)
        case isFile(".signed"):         return sdk.sign(t)
        case isFile(".apk"):            return sdk.align(t)
        case isFile(".jar"):            return sdk.packJar(t)
        case isFile("R.java"):          fallthrough
        case t.IsDir() && t.Name == "libs": fallthrough
        case t.IsDir() && t.Name == "res": fallthrough
        case t.IsDir() && strings.HasSuffix(t.Name, ".jar"): fallthrough
	//case strings.HasPrefix(t.Name, sdk.proj.out+separator): fallthrough
	case strings.HasPrefix(t.Name, "res"+separator):
                return nil
        default:
                segs := strings.SplitN(t.Name, separator, 2);
                if 0 < len(segs) {
                        switch {
                        case segs[0] == "libs": fallthrough
                        case segs[0] == "assets": fallthrough
                        case strings.HasSuffix(segs[0], ".jar"):
                                return nil
                        }
                }

                if smart.IsVerbose() {
                        smart.Info("ignored: %v (generate)", t)
                }
        }

        return nil
}

func (sdk *asdk) compileResource(t *smart.Target) (e error) {
        //assert(t.IsDir)

        var top string
        if top = t.Var("top"); top == "" {
                smart.Fatal("no top variable in %v", t)
        }

        args := []string{
                "package", "-m", "-J", t.Name, // should be out/res
		"-P", filepath.Join(sdk.out, filepath.Base(top), "public_resources.xml"),
                "-I", asdkGetPlatformFile("android.jar"),
        }

	for _, s := range sdk.staticLibs {
		args = append(args, "-I", s);
	}

        for _, d := range t.Dependees {
                ext := filepath.Ext(d.Name)
                switch {
                case ext == ".jar":
			args = append(args, "-I", d.Name)
                }
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
	aapt := asdkGetPlatformTool("aapt")
        p := smart.Command(aapt, args...)
        //if !smart.IsVerbose() {
        //        p.Stdout, p.Stderr = nil, nil
        //}
        e = p.Run() //return run("aapt", args...)
        return
}

func (sdk *asdk) compileAidl(t *smart.Target) (e error) {
        //smart.Fatal("aidl: %v %v", t, t.Dependees)

        if e = os.MkdirAll(filepath.Dir(t.Name), 0755); e != nil {
		smart.Fatal("mkdir: for %v", t)
                return
        }

        smart.Info("compile -o %v %v", t, t.Dependees[0])

        var top string
        if top = t.Var("top"); top == "" {
                //smart.Fatal("no top variable in %v", t)
		top = sdk.top
        }

	args := []string{
		//"-b",
		"-I" + filepath.Join(top, "src"),
		t.Dependees[0].Name, t.Name,
	}
	aidl := asdkGetPlatformTool("aidl")
        p := smart.Command(aidl, args...)
        //if !smart.IsVerbose() {
        //        p.Stdout, p.Stderr = nil, nil
        //}
	e = p.Run()
	return
}

// copyJNISharedLib copies libs/XXX/YYY into out/lib/XXX/YYY.
func (sdk *asdk) copyJNISharedLib(t *smart.Target) (e error) {
        if len(t.Dependees) != 1 {
                return smart.NewErrorf("no source JNI lib for %v", t)
        }

        if e = os.MkdirAll(filepath.Dir(t.Name), 0755); e != nil {
                return
        }

        lib := t.Dependees[0]

        smart.Info("copy -o %v %v", t, lib)
        if e = smart.CopyFile(lib.Name, t.Name); e != nil {
                return
        }
        return
}

func (sdk *asdk) compileJava(t *smart.Target) (e error) {
        classpath := []string{
                asdkGetPlatformFile("android.jar"),
        }
        classpath = append(classpath, sdk.staticLibs...)

        var sources []string
        for _, d := range t.Dependees {
                ext := filepath.Ext(d.Name)
                switch {
                case ext == ".java":
			if d.IsIntermediate() && 0 < len(d.Dependees) && strings.HasSuffix(d.Dependees[0].Name, ".aidl") {
				if d.Stat() != nil {
					// .aidl may contains no interfaces
					sources = append(sources, d.Name)
				}
			} else {
				sources = append(sources, d.Name)
			}
                case ext == ".jar":
			classpath = append(classpath, d.Name)
                default:
                        if smart.IsVerbose() {
                                smart.Info("ignored: %v (not Java)", d)
                        }
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
                "-cp", strings.Join(classpath, ":"),
        }
        args = append(args, sources...)

        smart.Info("compile -o %v <%d java files>", t.Name, len(sources))
	if false && smart.IsVerbose() {
		for _, s := range sources {
			smart.Info("\t%v", s)
		}
        }

        p := smart.Command("javac", args...)
        if e = p.Run(); e != nil {
                return
        }

        //smart.Info("==== %v ====", t)
	//smart.Command("find", t.Name, "-type", "f", "-name", "*.class").Run()
        return
}

func asdkExtractClasses(outclasses, lib string, cs []string) (classes []string) {
        f, err := os.Open(lib)
        if err != nil {
                smart.Fatal("open: %v (%v)", lib, err)
        }
        defer f.Close()

        var wd string
        if s, e := os.Getwd(); e != nil {
                smart.Fatal("getwd: %v", e)
                return
        } else {
                wd = s
        }

        if e := os.Chdir(outclasses); e != nil {
                smart.Fatal("chdir: %v", e)
                return
        }
        defer func() {
                if e := os.Chdir(wd); e != nil {
                        smart.Fatal("chdir: %v", e)
                }
        }()

        args := append([]string{ "-x" }, cs...)
        p := smart.Command("jar", args...)
        p.Stdin = f
        if e := p.Run(); e != nil {
                smart.Fatal("error: extract classes %v (%v)\n", lib, e)
        }

        for _, s := range cs {
                if fi, er := os.Stat(s); er != nil || fi == nil {
                        smart.Fatal("error: class `%v' not extracted (%v)", s, lib);
                        return
                }
                classes = append(classes, s)
        }

        return
}

func asdkExtractStaticLibsClasses(outclasses string, libs []string) (classes []string) {
        out := new(bytes.Buffer)

        for _, lib := range libs {
                if lib = strings.TrimSpace(lib); lib == "" { continue }

                out.Reset()

                args := []string{ "-tf", lib }
                p := smart.Command("jar", args...)
                p.Stdout = out
                if e := p.Run(); e != nil {
                        smart.Fatal("error: extract %v (%v)\n", lib, e)
                }

                var cs []string
                for _, s := range strings.Split(out.String(), "\n") {
                        if strings.HasSuffix(s, ".class") {
                                cs = append(cs, s)
                        }
                }

                //fmt.Printf("jar: %v: %v\n", lib, cs)

                classes = append(classes, asdkExtractClasses(outclasses, lib, cs)...)
        }

        //fmt.Printf("embeded-classes: %v\n", classes)
        return
}

func (sdk *asdk) dx(t *smart.Target) error {
        var classes *smart.Target
        if len(t.Dependees) == 1 {
                classes = t.Dependees[0]
        } else {
                return smart.NewErrorf("expect 1 depend: %v->%v\n", t, t.Dependees)
        }

	if asdkExtractStaticLibs {
		outclasses := filepath.Join(filepath.Dir(t.Name), "classes")
		embclasses := asdkExtractStaticLibsClasses(outclasses, sdk.staticLibs)
		if embclasses == nil {
		}
	}

	if (smart.T(sdk.proj.proguard) != nil) {
		os.Remove(sdk.proj.proguard)

		proguard := filepath.Join(asdkGetTool("proguard"), "lib", "proguard.jar")
		p := smart.Command("java", "-jar", proguard, "@"+smart.T(sdk.proj.proguard).Dependees[0].Name)
		if e := p.Run(); e != nil {
			return e
		}
	}

        var args []string

        switch runtime.GOOS {
        case "windows": args = append(args, "-JXms16M", "-JXmx1536M")
        }

	inputClasses := classes.Name
	if smart.T(sdk.proj.proguard) != nil {
		os.RemoveAll(classes.Name)
		inputClasses = sdk.proj.proguard
	}

        //args = append(args, "--verbose")
        args = append(args, "--dex", "--output="+t.Name)
        args = append(args, inputClasses)

	if !asdkExtractStaticLibs {
		for _, s := range sdk.staticLibs {
			args = append(args, s)
		}
	}

	for _, d := range classes.Dependees {
		ext := filepath.Ext(d.Name)
		switch {
		case ext == ".jar":
			args = append(args, d.Name)
		}
	}

	//smart.Info("classes: %v", inputClasses)
	//smart.Command("find", inputClasses, "-type", "f", "-name", "*.class").Run()

        smart.Info("dex %v", args)

        smart.Info("dex -o %v %v", t, inputClasses)
	dx := asdkGetPlatformTool("dx")
        p := smart.Command(dx, args...)
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

func (sdk *asdk) createEmptyPackage(t *smart.Target) error {
	pkg := t.Name
        dir := filepath.Dir(pkg)
        name := filepath.Base(pkg)
        dummy := filepath.Join(dir, "dummy")

        if f, e := os.Create(dummy); e != nil {
                return e
        } else {
                f.Close()
        }

        defer os.Remove(dummy)

        smart.Info("pack -o %v <empty>", pkg)
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
        if smart.T(unsigned) == nil {
                smart.Fatal("no unsigned for %v", t)
        }

        keystore, keypass, storepass := sdk.getKeystore()
        if keystore == "" || keypass == "" || storepass == "" {
                //return smart.NewErrorf("no available keystore")
        }

        if e = smart.CopyFile(unsigned, t.Name); e != nil {
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
        if smart.T(signed) == nil {
                smart.Fatal("no signed for %v", t)
        }

        smart.Info("align -o %v %v", t.Name, signed)

        zipalign := asdkGetTool("zipalign")
        args := []string{ zipalign, "-f", "4", signed, t.Name, }
        p := smart.Command("linux32", args...)
        return p.Run() //run32(zipalign, args...)
}

func (sdk *asdk) packResource(t *smart.Target) (e error) {
        os.Remove(t.Name) // remove the .jar or .apk first
        if e = os.MkdirAll(filepath.Dir(t.Name), 0755); e != nil { // make empty out dir
                return e
        }

        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        if e = sdk.createEmptyPackage(t); e != nil {
                return e
        }

        top := t.Var("top")
        if top == "" {
                smart.Fatal("empty top variable for %v", t)
        }

        args := []string{ "package", "-u",
                "-F", t.Name, // e.g. "out/_.unsigned", "foo.jar/_.jar"
                "-I", asdkGetPlatformFile("android.jar"),
        }

	for _, s := range sdk.staticLibs {
		args = append(args, "-I", s);
	}

        for _, d := range t.Dependees {
                ext := filepath.Ext(d.Name)
                switch {
                case ext == ".jar":
			args = append(args, "-I", d.Name)
                }
        }

        if t.Type == ".jar" || strings.HasSuffix(t.Name, ".jar") {
                args = append(args, "-x")
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

        if len(sources) == 0 {
                return
        }

        smart.Info("pack -o %v %v", t, strings.Join(sources, " "))
	aapt := asdkGetPlatformTool("aapt")
        p := smart.Command(aapt, args...)
        if true || !smart.IsVerbose() {
                p.Stdout, p.Stderr = nil, nil
        }
        if e = p.Run(); e != nil {
                return
        }
        return
}

func (sdk *asdk) packUnsigned(t *smart.Target) (e error) {
        var dex *smart.Target
        for _, d := range t.Dependees {
                if d.IsFile() && strings.HasSuffix(d.Name, ".dex") {
                        dex = d; break
                }
        }

        if dex == nil {
                return smart.NewErrorf("no dex for %v (%v)", t, t.Dependees)
        }

        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        if e = sdk.packResource(t); e != nil {
                return
        }

        smart.Info("pack -o %v %v (%v)\n", t, dex, t.Dependees)
        dexName := filepath.Base(dex.Name)
        apkName := filepath.Base(t.Name)
        aapt := asdkGetPlatformTool("aapt")
        p := smart.Command(aapt, "add", "-k", apkName, dexName)
        p.Stdout, p.Dir = nil, filepath.Dir(dex.Name)
        if e = p.Run(); e != nil {
                return
        }

        // add JNI libs
        if smart.T(sdk.proj.outLibs) == nil {
                return
        }

	projOutLib := filepath.Join(sdk.proj.out, "lib")

        var libs []string
        for _, lib := range smart.T(sdk.proj.outLibs).Dependees {
                if strings.HasPrefix(lib.Name, projOutLib) {
                        libs = append(libs, lib.Name[len(sdk.proj.out)+1:])
                } else if smart.IsVerbose() {
                        smart.Info("ignored: JNI library %v", lib)
                }
        }

        if len(libs) == 0 {
                //smart.Fatal("pack: no libs: %v, %v", sdk.proj.outLibs, sdk.proj.outLibs.Dependees)
		return
        }

        smart.Info("pack -o %v %v\n", t, strings.Join(libs, ", "))
        args := []string{ "-r", apkName, }
        args = append(args, libs...)
        p = smart.Command("zip", args...)
        p.Dir = sdk.proj.out
        //p.Stdout = nil
        if e = p.Run(); e != nil {
                return
        }

        return
}

func (sdk *asdk) packJar(t *smart.Target) (e error) {
	if t == smart.T(sdk.proj.proguard) {
		return nil
	}

	//smart.Info("pack: %v -> %v", t, t.Dependees);

        var classes *smart.Target
        sep := string(filepath.Separator)
        for _, d := range t.Dependees {
                if strings.HasPrefix(d.Name, "out"+sep) && strings.HasSuffix(d.Name, sep+"classes") {
                        classes = d; break
                }
        }

        defer func() {
                if e != nil {
                        os.Remove(t.Name)
                }
        }()

        if e = sdk.packResource(t); e != nil {
                return
        }

        var args []string
        var manifest string
        if manifest != "" {
                args = []string{ "-ufm" }
        } else {
                args = []string{ "-uf" }
        }

        //smart.Info("==== %v ====", t)
	//smart.Command("jar", "-tf", t.Name).Run()

        smart.Info("pack -o %v %v (%v)", t, classes, t.Dependees)
        args = append(args, t.Name, "-C", classes.Name, ".")
        p := smart.Command("jar", args...)
        if e = p.Run(); e != nil {
                return
        }

        //smart.Info("==== %v ==== (%v)", classes, t.Dependees)
	//smart.Command("find", classes.Name, "-type", "f", "-name", "*.class").Run()

        //smart.Info("==== %v ==== (%v)", t, classes)
	//smart.Command("jar", "-tf", t.Name).Run()
        return
}

type asdkCollector struct {
        sdk *asdk
        proj *asdkProject
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

func (coll *asdkCollector) addLibsDir(dir string) (t *smart.Target) {
        if smart.T(coll.proj.outLibs) != nil {
                smart.Fatal("more than one 'libs' subdirectories")
                return
        }

        if smart.T(coll.proj.unsigned) == nil {
                smart.Fatal("no unsigned target for %v", dir)
                return
        }

        coll.proj.outLibs = smart.T(coll.proj.unsigned).Dep(dir, smart.Dir).Name
        smart.Collect(dir, ".*", &asdkLibsCollector{ coll.proj })

        //smart.Info("libs: %v %v %v", coll.proj.outLibs, coll.proj.outLibs.Dependees, coll.proj.unsigned.Dependees)

        t = smart.T(coll.proj.outLibs)
        return
}

// add res and assets
func (coll *asdkCollector) addResDir(dir string) (t *smart.Target) {
        if coll.proj.pkg == "" {
                smart.Fatal("no package name")
        }

	classes := smart.T(coll.proj.outClasses)
        if classes == nil {
                smart.Fatal("no classes for %v", dir)
        }

	if coll.proj.outRes == "" {
		outDir := filepath.Dir(coll.proj.outClasses)
		coll.proj.outRes = filepath.Join(outDir, "res")
	}

	res := smart.T(coll.proj.outRes)
	res.Class = smart.IntermediateDir
        res.SetVar("top", filepath.Dir(dir))

        // Add R.java target
        pkg := strings.Replace(coll.proj.pkg, ".", smart.Separator, -1)
        rjava := filepath.Join(coll.proj.outRes, pkg, "R.java")

        r := classes.Dep(rjava, smart.IntermediateFile)
        r.Dep(res, smart.None)
        if r == nil {
                smart.Fatal("inter: %v:%v", rjava, coll.proj.outRes)
        }

        t = res.Dep(dir, smart.Dir)
        collRes := smart.NewDependeeCollector(t, smart.File)
        collRes.AddIgnorePattern(`[^~]*~`)
        collRes.AddPattern(`.*`)
        smart.Collect(dir, `.*`, collRes)
        return
}

func (coll *asdkCollector) addJarDir(dir string) (t *smart.Target) {
	classes := smart.T(coll.proj.outClasses)
        if classes == nil {
                smart.Fatal("no class output for %v", dir)
        }

        name := filepath.Join(coll.sdk.out, dir, "_.jar")
        t = classes.Dep(name, smart.IntermediateFile)
        t.Dep(dir, smart.Dir)
	t.SetVar("top", dir)

	res := smart.T(coll.proj.outRes)
	if res != nil {
		// the "res" also requires the "jar"
		res.Dep(t, t.Class)
	}

        jarColl := coll.sdk.NewCollector(t).(*asdkCollector)
        jarColl.proj.init(coll.sdk.out, dir)
        smart.Scan(jarColl, coll.sdk.top, dir)
        return
}

func (coll *asdkCollector) addAidl(dir string, info smart.FileInfo) (t *smart.Target) {
        //smart.Info("aidl: %v/%v", dir, info.Name())

        if smart.T(coll.proj.outClasses) == nil {
                smart.Fatal("no class output for %v/%v", dir, info.Name())
        }

	if !strings.HasSuffix(info.Name(), ".aidl") {
                smart.Fatal("not aidl: %v/%v", dir, info.Name())
	}

	jname := info.Name()
	jname = jname[0:len(jname)-5] + ".java"
        name := filepath.Join(coll.sdk.out, dir, jname)
        t = smart.T(coll.proj.outClasses).Dep(name, smart.IntermediateFile)
        t.Dep(filepath.Join(dir, info.Name()), smart.File)
	return
}

func (coll *asdkCollector) setProguard(dir string, info smart.FileInfo) (t *smart.Target) {
	//smart.Info("proguard: %s", info.Name());
	coll.proj.proguard = smart.T(coll.proj.target).Dep(filepath.Join(coll.sdk.out, "classes-processed.jar"), smart.File).Name
	//coll.proj.target.Dep(filepath.Join(coll.sdk.out, "classes-processed.map"), smart.File)
	t = smart.T(coll.proj.proguard).Dep(filepath.Join(dir, info.Name()), smart.File);
	return
}

func (coll *asdkCollector) Add(dir string, info smart.FileInfo) (t *smart.Target) {
        //smart.Info("add: %v", filepath.Join(dir, info.Name()))
        if info.IsDir() {
                return coll.addDir(dir, info)
        }
        return coll.addFile(dir, info)
}

func (coll *asdkCollector) addDir(dir string, info smart.FileInfo) (t *smart.Target) {
        dname := filepath.Join(dir, info.Name())

        switch {
        case info.Name() == "jni": fallthrough
        case info.Name() == "obj": fallthrough
        case info.Name() == "out":
                return nil

        case info.Name() == "src":
                smart.Collect(dname, `^.*?\.(java|aidl)$`, coll)

        case info.Name() == "libs":
                return coll.addLibsDir(dname)

        case info.Name() == "res": fallthrough
        case info.Name() == "assets":
                return coll.addResDir(dname)

        case strings.HasSuffix(info.Name(), ".jar"):
                return coll.addJarDir(dname)

        default:
                if smart.IsVerbose() {
                        smart.Info("ignored: %v (unknown dir)", filepath.Join(dir, info.Name()))
                }
        }

        return
}

func (coll *asdkCollector) addFile(dir string, info smart.FileInfo) (t *smart.Target) {
        //smart.Info("addFile: %v", filepath.Join(dir, info.Name()))
        dname := filepath.Join(dir, info.Name())

        if smart.T(coll.proj.target) == nil {
                smart.Fatal("no target for %v", dname)
        }

        if smart.T(coll.proj.outClasses) == nil {
                smart.Fatal("no classes for %v", dname)
        }

        switch {
        case info.Name() == "AndroidManifest.xml":
                t = smart.T(coll.proj.target).Dep(dname, smart.File)

	case info.Name() == "proguard.pro": fallthrough
	case info.Name() == "proguard.cfg":
		t = coll.setProguard(dir, info)

        case strings.HasSuffix(info.Name(), ".java"):
                t = smart.T(coll.proj.outClasses).Dep(dname, smart.File)

        case strings.HasSuffix(info.Name(), ".aidl"):
                t = coll.addAidl(dir, info)

        case strings.HasSuffix(info.Name(), ".apk"): fallthrough
	//case dir == "" && info.Name() == "proguard.cfg": fallthrough
	case dir == "" && info.Name() == "ant.properties": fallthrough
	case dir == "" && info.Name() == "project.properties": fallthrough
	case dir == "" && info.Name() == "build.xml":
		return

        default:
                if smart.IsVerbose() {
                        smart.Info("ignored: %v (unknown file)", dname)
                }
        }

        return
}

type asdkLibsCollector struct {
	proj *asdkProject
}

func (coll *asdkLibsCollector) Add(dir string, info smart.FileInfo) (t *smart.Target) {
        if info.IsDir() {
                return
        }

        //smart.Info("AddLib: %v/%v (%v)", dir, info.Name(), filepath.Dir(dir))
        if filepath.Dir(dir) != "libs" {
                smart.Info("ignore: %v/%v (not in libs)", dir, info.Name())
                return
        }

        dname := filepath.Join(dir, info.Name())
        libName := filepath.Join(coll.proj.out, "lib", filepath.Base(dir), info.Name())

        //smart.Info("lib: %v -> %v", dname, libName)

        // TODO: check dname: libXXX.so, or executibles

        lib := smart.T(coll.proj.outLibs).Dep(libName, smart.IntermediateFile)
        t = lib.Dep(dname, smart.File)
        return
}

func SetPlatformLevel(platformLevel uint) error {
        var versionCodes []string

        if n := int(platformLevel); n <= 0 || len(apiLevels) < n {
		return smart.NewErrorf("Platform %d not supported!", platformLevel)
        } else {
                versionCodes = apiLevels[n]
        }

        smart.Info("asdk: Using %s API (Android %s)", versionCodes[0], strings.Join(versionCodes[1:], ", "));

	platform := fmt.Sprintf("android-%d", platformLevel)
	s := filepath.Join(asdkRoot, "platforms", platform)
	if !smart.IsDir(s) {
		return smart.NewErrorf("Android %s not installed!", versionCodes[0])
	}

	asdkPlatform = platform
	return nil
}

func parseToolArgs(tool *asdk, name string, args[]string) []string {
	compats := make(smart.FlagArrayValue, 5)
        extras := make(smart.FlagArrayValue, 5)
        libs := make(smart.FlagArrayValue, 5)

        fs := flag.NewFlagSet(name, flag.ContinueOnError)
        fs.Var(&compats, "support", "specify extra libs of Android SDK")
        fs.Var(&extras, "extra", "specify extra libs of Android SDK")
        fs.Var(&libs, "lib", "specify static Jar libs for the project")

        if e := fs.Parse(args); e != nil {
                smart.Warn("%v", e)
        }

        for _, s := range extras {
                if s == "" { continue }

                s = filepath.Join(asdkRoot, "extras", s)
                if !smart.IsFile(s) {
                        smart.Fatal("error: extra \"%v\" is not file", s)
                }

                if strings.HasSuffix(s, ".jar") {
                        tool.staticLibs = append(tool.staticLibs, s)
                } else {
                        smart.Warn("unknown extra \"%v\"", s)
                }
        }

	for _, s := range compats {
                if s == "" { continue }
		j := fmt.Sprintf("android-support-%s.jar", s);
                j = filepath.Join(asdkRoot, "android-compatibility", s, j)
                if !smart.IsFile(j) {
                        smart.Fatal("error: unsupported compatibility \"%v\"", s)
                }
                tool.staticLibs = append(tool.staticLibs, j)
	}

        for _, s := range libs {
                if s == "" { continue }
                if !smart.IsFile(s) {
                        smart.Fatal("error: \"%v\" is not file", s)
                }
                tool.staticLibs = append(tool.staticLibs, s)
        }

        return fs.Args()
}

// Build builds a project
func build(args []string) (e error) {
        tool := New()
        parseToolArgs(tool, "build", args)
        e = smart.Build(tool)
	return
}

// Install invokes "adb install" command
func install(args []string) (e error) {
        tool := New()

        args = parseToolArgs(tool, "install", args)

        if e = smart.Build(tool); e != nil {
                return
        }

        goals := tool.Goals()

        // should be only one APK goal for a project
        if len(goals) != 1 {
                return smart.NewErrorf("wrong goals: %v", goals)
        }

        apk := goals[0]

        if !apk.IsFile() {
                return smart.NewErrorf("wrong APK type: %v", apk.Class)
        }

        if !smart.IsFile(apk.String()) {
                return smart.NewErrorf("APK not found: %v", apk)
        }

        for _, arg := range args {
                if arg[0] != '-' {
                        return smart.NewErrorf("unknown argument: %v", arg)
                }
        }

        args = append([]string{ "install", "-r" }, args...)
        args = append(args, apk.String())

        smart.Info("install %v..", apk)

        adb := asdkGetPlatformTool("adb")
        p := smart.Command(adb, args...)
        e  = p.Run()
        return
}

func proguard(args []string) (err error) {
        tool := New()
        args = parseToolArgs(tool, "install", args)

	if err = smart.ScanTargetGraph(tool); err != nil {
		return
	}

	cfgName := "proguard.cfg"

	if fi, e := os.Stat(cfgName); e == nil && fi != nil {
		os.Rename(cfgName, cfgName+"_")
	}

        cfg, err := os.Create(cfgName)
	fmt.Fprintf(cfg, "-injars %s\n", tool.proj.outClasses)
	if fi, e := os.Stat("libs"); e == nil && fi.IsDir() {
		fmt.Fprintf(cfg, "-injars %s\n", fi.Name())
	}

	fmt.Fprintf(cfg, "-libraryjars %v\n", asdkGetPlatformFile("android.jar"))
	/*
	for _, s := range tool.staticLibs {
		fmt.Fprintf(cfg, "-libraryjars %v\n", s)
	}
	 */

	fmt.Fprintf(cfg, "-outjars %s\n", filepath.Join(tool.out, "classes-processed.jar"))
	fmt.Fprintf(cfg, "-printmapping %s\n", filepath.Join(tool.out, "classes-processed.map"))
	fmt.Fprintf(cfg, `
-dontpreverify
-repackageclasses ''
-allowaccessmodification
-optimizations !code/simplification/arithmetic

-renamesourcefileattribute SourceFile
-keepattributes SourceFile,LineNumberTable
-keepattributes *Annotation*

-keep public class * extends android.app.Activity
-keep public class * extends android.app.Application
-keep public class * extends android.app.Service
-keep public class * extends android.content.BroadcastReceiver
-keep public class * extends android.content.ContentProvider

-keep public class * extends android.view.View {
    public <init>(android.content.Context);
    public <init>(android.content.Context, android.util.AttributeSet);
    public <init>(android.content.Context, android.util.AttributeSet, int);
    public void set*(...);
}

-keepclasseswithmembers class * {
    public <init>(android.content.Context, android.util.AttributeSet);
}

-keepclasseswithmembers class * {
    public <init>(android.content.Context, android.util.AttributeSet, int);
}

-keepclassmembers class * implements android.os.Parcelable {
    static android.os.Parcelable$Creator CREATOR;
}

-keepclassmembers class **.R$* {
  public static <fields>;
}

-keep public interface com.android.vending.licensing.ILicensingService

-dontnote com.android.vending.licensing.ILicensingService
-dontnote com.google.analytics.tracking.android.AdMobInfo

-dontwarn android.support.**
-dontwarn android.annotation.TargetApi

-keepclasseswithmembernames class * {
    native <methods>;
}

-keepclassmembers class * extends java.lang.Enum {
    public static **[] values();
    public static ** valueOf(java.lang.String);
}

-keepclassmembers class * implements java.io.Serializable {
    static final long serialVersionUID;
    static final java.io.ObjectStreamField[] serialPersistentFields;
    private void writeObject(java.io.ObjectOutputStream);
    private void readObject(java.io.ObjectInputStream);
    java.lang.Object writeReplace();
    java.lang.Object readResolve();
}

## Your application may contain more items that need to be preserved; 
## typically classes that are dynamically created using Class.forName:
# -keep public class mypackage.MyClass
# -keep public interface mypackage.MyInterface
# -keep public class * implements mypackage.MyInterface
`)
	cfg.Close()
	return
}

// Create invokes "android create" command
func create(args []string) (err error) {
        p := smart.Command(asdkGetTool("android"), args...)
        return p.Run()
}

func clean(args []string) error {
        tool := New()
        return smart.Clean(tool)
}

func level(args []string) error {
        smart.Info("Smart supported Android API Levels:");

        var maxCol = 0
        for _, codes := range apiLevels {
                if codes == nil { continue }
                if n := len(codes[0]); maxCol < n {
                        maxCol = n
                }
        }

        var req []int
        for n, s := range args {
                if j, e := fmt.Sscanf(s, "%d", &n); j == 1 && e == nil {
                        req = append(req, n)
                }
        }

        for l, codes := range apiLevels {
                if codes == nil { continue }
                s := strings.Repeat(" ", (maxCol - len(codes[0])))

                var v bool
                if len(req) == 0 {
                        v = true
                } else {
                        for _, i := range req {
                                if l == i { v = true; break }
                        }
                }

                if !v { continue }
                smart.Info("    %d\t%s %s (Android %s)", l, codes[0], s, strings.Join(codes[1:], ", "));
        }
        return nil
}

func logcat(args []string) error {
        var a []string

        fs := flag.NewFlagSet("logcat", flag.ContinueOnError)
        flagS := fs.String("s", "", "directs commands to the specified device")

        if e := fs.Parse(args); e != nil {
                return e
        }

        if *flagS != "" { a = append(a, "-s", *flagS) }

        args = fs.Args()

        var regs []*regexp.Regexp
        for _, s := range args {
                re, err := regexp.Compile(s)
                if err == nil {
                        regs = append(regs, re)
                }
        }

        a = append(a, "logcat")

        logsR, logsW := io.Pipe()
	adb := asdkGetPlatformTool("adb")
        p := exec.Command(adb, a...)
        p.Stdout = logsW
        go func() {
                in, log := bufio.NewReader(logsR), []byte{}
                for {
                        line, isPrefix, err := in.ReadLine()
                        log = append(log, line...)

                        for isPrefix && err == nil {
                                line, isPrefix, err = in.ReadLine()
                                log = append(log, line...)
                        }

                        for _, re := range regs {
                                if !re.Match(log) { continue }
                                fmt.Printf("%s\n", string(log))
                        }

                        log = []byte{}
                        if err != nil { break }
                }
                fmt.Printf("logcat finished\n")
        }()
        return p.Run()
}

func processPlatformLevelFlags(args []string) (a []string) {
	platformLevel := 10 // the default level

	for _, arg := range args {
		var level int
		if n, se := fmt.Sscanf(arg, "-%d", &level); n == 1 && se == nil {
			platformLevel = level
			continue
		}
		a = append(a, arg)
	}

	if e := SetPlatformLevel(uint(platformLevel)); e != nil {
                fmt.Fprintf(os.Stderr, "asdk: %v\n", e)
                os.Exit(-1)
	}

        return
}

func CommandLine(args []string) {
        if args = processPlatformLevelFlags(args); len(args) < 1 {
                fmt.Fprintf(os.Stderr, "asdk: no arguments\n")
                os.Exit(-1)
        }

        var commands = map[string] func(args []string) error {
                "build": build,
                "install": install,
                "create": create,
                "clean": clean,
                "level": level,
                "levels": level,
                "logcat": logcat,
		"proguard": proguard,
        }

        smart.CommandLine(commands, args)
}
