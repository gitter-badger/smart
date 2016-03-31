//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        "fmt"
        "bytes"
        "time"
        "testing"
)

func TestTraverse(t *testing.T) {
        m := map[string]bool{}
        err := traverse("../data", func(fn string, fi os.FileInfo) bool {
                m[fi.Name()] = true
                return true
        })
        if err != nil      { t.Errorf("error: %v\n", err) }
        //if !m["main.go"] { t.Error("main.go not found") }
        if !m["keystore"]  { t.Error("keystore not found") }
        if !m["keypass"]   { t.Error("keypass not found") }
        if !m["storepass"] { t.Error("storepass not found") }
}

func TestBuildRules(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestBuildRules", `
all: foo.txt bar.txt
foo.txt:; @touch $@ $(info noop: $@)
bar.txt:
	@touch $@ $(info noop: $@.1)
	@echo $@ >> $@ $(info noop: $@.2)
foobar.txt: foo.txt
	@echo $^ > $@ $(info $@,$<,$^,$?)
`);     if err != nil { t.Errorf("parse error:", err) }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        Update(ctx)
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        Update(ctx, "all")
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        Update(ctx, "foo.txt")
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }
        if fi, e := os.Stat("bar.txt"); fi != nil || e == nil { t.Errorf("TestBuildRules: bar.txt should not exists!") }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        Update(ctx, "bar.txt")
        if fi, e := os.Stat("foo.txt"); fi != nil || e == nil { t.Errorf("TestBuildRules: foo.txt should not exists!") }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }

        if s, x := info.String(), fmt.Sprintf(`noop: foo.txt
noop: bar.txt.1
noop: bar.txt.2
noop: foo.txt
noop: bar.txt.1
noop: bar.txt.2
noop: foo.txt
noop: bar.txt.1
noop: bar.txt.2
`); s != x { t.Errorf("'%s' != '%s'", s, x) }

        Update(ctx, "foobar.txt")
        if fiFoo, e := os.Stat("foo.txt"); fiFoo == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else {
                var t1Foo, t1Foobar, t2Foo, t2Foobar time.Time
                if fi, e := os.Stat("foo.txt"); fiFoo == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else { t1Foo = fi.ModTime() }
                if fi, e := os.Stat("foobar.txt"); fiFoo == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else { t1Foobar = fi.ModTime() }

                time.Sleep(1 * time.Second)
                tt := time.Now() // fiFoo.ModTime().Add(1 * time.Second)
                if e := os.Chtimes("foo.txt", tt, tt); e != nil { t.Errorf("TestBuildRules: %s", e) }

                Update(ctx, "foobar.txt")
                if fi, e := os.Stat("foo.txt"); fiFoo == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else { t2Foo = fi.ModTime() }
                if fi, e := os.Stat("foobar.txt"); fiFoo == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else { t2Foobar = fi.ModTime() }
                if !t1Foo.Before(t2Foo) { t.Errorf("!(%v < %v)", t1Foo, t2Foo) }
                if !t2Foobar.After(t1Foobar) { t.Errorf("!(%v < %v)", t1Foobar, t2Foobar) }
                if !t1Foobar.Before(t2Foobar) { t.Errorf("!(%v < %v)", t1Foobar, t2Foobar) }
        }
        if s, x := info.String(), fmt.Sprintf(`noop: foo.txt
noop: bar.txt.1
noop: bar.txt.2
noop: foo.txt
noop: bar.txt.1
noop: bar.txt.2
noop: foo.txt
noop: bar.txt.1
noop: bar.txt.2
noop: foo.txt
foobar.txt foo.txt foo.txt foo.txt
foobar.txt foo.txt foo.txt foo.txt
`); s != x { t.Errorf("'%s' != '%s'", s, x) }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        os.Remove("foobar.txt")
}

func TestBuildRuleTargetChecker(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestBuildRuleTargetChecker", `
foo:!: foobar
	@echo -n foo > $@.txt
foo:?:
	@test -f $@.txt && test "$$(cat $@.txt)" = "foo"
foobar:!:
`);     if err != nil { t.Errorf("parse error:", err) }
        if ctx == nil { t.Errorf("nil context") } else {
                if r, ok := ctx.g.rules["foo"]; !ok { t.Errorf("rule 'foo' not defined") } else {
                        if n := len(r.targets); n != 1 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) } else {
                                if g := ctx.g.getGoalRule(); g != r.targets[0] { t.Errorf("wrong goal rule: %v", g) }
                        }
                        if n := len(r.prerequisites); n != 0 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.recipes); n != 1 { t.Errorf("incorrect number of recipes: %v %v", n, r.recipes) }
                        if k, x := r.node.kind, nodeRuleChecker; k != x { t.Errorf("%v != %v", k, x) }
                }
                {
                        os.Remove("foo.txt")
                        Update(ctx)
                        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRuleTargetChecker: %v", e) }
                }
                {
                        os.Remove("foo.txt")
                        Update(ctx, "foo")
                        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRuleTargetChecker: %v", e) }
                }
        }

        ctx, err = newTestContext("TestBuildRuleTargetChecker", `
foo:!: foobar
	@echo -n foo > $@.txt
foo:?:
	@test -f $@.txt && test "$$(cat $@.txt)" = "foo"
foobar:!: ; @echo $@ $(info $@)
`);     if err != nil { t.Errorf("parse error:", err) }
        if ctx == nil { t.Errorf("nil context") } else {
                {
                        os.Remove("foo.txt")
                        Update(ctx)
                        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRuleTargetChecker: %v", e) }
                }
                {
                        os.Remove("foo.txt")
                        Update(ctx, "foo")
                        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRuleTargetChecker: %v", e) }
                }
        }

        os.Remove("foo.txt")
}

func TestBuildModules(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestBuildModules", `
$(module foo)

me.a := aaa

foobar.txt: foo.txt bar.txt
	@echo "$(me.a): $^" > $@ $(info 0: $@,$<,$^,$?)
foo.txt:
	@touch $@ $(info 1: $@ $(me.a))
bar.txt:
	@touch $@ $(info 2: $@ $(me.a))
	@echo $@ >> $@ $(info 3: $@)
$(commit)

foo:!:
	@echo "rule 'foo' is also called along with module 'foo'" $(info 4: $@)
`);     if err != nil { t.Errorf("parse error:", err) }

        os.Remove("bar.txt")
        os.Remove("foo.txt")
        os.Remove("foobar.txt")
        Update(ctx)
        if s, x := info.String(), fmt.Sprintf(`4: foo
1: foo.txt aaa
2: bar.txt aaa
3: bar.txt
0: foobar.txt foo.txt foo.txt bar.txt foo.txt bar.txt
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else {
                
        }
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else {
        }
        if fi, e := os.Stat("foobar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else {
        }

        info.Reset()
        os.Remove("bar.txt")
        os.Remove("foo.txt")
        os.Remove("foobar.txt")
        Update(ctx, "foo")
        if s, x := info.String(), fmt.Sprintf(`4: foo
1: foo.txt aaa
2: bar.txt aaa
3: bar.txt
0: foobar.txt foo.txt foo.txt bar.txt foo.txt bar.txt
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else {
        }
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else {
        }
        if fi, e := os.Stat("foobar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) } else {
        }

        os.Remove("bar.txt")
        os.Remove("foo.txt")
        os.Remove("foobar.txt")
}
