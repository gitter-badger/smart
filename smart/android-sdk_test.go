package smart

import (
        //"path/filepath"
        "testing"
        //"fmt"
        "os"
)

func testToolsetAndroidSDK(t *testing.T) {
        if l := len(modules); l != 0 { t.Errorf("expecting len(modules) for 0, but %v", l); return }
        if l := len(moduleOrderList); l != 0 { t.Errorf("expecting len(moduleOrderList) for 0, but %v", l); return }
        if l := len(moduleBuildList); l != 0 { t.Errorf("expecting len(moduleBuildList) for 0, but %v", l); return }
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }

        Build(computeTestRunParams())

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/res"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/res/org/smart/test/foo"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/res/org/smart/test/foo/R.java"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/Foo.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$attr.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$id.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$layout.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$string.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/foo_androidsdk_jar.jar"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/res"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/res/org/smart/test"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/res/org/smart/test/R.java"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/Foobar.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R$attr.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R$layout.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R$string.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes.dex"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/unsigned.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/signed.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/foo_androidsdk_use_jar.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/res"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/classes"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/classes.dex"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/unsigned.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/signed.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/foo_androidsdk_apk.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        os.RemoveAll("out")
}

func TestToolsetAndroidSDK(t *testing.T) {
        runToolsetTestCase(t, "android-sdk", testToolsetAndroidSDK)
}
