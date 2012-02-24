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

        countComments := 0
        for _, n := range l.nodes {
                if n.kind == node_comment { countComments += 1 }
        }
        if countComments != 9 { t.Error("expecting 9 comments but", countComments) }

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

func TestParse(t *testing.T) {
        var s = `
a = a
i = i
sh$ared = shared
stat$ic = static
a$$a = foo
xxx$(use $(sh$ared),$(stat$ic))-$(a$$a)-xxx
`
        p := newTestParser("TestParse", s)
        if err := p.parse(); err != nil {
                t.Error("parse failed: %v", err); return
        }
}
