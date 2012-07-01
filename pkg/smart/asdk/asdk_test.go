package asdk

import (
        ".." // smart
        "../t"
        "bytes"
        "os"
        "os/exec"
        "testing"
)

func newTestAsdk() *asdk {
        tool := New()
        if top, e := os.Getwd(); e != nil {
                // TODO: error report
        } else {
                tool.SetTop(top)
        }
        smart.ResetTargets()
        return tool
}

func TestBuildAPK(t *testing.T) {
        tt.Chdir(t, "+testdata/APK"); defer tt.Chdir(t, "-")
        tt.Checkf(t, "AndroidManifest.xml")
        tt.Checkd(t, "res")
        tt.Checkd(t, "res/layout")
        tt.Checkf(t, "res/layout/main.xml")
        tt.Checkd(t, "res/values")
        tt.Checkf(t, "res/values/strings.xml")
        tt.Checkd(t, "src")
        tt.Checkd(t, "src/org")
        tt.Checkd(t, "src/org/smart")
        tt.Checkd(t, "src/org/smart/test")
        tt.Checkd(t, "src/org/smart/test/ASDK")
        tt.Checkf(t, "src/org/smart/test/ASDK/Foo.java")

        os.RemoveAll("out")
        os.RemoveAll("org.smart.test.ASDK.apk")

        sdk := newTestAsdk()
        if e := smart.Build(sdk); e != nil {
                t.Errorf("build: %v", e)
        }

        tt.Checkf(t, "org.smart.test.ASDK.apk")
        tt.Checkd(t, "out")
        tt.Checkf(t, "out/_.signed")
        tt.Checkf(t, "out/_.unsigned")
        tt.Checkd(t, "out/classes")
        tt.Checkf(t, "out/classes.dex")
        tt.Checkd(t, "out/classes/org")
        tt.Checkd(t, "out/classes/org/smart")
        tt.Checkd(t, "out/classes/org/smart/test")
        tt.Checkd(t, "out/classes/org/smart/test/ASDK")
        tt.Checkf(t, "out/classes/org/smart/test/ASDK/Foo.class")
        tt.Checkf(t, "out/classes/org/smart/test/ASDK/R.class")
        tt.Checkf(t, "out/classes/org/smart/test/ASDK/R$attr.class")
        tt.Checkf(t, "out/classes/org/smart/test/ASDK/R$layout.class")
        tt.Checkf(t, "out/classes/org/smart/test/ASDK/R$string.class")
        tt.Checkd(t, "out/res")
        tt.Checkd(t, "out/res/org")
        tt.Checkd(t, "out/res/org/smart")
        tt.Checkd(t, "out/res/org/smart/test")
        tt.Checkd(t, "out/res/org/smart/test/ASDK")
        tt.Checkf(t, "out/res/org/smart/test/ASDK/R.java")

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
        tt.Chdir(t, "+testdata/use-jar"); defer tt.Chdir(t, "-")
        tt.Checkf(t, "AndroidManifest.xml")
        tt.Checkd(t, "res")
        tt.Checkd(t, "res/layout")
        tt.Checkf(t, "res/layout/main.xml")
        tt.Checkd(t, "res/values")
        tt.Checkf(t, "res/values/strings.xml")
        tt.Checkd(t, "src")
        tt.Checkd(t, "src/org")
        tt.Checkd(t, "src/org/smart")
        tt.Checkd(t, "src/org/smart/test")
        tt.Checkf(t, "src/org/smart/test/Foobar.java")

        os.RemoveAll("out")
        os.RemoveAll("org.smart.test.apk")

        sdk := newTestAsdk()
        if e := smart.Build(sdk); e != nil {
                t.Errorf("build: %v", e)
        }
}
