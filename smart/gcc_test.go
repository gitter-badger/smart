package smart

import (
        "testing"
        "os"
)

func TestToolsetGCC(t *testing.T) {
        if wd, e := os.Getwd(); e != nil { t.Errorf("Getwd: %v", e); return } else {
                fmt.Printf("TestToolsetGCC: Entering directory `%v'\n", tc)
                if e := os.Chdir("../test/gcc"); e != nil { t.Errorf("Chdir: %v", e); return }

                modules = map[string]*module{}
                moduleOrderList = []*module{}
                moduleBuildList = []pendedBuild{}

                // ...

                if e := os.Chdir(wd); e != nil { t.Errorf("Chdir: %v", e); return }
                fmt.Printf("TestToolsetGCC: Leaving directory `%v'\n", tc)
        }
}

