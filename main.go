//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package main

import (
        "github.com/duzy/smart/build"
        _ "github.com/duzy/smart/tools/gcc"
        _ "github.com/duzy/smart/tools/clang"
        _ "github.com/duzy/smart/tools/android/ndk"
        _ "github.com/duzy/smart/tools/android/sdk"
)

func main() {
        smart.Main()
}
