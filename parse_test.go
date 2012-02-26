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

        if len(l.nodes) != 16 { t.Error("expecting 16 nodes but", len(l.nodes)); return }

        count := 0
        for _, n := range l.nodes { if n.kind == node_comment { count += 1 } }
        if count != 9 { t.Error("expecting 9 comments but", count); return }

        var c *node

        checkNode := func(n int, k nodeType, s string) (quit bool) {
                if len(l.nodes) < n+1 { t.Error("expecting at list", n+1, "nodes but", len(l.nodes)); return true }
                if c = l.nodes[n]; c.kind != k { t.Error("node", n ,"is not comment:", c.kind) }
                if l.str(c) != s { t.Error("node", n, "is:", c.kind, "'"+l.str(c)+"'", ", not", "'"+s+"'") }
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

        if len(l.nodes) != 6 { t.Error("expecting 6 nodes but", len(l.nodes)); return }
        
        count := 0
        for _, n := range l.nodes {
                if n.kind == node_assign || n.kind == node_simple_assign { count += 1 }
        }
        if count != 6 { t.Error("expecting 6 assigns, but", count); return }

        checkNode := func(c *node, k nodeType, cc int, s string, cs ...string) {
                if c.kind != k { t.Error("expecting", k, ", but", c.kind); return }
                if len(c.children) != cc { t.Error("expecting", cc, " children, but", len(c.children)); return }

                var cn int
                for cn = 0; cn < len(c.children) && cn < len(cs); cn++ {
                        if s := l.str(c.children[cn]); s != cs[cn] {
                                t.Error("expected child", cn, "'"+cs[cn]+"'", ", but '"+s+"',", "in '"+l.str(c)+"'")
                                break
                        }
                }
                if cn != len(cs) { t.Error("expecting at least", len(cs), "children, but", cn); return }
        }

        var c *node
        c = l.nodes[0]; checkNode(c, node_assign, 3, `a = a`, "a", "=", "a")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind); return }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind); return }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind); return }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn); return }

        c = l.nodes[1]; checkNode(c, node_assign, 3, `b= b`, "b", "=", "b")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind); return }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind); return }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind); return }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn); return }

        c = l.nodes[2]; checkNode(c, node_assign, 3, `c=c`, "c", "=", "c")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind); return }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind); return }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind); return }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn); return }

        c = l.nodes[3]; checkNode(c, node_assign, 3, `d       =           d`, "d", "=", "d")
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind); return }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind); return }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind); return }
        if cn := len(c.children[2].children); cn != 0 { t.Error("expect child number 0, but:", cn); return }

        c = l.nodes[4]; checkNode(c, node_simple_assign, 3, `foo := $(a) \
 $b\
 ${c}\
`, "foo", ":=", `$(a) \
 $b\
 ${c}\
`)
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind); return }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind); return }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind); return }
        if cn := len(c.children[2].children); cn != 9 { t.Error("expect 9 children, but:", cn); return }

        c = l.nodes[5]; checkNode(c, node_assign, 3, `bar = $(foo) \
$(a) \
 $b $c`, `bar`, `=`, `$(foo) \
$(a) \
 $b $c`)
        if c.children[0].kind != node_text { t.Error("expect child 0 to be text, but", c.children[0].kind); return }
        if c.children[1].kind != node_text { t.Error("expect child 1 to be text, but", c.children[1].kind); return }
        if c.children[2].kind != node_text { t.Error("expect child 2 to be text, but", c.children[2].kind); return }
        if cn := len(c.children[2].children); cn != 10 { t.Error("expect 10 children, but:", cn); return }
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

        if len(l.nodes) != 9 { t.Error("expecting 8 nodes but", len(l.nodes)); return }
        
        count := 0
        for _, n := range l.nodes { if n.kind == node_call { count += 1 } }
        if count != 5 { t.Error("expecting 4 calls, but", count); return }

        var c, cc *node

        if c = l.nodes[2]; c.kind != node_simple_assign { t.Error("expecting assign node, but:", c.kind); return }
        if l.str(c) != `foobar := $(foo)$(bar)` { t.Error("expecting 'foobar := $(foo)$(bar)', but:", "'"+l.str(c)+"'"); return }
        if len(c.children) != 3 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.str(c)); return }
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if cc = c.children[1]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if cc = c.children[2]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if len(cc.children) != 2 { t.Error("expecting 2 children, but:", len(cc.children)); return }
        if cc.children[0].kind != node_call { t.Error("expecting call, but:", cc.children[0].kind, l.str(cc.children[0])); return }
        if cc.children[1].kind != node_call { t.Error("expecting call, but:", cc.children[1].kind, l.str(cc.children[1])); return }
        if l.str(cc.children[0]) != `$(foo)` { t.Error("expecting '$(foo)', but:", "'"+l.str(cc.children[0])+"'"); return }
        if l.str(cc.children[1]) != `$(bar)` { t.Error("expecting '$(bar)', but:", "'"+l.str(cc.children[1])+"'"); return }

        if c = l.nodes[4]; c.kind != node_call { t.Error("expecting call node, but:", c.kind); return }
        if len(c.children) != 2 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.str(c)); return }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if l.str(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.str(cc)+"'"); return }
        // arg 1
        if cc = c.children[1]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if l.str(cc) != "$(foo)$(bar)" { t.Error("expecting '$(foo)$(bar)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 2 { t.Error("expecting 2 children, but:", len(cc.children), "'"+l.str(cc)+"'"); return }
        // arg 1 -> child 1, 2
        if l.str(cc.children[0]) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.str(cc.children[0])+"'"); return }
        if l.str(cc.children[1]) != "$(bar)" { t.Error("expecting '$(bar)', but:", "'"+l.str(cc.children[1])+"'"); return }
        if cc.children[0].kind != node_call { t.Error("expecting call, but:", cc.children[0].kind); return }
        if cc.children[1].kind != node_call { t.Error("expecting call, but:", cc.children[1].kind); return }

        if c = l.nodes[5]; c.kind != node_call { t.Error("expecting call node, but:", c.kind); return }
        if len(c.children) != 2 { t.Error("expecting 2 children, but:", len(c.children), ", node:", l.str(c)); return }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if l.str(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        // arg 1
        if cc = c.children[1]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$(foobar)" { t.Error("expecting '$(foobar)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }

        if c = l.nodes[6]; c.kind != node_call { t.Error("expecting call node, but:", c.kind); return }
        if len(c.children) != 4 { t.Error("expecting 4 children, but:", len(c.children), ", node:", l.str(c)); return }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if l.str(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        // arg 1
        if cc = c.children[1]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        // arg 2
        if cc = c.children[2]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$(bar)" { t.Error("expecting '$(foo)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        // arg 3
        if cc = c.children[3]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$(foobar)" { t.Error("expecting '$(foo)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }

        if c = l.nodes[7]; c.kind != node_call { t.Error("expecting call node, but:", c.kind); return }
        if len(c.children) != 3 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.str(c)); return }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if l.str(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        // arg 1
        if cc = c.children[1]; cc.kind != node_text { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$(foo) $(bar)" { t.Error("expecting '$(foo) $(bar)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 3 { t.Error("expecting 3 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        // arg 2
        if cc = c.children[2]; cc.kind != node_text { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != " $(foobar) " { t.Error("expecting '$(foobar)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 3 { t.Error("expecting 2 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }

        if c = l.nodes[8]; c.kind != node_call { t.Error("expecting call node, but:", c.kind); return }
        if l.str(c) != "$(info $($(foo)),$($($(foo)$(bar))))" { t.Error("expecting '$(info $($(foo)),$($($(foo)$(bar))))', but:", "'"+l.str(c)+"'"); return }
        if len(c.children) != 3 { t.Error("expecting 3 children, but:", len(c.children), ", node:", l.str(c)); return }
        // name
        if cc = c.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if len(cc.children) != 0 { t.Error("expecting 0 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        if l.str(cc) != "info" { t.Error("expecting 'info', but:", "'"+l.str(cc)+"'"); return }
        // arg 1
        if cc = c.children[1]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$($(foo))" { t.Error("expecting '$($(foo))', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        if cc = cc.children[0]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        // arg 2
        if cc = c.children[2]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$($($(foo)$(bar)))" { t.Error("expecting '$($($(foo)$(bar)))', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        if cc = cc.children[0]; cc.kind != node_call { t.Error("expecting call node, but:", cc.kind); return }
        if l.str(cc) != "$($(foo)$(bar))" { t.Error("expecting '$($(foo)$(bar))', but:", "'"+l.str(cc)+"'"); return }
        if len(cc.children) != 1 { t.Error("expecting 1 children, but:", len(cc.children), ", '"+l.str(cc)+"'"); return }
        if cc = cc.children[0]; cc.kind != node_text { t.Error("expecting text node, but:", cc.kind); return }
        if l.str(cc.children[0]) != "$(foo)" { t.Error("expecting '$(foo)', but:", "'"+l.str(cc.children[0])+"'"); return }
        if l.str(cc.children[1]) != "$(bar)" { t.Error("expecting '$(bar)', but:", "'"+l.str(cc.children[1])+"'"); return }
        if cc.children[0].kind != node_call { t.Error("expecting call node, but:", cc.children[0].kind); return }
        if cc.children[1].kind != node_call { t.Error("expecting call node, but:", cc.children[1].kind); return }
}

func TestParse(t *testing.T) {
        var s = `
a = a
i = i
ii = i $a i a \
 $a i
sh$ared = shared
stat$ic = static
a$$a = foo
aaaa = xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx
bbbb := xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx
cccc = xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
dddd := xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
`
        p := newTestParser("TestParse", s)

        if err := p.parse(); err != nil { t.Error("parse error:", err); return }

        var nd *node

        nd = p.l.nodes[0]
        if s := p.l.str(nd); s != "a = a" { t.Error("expect 'a = a', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[0]); s != "a" { t.Error("expect 'a', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[1]); s != "=" { t.Error("expect '=', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[2]); s != "a" { t.Error("expect 'a', but", "'"+s+"'"); return }

        nd = p.l.nodes[1]
        if s := p.l.str(nd); s != "i = i" { t.Error("expect 'i = i', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[0]); s != "i" { t.Error("expect 'i', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[1]); s != "=" { t.Error("expect '=', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[2]); s != "i" { t.Error("expect 'i', but", "'"+s+"'"); return }

        nd = p.l.nodes[2]
        if s := p.l.str(nd); s != `ii = i $a i a \
 $a i` { t.Error(`expect 'ii = i $a i a \
 $a i', but`, "'"+s+"'"); return }
        if s := p.l.str(nd.children[0]); s != "ii" { t.Error("expect 'ii', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[1]); s != "=" { t.Error("expect '=', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[2]); s != `i $a i a \
 $a i` { t.Error(`expect 'i $a i a \
 $a i', but`, "'"+s+"'"); return }

        nd = p.l.nodes[3]
        if s := p.l.str(nd); s != "sh$ared = shared" { t.Error("expect 'sh$ared = shared', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[0]); s != "sh$ared" { t.Error("expect 'sh$ared', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[1]); s != "=" { t.Error("expect '=', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[2]); s != "shared" { t.Error("expect 'shared', but", "'"+s+"'"); return }

        nd = p.l.nodes[4]
        if s := p.l.str(nd); s != "stat$ic = static" { t.Error("expect 'stat$ic = static', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[0]); s != "stat$ic" { t.Error("expect 'stat$ic', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[1]); s != "=" { t.Error("expect '=', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[2]); s != "static" { t.Error("expect 'static', but", "'"+s+"'"); return }

        nd = p.l.nodes[5]
        if s := p.l.str(nd); s != "a$$a = foo" { t.Error("expect 'a$$a = foo', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[0]); s != "a$$a" { t.Error("expect 'a$$a', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[1]); s != "=" { t.Error("expect '=', but", "'"+s+"'"); return }
        if s := p.l.str(nd.children[2]); s != "foo" { t.Error("expect 'foo', but", "'"+s+"'"); return }

        nd = p.l.nodes[6]
        if s := p.l.str(nd.children[0]); s != "aaaa" { t.Error("expect aaaa, but", s); return }
        if s := p.l.str(nd.children[1]); s != "=" { t.Error("expect =, but", s); return }
        if s := p.l.str(nd.children[2]); s != "xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx" { t.Error("expect xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx, but", s); return }
        if nd = nd.children[2]; nd == nil {
                t.Error("expecting a node, but nil"); return
        } else {
                if num := len(nd.children); num != 5 { t.Error("expecting 5 children for 'aaaa', but", num); return }
                if s := p.l.str(nd.children[0]); s != "xxx" { t.Error("expecting xxx for 'aaaa', but", s); return }
                if s := p.l.str(nd.children[1]); s != "$(info $(sh$ared),$(stat$ic))" { t.Error("expecting $(info $(sh$ared),$(stat$ic)) for 'aaaa', but", s); return }
                if s := p.l.str(nd.children[2]); s != "-" { t.Error("expecting - for 'aaaa', but", s); return }
                if s := p.l.str(nd.children[3]); s != "$(a$$a)" { t.Error("expecting $(a$$a) for 'aaaa', but", s); return }
                if s := p.l.str(nd.children[4]); s != "-xxx" { t.Error("expecting -xxx for 'aaaa', but", s); return }

                var cc *node
                cn := nd.children[1]
                if len(cn.children) != 3 { t.Error("expecting 3 children, but", len(cn.children)); return }
                if s := p.l.str(cn.children[0]); s != "info" { t.Error("expecting 'info', but", s); return }

                cc = cn.children[1]
                if s := p.l.str(cc); s != "$(sh$ared)" { t.Error("expecting '$(sh$ared)', but", s); return }
                if cc.kind != node_call { t.Error("expecting call, but", cc.kind); return }
                if l := len(cc.children); l != 1 { t.Error("expecting 1 child, but", l); return }
                cc = cc.children[0]
                if l := len(cc.children); l != 3 { t.Error("expecting 3 children, but", l, "'"+p.l.str(cc)+"'"); return }
                if ccc := cc.children[0]; p.l.str(ccc) != "sh" { t.Error("expecting sh, but", p.l.str(ccc)); return }
                if ccc := cc.children[1]; p.l.str(ccc) != "$a" { t.Error("expecting $a, but", p.l.str(ccc)); return }
                if ccc := cc.children[2]; p.l.str(ccc) != "red" { t.Error("expecting red, but", p.l.str(ccc)); return }
                
                cc = cn.children[2]
                if s := p.l.str(cc); s != "$(stat$ic)" { t.Error("expecting '$(stat$ic)', but", s); return }
                if cc.kind != node_call { t.Error("expecting call, but", cc.kind); return }
                if l := len(cc.children); l != 1 { t.Error("expecting 1 child, but", l); return }
                cc = cc.children[0]
                if l := len(cc.children); l != 3 { t.Error("expecting 3 children, but", l, "'"+p.l.str(cc)+"'"); return }
                if ccc := cc.children[0]; p.l.str(ccc) != "stat" { t.Error("expecting stat, but", p.l.str(ccc)); return }
                if ccc := cc.children[1]; p.l.str(ccc) != "$i" { t.Error("expecting $i, but", p.l.str(ccc)); return }
                if ccc := cc.children[2]; p.l.str(ccc) != "c" { t.Error("expecting c, but", p.l.str(ccc)); return }
        }

        checkVar := func(name, value string) bool {
                if v, ok := p.variables[name]; !ok {
                        t.Error(name, "does not exist"); return false
                } else if v.value != value {
                        t.Error(name, "is", v.value, ", but expect", value); return false
                }
                return true
        }

        if _, ok := p.variables["aa"]; ok {
                t.Error("should not have 'aa' variable"); return
        }


        if !checkVar("a", "a") { return }
        if !checkVar("i", "i") { return }
        if !checkVar("ii", `i $a i a \
 $a i`) { return }
        if !checkVar("shared", "shared") { return }
        if !checkVar("static", "static") { return }
        if !checkVar("a$a", "foo") { return }
        if !checkVar("aaaa", "xxx$(info $(sh$ared),$(stat$ic))-$(a$$a)-xxx") { return }
        if !checkVar("bbbb", "xxx-foo-xxx") { return }
        if !checkVar("cccc", "xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx") { return }
        if !checkVar("dddd", "xxx-shared-static-foo-xxx") { return }
}
