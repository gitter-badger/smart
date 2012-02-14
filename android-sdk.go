package smart

import (
        //"bytes"
        "os"
        "path/filepath"
        "strings"
        "fmt"
)

func init() {
        registerToolset("android-sdk", &_androidsdk{})
}

type _androidsdk struct {
}

func (sdk *_androidsdk) setupModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); panic(p.newError(0, "no module"))
        }

        var sources []string
        d := filepath.Dir(p.file)
        err := traverse(filepath.Join(d, "src"), func(dname string, fi os.FileInfo) bool {
                if strings.HasSuffix(dname, ".java") {
                        //fmt.Printf("java: %v\n", dname[len(d)+1:])
                        sources = append(sources, dname[len(d)+1:])
                }
                return true
        })

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
        var c *androidsdkCommand
        var hasRes, hasAssets, hasSrc bool
        if fi, err := os.Stat(filepath.Join(d, "src")); err == nil && fi.IsDir() { hasSrc = true }
        if fi, err := os.Stat(filepath.Join(d, "res")); err == nil && fi.IsDir() { hasRes = true }
        if fi, err := os.Stat(filepath.Join(d, "assets")); err == nil && fi.IsDir() { hasAssets = true }
        if hasRes || hasAssets {
                args := []string{
                        "package", "-m",
                        "-J", "out/res",
                        "-M", filepath.Join(d, "AndroidManifest.xml"),
                        "-I", "/home/duzy/open/android-sdk-linux_x86/platforms/android-10/android.jar",
                }

                // TODO: -P -G
                if hasRes { args = append(args, "-S", filepath.Join(d, "res")) }
                if hasAssets { args = append(args, "-A", filepath.Join(d, "assets")) }

                c = androidsdkNewCommand("aapt", args...)
                c.path = "~/open/android-sdk-linux_x86/platform-tools/aapt"
                c.mkdir = "out/res"
                a = newAction("out/res", c)
                a = newAction("R.java", &genR{}, a)
        }

        var ps []*action
        if a != nil { ps = append(ps, a) }
        if hasSrc {
                if sources := p.getModuleSources(); 0 < len(sources) {
                        for _, src := range sources { ps = append(ps, newAction(src, nil)) }
                }
                args := []string{
                        "-d", "out/classes",
                        "-sourcepath", filepath.Join(filepath.Dir(p.file), "src"),
                        "-cp", "/home/duzy/open/android-sdk-linux_x86/platforms/android-10/android.jar",
                }
                c = androidsdkNewCommand("javac", args...)
                c.mkdir = "out/classes"
                a = newAction("out/classes", c, ps...)
        }
        
        a = newAction("*.class", &genClasses{}, a)
        m.action.prequisites = append(m.action.prequisites, a)

        return true
}

type androidsdkCommand struct {
        execCommand
        args []string
}

func androidsdkNewCommand(name string, args ...string) *androidsdkCommand {
        return &androidsdkCommand{
                execCommand{ name: name, },
                args,
        }
}

func (c *androidsdkCommand) execute(target string, prequisites []string) bool {
        var args []string

        args = append([]string{}, c.args...)

        if c.name == "javac" {
                for _, p := range prequisites {
                        args = append(args, p)
                }
        }

        return c.run(target, args...)
}

type genR struct{
        r string // holds the R.java file path
}
func (ic *genR) target() string { return ic.r }
func (ic *genR) needsUpdate() bool { return ic.r == "" }
func (ic *genR) execute(target string, prequisites []string) bool {
        ic.r = ""
        e := traverse("out/res", func(dname string, fi os.FileInfo) bool {
                if ic.r != dname && fi.Name() == "R.java" {
                        fmt.Printf("smart: android `%v'\n", dname)
                        ic.r = dname; return false
                }
                return true
        })
        return ic.r != "" && e == nil
}

type genClasses struct{
        classes []string // holds the *.class file
}
func (ic *genClasses) target() string { return strings.Join(ic.classes, " ") }
func (ic *genClasses) needsUpdate() bool { return len(ic.classes) == 0 }
func (ic *genClasses) execute(target string, prequisites []string) bool {
        e := traverse("out/classes", func(dname string, fi os.FileInfo) bool {
                if strings.HasSuffix(fi.Name(), ".class") {
                        fmt.Printf("smart: android `%v'\n", dname)
                        ic.classes = append(ic.classes, dname)
                }
                return true
        })
        return e == nil && 0 < len(ic.classes)
}
