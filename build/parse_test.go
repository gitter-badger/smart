//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "testing"
        //"strings"
        "bytes"
        "fmt"
        "os"
        "os/exec"
        "path/filepath"
)

type testToolset struct {
        BasicToolset
        tag string
        ns namespaceEmbed
}

func (tt *testToolset) getNamespace() namespace { return &tt.ns }

func newTestToolset(tag string) (ts *testToolset) {
        ts = &testToolset{
                tag:tag, ns:namespaceEmbed{
                        defines: make(map[string]*define),
                        rules: make(map[string]*rule),
                },
        }
        ts.ns.defines["name"] = &define{ name:tag, value:Items{ stringitem("test-"+tag) } }
        return
}

func newTestLex(file, s string) (l *lex) {
        l = &lex{ parseBuffer:&parseBuffer{ scope:file, s:[]byte(s) }, pos:0, }
        return
}

func newTestContext(file, s string) (p *Context, e error) {
        p, e = NewContext(file, []byte(s), nil)
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
                if c = l.nodes[n]; c.kind != k { t.Errorf("expecting node %v as %v but got %v(%v)", n, k, c.kind, c.str()); okay = false }
                if c.str() != s { t.Errorf("expecting node %v as %v(%v) but got %v(%v)", n, k, s, c.kind, c.str()); okay = false }
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

empty1 =
empty2 :=
empty3 = 
empty4 =    
empty5 =	

foo.bar = hierarchy
`)
        l.parse()

        if ex := 20; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }

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
        if ex := 12; countDeferredDefines != ex         { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDeferred,      countDeferredDefines) }
        if ex := 1; countQuestionedDefines != ex        { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineQuestioned,    countQuestionedDefines) }
        if ex := 3; countSingleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineSingleColoned, countSingleColonedDefines) }
        if ex := 1; countDoubleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDoubleColoned, countDoubleColonedDefines) }
        if ex := 2; countAppendDefines != ex            { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineAppend,        countAppendDefines) }
        if ex := 1; countNotDefines != ex               { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineNot,           countNotDefines) }
        if n := countDeferredDefines+countQuestionedDefines+countSingleColonedDefines+countDoubleColonedDefines+countAppendDefines+countNotDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        if c, x := l.nodes[0], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[1], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[2], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[3], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "d"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "d"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[4], nodeDefineSingleColoned; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), ":="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "$(a) \\\n $b\\\n ${c}\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c1.children), 6; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2, c3, c4, c5 := c1.children[0], c1.children[1], c1.children[2], c1.children[3], c1.children[4], c1.children[5]
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$(a)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.children[0].str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.kind, nodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c2.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.children[0].str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c3.kind, nodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c3.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c4.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c4.str(), "${c}"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c4.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c4.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c4.children[0].str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c5.kind, nodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c5.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                        }
                }
        }

        if c, x := l.nodes[5], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "$(foo) \\\n$(a) \\\n $b $c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c1.children), 6; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2, c3, c4, c5 := c1.children[0], c1.children[1], c1.children[2], c1.children[3], c1.children[4], c1.children[5]
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.children[0].str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.kind, nodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c2.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$(a)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.children[0].str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c3.kind, nodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c3.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c4.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c4.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c4.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c4.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c4.children[0].str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c5.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c5.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c5.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c5.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c5.children[0].str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[6], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "f_$a_$b_$c_1"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "f_a_b_c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c0.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.children[0], c0.children[1], c0.children[2]
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.children[0].str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.children[0].str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.children[0].str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                }
        }
        
        if c, x := l.nodes[7], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "f_$a_$b_$c_2"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "f_$a_$b_$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c0.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.children[0], c0.children[1], c0.children[2]
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.children[0].str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.children[0].str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.children[0].str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                        if a, b := len(c1.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c1.children[0], c1.children[1], c1.children[2]
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.children[0].str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.children[0].str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.children[0].kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.children[0].str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[8], nodeDefineAppend; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "+="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[9], nodeDefineAppend; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "+="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[10], nodeDefineQuestioned; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "?="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[11], nodeDefineDoubleColoned; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "::="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "cc"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "cc"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[12], nodeDefineNot; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "!="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[13], nodeDefineSingleColoned; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), ":="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "f_$a_$b_$c_3"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "f_$($a)_$($($b))_$(${$($c)})"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c0.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.children[0], c0.children[1], c0.children[2]
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.children[0]
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c1.children[0]
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.children[0]
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                        if a, b := len(c1.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c1.children[0], c1.children[1], c1.children[2]
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$($a)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.children[0]
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.children[0]
                                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.children[0]
                                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                                if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$($($b))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c1.children[0]
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "$($b)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.children[0]
                                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.str(), "$($b)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.children[0]
                                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                        c0 := c0.children[0]
                                                                                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                        if a, b := c0.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                        c0 := c0.children[0]
                                                                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                        if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                }
                                                                                        }
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                                if a, b := c2.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$(${$($c)})"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.children[0]
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "${$($c)}"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.children[0]
                                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.str(), "${$($c)}"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.children[0]
                                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.str(), "$($c)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                c0 := c0.children[0]
                                                                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                if a, b := c0.str(), "$($c)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                c0 := c0.children[0]
                                                                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                if a, b := c0.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                c0 := c0.children[0]
                                                                                                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                                if a, b := c0.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                                c0 := c0.children[0]
                                                                                                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                                                if a, b := c0.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                                                }
                                                                                                                                        }
                                                                                                                                }
                                                                                                                        }
                                                                                                                }
                                                                                                        }
                                                                                                }
                                                                                        }
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[14], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty1"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[15], nodeDefineSingleColoned; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), ":="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty2"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[16], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty3"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[17], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty4"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[18], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty5"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[19], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "foo.bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c0.children[0]
                                        if a, b := c0.kind, nodeNamePart; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.str(), "."; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "hierarchy"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
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

name = $(prefix:name)
name = $(prefix:part1.part2.part3)
`)
        l.parse()

        if ex := 12; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }
        
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
        if ex := 4; countDeferredDefines != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineDeferred,      countDeferredDefines) }
        if ex := 2; countSingleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeDefineSingleColoned, countSingleColonedDefines) }
        if n := countImmediateTexts+countDeferredDefines+countSingleColonedDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        if c, x := l.nodes[4], nodeImmediateText; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "$(info $(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.children[0]
                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1 := c0.children[0], c0.children[1]
                                        if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.str(), "$(foo)$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0, c1 := c1.children[0], c1.children[1]
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.children[0]
                                                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        }
                                                }
                                                if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c1.children[0]
                                                        if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[5], nodeImmediateText; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "$(info $(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.children[0]
                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1 := c0.children[0], c0.children[1]
                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                        if a, b := c1.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c1.children[0]
                                                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.children[0]
                                                                if a, b := c0.str(), "foobar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[6], nodeImmediateText; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "$(info $(foo),$(bar),$(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.children[0]
                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foo),$(bar),$(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 4; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1, c2, c3 := c0.children[0], c0.children[1], c0.children[2], c0.children[3]
                                        if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c1.children[0]
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.children[0]
                                                        if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                        if a, b := c2.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c2.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.children[0]
                                                if a, b := c0.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.children[0]
                                                        if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                        if a, b := c3.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c3.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c3.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c3.children[0]
                                                if a, b := c0.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.children[0]
                                                        if a, b := c0.str(), "foobar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[7], nodeImmediateText; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "$(info $(foo) $(bar), $(foobar) )"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.children[0]
                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foo) $(bar), $(foobar) )"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1, c2 := c0.children[0], c0.children[1], c0.children[2]
                                        if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.str(), "$(foo) $(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0, c1 := c1.children[0], c1.children[1]
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.children[0]
                                                        if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                                if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c1.children[0]
                                                        if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                        if a, b := c2.str(), " $(foobar) "; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c2.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.children[0]
                                                if a, b := c0.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.children[0]
                                                        if a, b := c0.str(), "foobar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[8], nodeImmediateText; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "$(info $($(foo)),$($($(foo)$(bar))))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.children[0]
                        if a, b := c0.str(), "$(info $($(foo)),$($($(foo)$(bar))))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.children[0], c0.children[1], c0.children[2]
                                if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c1.str(), "$($(foo))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c1.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c1.children[0]
                                        if a, b := c0.str(), "$($(foo))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.children[0]
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.children[0]
                                                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.children[0]
                                                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                }
                                                        }
                                                }
                                        }
                                }
                                if a, b := c2.str(), "$($($(foo)$(bar)))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c2.kind, nodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c2.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c2.children[0]
                                        if a, b := c0.str(), "$($($(foo)$(bar)))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.children[0]
                                                if a, b := c0.str(), "$($(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.children[0]
                                                        if a, b := c0.str(), "$($(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.children[0]
                                                                if a, b := c0.str(), "$(foo)$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        c0, c1 := c0.children[0], c0.children[1]
                                                                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.children[0]
                                                                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                }
                                                                        }
                                                                        if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c1.children[0]
                                                                                if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[9], nodeImmediateText; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "aaa |$(foo)|$(bar)| aaa"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c0.children[0]
                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                }
                        }
                        if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c1.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c1.children[0]
                                if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                }
                        }
                }
        }

        if c, x := l.nodes[10], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.str(), "name"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        }
                        if a, b := c1.str(), "$(prefix:name)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c1.children[0]
                                if a, b := c0.str(), "$(prefix:name)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c0.children[0]
                                        if a, b := c0.str(), "prefix:name"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.children[0]
                                                if a, b := c0.str(), ":"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeNamePrefix; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[11], nodeDefineDeferred; c.kind != x { t.Errorf("expecting %v but %v", x, c.kind) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.children[0], c.children[1]
                        if a, b := c0.str(), "name"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.kind, nodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        }
                        if a, b := c1.str(), "$(prefix:part1.part2.part3)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c1.kind, nodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c1.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c1.children[0]
                                if a, b := c0.str(), "$(prefix:part1.part2.part3)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.kind, nodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c0.children[0]
                                        if a, b := c0.str(), "prefix:part1.part2.part3"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.kind, nodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0, c1, c2 := c0.children[0], c0.children[1], c0.children[2]
                                                if a, b := c0.str(), ":"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.kind, nodeNamePrefix; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                                if a, b := c1.str(), "."; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.kind, nodeNamePart; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                                if a, b := c2.str(), "."; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.kind, nodeNamePart; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c2.children), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                        }
                                }
                        }
                }
        }
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
                if ss := c.str(); ss != s { t.Errorf("%v: expecting %v but got %v", i, s, ss) }
                if len(c.children) != cc { t.Errorf("%v: expecting %v children but got %v", i, cc, len(c.children)) }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        nd := c.children[cn]
                        if nd.end < nd.pos {
                                t.Errorf("%v: child %v has bad range [%v, %v) (%v)", i, cn, nd.pos, nd.end, c.str())
                        }
                        if s := nd.str(); s != cs[cn] {
                                t.Errorf("%v: expecting child %v '%v', but '%v', in '%v'", i, cn, cs[cn], s, nd.str())
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

func TestLexRules(t *testing.T) {
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
        l := newTestLex("TestLexRules", `
foobar : foo bar blah

foo: foo.c ; gcc -c $< -o $@
bar: bar.c
	# this is command line comment
# this is script comment being ignored
	@echo "compiling..."
	@gcc -c $< -o $@

baz:bar
zz::foo

a:
	echo blah blah

blah : blah.c
	gcc -c $< -o $@
`)
        l.parse()

        if ex, n := 7, len(l.nodes); n != ex { t.Errorf("expecting %v nodes but got %v", ex, n) }
        
        var (
                countRuleSingleColoned = 0
                countRuleDoubleColoned = 0
        )
        for _, n := range l.nodes {
                //fmt.Fprintf(os.Stderr, "TestLexRules: %v: %v\n", n.kind, n.children[0].str())
                switch n.kind {
                case nodeRuleSingleColoned:     countRuleSingleColoned++
                case nodeRuleDoubleColoned:     countRuleDoubleColoned++
                }
        }
        if ex := 6; countRuleSingleColoned != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeRuleSingleColoned,      countRuleSingleColoned) }
        if ex := 1; countRuleDoubleColoned != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeRuleDoubleColoned,      countRuleDoubleColoned) }
        if n := countRuleSingleColoned+countRuleDoubleColoned; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        var (
                c *node
                i int
        )
        checkNode := func(c *node, k nodeType, cc int, s string, cs ...string) {
                if c.kind != k { t.Errorf("%v: expecting kind %v but got %v", i, k, c.kind) }
                if ss := c.str(); ss != s { t.Errorf("%v: expecting '%v' but got '%v'", i, s, ss) }
                if len(c.children) != cc { t.Errorf("%v: expecting '%v' children but got '%v'", i, cc, len(c.children)) }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        nd := c.children[cn]
                        if nd.end < nd.pos {
                                t.Errorf("%v: child %v has bad range [%v, %v) (%v)", i, cn, nd.pos, nd.end, c.str())
                        }
                        if s := nd.str(); s != cs[cn] {
                                t.Errorf("%v: expecting child %v '%v', but '%v', in '%v'", i, cn, cs[cn], s, nd.str())
                        }
                }
                if cn != len(cs) { t.Errorf("%v: expecting at least %v children, but got %v", i, len(cs), cn) }
        }

        var (
                cc, cx *node
                ex nodeType
        )
        i = 0; c = l.nodes[i]; checkNode(c, nodeRuleSingleColoned, 2, `:`, `foobar`, `foo bar blah`)
        cc = c.children[0]; checkNode(cc, nodeTargets, 0, `foobar`)
        cc = c.children[1]; checkNode(cc, nodePrerequisites, 0, `foo bar blah`)

        i = 1; c = l.nodes[i]; checkNode(c, nodeRuleSingleColoned, 3, `:`, `foo`, "foo.c", "gcc -c $< -o $@")
        cc = c.children[0]; checkNode(cc, nodeTargets, 0, `foo`)
        cc = c.children[1]; checkNode(cc, nodePrerequisites, 0, "foo.c")

        i = 2; c = l.nodes[i]; checkNode(c, nodeRuleSingleColoned, 3, `:`, `bar`, "bar.c", "\t# this is command line comment\n# this is script comment being ignored\n\t@echo \"compiling...\"\n\t@gcc -c $< -o $@\n")
        cc = c.children[0]; checkNode(cc, nodeTargets, 0, `bar`)
        cc = c.children[1]; checkNode(cc, nodePrerequisites, 0, "bar.c")
        cc = c.children[2]; checkNode(cc, nodeActions, 4, "\t# this is command line comment\n# this is script comment being ignored\n\t@echo \"compiling...\"\n\t@gcc -c $< -o $@\n", "# this is command line comment", "# this is script comment being ignored", "@echo \"compiling...\"", "@gcc -c $< -o $@")
        if ex = nodeActions; cc.kind != ex { t.Errorf("expecting %v but %v", ex, cc.kind) }
        if cx, ex = cc.children[0], nodeAction;  cx.kind != ex { t.Errorf("expecting %v but %v", ex, cx.kind) } else {
                if s, ss := cx.str(), "# this is command line comment"; s != ss { t.Errorf("expecting %v but %v", ss, s) }
        }
        if cx, ex = cc.children[1], nodeComment; cx.kind != ex { t.Errorf("expecting %v but %v", ex, cx.kind) } else {
                if s, ss := cx.str(), "# this is script comment being ignored"; s != ss { t.Errorf("expecting %v but %v", ss, s) }
        }
        if cx, ex = cc.children[2], nodeAction;  cx.kind != ex { t.Errorf("expecting %v but %v", ex, cx.kind) } else {
                if s, ss := cx.str(), "@echo \"compiling...\""; s != ss { t.Errorf("expecting %v but %v", ss, s) }
        }
        if cx, ex = cc.children[3], nodeAction;  cx.kind != ex { t.Errorf("expecting %v but %v", ex, cx.kind) } else {
                if s, ss := cx.str(), "@gcc -c $< -o $@"; s != ss { t.Errorf("expecting %v but %v", ss, s) }
        }

        i = 3; c = l.nodes[i]; checkNode(c, nodeRuleSingleColoned, 2, `:`, `baz`, "bar")
        i = 4; c = l.nodes[i]; checkNode(c, nodeRuleDoubleColoned, 2, `::`, `zz`, "foo")
        i = 5; c = l.nodes[i]; checkNode(c, nodeRuleSingleColoned, 3, `:`, `a`, "", "\techo blah blah\n")
        i = 6; c = l.nodes[i]; checkNode(c, nodeRuleSingleColoned, 3, `:`, `blah`, "blah.c", "\tgcc -c $< -o $@\n")
}

func TestParse(t *testing.T) {
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestParse#1", `
a = a
i = i
ii = x x $a $i x x
i1 = $a$i-$(ii)
i2 = $a$($i)-$($i$i)
`);     if err != nil { t.Error("parse error:", err) }
        if ex, nl := 5, len(ctx.l.nodes); nl != ex { t.Error("expect", ex, "but", nl) }

        for _, s := range []string{ "a", "i", "ii", "i1", "i2" } {
                if d, ok := ctx.g.defines[s]; !ok || d == nil { t.Errorf("missing '%v'", s) } else {
                        if s := d.value.Expand(ctx); s == "" { t.Errorf("empty '%v'", s) }
                }
        }

        if l1, l2 := len(ctx.g.defines), 5; l1 != l2 { t.Errorf("expects '%v' defines but got '%v'", l2, l1) }

        if s, ex := ctx.Call("a").Expand(ctx), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i").Expand(ctx), "i";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("ii").Expand(ctx), "x x a i x x";        s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i1").Expand(ctx), "ai-x x a i x x";     s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i2").Expand(ctx), "ai-x x a i x x";     s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }

        //////////////////////////////////////////////////
        ctx, err = newTestContext("TestParse#2", `
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
`);     if err != nil { t.Error("parse error:", err) }
        if ex, nl := 10, len(ctx.g.defines); nl != ex { t.Error("expect", ex, "defines, but", nl) }

        for _, s := range []string{ "a", "i", "ii", "shared", "static", "a$a", "aaaa", "bbbb", "cccc", "dddd" } {
                if d, ok := ctx.g.defines[s]; !ok || d == nil { t.Errorf("missing '%v' (%v)", s, ctx.g.defines) } else {
                        if s := d.value.Expand(ctx); s == "" { t.Errorf("empty '%v'", s) }
                }
        }

        if s, ex := ctx.Call("a").Expand(ctx), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i").Expand(ctx), "i";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("ii").Expand(ctx), "i a i a   a i";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("shared").Expand(ctx), "shared";         s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("static").Expand(ctx), "static";         s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("a").Expand(ctx), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("a$a").Expand(ctx), "foo";               s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("a$$a").Expand(ctx), "";                 s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("aaaa").Expand(ctx), "xxx-foo-xxx";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("bbbb").Expand(ctx), "xxx-foo-xxx";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("cccc").Expand(ctx), "xxx-shared-static-foo-xxx";       s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("dddd").Expand(ctx), "xxx-shared-static-foo-xxx";       s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := info.String(), "2:shared static\n1:shared static\n1:shared static\n"; s != ex {
                t.Errorf("expects '%v' but got '%v'", ex, s)
        }
}

func TestEmptyValue(t *testing.T) {
        ctx, err := newTestContext("TestEmptyValue", `
foo =
bar = bar
foobar := 
foobaz := foo-baz
`);     if err != nil { t.Errorf("parse error:", err) }
        if s := ctx.Call("foo").Expand(ctx); s != "" { t.Errorf("foo: '%s'", s) }
        if s := ctx.Call("bar").Expand(ctx); s != "bar" { t.Errorf("bar: '%s'", s) }
        if s := ctx.Call("foobar").Expand(ctx); s != "" { t.Errorf("foobar: '%s'", s) }
        if s := ctx.Call("foobaz").Expand(ctx); s != "foo-baz" { t.Errorf("foobaz: '%s'", s) }
}

func TestSetEmptyValue(t *testing.T) {
        ctx, err := newTestContext("TestEmptyValue", `
foo =
bar = bar
foobar := 
foobaz := foo-baz
`);     if err != nil { t.Errorf("parse error:", err) }
        if s := ctx.Call("foo").Expand(ctx); s != "" { t.Errorf("foo: '%s'", s) }
        if s := ctx.Call("bar").Expand(ctx); s != "bar" { t.Errorf("bar: '%s'", s) }
        if s := ctx.Call("foobar").Expand(ctx); s != "" { t.Errorf("foobar: '%s'", s) }
        if s := ctx.Call("foobaz").Expand(ctx); s != "foo-baz" { t.Errorf("foobaz: '%s'", s) }
}

func TestMultipartNames(t *testing.T) {
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ts := newTestToolset("test")
        toolsets["test"] = &toolsetStub{ name:"test", toolset:ts }

        ctx, err := newTestContext("TestMultipartNames", `
$(= test:foo, f o o)
#$(= test:foo.bar,  foo bar)

$(= test:foobar, f)     $(info [$(test:foobar 1,2,3)])
$(+= test:foobar, o, o) $(info [$(test:foobar 1,2,3)])

$(module test)
me.foo = fooo
$(info $(me.foo))

$(module a)
me.foo = foooo
$(info $(me.foo))
$(commit) # test.a
me.a.bar = bar
$(commit) # test

$(info $(test.foo))
$(info $(test.a.foo))
$(info $(test.a.name))

test.foo = FOOO
test.a.foo = FOOOO
$(info $(test.foo))
$(info $(test.a.foo))
`);     if err != nil { t.Errorf("parse error:", err) }
        if s, x := ctx.Call("test:foo").Expand(ctx), " f o o"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        //if s, x := ctx.Call("test:foo.bar").Expand(ctx), "  foo bar (test:foo.bar:)"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        if s, x := ctx.Call("test.foo").Expand(ctx), "FOOO"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        if s, x := ctx.Call("test.a.foo").Expand(ctx), "FOOOO"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        if s, x := ctx.Call("test.a.bar").Expand(ctx), "bar"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        if d, b := ts.ns.defines["foo"]; !b { t.Errorf("expects 'foo'") } else {
                if x, s := " f o o", d.value.Expand(ctx); s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        }
        /*
        if d, b := ts.ns.defines["foo.bar"]; !b { t.Errorf("expects 'foo.bar'") } else {
                if x, s := "  foo bar", d.value.Expand(ctx); s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        } */

        if v, s := info.String(), fmt.Sprintf(`[ f]
[ f test:foobar  o  o]
fooo
foooo
fooo
foooo
a
FOOO
FOOOO
`); v != s { t.Errorf("`%s` != `%s`", v, s) }

        delete(toolsets, "test")
}

func TestContinualInCall(t *testing.T) {
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                //fmt.Fprintf(info, "%v\n", args.Expand(ctx))
                for n, a := range args {
                        if n == 0 {
                                fmt.Fprint(info, a.Expand(ctx))
                        } else {
                                fmt.Fprintf(info, ",%v", a.Expand(ctx))
                        }
                }
                fmt.Fprintf(info, "\n")
        }

        ctx, err := newTestContext("TestContinualInCall", `
foo = $(info a,  ndk  , \
  PLATFORM=android-9, \
  ABI=x86 armeabi, \
)
`);     if err != nil { t.Errorf("parse error:", err) }
        if s := ctx.Call("foo").Expand(ctx); s != "" { t.Errorf("foo: '%s'", s) }
        // FIXIME: if s := info.String(); s != `a,  ndk  , PLATFORM=android-9, ABI=x86 armeabi, ` { t.Errorf("info: '%s'", s) }
        if a, b := info.String(), "a,  ndk  ,    PLATFORM=android-9,    ABI=x86 armeabi,  \n"; a != b { t.Errorf("expects '%s' but '%s'", b, a) }
}

func TestModuleVariables(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestModuleVariables", `
$(module test)
$(info $(me) $(me.name) $(me.dir))
$(info $(me.export.nothing))
$(commit)
$(info $(test.name) $(test.dir))
#$(module test) ## error
`);     if err != nil { t.Errorf("parse error:", err) }

        if ctx.modules == nil { t.Errorf("nil modules") }
        if m, ok := ctx.modules["test"]; !ok || m == nil { t.Errorf("nil 'test' module") } else {
                if c, ok := m.Children["export"]; !ok || c == nil { t.Errorf("'me.export' is undefined") } else {
                        if d, ok := c.defines["name"]; !ok || d == nil { t.Errorf("no 'name' defined") } else {
                                if s := d.value.Expand(ctx); s != "export" { t.Errorf("name != 'export' (%v)", s) }
                        }
                }
                if d, ok := m.defines["name"]; !ok || d == nil { t.Errorf("no 'name' defined") } else {
                        if s := d.value.Expand(ctx); s != "test" { t.Errorf("name != 'test' (%v)", s) }
                }
                if d, ok := m.defines["dir"]; !ok || d == nil { t.Errorf("no 'dir' defined") } else {
                        if s := d.value.Expand(ctx); s != workdir { t.Errorf("dir != '%v' (%v)", workdir, s) }
                }
                ctx.With(m, func() {
                        if s := ctx.Call("me").Expand(ctx); s != "test" { t.Errorf("$(me) != test (%v)", s) }
                        if s := ctx.Call("me.name").Expand(ctx); s != "test" { t.Errorf("$(me.name) != test (%v)", s) }
                        if s := ctx.Call("me.dir").Expand(ctx); s != workdir { t.Errorf("$(me.dir) != %v (%v)", workdir, s) }
                })
        }

        if a, b := info.String(), fmt.Sprintf("test test %s\n\ntest %s\n", workdir, workdir); a != b { t.Errorf("expects '%v' but '%v'", b, a) }
}

func TestModuleTargets(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestModuleTargets", `
$(module test)

## test.foo
foo:
	@echo "$..$@"

foobar: foo
	@echo "$..$@ : $<"

$(commit)
`);     if err != nil { t.Errorf("parse error:", err) }

        if ctx.modules == nil { t.Errorf("nil modules") }
        if _, ok := ctx.g.rules["foo"]; ok { t.Errorf("foo defined in context") }
        if m, ok := ctx.modules["test"]; !ok || m == nil { t.Errorf("nil 'test' module") } else {
                if r, ok := m.rules["foo"]; !ok { t.Errorf("foo not defined in %v", m.GetName(ctx)) } else {
                        if n := len(r.targets); n != 1 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) }
                        if n := len(r.prerequisites); n != 0 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.actions); n != 1 { t.Errorf("incorrect number of actions: %v %v", n, r.actions) }
                }
                if r, ok := m.rules["foobar"]; !ok { t.Errorf("foobar not defined in %v", m.GetName(ctx)) } else {
                        if n := len(r.targets); n != 1 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) }
                        if n := len(r.prerequisites); n != 1 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.actions); n != 1 { t.Errorf("incorrect number of actions: %v %v", n, r.actions) }
                }
        }
        if s := info.String(); s != `` { t.Errorf("info: '%s'", s) }
}

func TestToolsetVariables(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ndk, _ := exec.LookPath("ndk-build")
        sdk, _ := exec.LookPath("android")
        if ndk == "" { t.Errorf("'ndk-build' is not in the PATH") }
        if sdk == "" { t.Errorf("'android' is not in the PATH") }

        ndk = filepath.Dir(ndk)
        sdk = filepath.Dir(filepath.Dir(sdk))

        toolsets["test-sdk"] = &toolsetStub{ name:"test-sdk", toolset:newTestToolset("sdk") }
        toolsets["test-ndk"] = &toolsetStub{ name:"test-ndk", toolset:newTestToolset("ndk") }
        toolsets["test-shell"] = &toolsetStub{ name:"test-shell", toolset:newTestToolset("shell") }

        _, err := newTestContext("TestToolsetVariables", `
$(info $(test-shell:name))
$(info $(test-sdk:name))
$(info $(test-sdk:support a,b,c))
$(info $(test-ndk:name))
`);     if err != nil { t.Errorf("parse error:", err) }
        if v, s := info.String(), fmt.Sprintf(`test-shell
test-sdk

test-ndk
`); v != s { t.Errorf("`%s` != `%s`", v, s) }
        delete(toolsets, "test-sdk")
        delete(toolsets, "test-ndk")
        delete(toolsets, "test-shell")
}

func TestToolsetRules(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestToolsetRules", `
all: foo bar
foo:; @touch $@.txt 
bar:
	@echo $@ > $@.txt
	@touch $@.txt
`);     if err != nil { t.Errorf("parse error:", err) }

        if r, ok := ctx.g.rules["all"]; !ok && r == nil { t.Errorf("'all' not defined") } else {
                if n, x := len(r.node.children), 2; n != x { t.Errorf("children %d != %d", n, x) }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "all"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 2; n != x { t.Errorf("prerequisites %d != %d", n, x) } else {
                        if s, x := r.prerequisites[0], "foo"; s != x { t.Errorf("prerequisites[0] %v != %v", s, x) }
                        if s, x := r.prerequisites[1], "bar"; s != x { t.Errorf("prerequisites[0] %v != %v", s, x) }
                }
                if n, x := len(r.actions), 0; n != x { t.Errorf("actions %d != %d", n, x) }
        }
        if r, ok := ctx.g.rules["foo"]; !ok && r == nil { t.Errorf("'foo' not defined") } else {
                if n, x := len(r.node.children), 3; n != x { t.Errorf("children %d != %d", n, x) } else {
                        if c, x := r.node.children[0], nodeTargets; c.kind != x { t.Errorf("children %v != %v", c.kind, x) }
                        if c, x := r.node.children[1], nodePrerequisites; c.kind != x { t.Errorf("children %v != %v", c.kind, x) }
                        if c, x := r.node.children[2], nodeAction; c.kind != x { t.Errorf("children %v != %v", c.kind, x) }
                }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "foo"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                if n, x := len(r.actions), 1; n != x { t.Errorf("actions %d != %d", n, x) } else {
                        if c, ok := r.actions[0].(*node); !ok { t.Errorf("actions[0] '%v' is not node", r.actions[0]) } else {
                                if k, x := c.kind, nodeAction; k != x { t.Errorf("actions[0] %v != %v", k, x) }
                                if s, x := c.str(), "@touch $@.txt "; s != x { t.Errorf("actions[0] %v != %v", s, x) }
                                if s, x := c.Expand(ctx), "@touch .txt "; s != x { t.Errorf("actions[0] %v != %v", s, x) }
                        }
                }
        }
        if r, ok := ctx.g.rules["bar"]; !ok && r == nil { t.Errorf("'bar' not defined") } else {
                if n, x := len(r.node.children), 3; n != x { t.Errorf("children %d != %d", n, x) } else {
                        if c, x := r.node.children[0], nodeTargets; c.kind != x { t.Errorf("children %v != %v", c.kind, x) }
                        if c, x := r.node.children[1], nodePrerequisites; c.kind != x { t.Errorf("children %v != %v", c.kind, x) }
                        if c, x := r.node.children[2], nodeActions; c.kind != x { t.Errorf("children %v != %v", c.kind, x) }
                }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "bar"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                if n, x := len(r.actions), 2; n != x { t.Errorf("actions %d != %d", n, x) } else {
                        if c, ok := r.actions[0].(*node); !ok { t.Errorf("actions[0] '%v' is not node", r.actions[0]) } else {
                                if k, x := c.kind, nodeAction; k != x { t.Errorf("actions[0] %v != %v", k, x) }
                                if s, x := c.str(), "@echo $@ > $@.txt"; s != x { t.Errorf("actions[0] %v != %v", s, x) }
                                if s, x := c.Expand(ctx), "@echo  > .txt"; s != x { t.Errorf("actions[0] %v != %v", s, x) }
                        }
                        if c, ok := r.actions[1].(*node); !ok { t.Errorf("actions[1] '%v' is not node", r.actions[1]) } else {
                                if k, x := c.kind, nodeAction; k != x { t.Errorf("actions[1] %v != %v", k, x) }
                                if s, x := c.str(), "@touch $@.txt"; s != x { t.Errorf("actions[1] %v != %v", s, x) }
                                if s, x := c.Expand(ctx), "@touch .txt"; s != x { t.Errorf("actions[1] %v != %v", s, x) }
                        }
                }
        }

        if v, s := info.String(), fmt.Sprintf(``); v != s { t.Errorf("`%s` != `%s`", v, s) }
}

func TestDefineToolset(t *testing.T) {
        wd, e := os.Getwd()
        if e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestDefineToolset", `
### Defining the toolset template
$(template test)

~.modules += $(me.name)

# Defered assignment.
me.out = $(me.dir)/out

$(info $(me.name): $(~.name) - "$(~.modules)")
$(info $(me.name): dir = "$(me.dir)")
$(info $(me.name): source = "$(me.source)")
$(info $(me.name): before-post) $(post) $(info $(me.name): after-post) # 'post' is a declaration that we're going to intersect the module 
$(info $(me.name): source = "$(me.source)")

$(me.name)/test:; @echo $@ $(me.source)

$(info $(me.name): before-commit) $(commit) $(info after-commit)

### Using the new toolset template
$(module a, test, a, b, c)    $(info a - $(me.dir),$(me.out))
me.source := a.cpp
$(commit)$(info a commited)

$(module b, test, a, b, c)    $(info b - $(me.dir),$(me.out))
me.source := b.cpp
$(commit)$(info b commited)
`);     if err != nil { t.Errorf("parse error:", err) }
        if ctx.modules == nil { t.Errorf("nil modules") }
        if ctx.templates == nil { t.Errorf("nil templates") }
        if temp, ok := ctx.templates["test"]; !ok || temp == nil { t.Errorf("no test template") } else {
                if s, x := temp.name, "test"; s != x { t.Errorf("expects %v but %v", s, x) }
                if n, x := len(temp.declNodes), 6; n != x { t.Errorf("expects %v but %v", x, n) } else {
                        if c, x := temp.declNodes[0], nodeDefineAppend; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), "+="; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.declNodes[1], nodeDefineDeferred; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), "="; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.declNodes[2], nodeImmediateText; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), `$(info $(me.name): $(~.name) - "$(~.modules)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.declNodes[3], nodeImmediateText; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), `$(info $(me.name): dir = "$(me.dir)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.declNodes[4], nodeImmediateText; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), `$(info $(me.name): source = "$(me.source)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.declNodes[5], nodeImmediateText; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), `$(info $(me.name): before-post)`; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                }
                if n, x := len(temp.postNodes), 4; n != x { t.Errorf("expects %v but %v", x, n) } else {
                        if c, x := temp.postNodes[0], nodeImmediateText; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), ` $(info $(me.name): after-post) `; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.postNodes[1], nodeImmediateText; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), `$(info $(me.name): source = "$(me.source)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.postNodes[2], nodeRuleSingleColoned; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), ":"; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                        if c, x := temp.postNodes[3], nodeImmediateText; c.kind != x { t.Errorf("expects %v but %v", x, c.kind) } else {
                                if s, x := c.str(), `$(info $(me.name): before-commit)`; s != x { t.Errorf("expects %v but %v", x, s) }
                        }
                }
        }
/*
a: after-post
a: source = "a.cpp"
a: before-commit
---------------------
b: after-post
b: source = "b.cpp"
b: before-commit
*/
        if s, x := info.String(), fmt.Sprintf(`after-commit
a: test - "a"
a: dir = "%s"
a: source = ""
a: before-post
a - %s %s/out
a commited
b: test - "a b"
b: dir = "%s"
b: source = ""
b: before-post
b - %s %s/out
b commited
`, wd, wd, wd, wd, wd, wd); s != x { t.Errorf("'%s' != '%s'", s, x) }
}

func TestSpeakSomething(t *testing.T) {
        wd, e := os.Getwd()
        if e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestSpeakSomething", `
script = $(speak template,
--------------------------
echo "smart speak - hello"
-------------------------)

text = $(speak /bin/bash, $(script))

$(info $(script))
$(info $(text))
`);     if err != nil { t.Errorf("parse error:", err) }
        if ctx == nil { t.Errorf("nil context") } else {
                
        }
        if s, x := info.String(), fmt.Sprintf(`echo "smart speak - hello"
smart speak - hello
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
}
