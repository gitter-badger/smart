package smart

import (
        "bytes"
        //"errors"
        "fmt"
        "unicode"
        "unicode/utf8"
        "io/ioutil"
        "os"
        "path/filepath"
        //"reflect"
        "strings"
)

type location struct {
        file *string
        lineno int
        colno int
}

func (l *location) String() string {
        return fmt.Sprintf("%v:%v:%v", *l.file, l.lineno, l.colno)
}

type variable struct {
        name string
        value string
        loc location
        readonly bool
}

type nodeType int

const (
        node_comment nodeType = iota
        node_continual
        node_spaces
        node_text
        node_assign             // =
        node_simple_assign      // :=
        node_question_assign    // ?=
        node_add_assign         // +=
        node_rule
        node_double_colon_rule
        node_call
)

var nodeTypeNames = []string {
        node_comment: "comment",
        node_continual: "continual",
        node_spaces: "spaces",
        node_text: "text",
        node_assign: "assign",
        node_simple_assign: "node-simple-assign",
        node_question_assign: "node-question-assign",
        node_rule: "rule",
        node_double_colon_rule: "double-colon-rule",
        node_call: "call",
}

func (k nodeType) String() string {
        return nodeTypeNames[int(k)]
}

type node struct {
        kind nodeType
        children []*node
        pos, end, lineno, colno int
}

func (n *node) len() int {
        return n.end - n.pos
}

type lex struct {
        file string
        s []byte // the content of the file
        pos int // the current read position
        start int // begins of the current node
        lnod *node // the end position of the last node
        rune rune // the rune last time returned by getRune
        runeLen int // the size in bytes of the rune last returned by getRune
        lineno, colno, prevColno int
        nodes, list, stack []*node // top level parsed node, and temporary parse list, and stack
}

func (l *lex) location() *location {
        return &location{ &l.file, l.lineno, l.colno }
}

func (l *lex) str(n *node) string {
        return string(l.s[n.pos:n.end])
}

func (l *lex) peek() (r rune) {
        if l.pos < len(l.s) {
                r, _ = utf8.DecodeRune(l.s[l.pos:])
        }
        return
}

func (l *lex) get() bool {
        if len(l.s) == l.pos { return false }
        if len(l.s) < l.pos { errorf(-2, "over reading (at %v)", l.pos) }

        l.rune, l.runeLen = utf8.DecodeRune(l.s[l.pos:])
        l.pos = l.pos+l.runeLen
        switch {
        case l.rune == 0:
                return false //errorf(-2, "zero reading (at %v)", l.pos)
        case l.rune == utf8.RuneError:
                errorf(-2, "invalid UTF8 encoding")
        case l.rune == '\n':
                l.lineno, l.prevColno, l.colno = l.lineno+1, l.colno, 0
        case l.runeLen > 1:
                l.colno += 2
        default:
                l.colno += 1
        }
        return true
}

func (l *lex) unget() {
        switch {
        case l.rune == 0:
                errorf(0, "wrong invocation of unget")
        case l.pos == 0:
                errorf(0, "get to the beginning of the bytes")
        case l.pos < 0:
                errorf(0, "get to the front of beginning of the bytes")
                //case l.lineno == 1 && l.colno <= 1: return
        }
        if l.rune == '\n' {
                l.lineno, l.colno, l.prevColno = l.lineno-1, l.prevColno, 0
        } else {
                l.colno--
        }
        // assert(utf8.RuneLen(l.rune) == l.runeLen)
        l.pos, l.rune, l.runeLen = l.pos-l.runeLen, 0, 0
        return
}

func (l *lex) new(t nodeType, off int) *node {
        pos := l.pos + off
        n := &node{ kind:t, pos:pos, end:l.pos, lineno:l.lineno, colno:l.colno }
        l.lnod = n
        return n
}

func (l *lex) parseComment() *node {
        n, r := l.new(node_comment, -1), rune(0)
        for {
                for r != '\n' { if !l.get() { break } else { r = l.rune } }
                if r == '\n' && l.peek() == '#' {
                        if l.get() { r = l.rune; continue }
                } else if r == '\n' {
                        // return the '\n' because the consequenced node may need
                        // this as separator.
                        l.unget()
                }
                break
        }
        n.end = l.pos

        //fmt.Printf("%v:%v: %v, '%v'\n", l.file, n.lineno, n.kind, l.str(n))
        return n
}

func (l *lex) parseText(off int) *node {
        n := l.new(node_text, off)
        for {
                if !l.get() { break }
                if strings.IndexRune(" $:=", l.rune) != -1 { l.unget(); break }
        }
        n.end = l.pos
        return n
}

func (l *lex) parseSpaces(off int) *node {
        n := l.new(node_spaces, off)
        for {
                if !l.get() { break }
                if !unicode.IsSpace(l.rune) { l.unget(); break }
        }
        n.end = l.pos
        return n
}

//func (l *lex) parseContinual(off int) *node {
//        return
//}

func (l *lex) parseCall() *node {
        n, rr := l.new(node_call, -1), rune(0)
        if !l.get() { errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos])) }
        switch l.rune {
        case '(': rr = ')'
        case '{': rr = '}'
        default:
                n.children, n.end = append(n.children, l.new(node_text, -1)), l.pos
                return n
        }
        nn, t, parentheses := l.new(node_text, 0), l.new(node_text, 0), []rune{}
out_loop: for {
                if !l.get() { errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos])) }
                switch {
                default: t.end = l.pos
                case l.rune == 0: 
                case l.rune == '(': parentheses = append(parentheses, ')')
                case l.rune == '{': parentheses = append(parentheses, '}')
                case l.rune == '$':
                        c := l.parseCall()
                        if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(node_text, 0) }
                        nn.children = append(nn.children, c)
                case l.rune == rr:
                        if 0 < len(parentheses) && rr == parentheses[len(parentheses)-1] {
                                parentheses = parentheses[0:len(parentheses)-1]; break
                        }
                        fallthrough
                case l.rune == ' ':
                        fallthrough
                case l.rune == ',':
                        nn.end = l.pos-1

                        if l.rune == ' ' && 0 < len(n.children) {
                                nn.children = append(nn.children, l.parseSpaces(-1))
                                break
                        }else if 0 < t.len() {
                                nn.children, t = append(nn.children, t), l.new(node_text, 0)
                        }

                        if len(nn.children) == 1 {
                                n.children = append(n.children, nn.children[0])
                        } else {
                                n.children = append(n.children, nn)
                        }

                        if l.rune == rr {
                                break out_loop
                        } else {
                                nn = l.new(node_text, 0)
                        }
                }
        }
        n.end = l.pos
        return n
}

func (l *lex) parseAssign(at nodeType) *node {
        off := -1
        switch at {
        case node_assign:
        case node_question_assign: fallthrough
        case node_simple_assign: off = -2
        default: errorf(0, "unknown assignment")
        }

        for cn, c := range l.list {
                //fmt.Printf("assign:1: %v '%v'\n", c.kind, l.str(c))
                if c.kind != node_spaces { l.list = l.list[cn:]; break }
        }

        for li := len(l.list)-1; 0 <= li; li-- {
                //fmt.Printf("assign:2: %v '%v'\n", l.list[li].kind, l.str(l.list[li]))
                if l.list[li].kind != node_spaces { l.list = l.list[0:li+1]; break }
        }

        if len(l.list) == 0 {
                errorf(0, "illigal assignment with no variable name")
        }

        // the name
        nn := l.new(node_text, 0)
        nn.children = append(nn.children, l.list...)
        nn.pos, nn.end, l.list = l.list[0].pos, l.list[len(l.list)-1].end, l.list[0:0]

        // 'n' is the whole assign statemetn, e.g. "foo = bar"
        n := l.new(at, nn.pos-l.pos)
        n.children = append(n.children, nn)
        //fmt.Printf("assign: '%v'\n", l.str(n))

        // the equal signs: '=', ':=', '?='
        nn = l.new(node_text, off)
        nn.end = nn.pos - off
        n.children = append(n.children, nn)

        // value
        nn, t := l.new(node_text, 0), l.new(node_text, 0)
        n.children = append(n.children, nn)

        var r rune
out_loop: for {
                if !l.get() { r = 0 } else { r = l.rune }
        the_sw: switch {
                default: t.end = l.pos
                case r == 0 || r == '\n' || r == '#':
                        if r != 0 { l.unget() }
                        if 0 < t.len() { nn.children = append(nn.children, t) }
                        nn.end = l.pos
                        break out_loop
                case r == ' ':
                        ss := l.parseSpaces(-1)
                        if ss.pos == nn.pos { nn.pos, t.pos, t.end = ss.end, ss.end, ss.end } // ignore the first space just after '='
                        if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(node_text, 0) }
                        nn.children = append(nn.children, ss)
                case r == '$':
                        cc := l.parseCall()
                        if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(node_text, 0) }
                        nn.children = append(nn.children, cc)
                case r == '\\':
                        if !l.get() { break out_loop }
                        switch l.rune {
                        case '\n':
                                if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(node_text, 0) }
                                cn := l.new(node_continual, -2); cn.end = cn.pos+2
                                nn.children = append(nn.children, cn)
                                break
                        default: r = l.rune; goto the_sw
                        }
                }
        }

        for cn, c := range nn.children {
                if c.kind != node_spaces { nn.children = nn.children[cn:]; break }
        }
        
        if len(nn.children) == 1 /*&& nn.children[0].kind == nn.kind*/ {
                nn.children = nn.children[0:0]
        }

        n.end = nn.end

        //fmt.Printf("%v:%v: %v, '%v', '%v'\n", l.file, n.lineno, n.kind, l.str(n), l.str(nn))
        return n
}

func (l *lex) parseDoubleColonRule() *node {
        n := l.new(node_double_colon_rule, -2)
        fmt.Printf("%v:%v: '%v'\n", l.file, n.lineno, n.kind)
        return n
}

func (l *lex) parseRule() *node {
        n := l.new(node_rule, -1)
        fmt.Printf("%v:%v: '%v'\n", l.file, n.lineno, n.kind)
        return n
}

func (l *lex) parse() {
        l.lineno, l.colno = 1, 0
        var r rune
main_loop:
        for {
                if !l.get() { break main_loop } else { r = l.rune }
        the_sw: switch {
                case r == '#':
                        l.nodes = append(l.nodes, l.parseComment())
                case r == '\\':
                        if !l.get() { break main_loop }
                        switch l.rune {
                        case '\n': l.list = append(l.list, l.new(node_continual, -2))
                        default: r = l.rune; goto the_sw
                        }
                case r == '=':
                        l.nodes = append(l.nodes, l.parseAssign(node_assign))
                case r == '?':
                        if !l.get() { break main_loop }
                        switch l.rune {
                        case '=': l.nodes = append(l.nodes, l.parseAssign(node_question_assign))
                        default: r = l.rune; goto the_sw
                        }
                case r == ':':
                        if !l.get() { break main_loop }
                        switch l.rune {
                        case '=': l.nodes = append(l.nodes, l.parseAssign(node_simple_assign))
                        case ':': l.nodes = append(l.nodes, l.parseDoubleColonRule())
                        default:  l.unget(); l.nodes = append(l.nodes, l.parseRule())
                        }
                case r == '$':
                        l.list = append(l.list, l.parseCall())
                case r == '\n':
                        l.nodes, l.list = append(l.nodes, l.list...), l.list[0:0]
                case r != '\n' && unicode.IsSpace(r):
                        l.list = append(l.list, l.parseSpaces(-1))
                default:
                        l.list = append(l.list, l.parseText(-1))
                }
                if len(l.s) == l.pos { break }
        }
        return
}

type parser struct {
        l lex
        module *module
        line bytes.Buffer // line accumulator
        variables map[string]*variable
}

func (p *parser) setModule(m *module) (prev *module) {
        prev = p.module
        p.module = m
        return
}

func (p *parser) getModuleSources() (sources []string) {
        if p.module == nil {
                return
        }

        if s, ok := p.module.variables["this.sources"]; ok {
                dir, str := p.module.dir, p.expand(s.value)
                sources = strings.Split(str, " ")
                for i, _ := range sources {
                        if sources[i][0] == '/' { continue }
                        sources[i] = filepath.Join(dir, sources[i])
                }
        }
        return
}

func (p *parser) expand(str string) string {
        var buf bytes.Buffer
        var exp func(s []byte) (out string, l int)
        var getRune = func(s []byte) (r rune, l int) {
                if r, l = utf8.DecodeRune(s); r == utf8.RuneError || l <= 0 {
                        errorf(1, "bad UTF8 encoding")
                }
                return
        }

        exp = func(s []byte) (out string, l int) {
                var r, rr rune
                var rs = 0

                r, rs = getRune(s); s, l = s[rs:], l + rs
                switch r {
                case '(': rr = ')'
                case '{': rr = '}'
                case '$': out = "$"; return // for "$$"
                }

                var name string
                var args []string
                var t bytes.Buffer
                if rr == 0 {
                        t.WriteRune(r)
                        out = p.call(t.String(), args...)
                        return
                }

                var parentheses []rune
                for 0 < len(s) {
                        r, rs = getRune(s)

                        switch r {
                        default:  t.WriteRune(r)
                        case ' ':
                                if name == "" {
                                        name = t.String(); t.Reset()
                                } else {
                                        t.WriteRune(r); break
                                }
                        case ',':
                                args = append(args, t.String()); t.Reset()
                        case '$':
                                //fmt.Printf("inner: %v, %v, %v\n", string(s), rs, l)
                                if ss, ll := exp(s[rs:]); 0 < ll {
                                        t.WriteString(ss)
                                        s, l = s[rs+ll:], l + rs + ll
                                        //fmt.Printf("inner: %v, %v, %v, %v\n", string(s), ll, ss, rs)
                                        continue
                                } else {
                                        errorf(1, string(s))
                                }
                        case '(': t.WriteRune(r); parentheses = append(parentheses, ')')
                        case '{': t.WriteRune(r); parentheses = append(parentheses, '}')
                        case rr:
                                if 0 < len(parentheses) && rr == parentheses[len(parentheses)-1] {
                                        parentheses = parentheses[0:len(parentheses)-1]
                                        t.WriteRune(r); break
                                }
                                if 0 < t.Len() {
                                        if 0 < len(name) /*0 < len(args)*/ {
                                                args = append(args, t.String())
                                        } else {
                                                name = t.String()
                                        }
                                        t.Reset()
                                }
                                //fmt.Printf("expcall: %v, %v, %v, %v\n", name, string(s[0:rs]), string(s[rs:]), rs)
                                out, l = p.call(name, args...), l + rs
                                return /* do not "break" */
                        }

                        //fmt.Printf("exp: %v, %v, %v, %v\n", name, args, string(s[0:rs]), rs)

                        s, l = s[rs:], l + rs
                }
                return
        }

        s := []byte(str)
        for 0 < len(s) {
                r, l := getRune(s)
                s = s[l:]
                if r == '$' {
                        if ss, ll := exp(s); ll <= 0 {
                                errorf(0, "bad variable")
                        } else {
                                s = s[ll:]
                                buf.WriteString(ss)
                        }
                } else {
                        buf.WriteRune(r)
                }
        }

        return buf.String()
}

func (p *parser) call(name string, args ...string) string {
        //fmt.Printf("call: %v %v\n", name, args)
        vars := p.variables

        switch {
        default:
                if f, ok := builtins[name]; ok {
                        // All arguments should be expended.
                        for i, _ := range args { args[i] = p.expand(args[i]) }
                        return f(p, args)
                }
        case name == "$": return "$";
        case name == "call":
                if 0 < len(args) {
                        return p.call(args[0], args[1:]...)
                }
                return ""
        case name == "this":
                if p.module != nil {
                        return p.module.name
                } else {
                        return ""
                }
        case strings.HasPrefix(name, "this.") && p.module != nil:
                vars = p.module.variables
        }

        if vars != nil {
                if v, ok := vars[name]; ok {
                        return v.value
                }
        }

        return ""
}

func (p *parser) setVariable(name, value string) (v *variable) {
        loc := p.l.location()

        if name == "this" {
                fmt.Printf("%v:warning: ignore attempts on \"this\"\n", loc)
                return
        }

        vars := p.variables
        if strings.HasPrefix(name, "this.") && p.module != nil {
                vars = p.module.variables
        }
        if vars == nil {
                fmt.Printf("%v:warning: no \"this\" module\n", &loc)
                return
        }

        var has = false
        if v, has = vars[name]; !has {
                v = &variable{}
                vars[name] = v
        }

        if v.readonly {
                fmt.Printf("%v:warning: `%v' is readonly\n", &loc, name)
                return
        }
        
        v.name = name
        v.value = value
        v.loc = *p.l.location()

        //fmt.Printf("%v: '%s' = '%s'\n", &v.loc, name, value)
        return
}

func (p *parser) expandNode(n *node) string {
        //fmt.Printf("%v:%v:%v: expand '%v' (%v)\n", p.l.file, n.lineno, n.colno, p.l.str(n), len(n.children))

        if len(n.children) == 0 {
                switch n.kind {
                case node_comment: errorf(0, "can't expand comment: %v", p.l.str(n))
                case node_call: errorf(0, "invalid call: %v", p.l.str(n))
                case node_continual: return " "
                }
                //fmt.Printf("%v:%v:%v: %v '%v' (%v)\n", p.l.file, n.lineno, n.colno, n.kind, p.l.str(n), len(n.children))
                return p.l.str(n)
        }

        if n.kind == node_call {
                //fmt.Printf("expand: call: %v, %v\n", p.l.str(n), len(n.children))
                name, args := p.expandNode(n.children[0]), []string{}
                for _, an := range n.children[1:] {
                        s := p.expandNode(an); args = append(args, s)
                        //fmt.Printf("%v:%v:%v: arg '%v' ((%v) '%v') (%v)\n", p.l.file, an.lineno, an.colno, p.l.str(an), len(an.children), s, name)
                }
                v := p.call(name, args...)
                //fmt.Printf("%v:%v:%v: call '%v' %v '%v'\n", p.l.file, n.lineno, n.colno, name, args, v)
                return v
        }

        //fmt.Printf("%v:%v:%v: %v '%v' (%v)\n", p.l.file, n.lineno, n.colno, n.kind, p.l.str(n), len(n.children))
        var b bytes.Buffer
        for _, cn := range n.children {
                v := p.expandNode(cn)
                b.WriteString(v)
                //fmt.Printf("%v:%v:%v: %v '%v' '%v'\n", p.l.file, cn.lineno, cn.colno, cn.kind, p.l.str(cn), v)
        }
        return b.String()
}

func (p *parser) processNode(n *node) (err error) {
        //fmt.Printf("%v:%v:%v: node '%v' (%v, %v)\n", p.l.file, n.lineno, n.colno, p.l.str(n), n.kind, len(n.children))

        switch n.kind {
        case node_comment:
        case node_spaces:
        case node_assign:
                nn, nv := n.children[0], n.children[2]
                p.setVariable(p.expandNode(nn), p.l.str(nv))
                //fmt.Printf("%v:%v:%v: %v %v\n", p.l.file, n.lineno, n.colno, p.l.str(nn), p.l.str(nv))
                //fmt.Printf("%v:%v:%v: '%v' '%v'\n", p.l.file, n.lineno, n.colno, p.expandNode(nn), p.l.str(nv))
        case node_simple_assign:
                nn, nv := n.children[0], n.children[2]
                p.setVariable(p.expandNode(nn), p.expandNode(nv))
                //fmt.Printf("%v:%v:%v: %v %v\n", p.l.file, n.lineno, n.colno, p.l.str(nn), p.l.str(nv))
                //fmt.Printf("%v:%v:%v: '%v' '%v'\n", p.l.file, n.lineno, n.colno, p.expandNode(nn), p.expandNode(nv))
        case node_question_assign:
                // TODO: ...
        case node_call:
                //fmt.Printf("%v:%v:%v: call %v\n", p.l.file, n.lineno, n.colno, p.l.str(n))
                if s := p.expandNode(n); s != "" {
                        errorf(0, "illigal: %v (%v)", s, p.l.str(n))
                }
        }
        return
}

func (p *parser) parse() (err error) {
        p.l.parse()

        for _, n := range p.l.nodes {
                if n.kind == node_comment { continue }
                if e := p.processNode(n); e != nil {
                        break
                }
        }
        return
}

func newParser(fn string) (p *parser, err error) {
        var f *os.File

        f, err = os.Open(fn)
        if err != nil {
                return
        }

        defer f.Close()

        s, err := ioutil.ReadAll(f)
        if err != nil {
                return
        }

        p = &parser{
        l: lex{ file: fn, s: s, pos: 0, },
        variables: make(map[string]*variable, 128),
        }

        return
}

func parse(conf string) (p *parser, err error) {
        p, err = newParser(conf)

        defer func() {
                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("%v: %v\n", p.l.location(), se)
                        } else {
                                panic(e)
                        }
                }
        }()

        if err = p.parse(); err != nil {
                return
        }

        return
}
