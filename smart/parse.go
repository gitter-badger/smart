//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
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
        nodeComment nodeType = iota
        nodeContinual
        nodeSpaces
        nodeText
        nodeAssign             // =
        nodeSimpleAssign      // :=
        nodeQuestionAssign    // ?=
        nodeAddAssign         // +=
        nodeRule
        nodeDoubleColonRule
        nodeCall
)

var nodeTypeNames = []string {
        nodeComment: "comment",
        nodeContinual: "continual",
        nodeSpaces: "spaces",
        nodeText: "text",
        nodeAssign: "assign",
        nodeSimpleAssign: "node-simple-assign",
        nodeQuestionAssign: "node-question-assign",
        nodeRule: "rule",
        nodeDoubleColonRule: "double-colon-rule",
        nodeCall: "call",
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
                l.colno ++
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
        n, r := l.new(nodeComment, -1), rune(0)
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
        n := l.new(nodeText, off)
        for {
                if !l.get() { break }
                if strings.IndexRune(" $:=", l.rune) != -1 { l.unget(); break }
        }
        n.end = l.pos
        return n
}

func (l *lex) parseSpaces(off int) *node {
        n := l.new(nodeSpaces, off)
        for {
                if !l.get() { break }
                if l.rune == '\n' || !unicode.IsSpace(l.rune) { l.unget(); break }
        }
        n.end = l.pos
        return n
}

//func (l *lex) parseContinual(off int) *node {
//        return
//}

func (l *lex) parseCall() *node {
        n, rr := l.new(nodeCall, -1), rune(0)
        if !l.get() { errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos])) }
        switch l.rune {
        case '(': rr = ')'
        case '{': rr = '}'
        default:
                n.children, n.end = append(n.children, l.new(nodeText, -1)), l.pos
                return n
        }
        nn, t, parentheses := l.new(nodeText, 0), l.new(nodeText, 0), []rune{}
out_loop: for {
                if !l.get() { errorf(0, "unexpected end of file: '%v'", string(l.s[n.pos:l.pos])) }
                switch {
                default: t.end = l.pos
                case l.rune == 0: 
                case l.rune == '(': parentheses = append(parentheses, ')')
                case l.rune == '{': parentheses = append(parentheses, '}')
                case l.rune == '$':
                        c := l.parseCall()
                        if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(nodeText, 0) }
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
                                nn.children, t = append(nn.children, t), l.new(nodeText, 0)
                        }

                        if len(nn.children) == 1 {
                                n.children = append(n.children, nn.children[0])
                        } else {
                                n.children = append(n.children, nn)
                        }

                        if l.rune == rr {
                                break out_loop
                        } else {
                                nn = l.new(nodeText, 0)
                        }
                }
        }
        n.end = l.pos
        return n
}

func (l *lex) parseAssign(at nodeType) *node {
        off := -1
        switch at {
        case nodeAssign:
        case nodeQuestionAssign: fallthrough
        case nodeSimpleAssign: off = -2
        default: errorf(0, "unknown assignment")
        }

        for cn, c := range l.list {
                //fmt.Printf("assign:1: %v '%v'\n", c.kind, l.str(c))
                if c.kind != nodeSpaces { l.list = l.list[cn:]; break }
        }

        for li := len(l.list)-1; 0 <= li; li-- {
                //fmt.Printf("assign:2: %v '%v'\n", l.list[li].kind, l.str(l.list[li]))
                if l.list[li].kind != nodeSpaces { l.list = l.list[0:li+1]; break }
        }

        if len(l.list) == 0 {
                errorf(0, "illigal assignment with no variable name")
        }

        // the name
        nn := l.new(nodeText, 0)
        nn.children = append(nn.children, l.list...)
        nn.pos, nn.end, l.list = l.list[0].pos, l.list[len(l.list)-1].end, l.list[0:0]

        // 'n' is the whole assign statemetn, e.g. "foo = bar"
        n := l.new(at, nn.pos-l.pos)
        n.children = append(n.children, nn)
        //fmt.Printf("assign: '%v'\n", l.str(n))

        // the equal signs: '=', ':=', '?='
        nn = l.new(nodeText, off)
        nn.end = nn.pos - off
        n.children = append(n.children, nn)

        // value
        nn, t := l.new(nodeText, 0), l.new(nodeText, 0)
        n.children = append(n.children, nn)

        //fmt.Printf("assign: '%v', %v\n", l.str(n), l.rune)

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
                        if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(nodeText, 0) }
                        nn.children = append(nn.children, ss)
                case r == '$':
                        cc := l.parseCall()
                        if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(nodeText, 0) }
                        nn.children = append(nn.children, cc)
                case r == '\\':
                        if !l.get() { break out_loop }
                        switch l.rune {
                        case '\n':
                                if 0 < t.len() { nn.children, t = append(nn.children, t), l.new(nodeText, 0) }
                                cn := l.new(nodeContinual, -2); cn.end = cn.pos+2
                                nn.children = append(nn.children, cn)
                                break
                        default: r = l.rune; goto the_sw
                        }
                }
        }

        for cn, c := range nn.children {
                if c.kind != nodeSpaces { nn.children = nn.children[cn:]; break }
        }
        
        if len(nn.children) == 1 /*&& nn.children[0].kind == nn.kind*/ {
                nn.children = nn.children[0:0]
        }

        n.end = nn.end

        //fmt.Printf("%v:%v: %v, '%v', '%v'\n", l.file, n.lineno, n.kind, l.str(n), l.str(nn))
        return n
}

func (l *lex) parseDoubleColonRule() *node {
        n := l.new(nodeDoubleColonRule, -2)
        fmt.Printf("%v:%v: '%v'\n", l.file, n.lineno, n.kind)
        return n
}

func (l *lex) parseRule() *node {
        n := l.new(nodeRule, -1)
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
                        case '\n': l.list = append(l.list, l.new(nodeContinual, -2))
                        default: r = l.rune; goto the_sw
                        }
                case r == '=':
                        l.nodes = append(l.nodes, l.parseAssign(nodeAssign))
                case r == '?':
                        if !l.get() { break main_loop }
                        switch l.rune {
                        case '=': l.nodes = append(l.nodes, l.parseAssign(nodeQuestionAssign))
                        default: r = l.rune; goto the_sw
                        }
                case r == ':':
                        if !l.get() { break main_loop }
                        switch l.rune {
                        case '=': l.nodes = append(l.nodes, l.parseAssign(nodeSimpleAssign))
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

// context hold a parse context and the current module being processed.
type context struct {
        l lex

        // module is the current module being processed
        module *module

        // line accumulates the current line of text
        line bytes.Buffer

        // variables holds the context
        variables map[string]*variable
}

func (ctx *context) setModule(m *module) (prev *module) {
        prev = ctx.module
        ctx.module = m
        return
}

func (ctx *context) getModuleSources() (sources []string) {
        if ctx.module == nil {
                return
        }

        if s, ok := ctx.module.variables["this.sources"]; ok {
                dir, str := ctx.module.dir, ctx.expand(s.value)
                sources = strings.Split(str, " ")
                for i := range sources {
                        if sources[i][0] == '/' { continue }
                        sources[i] = filepath.Join(dir, sources[i])
                }
        }
        return
}

func (ctx *context) expand(str string) string {
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
                        out = ctx.call(t.String(), args...)
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
                                out, l = ctx.call(name, args...), l + rs
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

func (ctx *context) call(name string, args ...string) string {
        //fmt.Printf("call: %v %v\n", name, args)
        vars := ctx.variables

        switch {
        default:
                if f, ok := builtins[name]; ok {
                        // All arguments should be expended.
                        for i := range args { args[i] = ctx.expand(args[i]) }
                        return f(ctx, args)
                }
        case name == "$": return "$";
        case name == "call":
                if 0 < len(args) {
                        return ctx.call(args[0], args[1:]...)
                }
                return ""
        case name == "this":
                if ctx.module != nil {
                        return ctx.module.name
                }
                return ""
        case strings.HasPrefix(name, "this.") && ctx.module != nil:
                vars = ctx.module.variables
        }

        if vars != nil {
                if v, ok := vars[name]; ok {
                        return v.value
                }
        }

        return ""
}

func (ctx *context) set(name, value string) (v *variable) {
        loc := ctx.l.location()

        if name == "this" {
                fmt.Printf("%v:warning: ignore attempts on \"this\"\n", loc)
                return
        }

        vars := ctx.variables
        if strings.HasPrefix(name, "this.") && ctx.module != nil {
                vars = ctx.module.variables
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
        v.loc = *ctx.l.location()

        //fmt.Printf("%v: '%s' = '%s'\n", &v.loc, name, value)
        return
}

func (ctx *context) expandNode(n *node) string {
        //fmt.Printf("%v:%v:%v: expand '%v' (%v)\n", ctx.l.file, n.lineno, n.colno, ctx.l.str(n), len(n.children))

        if len(n.children) == 0 {
                switch n.kind {
                case nodeComment: errorf(0, "can't expand comment: %v", ctx.l.str(n))
                case nodeCall: errorf(0, "invalid call: %v", ctx.l.str(n))
                case nodeContinual: return " "
                }
                //fmt.Printf("%v:%v:%v: %v '%v' (%v)\n", ctx.l.file, n.lineno, n.colno, n.kind, ctx.l.str(n), len(n.children))
                return ctx.l.str(n)
        }

        if n.kind == nodeCall {
                //fmt.Printf("expand: call: %v, %v\n", ctx.l.str(n), len(n.children))
                name, args := ctx.expandNode(n.children[0]), []string{}
                for _, an := range n.children[1:] {
                        s := ctx.expandNode(an); args = append(args, s)
                        //fmt.Printf("%v:%v:%v: arg '%v' ((%v) '%v') (%v)\n", ctx.l.file, an.lineno, an.colno, ctx.l.str(an), len(an.children), s, name)
                }
                v := ctx.call(name, args...)
                //fmt.Printf("%v:%v:%v: call '%v' %v '%v'\n", ctx.l.file, n.lineno, n.colno, name, args, v)
                return v
        }

        //fmt.Printf("%v:%v:%v: %v '%v' (%v)\n", ctx.l.file, n.lineno, n.colno, n.kind, ctx.l.str(n), len(n.children))
        var b bytes.Buffer
        for _, cn := range n.children {
                v := ctx.expandNode(cn)
                b.WriteString(v)
                //fmt.Printf("%v:%v:%v: %v '%v' '%v'\n", ctx.l.file, cn.lineno, cn.colno, cn.kind, ctx.l.str(cn), v)
        }
        return b.String()
}

func (ctx *context) processNode(n *node) (err error) {
        //fmt.Printf("%v:%v:%v: node '%v' (%v, %v)\n", ctx.l.file, n.lineno, n.colno, ctx.l.str(n), n.kind, len(n.children))

        switch n.kind {
        case nodeComment:
        case nodeSpaces:
        case nodeAssign:
                nn, nv := n.children[0], n.children[2]
                ctx.set(ctx.expandNode(nn), ctx.l.str(nv))
                //fmt.Printf("%v:%v:%v: %v %v\n", ctx.l.file, n.lineno, n.colno, ctx.l.str(nn), ctx.l.str(nv))
                //fmt.Printf("%v:%v:%v: '%v' '%v'\n", ctx.l.file, n.lineno, n.colno, ctx.expandNode(nn), ctx.l.str(nv))
        case nodeSimpleAssign:
                nn, nv := n.children[0], n.children[2]
                ctx.set(ctx.expandNode(nn), ctx.expandNode(nv))
                //fmt.Printf("%v:%v:%v: %v %v\n", ctx.l.file, n.lineno, n.colno, ctx.l.str(nn), ctx.l.str(nv))
                //fmt.Printf("%v:%v:%v: '%v' '%v'\n", ctx.l.file, n.lineno, n.colno, ctx.expandNode(nn), ctx.expandNode(nv))
        case nodeQuestionAssign:
                // TODO: ...
        case nodeCall:
                //fmt.Printf("%v:%v:%v: call %v\n", ctx.l.file, n.lineno, n.colno, ctx.l.str(n))
                if s := ctx.expandNode(n); s != "" {
                        errorf(0, "illigal: %v (%v)", s, ctx.l.str(n))
                }
        }
        return
}

func (ctx *context) parse() (err error) {
        ctx.l.parse()

        for _, n := range ctx.l.nodes {
                if n.kind == nodeComment { continue }
                if e := ctx.processNode(n); e != nil {
                        break
                }
        }
        return
}

func newContext(fn string) (ctx *context, err error) {
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

        ctx = &context{
                l: lex{ file: fn, s: s, pos: 0, },
                variables: make(map[string]*variable, 128),
        }

        return
}

func parse(conf string) (ctx *context, err error) {
        ctx, err = newContext(conf)

        defer func() {
                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("%v: %v\n", ctx.l.location(), se)
                        } else {
                                panic(e)
                        }
                }
        }()

        if err = ctx.parse(); err != nil {
                return
        }

        return
}
