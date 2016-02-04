//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "bytes"
        "errors"
        "fmt"
        "unicode"
        "unicode/utf8"
        "io/ioutil"
        "os"
        //"path/filepath"
        //"reflect"
        "strings"
)

type location struct {
        buf *parseBuffer
        offset, lineno, colno int
}

func (l *location) String() string {
        return fmt.Sprintf("%v:%v:%v", l.buf.scope, l.lineno, l.colno)
}

type define struct {
        name string
        node []interface{} // string, *node
        readonly bool
        loc location
}

type nodeType int

const (
        nodeComment nodeType = iota
        nodeEscape
        nodeDeferredText
        nodeImmediateText
        nodeValueText           // value text for defines (TODO: use it)
        nodeDefineDeferred      //  =     deferred
        nodeDefineQuestioned    // ?=     deferred
        nodeDefineSingleColoned // :=     immediate
        nodeDefineDoubleColoned // ::=    immediate
        nodeDefineNot           // !=     immediate
        nodeDefineAppend        // +=     deferred or immediate (parsed into deferred)
        nodeRuleSingleColoned   // :
        nodeRuleDoubleColoned   // ::
        nodePrerequisites
        nodeActions
        nodeCall
        nodeCallName
        nodeCallArg
)

/*
Variable definitions are parsed as follows:

immediate = deferred
immediate ?= deferred
immediate := immediate
immediate ::= immediate
immediate += deferred or immediate
immediate != immediate

define immediate
  deferred
endef

define immediate =
  deferred
endef

define immediate ?=
  deferred
endef

define immediate :=
  immediate
endef

define immediate ::=
  immediate
endef

define immediate +=
  deferred or immediate
endef

define immediate !=
  immediate
endef
*/
var nodeTypeNames = []string {
        nodeComment:                    "comment",
        nodeEscape:                     "escape",
        nodeDeferredText:               "deferred-text",
        nodeImmediateText:              "immediate-text",
        nodeValueText:                  "value-text",
        nodeDefineDeferred:             "define-deferred",
        nodeDefineQuestioned:           "define-questioned",
        nodeDefineSingleColoned:        "define-single-coloned",
        nodeDefineDoubleColoned:        "define-double-coloned",
        nodeDefineNot:                  "define-not",
        nodeDefineAppend:               "define-append",
        nodeRuleSingleColoned:          "rule-single-coloned",
        nodeRuleDoubleColoned:          "rule-double-coloned",
        nodePrerequisites:              "prerequisites",
        nodeActions:                    "actions",
        nodeCall:                       "call",
        nodeCallName:                   "call-name",
        nodeCallArg:                    "call-arg",
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

type parseBuffer struct {
        scope string // file or named scope
        s []byte // the content of the file
}

type lexStack struct {
        node *node
        state func()
        code int
        delm rune // delimeter
}
type lex struct {
        *parseBuffer
        pos int // the current read position

        lineno, colno, prevColno int

        rune rune // the rune last time returned by getRune
        runeLen int // the size in bytes of the rune last returned by getRune
        
        stack []*lexStack
        step func ()

        nodes []*node // parsed top level nodes
}

func (l *lex) location() *location {
        return &location{ l.parseBuffer, l.pos, l.lineno, l.colno }
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
        if len(l.s) == l.pos {
                if l.rune != rune(0) && 0 < l.runeLen {
                        l.rune, l.runeLen = 0, 0
                        return true
                }
                return false
        }
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

func (l *lex) new(t nodeType) *node {
        return &node{ kind:t, pos:l.pos, end:l.pos, lineno:l.lineno, colno:l.colno }
}

func (l *lex) push(t nodeType, ns func(), c int) *lexStack {
        ls := &lexStack{ node:l.new(t), state:l.step, code:c }
        l.stack, l.step = append(l.stack, ls), ns
        return ls
}

func (l *lex) pop() *lexStack {
        if i := len(l.stack)-1; 0 <= i {
                st := l.stack[i]
                l.stack, l.step = l.stack[0:i], st.state
                return st
        }
        return nil
}

func (l *lex) top() *lexStack {
        if i := len(l.stack)-1; 0 <= i {
                return l.stack[i]
        }
        return nil
}

func (l *lex) backwardNonSpace(i int) int {
        for 0 < i {
                r, l := utf8.DecodeLastRune(l.s[0:i])
                if unicode.IsSpace(r) {
                        i -= l
                } else {
                        break
                }
        }
        return i
}

func (l *lex) stateAppendNode() {
        t := l.pop().node

        /*
        fmt.Printf("AppendNode: %v: '%v' %v '%v' (%v, %v, %v)\n", t.kind,
                l.str(t.children[0]), l.str(t),
                l.str(t.children[1]), len(l.stack), l.pos, l.rune) //*/

        // Pop out and append the node.
        l.nodes = append(l.nodes, t)
}

func (l *lex) stateGlobal() {
state_loop:
        for l.get() {
                switch {
                case l.rune == '#':
                        st := l.push(nodeComment, l.stateComment, 0)
                        st.node.pos-- // for the '#'
                        break state_loop
                case l.rune != rune(0) && !unicode.IsSpace(l.rune):
                        l.unget() // Put back the rune.
                        l.push(nodeImmediateText, l.stateLineHeadText, 0)
                        break state_loop
                }
        }
}

func (l *lex) stateComment() {
state_loop:
        for l.get() {
                switch {
                case l.rune == '\\':
                        if l.peek() == '\n' {
                                l.get() // continual comment line
                        }

                case l.rune == '\n':
                        if l.peek() == '#' {
                                break // assemply continual comment lines in one node
                        }
                        fallthrough
                case l.rune == rune(0): // end of string
                        st := l.pop()
                        st.node.end = l.pos
                        if l.rune == '\n' {
                                st.node.end-- // exclude the '\n'
                        }

                        // append the comment node
                        l.nodes = append(l.nodes, st.node)
                        break state_loop
                }
        }
}

// stateLineHeadText process line-head text
func (l *lex) stateLineHeadText() {
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        st := l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == '=':
                        l.top().code = int(nodeDefineDeferred)
                        l.step = l.stateDefine
                        break state_loop

                case l.rune == '?':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '?='
                                l.top().code = int(nodeDefineQuestioned)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == '!':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '!='
                                l.top().code = int(nodeDefineNot)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == '+':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '+='
                                l.top().code = int(nodeDefineAppend)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == ':':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for ':='
                                l.top().code = int(nodeDefineSingleColoned)
                                l.step = l.stateDefine
                        } else {
                                l.top().node.end = l.backwardNonSpace(l.pos-1)
                                l.step = l.stateRule
                        }
                        break state_loop

                case l.rune == '#': fallthrough
                case l.rune == '\n':
                        st := l.pop() // pop out the node
                        st.node.end = l.pos-1

                        // append the island text
                        l.nodes = append(l.nodes, st.node)

                        if l.rune == '#' {
                                st = l.push(nodeComment, l.stateComment, 0)
                                st.node.pos-- // for the '#'
                        }
                        break state_loop

                default: l.escapeTextLine()
                }
        }
}

func (l *lex) stateDefine() {
        st := l.pop() // name

        var (
                name, t, n = st.node, nodeType(st.code), 2
                vt nodeType
        )
        switch t {
        case nodeDefineDoubleColoned: n = 3; fallthrough
        case nodeDefineSingleColoned:        fallthrough
        case nodeDefineNot:          vt = nodeImmediateText

        case nodeDefineDeferred:      n = 1; fallthrough
        default:                     vt = nodeDeferredText
        }

        name.end = l.backwardNonSpace(l.pos-n) // for '=', '+=', '?=', ':=', '::='

        st = l.push(t, l.stateAppendNode, 0)
        st.node.children = []*node{ name }
        st.node.pos -= n // for '=', '+=', '?=', ':='

        // Create the value node.
        value := l.push(vt, l.stateDefineTextLine, 0).node
        st.node.children = append(st.node.children, value)
}

func (l *lex) stateDefineTextLine() {
        st := l.top()

state_loop:
        for l.get() {
                if st.code == 0 && !unicode.IsSpace(l.rune) { // skip spaces after '='
                        st.node.pos, st.code = l.pos-1, 1
                }

                switch {
                case l.rune == '$':
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == '#':
                        l.unget() // Put back the '#', then fall through.
                        fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // The end of string.
                        st.node.end = l.pos
                        if l.rune == '\n' {
                                st.node.end-- // Exclude the '\n'.
                        }

                        l.pop() // Pop out the value node and forward to the define node.
                        break state_loop

                default: l.escapeTextLine()
                }
        }
}

func (l *lex) escapeTextLine() {
        if l.rune != '\\' { return }

        // Escape: \\n \#
        if l.get() { // get the char right next to '\\'
                switch l.rune {
                case '#': fallthrough
                case '\n':
                        en := l.new(nodeEscape)
                        en.pos -= 2 // for the '\\\n', '\\#', etc.
                        t := l.top().node
                        t.children = append(t.children, en)
                }
        }
}

func (l *lex) stateRule() {
        if l.get() {
                t, n := nodeRuleSingleColoned, 1 // Assuming single colon.
                switch {
                case l.rune == '\n': // targets :
                        targets := l.pop().node

                        // append the single-coloned-define
                        l.nodes = append(l.nodes, targets)

                case l.rune == ':': // targets :: blah blah blah
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '::='
                                l.top().code = int(nodeDefineDoubleColoned)
                                l.step = l.stateDefine
                                return
                        }
                        t, n = nodeRuleDoubleColoned, 2; fallthrough
                default: // targets : blah blah blah
                        targets := l.pop().node

                        st := l.push(t, l.stateAppendNode, 0)
                        st.node.children = []*node{ targets }
                        st.node.pos -= n // for the ':' or '::'

                        prerequisites := l.push(nodePrerequisites, l.stateRuleTextLine, 0).node
                        st.node.children = append(st.node.children, prerequisites)
                }
        }
}

func (l *lex) stateRuleTextLine() {
        st := l.top()

state_loop:
        for l.get() {
                if st.code == 0 && !unicode.IsSpace(l.rune) { // skip spaces after ':' or '::'
                        st.node.pos, st.code = l.pos-1, 1
                }

                switch {
                /* case l.rune != '\n' && unicode.IsSpace(l.rune):
                        t := l.new(nodeImmediateText)
                        i := len(st.node.children)-1
                        st.node.children[i].end = l.pos
                        st.node.children = append(st.node.children, t)
                        break state_loop */

                case l.rune == '$':
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == '#': fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        i := len(st.node.children)-1
                        st.node.children[i].end = l.pos
                        if l.rune != rune(0) {
                                st.node.children[i].end-- // exclude the '\n' or '#'
                        }

                        fmt.Printf("RuleTextLine: '%v'\n", l.str(st.node.children[i]))

                        st = l.pop() // pop out the node

                        if l.rune == '#' {
                                st = l.push(nodeComment, l.stateComment, 0)
                                st.node.pos-- // for the '#'
                        }
                        break state_loop

                default: l.escapeTextLine()
                }
        }
}

func (l *lex) stateTabAction() { // tab-indented action of a rule
}

func (l *lex) stateDollar() {
        st := l.top() // nodeCall 
        st.node.children = []*node{ l.new(nodeCallName) }

        l.step = l.stateCallee
}

func (l *lex) stateCallee() {
        const ( init int = iota; name; args )
        st := l.top() // Must be a nodeCall.

state_loop:
        for l.get() {
                switch {
                case l.rune == '(' && st.code == init: st.delm = ')'; fallthrough
                case l.rune == '{' && st.code == init: if st.delm == 0 { st.delm = '}' }
                        st.node.children[0].pos = l.pos
                        st.node.end, st.code = l.pos, name

                case l.rune == ' ' && st.code == name:
                        st.code = args; fallthrough
                case l.rune == ',' && st.code == args:
                        a := l.new(nodeCallArg)
                        i := len(st.node.children)-1
                        st.node.children[i].end = l.pos-1
                        st.node.children = append(st.node.children, a)

                case l.rune == '$' && st.code != init:
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == st.delm: //&& st.code != init:
                        fallthrough
                case st.code == init: // $$, $a, $<, $@, $^, etc.
                        call, i, n := st.node, len(st.node.children)-1, 1
                        if st.delm == rune(0) { n = 0 } // don't shift for single char like '$a'
                        call.end, call.children[i].end = l.pos, l.pos-n

                        l.pop() // pop out the current nodeCall

                        t := l.top().node
                        switch t.kind {
                        case nodeDeferredText: fallthrough
                        case nodeImmediateText:
                                t.children = append(t.children, call)
                        default:
                                // Add to the last child.
                                i = len(t.children)-1
                                if 0 <= i { t = t.children[i] }
                                t.children = append(t.children, call)
                        }

                        break state_loop
                }
        }
}

func (l *lex) parse() bool {
        l.step, l.lineno, l.colno, l.pos = l.stateGlobal, 1, 0, 0
        end := len(l.s); for l.pos < end { l.step() }

        for 0 < len(l.stack) {
                l.step() // Make extra step to allow handling rune(0),
                l.pop()  // then pop out the state.
        }
        return l.pos == end
}

// Context hold a parse context and the current module being processed.
type Context struct {
        stack []*lex

        l *lex

        // module is the current module being processed
        module *Module

        // variables holds the context
        defines map[string]*define

        // line accumulates the current line of text
        //line bytes.Buffer
}

func (ctx *Context) CurrentScope() string {
        return ctx.l.scope
}

func (ctx *Context) CurrentLocation() *location {
        return ctx.l.location()
}

func (ctx *Context) setModule(m *Module) (prev *Module) {
        prev = ctx.module
        ctx.module = m
        return
}

func (ctx *Context) expand(str string) string {
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
                                if ss, ll := exp(s[rs:]); 0 < ll {
                                        t.WriteString(ss)
                                        s, l = s[rs+ll:], l + rs + ll
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
                                        if 0 < len(name) { // 0 < len(args)
                                                args = append(args, t.String())
                                        } else {
                                                name = t.String()
                                        }
                                        t.Reset()
                                }
                                out, l = ctx.call(name, args...), l + rs
                                return // do not "break"
                        }

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

func (ctx *Context) CallWith(m *Module, name string, args ...string) string {
        return ctx.callWith(m, name, args...)
}

func (ctx *Context) Call(name string, args ...string) string {
        return ctx.call(name, args...)
}

func (ctx *Context) Set(name string, a ...interface{}) {
        ctx.set(name, a...)
}

func (ctx *Context) call(name string, args ...string) string {
        vars := ctx.defines

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
        case name == "me":
                if ctx.module != nil {
                        return ctx.module.Name
                }
                return ""
        case strings.HasPrefix(name, "me.") && ctx.module != nil:
                vars = ctx.module.defines
        }

        if vars != nil {
                if v, ok := vars[name]; ok {
                        b, f0, fn := new(bytes.Buffer), "%s", " %s"
                        for n, i := range v.node {
                                f := f0[0:]
                                if 0 < n { f = fn[0:] }
                                switch t := i.(type) {
                                case string: fmt.Fprintf(b, f, t)
                                case *node: fmt.Fprintf(b, f, ctx.expandNode(t.children[1]))
                                }
                        }
                        return b.String()
                }
        }

        return ""
}

func (ctx *Context) callWith(m *Module, name string, args ...string) (s string) {
        o := ctx.module
        ctx.module = m
        s = ctx.call("me."+name, args...)
        ctx.module = o
        return
}

func (ctx *Context) get(name string) *define {
        vars := ctx.defines
        if strings.HasPrefix(name, "me.") && ctx.module != nil {
                vars = ctx.module.defines
        }
        if vars == nil {
                //fmt.Printf("%v:warning: no \"me\" module\n", &loc)
                return nil
        }
        v, _ := vars[name]
        return v
}

func (ctx *Context) set(name string, a ...interface{}) (v *define) {
        loc := ctx.l.location()

        if name == "me" {
                fmt.Printf("%v:warning: ignore attempts on \"me\"\n", loc)
                return
        }

        vars := ctx.defines
        if strings.HasPrefix(name, "me.") && ctx.module != nil {
                vars = ctx.module.defines
        }
        if vars == nil {
                fmt.Printf("%v:warning: no \"me\" module\n", &loc)
                return
        }

        var has = false
        if v, has = vars[name]; !has {
                v = &define{}
                vars[name] = v
        }

        if v.readonly {
                fmt.Printf("%v:warning: `%v' is readonly\n", &loc, name)
                return
        }
        
        v.name, v.node, v.loc = name, a, *ctx.l.location()
        return
}

func (ctx *Context) expandNode(n *node) string {
        nc := len(n.children)

        switch n.kind {
        case nodeEscape:
                switch i := n.pos + 1; ctx.l.s[i] {
                case '\n': return " "
                case '#':  return "#"
                }
                return ""

        case nodeCall:
                name, args := ctx.expandNode(n.children[0]), []string{}
                for _, an := range n.children[1:] {
                        args = append(args, ctx.expandNode(an))
                }
                return ctx.call(name, args...)

        case nodeCallArg:       fallthrough
        case nodeCallName:      fallthrough
        case nodeDeferredText:  fallthrough
        case nodeImmediateText:
                if nc == 0 { return ctx.l.str(n) }
                b, pos := new(bytes.Buffer), n.pos
                for _, c := range n.children {
                        if pos < c.pos {
                                b.Write(ctx.l.s[pos:c.pos])
                        }
                        b.WriteString(ctx.expandNode(c))
                        pos = c.end
                }
                if pos < n.end {
                        b.Write(ctx.l.s[pos:n.end])
                }
                return b.String()

        default:
                panic(fmt.Sprintf("TODO: %v: %v (%v)\n", n.kind, ctx.l.str(n), nc))
        }

        return ""
}

func (ctx *Context) processNode(n *node) (err error) {
        switch n.kind {
        case nodeDefineQuestioned:
                if name := ctx.expandNode(n.children[0]); ctx.call(name) == "" {
                        ctx.set(name, n)
                }

        case nodeDefineDeferred:
                name := ctx.expandNode(n.children[0])
                ctx.set(name, n)

        case nodeDefineSingleColoned: fallthrough
        case nodeDefineDoubleColoned:
                name := ctx.expandNode(n.children[0])
                ctx.set(name, ctx.expandNode(n.children[1]))

        case nodeDefineAppend:
                name := ctx.expandNode(n.children[0])
                if d := ctx.get(name); d != nil {
                        deferred := true
                        if 0 < len(d.node) {
                                _, deferred = d.node[0].(*node)
                        }

                        if deferred {
                                d.node = append(d.node, n)
                        } else {
                                v := ctx.expandNode(n.children[1])
                                d.node = append(d.node, v)
                        }
                } else {
                        ctx.set(name, n)
                }
                

        case nodeDefineNot:
                panic("'!=' not implemented")

        case nodeCall:
                if s := ctx.expandNode(n); s != "" {
                        errorf(0, "illigal: %v (%v)", s, ctx.l.str(n))
                }

        case nodeComment:
        }
        return
}

func (ctx *Context) _parse() (err error) {
        if !ctx.l.parse() {
                err = errors.New("syntax error")
                return
        }

        for _, n := range ctx.l.nodes {
                if n.kind == nodeComment { continue }
                if e := ctx.processNode(n); e != nil {
                        break
                }
        }
        return
}

func (ctx *Context) append(scope string, s []byte) (err error) {
        l := &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0, }
        ctx.stack = append(ctx.stack, l)

        defer func() {
                ctx.stack = ctx.stack[len(ctx.stack)-1:]

                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                message("%v: %v", ctx.l.location(), se)
                        } else {
                                panic(e)
                        }
                }
        }()

        if err = ctx._parse(); err != nil {
                // ...
        }
        return
}

func (ctx *Context) include(fn string) (err error) {
        var (
                f *os.File
                s []byte
        )

        f, err = os.Open(fn)
        if err != nil {
                return
        }

        defer f.Close()

        s, err = ioutil.ReadAll(f)
        if err == nil {
                err = ctx.append(fn, s)
        }

        return
}

func NewContext(scope string, s []byte, vars map[string]string) (ctx *Context, err error) {
        ctx = &Context{
                l: &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0, },
                defines: make(map[string]*define, len(vars) + 32),
        }
        for k, v := range vars {
                ctx.set(k, v)
        }

        err = ctx.append(scope, s)
        return
}

func NewContextFromFile(fn string, vars map[string]string) (ctx *Context, err error) {
        s := []byte{} // TODO: needs init script
        if ctx, err = NewContext(fn, s, vars); err == nil && ctx != nil {
                err = ctx.include(fn)
        }
        return
}