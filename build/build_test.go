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
        "io/ioutil"
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
        if ctx.t != nil { t.Errorf("ctx.t: %v", ctx.t) }
        if ctx.m != nil { t.Errorf("ctx.m: %v", ctx.m) }
        if n, x := len(ctx.g.rules), 1; n != x { t.Errorf("wrong rules: %v", ctx.g.rules) } else {
                if r, ok := ctx.g.rules["foo"]; !ok && r == nil { t.Errorf("'all' not defined") } else {
                        if k, x := r.node.kind, nodeRulePhony; k != x { t.Errorf("%v != %v", k, x) }
                        if n, x := len(r.node.children), 3; n != x { t.Errorf("children %d != %d", n, x) }
                        if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                                if s, x := r.targets[0], "foo"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                        }
                        if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                        if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                                ctx.Set("@", stringitem("xxxxx"))
                                if c, ok := r.recipes[0].(*node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                        if k, x := c.kind, nodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                        if s, x := c.str(), `@echo "rule 'foo' is also called along with module 'foo'" $(info 4: $@)`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                        if s, x := c.Expand(ctx), `@echo "rule 'foo' is also called along with module 'foo'" `; s != x { t.Errorf("recipes[1]: '%v' != '%v'", s, x) }
                                }
                                ctx.Set("@", stringitem(""))
                        }
                        if c, ok := r.c.(*phonyTargetUpdater); !ok { t.Errorf("wrong type %v", c) }
                }
        }

        os.Remove("bar.txt")
        os.Remove("foo.txt")
        os.Remove("foobar.txt")
        Update(ctx)
        if s, x := info.String(), fmt.Sprintf(`4: xxxxx
4: foo
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

func TestBuildUseTemplate(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestBuildUseTemplate", `
################
$(template test)

foobar.txt: foo.txt bar.txt
	@echo "$(me.a): $^" > $@ $(info 0: $@,$<,$^,$?)
foo.txt:
	@touch $@ $(info 1: $@ $(me.a))
bar.txt:
	@touch $@ $(info 2: $@ $(me.a))
	@echo $@ >> $@ $(info 3: $@)

$(commit)

#############
$(module foo, test)

me.a := aaa

$(commit)

######
foo:!:
	@echo "rule 'foo' is also called along with module 'foo'" $(info 4: $@)
`);     if err != nil { t.Errorf("parse error:", err) }
        if ctx.t != nil { t.Errorf("ctx.t: %v", ctx.t) }
        if ctx.m != nil { t.Errorf("ctx.m: %v", ctx.m) }
        if n, x := len(ctx.g.rules), 1; n != x { t.Errorf("wrong rules: %v", ctx.g.rules) } else {
                if r, ok := ctx.g.rules["foo"]; !ok || r == nil { t.Errorf("'foo' not defined") } else {
                        if k, x := r.node.kind, nodeRulePhony; k != x { t.Errorf("%v != %v", k, x) }
                        if n, x := len(r.node.children), 3; n != x { t.Errorf("children %d != %d", n, x) }
                        if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                                if s, x := r.targets[0], "foo"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                        }
                        if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                        if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                                ctx.Set("@", stringitem("xxxxx"))
                                if c, ok := r.recipes[0].(*node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                        if k, x := c.kind, nodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                        if s, x := c.str(), `@echo "rule 'foo' is also called along with module 'foo'" $(info 4: $@)`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                        if s, x := c.Expand(ctx), `@echo "rule 'foo' is also called along with module 'foo'" `; s != x { t.Errorf("recipes[1]: '%v' != '%v'", s, x) }
                                }
                                ctx.Set("@", stringitem(""))
                        }
                        if c, ok := r.c.(*phonyTargetUpdater); !ok { t.Errorf("wrong type %v", c) }
                }
        }
        if n, x := len(ctx.modules), 1; n != x { t.Errorf("wrong modules: %v", ctx.modules) } else {
                if m, ok := ctx.modules["foo"]; !ok || m == nil { t.Errorf("foo not defined: %v", ctx.modules) } else {
                }
        }

        os.Remove("bar.txt")
        os.Remove("foo.txt")
        os.Remove("foobar.txt")
        Update(ctx)
        if s, x := info.String(), fmt.Sprintf(`4: xxxxx
4: foo
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

func TestBuildUseTemplate2(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestBuildUseTemplate2", `
all: foo bar

$(template test)

$(me.name).txt:
	@touch $@ $(info 1: $@ $(me.a))
	@echo $@ >> $@ $(info 2: $@)

$(commit)

$(module foo, test)
me.a := aaa1
$(commit)

$(module bar, test)
me.a := aaa2
$(commit)

foo:!:
	@echo "rule 'foo' is also called along with module 'foo'" $(info 3: $@)
bar:!:
	@echo "rule 'foo' is also called along with module 'foo'" $(info 4: $@)
`);     if err != nil { t.Errorf("parse error:", err) }
        if s, x := ctx.g.goal, "all"; s != x { t.Errorf("%v != %v", s, x) }
        if n, x := len(ctx.g.rules), 3; n != x { t.Errorf("wrong rules: %v", ctx.g.rules) } else {
                if r, ok := ctx.g.rules["all"]; !ok && r == nil { t.Errorf("'all' not defined") } else {
                        // TODO: ...
                }
                if r, ok := ctx.g.rules["foo"]; !ok && r == nil { t.Errorf("'foo' not defined") } else {
                        // TODO: ...
                }
                if r, ok := ctx.g.rules["bar"]; !ok && r == nil { t.Errorf("'bar' not defined") } else {
                        // TODO: ...
                }
        }
        if n, x := len(ctx.modules), 2; n != x { t.Errorf("wrong modules: %v", ctx.modules) } else {
                if m, ok := ctx.modules["foo"]; !ok || m == nil { t.Errorf("foo not defined: %v", ctx.modules) } else {
                        if s, x := m.goal, "foo.txt"; s != x { t.Errorf("%v != %v", s, x) }
                        if n, x := len(m.rules), 1; n != x { t.Errorf("wrong rules: %v", m.rules) } else {
                                if r, ok := m.rules["foo.txt"]; !ok && r == nil { t.Errorf("'foo.txt' not defined") } else {
                                        // TODO: ...
                                }
                        }
                }
                if m, ok := ctx.modules["bar"]; !ok || m == nil { t.Errorf("foo not defined: %v", ctx.modules) } else {
                        if s, x := m.goal, "bar.txt"; s != x { t.Errorf("%v != %v", s, x) }
                        if n, x := len(m.rules), 1; n != x { t.Errorf("wrong rules: %v", m.rules) } else {
                                if r, ok := m.rules["bar.txt"]; !ok && r == nil { t.Errorf("'foo.txt' not defined") } else {
                                        // TODO: ...
                                }
                        }
                }
        }
        
        info.Reset()
        os.Remove("bar.txt")
        os.Remove("foo.txt")
        Update(ctx)
        if s, x := info.String(), fmt.Sprintf(`3: foo
1: foo.txt aaa1
2: foo.txt
4: bar
1: bar.txt aaa2
2: bar.txt
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("%v", e) } else {
                if b, e := ioutil.ReadFile("bar.txt"); e != nil { t.Errorf("%v", e) } else {
                        if s, x := string(b), "bar.txt\n"; s != x { t.Errorf("%v", s) }
                }
        }
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("%v", e) } else {
                if b, e := ioutil.ReadFile("foo.txt"); e != nil { t.Errorf("%v", e) } else {
                        if s, x := string(b), "foo.txt\n"; s != x { t.Errorf("%v", s) }
                }
        }

        info.Reset()
        os.Remove("bar.txt")
        os.Remove("foo.txt")
        Update(ctx, "foo")
        if s, x := info.String(), fmt.Sprintf(`3: foo
1: foo.txt aaa1
2: foo.txt
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("%v", e) } else {
        }

        info.Reset()
        os.Remove("bar.txt")
        os.Remove("foo.txt")
        Update(ctx, "bar")
        if s, x := info.String(), fmt.Sprintf(`4: bar
1: bar.txt aaa2
2: bar.txt
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("%v", e) } else {
        }

        os.Remove("bar.txt")
        os.Remove("foo.txt")
}

func TestBuildTemplateHooks(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        hooksMap["test"] = HookTable{
                "some": func(ctx *Context, args Items) (res Items) {
                        res = append(res, stringitem("some"))
                        res = append(res, args...)
                        return
                },
        }

        ctx, err := newTestContext("TestBuildTemplateHooks", `
$(template test)
$(info $(test:some $(me.a),.,.,$(me.a)))
$(post)
$(info $(test:some $(me.a),.,.,$(me.a)))
$(commit)

$(module foo, test)
me.a := aaa
$(commit)
`);     if err != nil { t.Errorf("parse error:", err) }
        if s, x := ctx.g.goal, ""; s != x { t.Errorf("%v != %v", s, x) }
        Update(ctx, "foo") // invoke the "foo" module
        if s, x := info.String(), "some . .\nsome aaa . . aaa\n"; s != x { t.Errorf("'%s' != '%s'", s, x) }

        delete(hooksMap, "test")
}

/*
// intercommand represents a intermdiate action command
type intercommand interface {
        Command
        Targets(prerequisites []*Action) (names []string, needsUpdate bool)
}

func ComputeInterTargets(d, sre string, prerequisites []*Action) (targets []string, outdates int, outdateMap map[int]int) {
        re := regexp.MustCompile(sre)
        outdateMap = map[int]int{}
        traverse(d, func(fn string, fi os.FileInfo) bool {
                if !re.MatchString(fn) { return true }
                i := len(targets)
                outdateMap[i] = 0
                targets = append(targets, fn)
                for _, p := range prerequisites {
                        if pc, ok := p.Command.(intercommand); ok {
                                if _, needsUpdate := pc.Targets(p.Prerequisites); needsUpdate {
                                        outdateMap[i]++
                                }
                        } else {
                                for _, t := range p.Targets {
                                        if pfi, _ := os.Stat(t); pfi == nil {
                                                errorf("`%v' not found", t)
                                        } else if fi.ModTime().Before(pfi.ModTime()) {
                                                outdateMap[i]++
                                        }
                                }
                        }
                }
                outdates += outdateMap[i]
                return true
        })
        return
}

func ComputeKnownInterTargets(targets []string, prerequisites []*Action) (outdates int, outdateMap map[int]int) {
        outdateMap = map[int]int{}
        for i, fn := range targets {
                fi, e := os.Stat(fn)
                if e != nil || fi == nil { // Target not existed.
                        outdateMap[i]++
                        continue
                }
                for _, p := range prerequisites {
                        if pc, ok := p.Command.(intercommand); ok {
                                if _, needsUpdate := pc.Targets(p.Prerequisites); needsUpdate {
                                        outdateMap[i]++
                                }
                        } else {
                                for _, t := range p.Targets {
                                        if pfi, _ := os.Stat(t); pfi == nil {
                                                errorf("`%v' not found", t)
                                        } else if fi.ModTime().Before(pfi.ModTime()) {
                                                outdateMap[i]++
                                        }
                                }
                        }
                }
                outdates += outdateMap[i]
        }
        return
}

// Action performs a command for updating targets
type Action struct {
        Targets []string
        Prerequisites []*Action
        Command Command
}

func (a *Action) update() (updated bool, updatedTargets []string) {
        var targets []string
        var targetsNeedUpdate bool
        var isIntercommand bool
        if a.Command != nil {
                if c, ok := a.Command.(intercommand); ok {
                        targets, targetsNeedUpdate = c.Targets(a.Prerequisites)
                        isIntercommand = true
                }
        }

        if !isIntercommand {
                //fmt.Printf("targets: %v\n", a.targets)
                targets = append(targets, a.Targets...)
        }

        var missingTargets, outdatedTargets []int
        var fis []os.FileInfo
        for n, s := range targets {
                if i, _ := os.Stat(s); i != nil {
                        fis = append(fis, i)
                } else {
                        fis = append(fis, nil)
                        missingTargets = append(missingTargets, n)
                }
        }

        if len(fis) != len(targets) {
                panic("internal unmatched arrays") //errorf(-1, "internal")
        }

        updatedPreNum := 0
        prerequisites := []string{}
        for _, p := range a.Prerequisites {
                if u, pres := p.update(); u {
                        prerequisites = append(prerequisites, pres...)
                        updatedPreNum++
                } else if pc, ok := p.Command.(intercommand); ok {
                        pres, nu := pc.Targets(p.Prerequisites)
                        if nu { errorf("requiring updating %v for %v", pres, targets) }
                        prerequisites = append(prerequisites, pres...)
                } else {
                        prerequisites = append(prerequisites, p.Targets...)
                        for _, pt := range p.Targets {
                                if fi, err := os.Stat(pt); err != nil {
                                        errorf("`%v' not found", pt)
                                } else {
                                        for n, i := range fis {
                                                if i != nil && i.ModTime().Before(fi.ModTime()) {
                                                        outdatedTargets = append(outdatedTargets, n)
                                                }
                                        }
                                }
                        }
                }
        }

        if a.Command == nil {
                for n, i := range fis {
                        if i == nil {
                                errorf("`%s' not found", targets[n])
                        }
                }
                return
        }

        if 0 < updatedPreNum || targetsNeedUpdate {
                updated, updatedTargets = a.execute(targets, fis, prerequisites)
        } else {
                var rr []int
                var request []string
                var requestfis []os.FileInfo

                rr = append(rr, missingTargets...)
                rr = append(rr, outdatedTargets...)
                sort.Ints(rr)

                for n := range rr {
                        if n == 0 || rr[n-1] != rr[n] {
                                request = append(request, targets[rr[n]])
                                requestfis = append(requestfis, fis[rr[n]])
                        }
                }

                //fmt.Printf("targets: %v, %v, %v, %v\n", targets, request, len(a.prerequisites), prerequisites)
                if 0 < len(request) {
                        updated, updatedTargets = a.execute(request, requestfis, prerequisites)
                }
        }

        return
}

func (a *Action) execute(targets []string, tarfis []os.FileInfo, prerequisites []string) (updated bool, updatedTargets []string) {
        if updated = a.Command.Execute(targets, prerequisites); updated {
                var targetsNeedUpdate bool
                if c, ok := a.Command.(intercommand); ok {
                        updatedTargets, targetsNeedUpdate = c.Targets(a.Prerequisites)
                        updated = !targetsNeedUpdate
                } else {
                        for _, t := range a.Targets {
                                if fi, e := os.Stat(t); e != nil || fi == nil {
                                        errorf("`%s' was not built", t)
                                } else {
                                        updatedTargets = append(updatedTargets, t)
                                }
                        }
                }
        }
        return
}

func (a *Action) clean() {
        errorf("TODO: clean `%v'\n", a.Targets)
}

func newAction(target string, c Command, pre ...*Action) *Action {
        a := &Action{
                Command: c,
                Targets: []string{ target },
                Prerequisites: pre,
        }
        return a
}

func NewAction(target string, c Command, pre ...*Action) *Action {
        return newAction(target, c, pre...)
}

func NewInterAction(target string, c intercommand, pre ...*Action) *Action {
        return newAction(target, c, pre...)
}

func CreateSourceTransformActions(sources []string, namecommand func(src string) (string, Command)) []*Action {
        var inters []*Action
        if namecommand == nil {
                errorf("can't draw source rules (%v)", namecommand)
        }

        for _, src := range sources {
                aname, c := namecommand(src)
                if aname == "" { continue }
                if aname == src {
                        errorf("no intermediate name for `%v'", src)
                }

                if c == nil {
                        errorf("no command for `%v'", src)
                }

                asrc := newAction(src, nil)
                a := newAction(aname, c, asrc)
                inters = append(inters, a)
        }
        return inters
}

*/
