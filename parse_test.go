package smart

import (
        //. "smart"
        "testing"
)

func newTestLex(file, s string) (l *lex) {
        l = &lex{ file:file, s:[]byte(s), pos:0, }
        return
}

func newTestParser(file, s string) (p *parser) {
        p = &parser{
                l: lex{ file:file, s:[]byte(s), pos:0, },
                variables:make(map[string]*variable, 200),
        }
        return
}

func TestLexComments(t *testing.T) {
        var s = `
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
foobac=# comment 6	`
        l := newTestLex("TestLexComments", s)
        l.parse()

        if len(l.nodes) != 16 { t.Error("expecting 16 nodes but", len(l.nodes)) }

        count := 0
        for _, n := range l.nodes { if n.kind == node_comment { count += 1 } }
        if count != 9 { t.Error("expecting 9 comments but", count) }

        var c *node

        checkNode := func(n int, k nodeType, s string) (quit bool) {
                if len(l.nodes) < n+1 { t.Error("expecting at list", n+1, "nodes but", len(l.nodes)); return true }
                if c = l.nodes[n]; c.kind != k { t.Error("node", n ,"is not comment:", c.kind) }
                if l.get(c) != s { t.Error("node", n, "is:", c.kind, "'"+l.get(c)+"'", ", not", "'"+s+"'") }
                return false
        }

        if checkNode(0, node_comment, `# this is a comment
# this is the second line of the comment`) { return }
        if checkNode(1, node_comment, `# this is another comment
# this is the second line of the other comment
# this is the third line of the other comment`) { return }
        if checkNode(2, node_comment, `# more...`) { return }
        if checkNode(3, node_assign, `foo = foo `) { return }
        if checkNode(4, node_comment, `# comment 1`) { return }
        if checkNode(5, node_comment, `# comment 2`) { return }
        if checkNode(6, node_call, `$(info info)`) { return }
        if checkNode(7, node_spaces, ` `) { return }
        if checkNode(8, node_assign, `bar = bar`) { return }
        if checkNode(9, node_comment, `# comment 3`) { return }
        if checkNode(10, node_assign, `foobar=foobar`) { return }
        if checkNode(11, node_comment, `#comment 4`) { return }
        if checkNode(12, node_assign, `foobaz=blah`) { return }
        if checkNode(13, node_comment, `# comment 5 `) { return }
        if checkNode(14, node_assign, `foobac=`) { return }
        if checkNode(15, node_comment, `# comment 6	`) { return }
}

func TestLexAssigns(t *testing.T) {
        s := `
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
`
        l := newTestLex("TestLexAssigns", s)
        l.parse()

        if len(l.nodes) != 6 { t.Error("expecting 6 nodes but", len(l.nodes)) }
        
        count := 0
        for _, n := range l.nodes {
                if n.kind == node_assign || n.kind == node_simple_assign { count += 1 }
        }
        if count != 6 { t.Error("expecting 6 assigns, but", count) }

        checkNode := func(c *node, k nodeType, cc int, s string, cs ...string) {
                if c.kind != k { t.Error("expecting", k, ", but", c.kind) }
                if len(c.children) != cc { t.Error("expecting", cc, " children, but", len(c.children)) }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        if s := l.get(c.children[cn]); s != cs[cn] {
                                t.Error("expected child", cn, "'"+cs[cn]+"'", ", but '"+s+"',", "in '"+l.get(c)+"'")
                                break
                        }
                }
                if cn != len(cs) { t.Error("expecting at least", len(cs), "children, but", cn) }
        }

        var c *node
        c = l.nodes[0]; checkNode(c, node_assign, 3, `a = a`, "a", "=", "a")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind) }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind) }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind) }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn) }

        c = l.nodes[1]; checkNode(c, node_assign, 3, `b= b`, "b", "=", "b")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind) }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind) }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind) }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn) }

        c = l.nodes[2]; checkNode(c, node_assign, 3, `c=c`, "c", "=", "c")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind) }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind) }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind) }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn) }

        c = l.nodes[3]; checkNode(c, node_assign, 3, `d       =           d`, "d", "=", "d")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind) }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind) }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind) }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn) }

        c = l.nodes[4]; checkNode(c, node_simple_assign, 3, `foo := $(a) \
 $b\
 ${c}\
`, "foo", ":=", `$(a) \
 $b\
 ${c}\
`)
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind) }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind) }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind) }
        if cn := len(c.children[2].children); cn != 9 { t.Error("expect 9 children, but:", cn) }

        c = l.nodes[5]; checkNode(c, node_assign, 3, `bar = $(foo) \
$(a) \
 $b $c`, `bar`, `=`, `$(foo) \
$(a) \
 $b $c`)
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind) }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind) }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind) }
        if cn := len(c.children[2].children); cn != 10 { t.Error("expect 10 children, but:", cn) }
}

func TestLexCalls(t *testing.T) {
        s := `
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
`
        l := newTestLex("TestLexCalls", s)
        l.parse()

        if len(l.nodes) != 9 { t.Error("expecting 8 nodes but", len(l.nodes)) }
        
        count := 0
        for _, n := range l.nodes { if n.kind == node_call { count += 1 } }
        if count != 5 { t.Error("expecting 4 calls, but", count) }

        var c, cc *node

        if c = l.nodes[2]; c.kind != node_simple_assign { t.Error("expecting assign node, but:", c.kind) }
        if l.get(c) != `foobar := $(foo)$(bar)` { t.Error("expecting 'foobar := $(foo)$(bar)', but:", "'"+l.get(c)+"'") }
        if len(c.children) != 3 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.get(c)) }
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if cc = c.children[1]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if cc = c.children[2]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if len(cc.children) != 2 { t.Error("expecting 2 children, but:", len(cc.children)) }
        if cc.children[0].kind != node_call { t.Error("expecting call, but:", cc.children[0].kind, l.get(cc.children[0])) }
        if cc.children[1].kind != node_call { t.Error("expecting call, but:", cc.children[1].kind, l.get(cc.children[1])) }
        if l.get(cc.children[0]) != `$(foo)` { t.Error("expecting '$(foo)', but:", "'"+l.get(cc.children[0])+"'") }
        if l.get(cc.children[1]) != `$(bar)` { t.Error("expecting '$(bar)', but:", "'"+l.get(cc.children[1])+"'") }

        if c = l.nodes[4]; c.kind != node_call { t.Error("expecting call node, but:", c.kind) }
        if len(c.children) != 2 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.get(c)) }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if l.get(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.get(cc)+"'") }
        // arg 1
        if cc = c.children[1]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if l.get(cc) != "$(foo)$(bar)" { t.Error("expecting '$(foo)$(bar)', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 2 { t.Error("expecting 2 children, but:", len(cc.children), "'"+l.get(cc)+"'") }
        // arg 1 -> child 1, 2
        if l.get(cc.children[0]) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.get(cc.children[0])+"'") }
        if l.get(cc.children[1]) != "$(bar)" { t.Error("expecting '$(bar)', but:", "'"+l.get(cc.children[1])+"'") }
        if cc.children[0].kind != node_call { t.Error("expecting call, but:", cc.children[0].kind) }
        if cc.children[1].kind != node_call { t.Error("expecting call, but:", cc.children[1].kind) }

        if c = l.nodes[5]; c.kind != node_call { t.Error("expecting call node, but:", c.kind) }
        if len(c.children) != 2 { t.Error("expecting 2 children, but:", len(c.children), ", node:", l.get(c)) }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if l.get(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        // arg 1
        if cc = c.children[1]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != "$(foobar)" { t.Error("expecting '$(foobar)', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }

        if c = l.nodes[6]; c.kind != node_call { t.Error("expecting call node, but:", c.kind) }
        if len(c.children) != 4 { t.Error("expecting 4 children, but:", len(c.children), ", node:", l.get(c)) }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if l.get(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        // arg 1
        if cc = c.children[1]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        // arg 2
        if cc = c.children[2]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != "$(bar)" { t.Error("expecting '$(foo)', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        // arg 3
        if cc = c.children[3]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != "$(foobar)" { t.Error("expecting '$(foo)', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }

        if c = l.nodes[7]; c.kind != node_call { t.Error("expecting call node, but:", c.kind) }
        if len(c.children) != 3 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.get(c)) }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if l.get(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        // arg 1
        if cc = c.children[1]; cc.kind != node_text { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != "$(foo) $(bar)" { t.Error("expecting '$(foo) $(bar)', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 3 { t.Error("expecting 3 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        // arg 2
        if cc = c.children[2]; cc.kind != node_text { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != " $(foobar) " { t.Error("expecting '$(foobar)', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 3 { t.Error("expecting 2 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }

        if c = l.nodes[8]; c.kind != node_call { t.Error("expecting call node, but:", c.kind) }
        if len(c.children) != 3 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.get(c)) }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if l.get(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        // arg 1
        if cc = c.children[1]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != "$($(foo))" { t.Error("expecting '$($(foo))', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        if cc = cc.children[0]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        if l.get(cc) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.get(cc)+"'") }
        // arg 2
        if cc = c.children[2]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if l.get(cc) != "$($($(foo)$(bar)))" { t.Error("expecting '$($($(foo)$(bar)))', but:", "'"+l.get(cc)+"'") }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        if cc = cc.children[0]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind) }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.get(cc)+"'") }
        if l.get(cc) != "$($(foo)$(bar))" { t.Error("expecting '$($(foo)$(bar))', but:", "'"+l.get(cc)+"'") }
        if cc = cc.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind) }
        if l.get(cc.children[0]) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.get(cc.children[0])+"'") }
        if l.get(cc.children[1]) != "$(bar)" { t.Error("expecting '$(bar)', but:", "'"+l.get(cc.children[1])+"'") }
        if cc.children[0].kind != node_call { t.Error("expecting call node, but:", cc.children[0].kind) }
        if cc.children[1].kind != node_call { t.Error("expecting call node, but:", cc.children[1].kind) }
}

func TestParse(t *testing.T) {
        var s = `
a = a
i = i
sh$ared = shared
stat$ic = static
a$$a = foo
aaaa = xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx
bbbb := xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx
cccc = xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
dddd := xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
`
        p := newTestParser("TestParse", s)

        if err := p.parse(); err != nil {
                t.Error("parse failed: %v", err); return
        }

        nd := p.l.nodes[5]
        if s := p.l.get(nd.children[0]); s != "aaaa" { t.Error("expect aaaa, but", s) }
        if s := p.l.get(nd.children[1]); s != "=" { t.Error("expect =, but", s) }
        if s := p.l.get(nd.children[2]); s != "xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx" { t.Error("expect xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx, but", s) }
        if nd = nd.children[2]; nd == nil {
                t.Error("expecting a node, but nil")
        } else {
                if num := len(nd.children); num != 5 { t.Error("expecting 5 children for 'aaaa', but", num) }
                if s := p.l.get(nd.children[0]); s != "xxx" { t.Error("expecting xxx for 'aaaa', but", s) }
                if s := p.l.get(nd.children[1]); s != "$(info $(sh$ared),$(stat$ic))" { t.Error("expecting $(info $(sh$ared),$(stat$ic)) for 'aaaa', but", s) }
                if s := p.l.get(nd.children[2]); s != "-" { t.Error("expecting - for 'aaaa', but", s) }
                if s := p.l.get(nd.children[3]); s != "$(a$$a)" { t.Error("expecting $(a$$a) for 'aaaa', but", s) }
                if s := p.l.get(nd.children[4]); s != "-xxx" { t.Error("expecting -xxx for 'aaaa', but", s) }
        }

        checkVar := func(name, value string) {
                if v, ok := p.variables[name]; !ok {
                        t.Error(name, "does not exist");
                } else if v.value != value {
                        t.Error(name, "is", v.value, ", but expect", value);
                }
        }

        checkVar("a", "a")
        checkVar("i", "i")
        checkVar("shared", "shared")
        checkVar("static", "static")
        checkVar("a$a", "foo")
        checkVar("aaaa", "xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx")
        checkVar("bbbb", "xxx-foo-xxx")
        checkVar("cccc", "xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx")
        checkVar("dddd", "xxx-shared-static-foo-xxx")
}
