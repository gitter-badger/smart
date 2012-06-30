package smart

import (
        "bytes"
        "os"
        "os/exec"
        "testing"
)

func newTestAsdk() *asdk {
        tool := &asdk{}
        if top, e := os.Getwd(); e != nil {
                // TODO: error report
        } else {
                tool.SetTop(top)
        }
        targets = make(map[string]*Target)
        return tool
}

func TestBuildAPK(t *testing.T) {
        chdir(t, "+testdata/asdk/APK"); defer chdir(t, "-")
        checkf(t, "AndroidManifest.xml")
        checkd(t, "res")
        checkd(t, "res/layout")
        checkf(t, "res/layout/main.xml")
        checkd(t, "res/values")
        checkf(t, "res/values/strings.xml")
        checkd(t, "src")
        checkd(t, "src/org")
        checkd(t, "src/org/smart")
        checkd(t, "src/org/smart/test")
        checkd(t, "src/org/smart/test/ASDK")
        checkf(t, "src/org/smart/test/ASDK/Foo.java")

        os.RemoveAll("out")
        os.RemoveAll("org.smart.test.ASDK.apk")

        sdk := newTestAsdk()
        if e := Build(sdk); e != nil {
                t.Errorf("build: %v", e)
        }

        checkf(t, "org.smart.test.ASDK.apk")
        checkd(t, "out")
        checkf(t, "out/_.signed")
        checkf(t, "out/_.unsigned")
        checkd(t, "out/classes")
        checkf(t, "out/classes.dex")
        checkd(t, "out/classes/org")
        checkd(t, "out/classes/org/smart")
        checkd(t, "out/classes/org/smart/test")
        checkd(t, "out/classes/org/smart/test/ASDK")
        checkf(t, "out/classes/org/smart/test/ASDK/Foo.class")
        checkf(t, "out/classes/org/smart/test/ASDK/R.class")
        checkf(t, "out/classes/org/smart/test/ASDK/R$attr.class")
        checkf(t, "out/classes/org/smart/test/ASDK/R$layout.class")
        checkf(t, "out/classes/org/smart/test/ASDK/R$string.class")
        checkd(t, "out/res")
        checkd(t, "out/res/org")
        checkd(t, "out/res/org/smart")
        checkd(t, "out/res/org/smart/test")
        checkd(t, "out/res/org/smart/test/ASDK")
        checkf(t, "out/res/org/smart/test/ASDK/R.java")

        v := func(name string) {
                out := bytes.NewBuffer(nil)
                p := exec.Command("jarsigner", "-verify", name)
                p.Stdout = out
                p.Stderr = out
                if e := p.Run(); e != nil {
                        t.Errorf("jarsigner: %v", e)
                } else {
                        if "jar verified.\n" != string(out.Bytes()) {
                                t.Errorf("jarsigner: %v", string(out.Bytes()))
                        }
                }
        }
        v("org.smart.test.ASDK.apk")
        v("out/_.signed")

        os.RemoveAll("out")
        os.RemoveAll("org.smart.test.ASDK.apk")
}

func TestBuildUseJAR(t *testing.T) {
        chdir(t, "+testdata/asdk/use-jar"); defer chdir(t, "-")
        checkf(t, "AndroidManifest.xml")
        checkd(t, "res")
        checkd(t, "res/layout")
        checkf(t, "res/layout/main.xml")
        checkd(t, "res/values")
        checkf(t, "res/values/strings.xml")
        checkd(t, "src")
        checkd(t, "src/org")
        checkd(t, "src/org/smart")
        checkd(t, "src/org/smart/test")
        checkf(t, "src/org/smart/test/Foobar.java")

        os.RemoveAll("out")
        os.RemoveAll("org.smart.test.apk")

        sdk := newTestAsdk()
        if e := Build(sdk); e != nil {
                t.Errorf("build: %v", e)
        }
}
