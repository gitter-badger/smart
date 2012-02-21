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
        node_rule
        node_double_colon_rule
        node_call
        node_call_arg
)

var nodeTypeNames = []string {
        node_comment: "comment",
        node_continual: "continual",
        node_spaces: "spaces",
        node_text: "text",
        node_assign: "assign",
        node_rule: "rule",
        node_double_colon_rule: "double-colon-rule",
        node_call: "call",
        node_call_arg: "arg",
}

func (k nodeType) String() string {
        return nodeTypeNames[int(k)]
}

type node struct {
        kind nodeType
        root *node
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
        var r rune
        n := l.new(node_comment, -1)
        for {
                for r != '\n' { if r = l.getRune(); r == 0 { break } }
                if r == '\n' && l.peekRune() == '#' { r = l.getRune(); continue }
                break
        }
        n.end = l.pos
        return n
}

func (l *lex) parseAssign(off int) *node {
        var r rune
        n := l.new(node_assign, off)
        n.children = append(n.children, l.list...)
        l.list = l.list[0:0]
        for {
                if r = l.getRune(); r == 0 || r == '\n' || r == '#' { break }
        }
        n.end = l.pos
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
        n := l.new(node_call, -1)
        rr := l.getRune()
        switch rr {
        case 0: errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos]))
        case '(': rr = ')'
        case '{': rr = '}'
        default: n.end = l.pos; return n
        }
        for {
                if r := l.getRune(); r == 0 {
                        errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos]))
                } else if r == '$' {
                        n.children = append(n.children, l.parseCall())
                } else if r == rr {
                        break
                }
        }
        n.end = l.pos
        return n
}

func (l *lex) parseAny() *node {
main_loop:
        for {
                var r rune
                if r = l.getRune(); r == 0 { break main_loop }
                switch {
                case r == '#': return l.parseComment()
                case r == '\\':
                        switch l.getRune() {
                        case 0: break main_loop
                        case '\n': l.new(node_continual, -2)
                        default: l.ungetRune()
                        }
                case r == '=':
                        return l.parseAssign(-1)
                case r == ':':
                        switch l.getRune() {
                        case '0': break main_loop
                        case '=': return l.parseAssign(-2)
                        case ':': return l.parseDoubleColonRule()
                        default:  l.ungetRune(); return l.parseRule()
                        }
                case r == '$':
                        return l.parseCall()
                case r == '\n':
                        l.nodes, l.list = append(l.nodes, l.list...), l.list[0:0]
                case r != '\n' && unicode.IsSpace(r):
                        n := l.new(node_spaces, -1)
                        for {
                                if r = l.getRune(); r == 0 { break main_loop }
                                if !unicode.IsSpace(r) { l.ungetRune(); break }
                        }
                        n.end = l.pos; return n
                default:
                        n := l.new(node_text, -1)
                        for {
                                if r = l.getRune(); r == 0 { break main_loop }
                                if strings.IndexRune(" $:=", r) != -1 { l.ungetRune(); break }
                        }
                        n.end = l.pos; return n
                }
        }
        return nil
}

func (l *lex) parse() {
        l.lineno, l.colno = 1, 0
        for {
                if n := l.parseAny(); n == nil {
                        break
                } else {
                        l.list = append(l.list, n)
                }
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
        //loc := location{ file:&(p.file), lineno:p.lineno, colno:p.colno+1 }
        loc := p.l.location()

        if name == "this" {
                fmt.Printf("%v:warning: ignore attempts on \"this\"\n", &loc)
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

func (p *parser) parse() (err error) {
        p.l.parse()

        for _, n := range p.l.nodes {
                //fmt.Printf("%v:%v:%v: %v, %v\n", p.l.file, n.lineno, n.colno, n.kind, string(p.l.s[n.pos:n.end]), n.children)
                fmt.Printf("%v:%v:%v: %v, %v children\n", p.l.file, n.lineno, n.colno, n.kind, len(n.children))
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
