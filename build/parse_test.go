//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "testing"
        "strings"
        "bytes"
        "fmt"
)

func newTestLex(file, s string) (l *lex) {
        l = &lex{ parseBuffer:&parseBuffer{ scope:file, s:[]byte(s) }, pos:0, }
        return
}

func newTestContext(file, s string) (p *context) {
        p, _ = newContext(file, []byte(s), nil)
        return
}

func TestLexComments(t *testing.T) {
        l := newTestLex("TestLexComments", `
# this is a comment
# this is the second line of the comment

# this is another comment
# this is the second line of the other comment
# this is the third line of the other comment

# more...

foo = foo # comment 1
$(info info) # comment 2
bar = bar# comment 3
foobar=foobar#comment 4
foobaz=blah# comment 5 
foobac=# comment 6	`)
        l.parse()

        if len(l.nodes) != 15 { t.Errorf("expecting 15 nodes but got %v", len(l.nodes)) }

        countComments := 0
        for _, n := range l.nodes { if n.kind == nodeComment { countComments++ } }
        if v := 9; countComments != v { t.Errorf("expecting %v comments but got %v", v, countComments) }

        var c *node
        checkNode := func(n int, k nodeType, s string) (okay bool) {
                okay = true
                if len(l.nodes) < n+1 { t.Errorf("expecting at least %v nodes but got %v", n+1, len(l.nodes)); okay = false }
                if c = l.nodes[n]; c.kind != k { t.Errorf("expecting node %v as %v but got %v(%v)", n, k, c.kind, l.str(c)); okay = false }
                if l.str(c) != s { t.Errorf("expecting node %v as %v(%v) but got %v(%v)", n, k, s, c.kind, l.str(c)); okay = false }
                return
        }

        if !checkNode(0, nodeComment, `# this is a comment
# this is the second line of the comment`) { return }
        if !checkNode(1, nodeComment, `# this is another comment
# this is the second line of the other comment
# this is the third line of the other comment`) { return }
        if !checkNode(2, nodeComment, `# more...`) { return }
        if !checkNode(3, nodeDefineDeferred, `=`) { return }
        if !checkNode(4, nodeComment, `# comment 1`) { return }
        if !checkNode(5, nodeImmediateText, `$(info info) `) { return }
        if !checkNode(6, nodeComment, `# comment 2`) { return }
        if !checkNode(7, nodeDefineDeferred, `=`) { return }
        if !checkNode(8, nodeComment, `# comment 3`) { return }
        if !checkNode(9, nodeDefineDeferred, `=`) { return }
        if !checkNode(10, nodeComment, `#comment 4`) { return }
        if !checkNode(11, nodeDefineDeferred, `=`) { return }
        if !checkNode(12, nodeComment, `# comment 5 `) { return }
        if !checkNode(13, nodeDefineDeferred, `=`) { return }
        if !checkNode(14, nodeComment, `# comment 6	`) { return }
}

func TestLexAssigns(t *testing.T) {
        l := newTestLex("TestLexAssigns", `
a = a
b= b
c=c
d       =           d
foo := $(a) \
 $b\
 ${c}\

bar = $(foo) \
$(a) \
 $b $c

f_$a_$b_$c_1 = f_a_b_c
f_$a_$b_$c_2 = f_$a_$b_$c

a += a
a += a
b ?= b
cc ::= cc
n != n

f_$a_$b_$c_3 := f_$($a)_$($($b))_$(${$($c)})
`)
        l.parse()

        if ex := 14; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }

        var (
                countDeferredDefines = 0
                countQuestionedDefines = 0
                countSingleColonedDefines = 0
                countDoubleColonedDefines = 0
                countAppendDefines = 0
                countNotDefines = 0
        )
        for _, n := range l.nodes {
                switch n.kind {
                case nodeDefineDeferred:        countDeferredDefines++
                case nodeDefineQuestioned:      countQuestionedDefines++
                case nodeDefineSingleColoned:   countSingleColonedDefines++
                case nodeDefineDoubleColoned:   countDoubleColonedDefines++
                case nodeDefineAppend:          countAppendDefines++
                case nodeDefineNot:             countNotDefines++
                }
        }
        if ex := 7; countDeferredDefines != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDeferred,      countDeferredDefines) }
        if ex := 1; countQuestionedDefines != ex        { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineQuestioned,    countQuestionedDefines) }
        if ex := 2; countSingleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineSingleColoned, countSingleColonedDefines) }
        if ex := 1; countDoubleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDoubleColoned, countDoubleColonedDefines) }
        if ex := 2; countAppendDefines != ex            { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineAppend,        countAppendDefines) }
        if ex := 1; countNotDefines != ex               { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineNot,           countNotDefines) }
        if n := countDeferredDefines+countQuestionedDefines+countSingleColonedDefines+countDoubleColonedDefines+countAppendDefines+countNotDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        var (
                c *node
                i int
        )
        checkNode := func(c *node, k nodeType, cc int, s string, cs ...string) {
                if c.kind != k { t.Errorf("%v: expecting kind %v but got %v", i, k, c.kind) }
                if ss := l.str(c); ss != s { t.Errorf("%v: expecting %v but got %v", i, s, ss) }
                if len(c.children) != cc { t.Errorf("%v: expecting %v children but got %v", i, cc, len(c.children)) }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        nd := c.children[cn]
                        if nd.end <= nd.pos {
                                t.Errorf("%v: child %v has bad range [%v, %v) (%v)", i, cn, nd.pos, nd.end, l.str(c))
                        }
                        if s := l.str(nd); s != cs[cn] {
                                t.Errorf("%v: expecting child %v '%v', but '%v', in '%v'", i, cn, cs[cn], s, l.str(nd))
                        }
                }
                if cn != len(cs) { t.Errorf("%v: expecting at least %v children, but got %v", i, len(cs), cn) }
        }

        var cc, ccc *node

        i = 0; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, "a", "a")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 1; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, "b", "b")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 2; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, "c", "c")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 3; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, "d", "d")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 4; c = l.nodes[i]; checkNode(c, nodeDefineSingleColoned, 2, `:=`, "foo", `$(a) \
 $b\
 ${c}\
`)
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, "foo")
        cc = c.children[1]; checkNode(cc, nodeImmediateText, 6, "$(a) \\\n $b\\\n ${c}\\\n", "$(a)", "\\\n", "$b", "\\\n", "${c}", "\\\n")
        cc = c.children[1].children[0]; checkNode(cc, nodeCall, 1, "$(a)", "a")
        cc = c.children[1].children[1]; checkNode(cc, nodeEscape, 0, "\\\n")
        cc = c.children[1].children[2]; checkNode(cc, nodeCall, 1, "$b", "b")
        cc = c.children[1].children[3]; checkNode(cc, nodeEscape, 0, "\\\n")
        cc = c.children[1].children[4]; checkNode(cc, nodeCall, 1, "${c}", "c")
        cc = c.children[1].children[5]; checkNode(cc, nodeEscape, 0, "\\\n")

        i = 5; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, `bar`, `$(foo) \
$(a) \
 $b $c`)
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, "bar")
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 6, "$(foo) \\\n$(a) \\\n $b $c", "$(foo)", "\\\n", "$(a)", "\\\n", "$b", "$c")

        i = 6; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, "f_$a_$b_$c_1", "f_a_b_c")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 3, "f_$a_$b_$c_1", "$a", "$b", "$c")
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 0, "f_a_b_c")

        i = 7; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, "f_$a_$b_$c_2", "f_$a_$b_$c")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 3, "f_$a_$b_$c_2", "$a", "$b", "$c")
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 3, "f_$a_$b_$c", "$a", "$b", "$c")

        i = 8; c = l.nodes[i]; checkNode(c, nodeDefineAppend, 2, `+=`, "a", "a")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 9; c = l.nodes[i]; checkNode(c, nodeDefineAppend, 2, `+=`, "a", "a")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 10; c = l.nodes[i]; checkNode(c, nodeDefineQuestioned, 2, `?=`, "b", "b")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeDeferredText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 11; c = l.nodes[i]; checkNode(c, nodeDefineDoubleColoned, 2, `::=`, "cc", "cc")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 12; c = l.nodes[i]; checkNode(c, nodeDefineNot, 2, `!=`, "n", "n")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 13; c = l.nodes[i]; checkNode(c, nodeDefineSingleColoned, 2, `:=`, "f_$a_$b_$c_3", "f_$($a)_$($($b))_$(${$($c)})")
        if n, ek := 0, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeImmediateText; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 3, "f_$a_$b_$c_3", "$a", "$b", "$c")
        cc = c.children[1]; checkNode(cc, nodeImmediateText, 3, "f_$($a)_$($($b))_$(${$($c)})", "$($a)", "$($($b))", "$(${$($c)})")

        cc = c.children[0]
        ccc = cc.children[0]; checkNode(ccc, nodeCall, 1, "$a", "a")
        ccc = cc.children[1]; checkNode(ccc, nodeCall, 1, "$b", "b")
        ccc = cc.children[2]; checkNode(ccc, nodeCall, 1, "$c", "c")

        cc = c.children[1]
        ccc = cc.children[0]; checkNode(ccc, nodeCall, 1, "$($a)", "$a")
        ccc = cc.children[1]; checkNode(ccc, nodeCall, 1, "$($($b))", "$($b)")

        cc = c.children[1].children[0]
        cc = cc.children[0]; checkNode(cc, nodeCallName, 1, "$a", "$a")
        cc = cc.children[0]; checkNode(cc, nodeCall, 1, "$a", "a")

        cc = c.children[1].children[1]
        cc = cc.children[0]; checkNode(cc, nodeCallName, 1, "$($b)", "$($b)")
        cc = cc.children[0]; checkNode(cc, nodeCall, 1, "$($b)", "$b")
        cc = cc.children[0]; checkNode(cc, nodeCallName, 1, "$b", "$b")
        cc = cc.children[0]; checkNode(cc, nodeCall, 1, "$b", "b")
}

func TestLexCalls(t *testing.T) {
        l := newTestLex("TestLexCalls", `
foo = foo
bar = bar
foobar := $(foo)$(bar)
foobaz := $(foo)\
$(bar)\

$(info $(foo)$(bar))
$(info $(foobar))
$(info $(foo),$(bar),$(foobar))
$(info $(foo) $(bar), $(foobar) )
$(info $($(foo)),$($($(foo)$(bar))))

aaa |$(foo)|$(bar)| aaa
`)
        l.parse()

        if ex := 10; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }
        
        var (
                countImmediateTexts = 0
                countDeferredDefines = 0
                countSingleColonedDefines = 0
        )
        for _, n := range l.nodes {
                //t.Logf("%v", n.kind)
                switch n.kind {
                case nodeImmediateText:         countImmediateTexts++
                case nodeDefineDeferred:        countDeferredDefines++
                case nodeDefineSingleColoned:   countSingleColonedDefines++
                }
        }
        if ex := 6; countImmediateTexts != ex           { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeImmediateText,       countImmediateTexts) }
        if ex := 2; countDeferredDefines != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDeferred,      countDeferredDefines) }
        if ex := 2; countSingleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineSingleColoned, countSingleColonedDefines) }
        if n := countImmediateTexts+countDeferredDefines+countSingleColonedDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        var (
                c *node
                i int
        )
        checkNode := func(c *node, k nodeType, cc int, s string, cs ...string) {
                if c.kind != k { t.Errorf("%v: expecting kind %v but got %v", i, k, c.kind) }
                if ss := l.str(c); ss != s { t.Errorf("%v: expecting %v but got %v", i, s, ss) }
                if len(c.children) != cc { t.Errorf("%v: expecting %v children but got %v", i, cc, len(c.children)) }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        nd := c.children[cn]
                        if nd.end <= nd.pos {
                                t.Errorf("%v: child %v has bad range [%v, %v) (%v)", i, cn, nd.pos, nd.end, l.str(c))
                        }
                        if s := l.str(nd); s != cs[cn] {
                                t.Errorf("%v: expecting child %v '%v', but '%v', in '%v'", i, cn, cs[cn], s, l.str(nd))
                        }
                }
                if cn != len(cs) { t.Errorf("%v: expecting at least %v children, but got %v", i, len(cs), cn) }
        }

        var cc *node

        i = 4; c = l.nodes[i]; checkNode(c, nodeImmediateText, 1, `$(info $(foo)$(bar))`, `$(info $(foo)$(bar))`)
        if n, ek := 0, nodeCall; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        c = c.children[0]; checkNode(c, nodeCall, 2, `$(info $(foo)$(bar))`, "info", "$(foo)$(bar)")
        if n, ek := 0, nodeCallName; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 5; c = l.nodes[i]; checkNode(c, nodeImmediateText, 1, `$(info $(foobar))`, `$(info $(foobar))`)
        if n, ek := 0, nodeCall; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        c = c.children[0]; checkNode(c, nodeCall, 2, `$(info $(foobar))`, "info", "$(foobar)")
        if n, ek := 0, nodeCallName; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 6; c = l.nodes[i]; checkNode(c, nodeImmediateText, 1, `$(info $(foo),$(bar),$(foobar))`, `$(info $(foo),$(bar),$(foobar))`)
        if n, ek := 0, nodeCall; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        c = c.children[0]; checkNode(c, nodeCall, 4, `$(info $(foo),$(bar),$(foobar))`, "info", "$(foo)", "$(bar)", "$(foobar)")
        if n, ek := 0, nodeCallName; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 2, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 3, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 7; c = l.nodes[i]; checkNode(c, nodeImmediateText, 1, `$(info $(foo) $(bar), $(foobar) )`, `$(info $(foo) $(bar), $(foobar) )`)
        if n, ek := 0, nodeCall; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        c = c.children[0]; checkNode(c, nodeCall, 3, `$(info $(foo) $(bar), $(foobar) )`, "info", "$(foo) $(bar)", " $(foobar) ")
        if n, ek := 0, nodeCallName; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 2, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }

        i = 8; c = l.nodes[i]; checkNode(c, nodeImmediateText, 1, `$(info $($(foo)),$($($(foo)$(bar))))`, `$(info $($(foo)),$($($(foo)$(bar))))`)
        if n, ek := 0, nodeCall; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        c = c.children[0]; checkNode(c, nodeCall, 3, `$(info $($(foo)),$($($(foo)$(bar))))`, "info", "$($(foo))", "$($($(foo)$(bar)))")
        if n, ek := 0, nodeCallName; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        if n, ek := 1, nodeCallArg; c.children[n].kind != ek { t.Errorf("%v: expect child %v to be %v, but got %v", i, n, ek, c.children[n].kind) }
        cc = c.children[0]; checkNode(cc, nodeCallName, 0, `info`)
        cc = c.children[1]; checkNode(cc, nodeCallArg, 1, `$($(foo))`, "$($(foo))")
        cc = c.children[1].children[0]; checkNode(cc, nodeCall, 1, `$($(foo))`, "$(foo)")
        cc = c.children[1].children[0].children[0]; checkNode(cc, nodeCallName, 1, `$(foo)`, "$(foo)")
        cc = c.children[1].children[0].children[0].children[0]; checkNode(cc, nodeCall, 1, `$(foo)`, "foo")
        cc = c.children[1].children[0].children[0].children[0].children[0]; checkNode(cc, nodeCallName, 0, "foo")

        cc = c.children[2]; checkNode(cc, nodeCallArg, 1, `$($($(foo)$(bar)))`, "$($($(foo)$(bar)))")
        cc = c.children[2].children[0]; checkNode(cc, nodeCall, 1, `$($($(foo)$(bar)))`, "$($(foo)$(bar))")
        cc = c.children[2].children[0].children[0]; checkNode(cc, nodeCallName, 1, `$($(foo)$(bar))`, "$($(foo)$(bar))")
        cc = c.children[2].children[0].children[0].children[0]; checkNode(cc, nodeCall, 1, `$($(foo)$(bar))`, "$(foo)$(bar)")
        cc = c.children[2].children[0].children[0].children[0].children[0]; checkNode(cc, nodeCallName, 2, `$(foo)$(bar)`, "$(foo)", "$(bar)")
        cc = c.children[2].children[0].children[0].children[0].children[0].children[0]; checkNode(cc, nodeCall, 1, "$(foo)", "foo")
        cc = c.children[2].children[0].children[0].children[0].children[0].children[0].children[0]; checkNode(cc, nodeCallName, 0, "foo")
        cc = c.children[2].children[0].children[0].children[0].children[0].children[1]; checkNode(cc, nodeCall, 1, "$(bar)", "bar")
        cc = c.children[2].children[0].children[0].children[0].children[0].children[1].children[0]; checkNode(cc, nodeCallName, 0, "bar")

        i = 9; c = l.nodes[i]; checkNode(c, nodeImmediateText, 2, `aaa |$(foo)|$(bar)| aaa`, "$(foo)", "$(bar)")
        c = l.nodes[i].children[0]; checkNode(c, nodeCall, 1, `$(foo)`, "foo")
        c = l.nodes[i].children[0].children[0]; checkNode(c, nodeCallName, 0, `foo`)
        c = l.nodes[i].children[1]; checkNode(c, nodeCall, 1, `$(bar)`, "bar")
        c = l.nodes[i].children[1].children[0]; checkNode(c, nodeCallName, 0, `bar`)
}

func TestLexEscapes(t *testing.T) {
        l := newTestLex("TestLexCalls", `
a = xxx\#xxx
b = xxx\
  yyy \
  zzz \

c = xxx \#\
  yyy \#\

`)
        l.parse()

        if ex := 3; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }
        
        var (
                countDeferredDefines = 0
        )
        for _, n := range l.nodes {
                switch n.kind {
                case nodeDefineDeferred:        countDeferredDefines++
                }
        }
        if ex := 3; countDeferredDefines != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDeferred,      countDeferredDefines) }
        if n := countDeferredDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        var (
                c *node
                i int
        )
        checkNode := func(c *node, k nodeType, cc int, s string, cs ...string) {
                if c.kind != k { t.Errorf("%v: expecting kind %v but got %v", i, k, c.kind) }
                if ss := l.str(c); ss != s { t.Errorf("%v: expecting %v but got %v", i, s, ss) }
                if len(c.children) != cc { t.Errorf("%v: expecting %v children but got %v", i, cc, len(c.children)) }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        nd := c.children[cn]
                        if nd.end <= nd.pos {
                                t.Errorf("%v: child %v has bad range [%v, %v) (%v)", i, cn, nd.pos, nd.end, l.str(c))
                        }
                        if s := l.str(nd); s != cs[cn] {
                                t.Errorf("%v: expecting child %v '%v', but '%v', in '%v'", i, cn, cs[cn], s, l.str(nd))
                        }
                }
                if cn != len(cs) { t.Errorf("%v: expecting at least %v children, but got %v", i, len(cs), cn) }
        }

        var cc *node

        i = 0; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, `a`, `xxx\#xxx`)
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, `a`)
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 1, `xxx\#xxx`, `\#`)

        i = 1; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, `b`, "xxx\\\n  yyy \\\n  zzz \\\n")
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, `b`)
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 3, "xxx\\\n  yyy \\\n  zzz \\\n", "\\\n", "\\\n", "\\\n")

        i = 2; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, `c`, "xxx \\#\\\n  yyy \\#\\\n")
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, `c`)
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 4, "xxx \\#\\\n  yyy \\#\\\n", "\\#", "\\\n", "\\#", "\\\n")
}

func _TestLexRules(t *testing.T) {
        /*
## Bash
foobar : foo bar blah {{
    gcc -c $1 $2
}}

## TCL
blah : blah.c [tcl]{{
    [ gcc -c $< -o $@ ]
}}
        */
        l := newTestLex("TestLexCalls", `
foobar : foo bar blah

foo: foo.c ; gcc -c $< -o $@
bar: bar.c
	# this is command line comment
# this is script comment being ignored
	@echo "compiling..."
	@gcc -c $< -o $@

blah : blah.c
	gcc -c $< -o $@
`)
        l.parse()

        if ex := 4; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }
        
        var (
                countDeferredDefines = 0
        )
        for _, n := range l.nodes {
                switch n.kind {
                case nodeDefineDeferred:        countDeferredDefines++
                }
        }
        if ex := 3; countDeferredDefines != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDeferred,      countDeferredDefines) }
        if n := countDeferredDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        var (
                c *node
                i int
        )
        checkNode := func(c *node, k nodeType, cc int, s string, cs ...string) {
                if c.kind != k { t.Errorf("%v: expecting kind %v but got %v", i, k, c.kind) }
                if ss := l.str(c); ss != s { t.Errorf("%v: expecting %v but got %v", i, s, ss) }
                if len(c.children) != cc { t.Errorf("%v: expecting %v children but got %v", i, cc, len(c.children)) }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        nd := c.children[cn]
                        if nd.end <= nd.pos {
                                t.Errorf("%v: child %v has bad range [%v, %v) (%v)", i, cn, nd.pos, nd.end, l.str(c))
                        }
                        if s := l.str(nd); s != cs[cn] {
                                t.Errorf("%v: expecting child %v '%v', but '%v', in '%v'", i, cn, cs[cn], s, l.str(nd))
                        }
                }
                if cn != len(cs) { t.Errorf("%v: expecting at least %v children, but got %v", i, len(cs), cn) }
        }

        var cc *node

        i = 0; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, `a`, `xxx\#xxx`)
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, `a`)
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 1, `xxx\#xxx`, `\#`)

        i = 1; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, `b`, "xxx\\\n  yyy \\\n  zzz \\\n")
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, `b`)
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 3, "xxx\\\n  yyy \\\n  zzz \\\n", "\\\n", "\\\n", "\\\n")

        i = 2; c = l.nodes[i]; checkNode(c, nodeDefineDeferred, 2, `=`, `c`, "xxx \\#\\\n  yyy \\#\\\n")
        cc = c.children[0]; checkNode(cc, nodeImmediateText, 0, `c`)
        cc = c.children[1]; checkNode(cc, nodeDeferredText, 4, "xxx \\#\\\n  yyy \\#\\\n", "\\#", "\\\n", "\\#", "\\\n")
}

func TestParse(t *testing.T) {
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(args ...string) {
                fmt.Fprintf(info, "%v\n", strings.Join(args, " "))
        }

        p := newTestContext("TestParse#1", `
a = a
i = i
ii = x x $a $i x x
i1 = $a$i-$(ii)
i2 = $a$($i)-$($i$i)
`)

        //if err := p.parse();          err != nil { t.Error("parse error:", err) }
        if ex, nl := 5, len(p.l.nodes); nl != ex { t.Error("expect", ex, "but", nl) }

        for _, s := range []string{ "a", "i", "ii", "i1", "i2" } {
                if _, ok := p.defines[s]; !ok { t.Errorf("missing '%v'", s) }
        }

        if l1, l2 := len(p.defines), 5;                 l1 != l2 { t.Errorf("expects '%v' defines but got '%v'", l2, l1) }

        if s, ex := p.call("a"), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("i"), "i";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("ii"), "x x a i x x";        s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("i1"), "ai-x x a i x x";     s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("i2"), "ai-x x a i x x";     s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }

        //////////////////////////////////////////////////
        p = newTestContext("TestParse#2", `
a = a
i = i
ii = i $a i a \
 $a i
sh$ared = shared
stat$ic = static
a$$a = foo
aaaa = xxx$(info 1:$(sh$ared),$(stat$ic))-$(a$$a)-xxx
bbbb := xxx$(info 2:$(sh$ared),$(stat$ic))-$(a$$a)-xxx
cccc = xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
dddd := xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
`)

        //if err := p.parse();           err != nil { t.Error("parse error:", err) }
        if ex, nl := 10, len(p.defines); nl != ex { t.Error("expect", ex, "defines, but", nl) }

        for _, s := range []string{ "a", "i", "ii", "shared", "static", "a$a", "aaaa", "bbbb", "cccc", "dddd" } {
                if _, ok := p.defines[s]; !ok { t.Errorf("missing '%v'", s) }
        }

        if s, ex := p.call("a"), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("i"), "i";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("ii"), "i a i a   a i";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("shared"), "shared";         s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("static"), "static";         s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("a"), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("a$a"), "foo";               s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("a$$a"), "";                 s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("aaaa"), "xxx-foo-xxx";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("bbbb"), "xxx-foo-xxx";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("cccc"), "xxx-shared-static-foo-xxx";       s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := p.call("dddd"), "xxx-shared-static-foo-xxx";       s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := info.String(), "2:shared static\n1:shared static\n"; s != ex {
                t.Errorf("expects '%v' but got '%v'", ex, s)
        }
}

func _TestEmptyValue(t *testing.T) {
        ctx := newTestContext("TestLexCalls", `
foo =
bar = bar
foobar := 
foobaz := foo-baz
`)
        //if err := ctx.parse(); err != nil { t.Errorf("parse error:", err) }
        if s := ctx.call("foo"); s != "" { t.Errorf("foo: %s", s) }
        if s := ctx.call("bar"); s != "bar" { t.Errorf("bar: %s", s) }
        if s := ctx.call("foobar"); s != "" { t.Errorf("foobar: %s", s) }
        if s := ctx.call("foobaz"); s != "foo-baz" { t.Errorf("foobaz: %s", s) }
}
