package smart

import (
        "testing"
        "os"
)

func testToolsetNdkBuild(t *testing.T) {
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }
        defer os.RemoveAll("out")

        Build(computeTestRunParams())
}

func TestToolsetNdkBuild(t *testing.T) {
        runToolsetTestCase(t, "ndkbuild", testToolsetNdkBuild)
}
