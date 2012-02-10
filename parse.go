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
        "strings"
)

type location struct {
        file *string
        lineno int
        colno int
}

func (l *location) String() string {
        return fmt.Sprintf("%v:%v:%v:", *l.file, l.lineno, l.colno)
}

type variable struct {
        name string
        value string
        loc location
}

type parseError struct {
        level int
        message string
        lineno int
        colno int
}

func (e *parseError) String() string {
        return fmt.Sprintf("%v (%v)", e.message, e.level)
}

type parser struct {
        file string
        in *bufio.Reader //io.RuneReader
        buf bytes.Buffer // token or word or line accumulator
        stack []string // token stack
        rune rune
        lineno int
        colno, prevColno int
        variables map[string]*variable
}

func (p *parser) newError(l int, s string) *parseError {
        return &parseError{ l, s, p.lineno, p.colno }
}

func (p *parser) stepLine() {
        p.lineno++
        p.prevColno, p.colno = p.colno, 0
}

func (p *parser) stepCol() {
        p.colno++
}

func (p *parser) push(w string) {
        if 5000 < len(p.stack) { panic(p.newError(-1, "stack overflow")) }
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
        if rs == 0 { panic(p.newError(-2, "zero reading")) }
        if r == '\n' { p.stepLine() } else { p.stepCol() }
        p.rune = r
        return
}

func (p *parser) ungetRune() (err error) {
        if p.lineno == 1 && p.colno <= 1 { return }
        if err = p.in.UnreadRune(); err != nil { panic(p.newError(-3, "unget")) }
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
                        p.ungetRune()
                        break
                }
        }
        return
}

func (p *parser) skipRune() (bytes int, err error) {
        if _, bytes, err = p.getRune(); err != nil {
                return
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
                                        if e := p.ungetRune(); e != nil {
                                                panic(p.newError(-1, "unget '\n'"))
                                        }
                                        break
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

func (p *parser) skipLine() (size int, err error) {
        return p.skip(func(r rune) bool {
                return r == '\n'
        })
}

// get a sequence of 
func (p *parser) get(stop func(*rune) bool) (w string, err error) {
        var r rune
        p.buf.Reset()
        for {
                r, _, err = p.getRune()
                if err != nil {
                        break
                }
                if stop(&r) {
                        err = p.ungetRune()
                        break
                }
                if r != 0 { p.buf.WriteRune(r) }
        }
        w = string(p.buf.Bytes())
        return
}

// getWord read a word(non-space rune sequence)
func (p *parser) getWord() (string, error) {
        return p.get(func(r *rune) bool {
                return unicode.IsSpace(*r)
        })
}

// getLine read a sequence of rune until '\n'
func (p *parser) getLine() (s string, err error) {
        s, e := p.get(func(r *rune) bool {
                if *r == '\\' {
                        rr, _, e := p.getRune()
                        if e != nil {
                                err = e; return true
                        }
                        if rr == '\n' {
                                for {
                                        if rr, _, e = p.getRune(); e != nil {
                                                err = e; return true
                                        }
                                        if rr == '\n' { *r = 0; return true }
                                        if !unicode.IsSpace(rr) {
                                                if e = p.ungetRune(); e != nil {
                                                        err = e; return true
                                                }
                                                break
                                        }
                                }
                                *r = ' ' // replace '\\' with a space
                                return false
                        }
                }
                return *r == '\n'
        })
        if err == nil && e != nil { err = e }
        return
}

func (p *parser) call(name string, args []string) string {
        //fmt.Printf("call: %v %v %v\n", name, args, p.variables[name])
        //fmt.Printf("call: %v %v\n", name, args)
        if v, ok := p.variables[name]; ok {
                return v.value
        }
        return ""
}

func (p *parser) expand(str string) string {
        var buf bytes.Buffer
        var exp func(s []byte) (out string, l int)
        var getRune func(s []byte) (r rune, l int)

        getRune = func(s []byte) (r rune, l int) {
                if r, l = utf8.DecodeRune(s); r == utf8.RuneError || l <= 0 {
                        panic(p.newError(1, "bad UTF8 encoding"))
                }
                return
        }

        exp = func(s []byte) (out string, l int) {
                var r, rr rune
                var sz = 0

                if r, sz = getRune(s); r == '$' {
                        s, l = s[sz:], l + sz
                } else {
                        panic(p.newError(1, "not a variable"))
                }

                r, sz = getRune(s); s, l = s[sz:], l + sz
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
                        out = p.call(t.String(), args)
                        return
                }

                for 0 < len(s) {
                        r, sz = getRune(s)

                        switch r {
                        default: t.WriteRune(r)
                        case ' ':
                                name = t.String(); t.Reset()
                        case ',':
                                args = append(args, t.String()); t.Reset()
                        case '$':
                                //fmt.Printf("inner: %v, %v, %v\n", string(s), sz, l)
                                if ss, ll := exp(s); 0 < ll {
                                        t.WriteString(ss)
                                        s, l = s[ll:], l + ll
                                        //fmt.Printf("inner: %v, %v, %v, %v\n", string(s), ll, ss, sz)
                                        continue
                                } else {
                                        panic(p.newError(1, string(s)))
                                }
                        case rr:
                                if 0 < t.Len() {
                                        if 0 < len(args) {
                                                args = append(args, t.String())
                                        } else {
                                                name = t.String()
                                        }
                                        t.Reset()
                                }
                                //fmt.Printf("rr: %v, %v, %v\n", string(s), rr, l)
                                out, l = p.call(name, args), l + sz
                                return /* do not "break" */
                        }
                        s, l = s[sz:], l + sz
                }
                return
        }

        s := []byte(str)
        for 0 < len(s) {
                r, l := getRune(s)
                if r == '$' {
                        if ss, ll := exp(s); ll <= 0 {
                                panic(p.newError(0, "bad variable"))
                        } else {
                                s = s[ll:]
                                buf.WriteString(ss)
                        }
                } else {
                        buf.WriteRune(r)
                        s = s[l:]
                }
        }
        return buf.String()
}

func (p *parser) parse() (err error) {
        p.lineno, p.colno = 1, 0

        var w, s string
        var del rune
        for {
                if _, err = p.skipSpace(false); err != nil {
                        break
                }

                w, err = p.get(func(r *rune) bool {
                        if *r == '=' || *r == ':' || *r == '\n' {
                                del = *r; return true
                        }
                        return false
                })//p.getWord()
                if err != nil { break }

                if w = strings.TrimSpace(w); w == "" {
                        p.stepCol(); panic(p.newError(0, "illegal"))
                }

                if _, err = p.skipRune(); err != nil { break }
                if _, err = p.skipSpace(true); err != nil { break }
                if s, err = p.getLine(); err != nil && err != io.EOF { break }

                w = strings.TrimSpace(p.expand(w))

                switch del {
                case '=':
                        p.saveVariable(w, s)
                        //print("parse: "+w+" = "+s+"\n")
                case ':':
                        //print("parse: "+w+" : "+s+"\n")
                case '\n':
                        fmt.Printf("line: %v\n", w)
                default:
                        if w != "" {
                                panic(p.newError(0, w))
                        }
                }
        }
        if err == io.EOF { err = nil }
        return
}

func (p *parser) saveVariable(name, value string) {
        var v *variable
        var has bool
        if v, has = p.variables[name]; !has {
                v = &variable{}
                p.variables[name] = v
        }
        v.name = name
        v.value = value
        v.loc.file = &p.file
        v.loc.lineno = p.lineno

        fmt.Printf("%v %s = %s\n", &v.loc, name, value)
}

func (m *module) parse(conf string) (err error) {
        var f *os.File

        f, err = os.Open(conf)
        if err != nil {
                return
        }

        defer func() {
                f.Close()

                if e := recover(); e != nil {
                        if pe, ok := e.(*parseError); ok {
                                fmt.Printf("%s:%v:%v: %v\n", conf, pe.lineno, pe.colno, pe)
                        } else {
                                panic(e)
                        }
                }
        }()

        p := &parser{ file:conf, in:bufio.NewReader(f), variables:make(map[string]*variable, 200) }
        if err = p.parse(); err != nil {
                return
        }

        m.variables = p.variables
        return
}
