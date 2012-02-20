package smart

import (
        "bufio"
        "bytes"
        //"errors"
        "fmt"
        "unicode"
        "unicode/utf8"
        "io"
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

type parser struct {
        module *module
        file string
        in *bufio.Reader //io.RuneReader
        buf bytes.Buffer // token or word or line accumulator
        stack []string // token stack
        rune rune
        lineno int
        colno, prevColno int
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
                dir := filepath.Dir(p.file)
                str := p.expand(s.value)
                sources = strings.Split(str, " ")
                for i, _ := range sources {
                        if sources[i][0] == '/' { continue }
                        sources[i] = filepath.Join(dir, sources[i])
                }
        }
        return
}

/*
func (p *parser) getModuleSourceActions(func f(a *action)) (sources []*action) {
        sources := p.getModuleSources()
        for _, src := range sources {
                asrc := newAction(src)
                
        }
}
*/

func (p *parser) location() *location {
        return &location{ &p.file, p.lineno, p.colno }
}

func (p *parser) stepLine() {
        p.lineno++
        p.prevColno, p.colno = p.colno, 0
}

func (p *parser) stepLineBack() {
        p.lineno, p.colno = p.lineno-1, p.prevColno+1
}

func (p *parser) stepCol() {
        p.colno++
}

func (p *parser) push(w string) {
        if 5000 < len(p.stack) { errorf(-1, "stack overflow") }
        p.stack = append(p.stack, w)
}

func (p *parser) pop() (w string) {
        if len(p.stack) == 0 { return }
        top := len(p.stack)-1
        w = p.stack[top]
        p.stack = p.stack[0:top]
        return
}

func (p *parser) getRune() (r rune, rs int, err error) {
        r, rs, err = p.in.ReadRune()
        if err != nil { return }
        if rs == 0 { errorf(-2, "zero reading") }
        if r == '\n' { p.stepLine() } else { p.stepCol() }
        p.rune = r
        return
}

func (p *parser) ungetRune() (err error) {
        if p.lineno == 1 && p.colno <= 1 { return }
        if err = p.in.UnreadRune(); err != nil { errorf(0, fmt.Sprintf("%v", err)) }
        if p.rune == '\n' {
                p.lineno, p.colno = p.lineno-1, p.prevColno
        } else {
                p.colno--
        }
        return
}

func (p *parser) skip(shouldSkip func(r rune) bool) (bytes int, err error) {
        var r rune
        var rs int
        for {
                r, rs, err = p.getRune()
                if err != nil {
                        return
                }
                if shouldSkip(r) {
                        bytes += rs;
                } else {
                        p.ungetRune(); break
                }
        }
        return
}

func (p *parser) skipRune(r rune) (bytes int, err error) {
        if v, b, e := p.getRune(); e == nil {
                if r == 0 || v == r {
                        bytes = b
                        return
                } else {
                        p.ungetRune()
                        errorf(0, "not rune '%v' (%v)", r, v)
                }
        } else {
                err = e
        }
        return
}

func (p *parser) skipSpace(inline bool) (bytes int, err error) {
        sz, e := p.skip(func(r rune) bool {
                if r == '#' {
                        for {
                                var rs int
                                if r, rs, err = p.getRune(); err != nil {
                                        return false
                                }
                                if r == '\n' {
                                        err = p.ungetRune(); break
                                }
                                bytes += rs
                        }
                        return true
                }
                if inline {
                        return r != '\n' && unicode.IsSpace(r)
                }
                return unicode.IsSpace(r)
        })
        bytes += sz
        if err == nil && e != nil { err = e }
        return
}

// getLine read a sequence of rune until delimeters
func (p *parser) getLine(delimeters string) (s string, del rune, err error) {
        var r rune
        p.buf.Reset()
main_loop: for {
                if r, _, err = p.getRune(); err != nil { break }
                switch {
                case r == '\\':
                        rr, _, e := p.getRune()
                        if e != nil { err = e; break main_loop }
                        if rr == '\n' { // line continual by "\\\n"
                                for {
                                        if rr, _, e = p.getRune(); e != nil { err = e; break main_loop }
                                        if rr == '\n' { p.ungetRune(); del = rr; break main_loop }
                                        if !unicode.IsSpace(rr) { err = p.ungetRune(); break }
                                }
                                r = ' ' // replace '\\' with a space
                        } else {
                                p.ungetRune()
                        }
                case r == '#':
                        //var b bytes.Buffer
                        for {
                                if rr, _, e := p.getRune(); e != nil {
                                        err = e; break main_loop
                                } else if rr == '\n' {
                                        //fmt.Printf("comment: %v\n", &b)
                                        break
                                } else {
                                        //b.WriteRune(rr)
                                }
                        }
                        p.ungetRune(); del = '\n'; break main_loop
                case strings.IndexRune(delimeters, r) != -1:
                        p.ungetRune(); del = r; break main_loop
                default:
                        p.buf.WriteRune(r)
                }
        }
        if err == nil || err == io.EOF {
                s, err = p.buf.String(), nil
        }
        //fmt.Printf("s: %v\n", p.buf.String())
        return
}

func (p *parser) parse() (err error) {
        p.lineno, p.colno = 1, 0

        var w, s string
        var del rune
parse_loop: for {
                if _, err = p.skipSpace(false); err != nil { break }

                if w, del, err = p.getLine("=:\n"); err != nil && err != io.EOF { break }

                if w = strings.TrimSpace(w); w == "" {
                        p.stepCol(); errorf(0, fmt.Sprintf("illegal: %v", w))
                }

                w = p.expand(w)

                // if it's the new line, we stop here
                if del == '\n' {
                        if w = strings.TrimSpace(w); w != "" {
                                p.colno -= utf8.RuneCount([]byte(w)) + 1
                                errorf(0, fmt.Sprintf("illegal: '%v'", w))
                        }
                        continue
                }

                // skip the delimiter
                if _, err = p.skipRune(del); err != nil { break parse_loop }

                var rr rune
                if rr, _, err = p.getRune(); err != nil { break } else {
                        if strings.IndexRune("=:", rr) == -1 { p.ungetRune() }
                }

                if _, err = p.skipSpace(true); err != nil && err != io.EOF { break }
                if s, _, err = p.getLine("\n"); err != nil && err != io.EOF { break }

                switch del {
                case '=': //print("parse: "+w+" = "+s+"\n")
                        if w == "" {
                                errorf(0, fmt.Sprintf("illegal: %v", w))
                        }
                        p.setVariable(w, s)

                case ':': //print("parse: "+w+" : "+s+"\n")
                        if w == "" {
                                errorf(0, fmt.Sprintf("illegal: %v", w))
                        }

                        switch rr {
                        case '=':
                                p.setVariable(w, p.expand(s))
                        case ':':
                                fmt.Printf("TODO: %v :: %v\n", w, s)
                        default:
                                fmt.Printf("TODO: %v : %v\n", w, s)
                        }

                default:
                        if w != "" {
                                errorf(0, fmt.Sprintf("illegal: %v", w))
                        }
                }
        }
        if err == io.EOF { err = nil }
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
        loc := location{ file:&(p.file), lineno:p.lineno, colno:p.colno+1 }

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
        v.loc = *p.location()

        //fmt.Printf("%v: '%s' = '%s'\n", &v.loc, name, value)
        return
}

func parse(conf string) (err error) {
        var f *os.File

        f, err = os.Open(conf)
        if err != nil {
                return
        }

        p := &parser{
                file: conf,
                in: bufio.NewReader(f),
                variables: make(map[string]*variable, 128),
        }

        defer func() {
                f.Close()

                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("%s:%v:%v: %v\n", conf, p.lineno, p.colno, se)
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
