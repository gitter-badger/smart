package smart

import (
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

        m.action = newAction(m.name+".apk")

        d := filepath.Dir(p.file)

        var a *action
        var hasRes, hasAssets bool
        if fi, err := os.Stat(filepath.Join(d, "res")); err == nil && fi.IsDir() { hasRes = true }
        if fi, err := os.Stat(filepath.Join(d, "assets")); err == nil && fi.IsDir() { hasAssets = true }
        if hasRes || hasAssets {
                args := []string{
                        "-J", "out/res",
                        "-M", filepath.Join(d, "AndroidManifest.xml"),
                        "-I", "~/open/android-sdk-linux_x86/platforms/android-10/android.jar",
                }

                // -P -G
                if hasRes { args = append(args, "-S", filepath.Join(d, "res")) }
                if hasAssets { args = append(args, "-A", filepath.Join(d, "assets")) }

                a = newAction("out/res")
                a.command = androidsdkNewCommand("mkdir -p out/res && ~/open/android-sdk-linux_x86/platform-tools/aapt", args...)

                a = newAction("out/res.list", a)
                a.command = androidsdkNewCommand("find", "out/res", "-type", "f", "-name", "R.java")
        }

        if a == nil {
                a = newAction("out/classes.list")
        } else {
                a = newAction("out/classes.list", a)
        }
        a.command = androidsdkNewCommand("find", "out/classes", "-type", "f", "-name", "\"*.class\"")
        m.action.prequisites = append(m.action.prequisites, a)
        sources := p.getModuleSources()
        for _, src := range sources {
                asrc := newAction(src)
                a.prequisites = append(a.prequisites, asrc)
        }

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

        args = append([]string{ "-o", target, }, c.args...)

        for _, p := range prequisites {
                args = append(args, p)
        }
        return c.run(target, args...)
}
