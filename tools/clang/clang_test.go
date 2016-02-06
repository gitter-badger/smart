//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"os"
        "testing"
        //. "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testToolsetClang(t *testing.T) {
}

func TestToolsetClang(t *testing.T) {
        RunToolsetTestCase(t, "../..", "gcc", testToolsetClang)
}
