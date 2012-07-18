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

        removeOut := func() {
                os.RemoveAll("out")
                os.RemoveAll("org.smart.test.apk")
        }

        defer removeOut()

        sdk := newTestAsdk()

        removeOut()

        if e := smart.Build(sdk); e != nil {
                t.Errorf("build: %v", e)
        }

        tt.Checkd(t, "out")
        tt.Checkf(t, "out/_.signed")
        tt.Checkf(t, "out/_.unsigned")
        tt.Checkd(t, "out/classes")
        tt.Checkf(t, "out/classes.dex")
        tt.Checkd(t, "out/res")
        tt.Checkd(t, "out/foo.jar")
        tt.Checkf(t, "out/foo.jar/_.jar")
        tt.Checkd(t, "out/foo.jar/classes")
        tt.Checkd(t, "out/foo.jar/classes/org")
        tt.Checkd(t, "out/foo.jar/classes/org/smart")
        tt.Checkd(t, "out/foo.jar/classes/org/smart/test")
        tt.Checkd(t, "out/foo.jar/classes/org/smart/test/foo")
        tt.Checkf(t, "out/foo.jar/classes/org/smart/test/foo/Bar.class")
        tt.Checkf(t, "out/foo.jar/classes/org/smart/test/foo/Foo.class")
        tt.Checkf(t, "out/foo.jar/classes/org/smart/test/foo/R.class")
        tt.Checkf(t, "out/foo.jar/classes/org/smart/test/foo/R$attr.class")
        tt.Checkf(t, "out/foo.jar/classes/org/smart/test/foo/R$id.class")
        tt.Checkf(t, "out/foo.jar/classes/org/smart/test/foo/R$layout.class")
        tt.Checkf(t, "out/foo.jar/classes/org/smart/test/foo/R$string.class")
        tt.Checkd(t, "out/foo.jar/res")
        tt.Checkd(t, "out/foo.jar/res/org")
        tt.Checkd(t, "out/foo.jar/res/org/smart")
        tt.Checkd(t, "out/foo.jar/res/org/smart/test")
        tt.Checkd(t, "out/foo.jar/res/org/smart/test/foo")
        tt.Checkf(t, "out/foo.jar/res/org/smart/test/foo/R.java")
}

func TestBuildAPKRebuild(t *testing.T) {
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

        sdk := newTestAsdk()
        if e := smart.Build(sdk); e != nil {
                t.Errorf("build: %v", e)
        }

        outFiles := []string {
                "org.smart.test.ASDK.apk",
                "out/_.signed",
                "out/_.unsigned",
                "out/classes",
                "out/classes.dex",
                "out/classes/org/smart/test/ASDK/Foo.class",
                "out/classes/org/smart/test/ASDK/R.class",
                "out/classes/org/smart/test/ASDK/R$attr.class",
                "out/classes/org/smart/test/ASDK/R$layout.class",
                "out/classes/org/smart/test/ASDK/R$string.class",
                "out/res/org/smart/test/ASDK/R.java",
        }

        // because it's the second build, these must already existed
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

        fis := make(map[string]os.FileInfo, len(outFiles))
        for _, s := range outFiles {
                if fi, e := os.Stat(s); e != nil {
                        t.Errorf("%v", e)
                } else {
                        fis[s] = fi
                }
        }

        sdk = newTestAsdk()
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

        for _, s := range outFiles {
                if fi, e := os.Stat(s); e != nil {
                        t.Errorf("%v", e)
                } else {
                        fi0, ok := fis[s]
                        if !ok {
                                t.Errorf("fi for %v not matched", s)
                        }
                        if fi0.ModTime() != fi.ModTime() {
                                t.Errorf("ModTime: %v: %v != %v", s, fi0.ModTime(), fi.ModTime())
                        }
                }
        }

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