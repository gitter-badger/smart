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
        node_assign
        node_simple_assign
        node_question_assign
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

func (l *lex) peekRune() (r rune) {
        if l.pos < len(l.s) {
                r, _ = utf8.DecodeRune(l.s[l.pos:])
        }
        return
}

func (l *lex) get(n *node) string {
        return string(l.s[n.pos:n.end])
}

func (l *lex) getRune() (r rune) {
        if len(l.s) == l.pos { r = 0; return }
        if len(l.s) < l.pos { errorf(-2, "over reading (at %v)", l.pos) }

        l.rune, l.runeLen = utf8.DecodeRune(l.s[l.pos:])
        r, l.pos = l.rune, l.pos+l.runeLen
        switch {
        //case l.rune == 0:
        //        errorf(-2, "zero reading (at %v)", l.pos)
        case l.rune == utf8.RuneError:
                errorf(-2, "bad UTF8 encoding")
        case l.rune == '\n':
                l.lineno, l.prevColno, l.colno = l.lineno+1, l.colno, 0
        case l.runeLen > 1:
                l.colno += 2
        default:
                l.colno += 1
        }
        return
}

func (l *lex) ungetRune() {
        switch {
        case l.rune == 0:
                errorf(0, "wrong invocation of ungetRune")
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

func (l *lex) skip(shouldSkip func(r rune) bool) (err error) {
        var r rune
        for {
                if r = l.getRune(); r == 0 {
                        return
                }
                if shouldSkip(r) {
                        //bytes += rs;
                } else {
                        l.ungetRune(); break
                }
        }
        return
}

func (l *lex) skipRune(r rune) (err error) {
        if v := l.getRune(); r == 0 || v == r {
                return
        } else {
                l.ungetRune()
                errorf(0, "not rune '%v' (%v)", r, v)
        }
        return
}

func (l *lex) skipSpace(inline bool) (err error) {
        e := l.skip(func(r rune) bool {
                if r == '#' {
                        for {
                                if r = l.getRune(); r == 0 {
                                        return false
                                }
                                if r == '\n' {
                                        l.ungetRune(); break
                                }
                        }
                        return true
                }
                if inline {
                        return r != '\n' && unicode.IsSpace(r)
                }
                return unicode.IsSpace(r)
        })
        if err == nil && e != nil { err = e }
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
                for r != '\n' { if r = l.getRune(); r == 0 { break } }
                if r == '\n' && l.peekRune() == '#' { r = l.getRune(); continue }
                break
        }
        n.end = l.pos
        return n
}

func (l *lex) parseText(off int) *node {
        n, r := l.new(node_text, off), rune(0)
        for {
                if r = l.getRune(); r == 0 { break }
                if strings.IndexRune(" $:=", r) != -1 { l.ungetRune(); break }
        }
        n.end = l.pos
        return n
}

func (l *lex) parseSpaces(off int) *node {
        n, r := l.new(node_spaces, off), rune(0)
        for {
                if r = l.getRune(); r == 0 { break }
                if !unicode.IsSpace(r) { l.ungetRune(); break }
        }
        n.end = l.pos
        return n
}

//func (l *lex) parseContinual(off int) *node {
//        return
//}

func (l *lex) parseAssign(at nodeType) *node {
        if len(l.list) == 0 {
                errorf(0, "illigal assignment")
        }

        off := -1
        switch at {
        case node_assign:
        case node_question_assign: fallthrough
        case node_simple_assign: off = -2
        default: errorf(0, "unknown assignment")
        }

        li := 0
        for {
                if l.list[li].kind != node_spaces { break }
                if li += 1; li == len(l.list) { break }
        }

        l.list = l.list[li:]
        li = len(l.list) - 1
        for {
                if l.list[li].kind == node_spaces { l.list = l.list[0:li] }
                if li -= 1; li == 0 { break }
        }

        nn := l.new(node_text, 0)
        nn.children = append(nn.children, l.list...)
        nn.pos, nn.end = l.list[0].pos, l.list[len(l.list)-1].end

        n, r := l.new(node_assign, off), rune(0)
        n.children, l.list = append(n.children, nn), l.list[0:0]

        nn = l.new(node_text, 0)
        n.children = append(n.children, nn)

out_loop: for {
                r = l.getRune()
        the_sw: switch {
                case r == 0 || r == '\n' || r == '#':
                        nn.end = l.pos - 1
                        if 0 < len(nn.children) {
                                lc := nn.children[len(nn.children)-1]
                                if lc.end < nn.end {
                                        t := l.new(node_text, lc.end-1-nn.end); t.end = nn.end
                                        if l.s[t.pos] == ' ' { break out_loop }
                                        //print("t: "+l.get(t)+"\n")
                                        nn.children = append(nn.children, t)
                                }
                        }
                        if r == '#' { l.nodes = append(l.nodes, l.parseComment()) }
                        break out_loop
                case r == ' ':
                        ss := l.parseSpaces(-1)
                        if 0 == len(nn.children) {
                                if nn.pos == ss.pos /*|| nn.pos == nn.end*/ {
                                        nn.pos = ss.end
                                } else {
                                        t := l.new(node_text, nn.pos-ss.pos-1); t.end = ss.pos
                                        //print("t: "+l.get(t)+"\n")
                                        nn.children = append(nn.children, t)
                                }
                        } else {
                                nn.children = append(nn.children, ss)
                        }
                case r == '$':
                        if 0 == len(nn.children) && nn.pos != l.pos-1 {
                                t := l.new(node_text, nn.pos-l.pos); t.end = l.pos-1
                                //print("t: "+l.get(t)+"\n")
                                nn.children = append(nn.children, t)
                        }
                        nn.children = append(nn.children, l.parseCall())
                case r == '\\':
                        switch l.getRune() {
                        case 0: break out_loop
                        case '\n':
                                if 0 == len(nn.children) && nn.pos != l.pos-2 {
                                        t := l.new(node_text, nn.pos-l.pos-1); t.end = l.pos-2
                                        //print("t: "+l.get(t)+"\n")
                                        nn.children = append(nn.children, t)
                                }
                                nn.children = append(nn.children, l.new(node_continual, -2))
                        default: goto the_sw //l.ungetRune()
                        }
                }
        }
        n.end = nn.end

        //fmt.Printf("assign: '%v', '%v'\n", string(l.s[n.children[0].pos:n.children[0].end]), string(l.s[n.children[1].pos:n.children[len(n.children)-1].end]))
        //fmt.Printf("assign: '%v'\n", string(l.s[n.children[0].pos:n.children[0].end]))
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

func (l *lex) parseCall() *node {
        n, r := l.new(node_call, -1), rune(0)
        rr := l.getRune()
        switch rr {
        case 0: errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos]))
        case '(': rr = ')'
        case '{': rr = '}'
        default:
                n.children, n.end = append(n.children, l.new(node_text, -1)), l.pos
                return n
        }
        nn, parentheses := l.new(node_text, 0), []rune{}
out_loop: for {
                r = l.getRune()
                switch {
                case r == 0: errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos]))
                case r == '(': parentheses = append(parentheses, ')')
                case r == '{': parentheses = append(parentheses, '}')
                case r == rr:
                        if 0 < len(parentheses) && rr == parentheses[len(parentheses)-1] {
                                parentheses = parentheses[0:len(parentheses)-1]
                                break
                        }
                        fallthrough
                case r == ' ' && len(n.children) == 0: fallthrough
                case r == ',':
                        if 0 < len(nn.children) {
                                lc := nn.children[len(nn.children)-1]
                                c := l.new(node_text, 0)
                                c.pos, c.end = lc.end + 1, l.pos
                                nn.children = append(nn.children, c)
                                //fmt.Printf("%v: call: %v\n", l.location(), len(nn.children))
                        }
                        nn.end, n.children = l.pos, append(n.children, nn)
                        if r == rr {
                                break out_loop
                        } else {
                                nn = l.new(node_text, 0)
                        }
                case r == '$': nn.children = append(nn.children, l.parseCall())
                case r == ' ' && len(n.children) > 0: nn.children = append(nn.children, l.parseSpaces(-1))
                }
        }
        n.end = l.pos
        return n
}

func (l *lex) parse() {
        l.lineno, l.colno = 1, 0
main_loop:
        for {
                var r rune
                if r = l.getRune(); r == 0 { break main_loop }
        the_sw: switch {
                case r == '#': l.nodes = append(l.nodes, l.parseComment())
                case r == '\\':
                        switch l.getRune() {
                        case 0: break main_loop
                        case '\n': l.list = append(l.list, l.new(node_continual, -2))
                        default: goto the_sw //l.ungetRune()
                        }
                case r == '=': l.nodes = append(l.nodes, l.parseAssign(node_assign))
                case r == '?':
                        switch l.getRune() {
                        case 0: break main_loop
                        case '=': l.list = append(l.list, l.parseAssign(node_question_assign))
                        default:  l.ungetRune(); l.list = append(l.list, l.parseRule())
                        }
                case r == ':':
                        switch l.getRune() {
                        case 0: break main_loop
                        case '=': l.list = append(l.list, l.parseAssign(node_simple_assign))
                        case ':': l.list = append(l.list, l.parseDoubleColonRule())
                        default:  l.ungetRune(); l.list = append(l.list, l.parseRule())
                        }
                case r == '$': l.list = append(l.list, l.parseCall())
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
                dir := filepath.Dir(p.l.file)
                str := p.expand(s.value)
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
        if len(n.children) == 0 {
                if n.kind == node_call {
                        errorf(0, "bad call: %v", p.l.get(n))
                }
                return p.l.get(n)
        }
        if n.kind == node_call {
                nn := p.expandNode(n.children[0])
                fmt.Printf("call: %v\n", nn)
        }
        return ""
}

func (p *parser) processNode(n *node) (err error) {
        switch n.kind {
        case node_assign:
                nn, nv := n.children[0], n.children[1]
                fmt.Printf("%v:%v:%v: %v = %v\n", p.l.file, n.lineno, n.colno, p.l.get(nn), p.l.get(nv))
                switch p.l.s[n.pos] {
                case ':': p.setVariable(p.l.get(nn), p.expandNode(nv))
                case '=': p.setVariable(p.l.get(nn), p.l.get(nv))
                }
        case node_call: //fmt.Printf("%v:%v:%v: %v, %v children\n", p.l.file, n.lineno, n.colno, n.kind, len(n.children))
        }
        return
}

func (p *parser) parse() (err error) {
        p.l.parse()

        for _, n := range p.l.nodes {
                //fmt.Printf("%v: %v, %v children\n", p.l.location(), n.kind, len(n.children))
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
