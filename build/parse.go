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
        "os/exec"
        //"path/filepath"
        //"reflect"
        "strings"
        "github.com/duzy/worker"
)

type Item interface{
        // Expand the item to string
        Expand(ctx *Context) string

        // Check if the item is empty (including all spaces)
        IsEmpty(ctx *Context) bool
}

type Items []Item

func (is Items) Len() int { return len(is) }
func (is Items) IsEmpty(ctx *Context) bool {
        if 0 < len(is) { return false }
        for _, i := range is {
                if !i.IsEmpty(ctx) { return false }
        }
        return true
}

func (is Items) Expand(ctx *Context) string { return is.Join(ctx, " ") }
func (is Items) Join(ctx *Context, sep string) string {
        b := new(bytes.Buffer)
        for i, a := range is {
                if s := a.Expand(ctx); s != "" {
                        if i == 0 {
                                fmt.Fprint(b, s)
                        } else {
                                fmt.Fprintf(b, "%s%s", sep, s)
                        }
                }
        }
        return b.String()
}

func (is Items) Concat(ctx *Context, args ...Item) (res Items) {
        for _, a := range is {
                if !a.IsEmpty(ctx) {
                        res = append(res, a)
                }
        }
        for _, a := range args {
                if !a.IsEmpty(ctx) {
                        res = append(res, a)
                }
        }
        return
}

type define struct {
        name string
        value Items
        readonly bool
        loc location
}

type rule struct {
        targets, prerequisites []string
        actions []interface{} // *node, string
        ns namespace
        node *node
}

type scoper interface {
        // Call variable.
        Call(ctx *Context, ids []string, args ...Item) Items
        // Set variable.        
        Set(ctx *Context, ids []string, items ...Item)
        // Check if a variable exists.        
        //Has(ctx *Context, ids []string) bool
}

type namespace interface {
        getNamespace(name string) namespace
        getDefineMap() map[string]*define
        getRuleMap() map[string]*rule
        findMatchedRule(ctx *Context, target string) (m *match, r *rule)
        isPhonyTarget(ctx *Context, target string) bool
        saveDefines(names ...string) (saveIndex int, m map[string]*define)
        restoreDefines(saveIndex int)
        Set(ctx *Context, ids []string, items ...Item)
}

type namespaceEmbed struct {
        defines map[string]*define
        saveList []map[string]*define // saveDefines, restoreDefines
        rules map[string]*rule
        goal *rule
}

func (ns *namespaceEmbed) saveDefines(names ...string) (saveIndex int, m map[string]*define) {
        var ok bool
        m = make(map[string]*define, len(names))
        for _, name := range names {
                m[name], ok = ns.defines[name]
                if ok { delete(ns.defines, name) }
        }
        saveIndex = len(ns.saveList)
        ns.saveList = append(ns.saveList, m)
        return
}

func (ns *namespaceEmbed) restoreDefines(saveIndex int) {
        m := ns.saveList[saveIndex]
        ns.saveList = ns.saveList[0:saveIndex]
        for name, d := range m {
                if d == nil {
                        delete(ns.defines, name)
                } else {
                        ns.defines[name] = d
                }
        }
}

/*
func (ns *namespaceEmbed) Call(ctx *Context, ids []string, args ...Item) (is Items) {
        if n := len(ids); n == 1 {
                if d, ok := ns.defines[ids[0]]; ok && d != nil {
                        is = d.value
                }
        } else {
                lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
                fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: nested referencing\n", ctx.l.scope, lineno, colno)

                // FIXME: nested
        }
        return
} */

func (ns *namespaceEmbed) Set(ctx *Context, ids []string, items ...Item) {
        if n := len(ids); n == 1 {
                name := ids[0]
                if d, ok := ns.defines[name]; ok && d != nil {
                        d.value = items
                } else {
                        ns.defines[name] = &define{ loc:ctx.CurrentLocation(), name:name, value:items }
                }
        } else {
                lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
                fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: nested referencing\n", ctx.l.scope, lineno, colno)

                // FIXME: nested
        }
}

func (ns *namespaceEmbed) getNamespace(name string) namespace {
        //lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
        //fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: nesting reference '%s'\n", ctx.l.scope, lineno, colno, name)
        return nil
}

func (ns *namespaceEmbed) getDefineMap() map[string]*define {
        return ns.defines
}

func (ns *namespaceEmbed) getRuleMap() map[string]*rule {
        return ns.rules
}

func (ns *namespaceEmbed) findMatchedRule(ctx *Context, target string) (m *match, r *rule) {
        if rr, ok := ns.rules[target]; ok && rr != nil {
                if m, ok = rr.match(target); ok && m != nil {
                        r = rr
                }
        } else {
                /// TODO: perform pattern match for a perfect rule
        }
        return
}

func (ns *namespaceEmbed) isPhonyTarget(ctx *Context, target string) bool {
        /// TODO: checking phony target
        return false
}

type nodeType int

const (
        nodeComment nodeType = iota
        nodeEscape
        nodeDeferredText
        nodeImmediateText
        nodeName
        nodeNamePrefix          // :
        nodeNamePart            // .
        nodeArg
        nodeValueText           // value text for defines (TODO: use it)
        nodeDefineDeferred      //  =     deferred
        nodeDefineQuestioned    // ?=     deferred
        nodeDefineSingleColoned // :=     immediate
        nodeDefineDoubleColoned // ::=    immediate
        nodeDefineNot           // !=     immediate
        nodeDefineAppend        // +=     deferred or immediate (parsed into deferred)
        nodeRuleSingleColoned   // :
        nodeRuleDoubleColoned   // ::
        nodeTargets
        nodePrerequisites
        nodeActions
        nodeAction
        nodeCall
        nodeSpeak               // $(speak dialect, ...)
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
var (
        meDot = "me."
        nodeTypeNames = []string {
                nodeComment:                    "comment",
                nodeEscape:                     "escape",
                nodeDeferredText:               "deferred-text",
                nodeImmediateText:              "immediate-text",
                nodeName:                       "name",
                nodeNamePrefix:                 "name-prefix",
                nodeNamePart:                   "name-part",
                nodeArg:                        "arg",
                nodeValueText:                  "value-text",
                nodeDefineDeferred:             "define-deferred",
                nodeDefineQuestioned:           "define-questioned",
                nodeDefineSingleColoned:        "define-single-coloned",
                nodeDefineDoubleColoned:        "define-double-coloned",
                nodeDefineNot:                  "define-not",
                nodeDefineAppend:               "define-append",
                nodeRuleSingleColoned:          "rule-single-coloned",
                nodeRuleDoubleColoned:          "rule-double-coloned",
                nodeTargets:                    "targets",
                nodePrerequisites:              "prerequisites",
                nodeActions:                    "actions",
                nodeAction:                     "action",
                nodeCall:                       "call",
                nodeSpeak:                      "speak",
        }
)

func (k nodeType) String() string {
        return nodeTypeNames[int(k)]
}

type location struct {
        offset, end int // (node.pos, node.end)
}

type stringitem string

func (si stringitem) Expand(ctx *Context) string { return string(si) }
func (si stringitem) IsEmpty(ctx *Context) bool { return string(si) == "" }

func StringItem(s string) stringitem { return stringitem(s) }

// flatitem is a expanded string with a location
type flatitem struct {
        s string
        l location
}

func (fi *flatitem) Expand(ctx *Context) string { return fi.s }
func (fi *flatitem) IsEmpty(ctx *Context) bool { return fi.s == "" }

type node struct {
        l *lex
        kind nodeType
        children []*node
        pos, end int
}

func (n *node) Expand(ctx *Context) (s string) {
        var is Items
        if nodeDefineDeferred <= n.kind && n.kind <= nodeDefineAppend {
                is = ctx.nodeItems(n.children[1])
        } else {
                is = ctx.nodeItems(n)
        }
        return is.Expand(ctx)
}

func (n *node) IsEmpty(ctx *Context) bool {
        if len(n.children) == 0 { return true }
        return n.Expand(ctx) == ""
}

func (n *node) len() int {
        return n.end - n.pos
}

func (n *node) str() string {
        return string(n.l.s[n.pos:n.end])
}

func (n *node) loc() location {
        return location{n.pos, n.end}
}

type parseBuffer struct {
        scope string // file or named scope
        s []byte // the content of the file
}

func (p *parseBuffer) caculateLocationLineColumn(loc location) (lineno, colno int) {
        for i, end := 0, len(p.s); i < loc.offset && i < end; {
                r, l := utf8.DecodeRune(p.s[i:])
                switch {
                case r == '\n':
                        lineno, colno = lineno + 1, 0
                case 0 < l:
                        colno += l
                }
                i += l
        }
        lineno++ // started from 1
        colno++  // started from 1
        return
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

        rune rune // the rune last time returned by getRune
        runeLen int // the size in bytes of the rune last returned by getRune
        
        stack []*lexStack
        step func ()

        nodes []*node // parsed top level nodes
}

func (l *lex) location() location {
        return location{ l.pos, l.pos }
}

func (l *lex) getLineColumn() (lineno, colno int) {
        lineno, colno = l.caculateLocationLineColumn(l.location())
        return 
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
        if len(l.s) < l.pos { errorf("over reading (at %v)", l.pos) }

        l.rune, l.runeLen = utf8.DecodeRune(l.s[l.pos:])
        l.pos = l.pos+l.runeLen
        switch {
        case l.rune == 0:
                return false //errorf(-2, "zero reading (at %v)", l.pos)
        case l.rune == utf8.RuneError:
                errorf("invalid UTF8 encoding")
        }
        return true
}

func (l *lex) unget() {
        switch {
        case l.rune == 0:
                errorf("wrong invocation of unget")
        case l.pos == 0:
                errorf("get to the beginning of the bytes")
        case l.pos < 0:
                errorf("get to the front of beginning of the bytes")
                //case l.lineno == 1 && l.colno <= 1: return
        }
        /*
        if l.rune == '\n' {
                l.lineno, l.colno, l.prevColno = l.lineno-1, l.prevColno, 0
        } else {
                l.colno--
        } */
        // assert(utf8.RuneLen(l.rune) == l.runeLen)
        l.pos, l.rune, l.runeLen = l.pos-l.runeLen, 0, 0
        return
}

func (l *lex) new(t nodeType) *node {
        return &node{ l:l, kind:t, pos:l.pos, end:l.pos }
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

/*
func (l *lex) forward(i, n int) int {
        for x = len(l.s); i < x && 0 < n {
                r, l := utf8.DecodeLastRune(l.s[0:i])
                if unicode.IsSpace(r) {
                        i += l
                } else {
                        break
                }
                n--
        }
        return i
}

func (l *lex) backward(i, n int) int {
        for 0 < i && 0 < n {
                r, l := utf8.DecodeLastRune(l.s[0:i])
                if unicode.IsSpace(r) {
                        i -= l
                } else {
                        break
                }
                n--
        }
        return i
}
*/

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
                t.children[0].str(), t.str(), t.children[1].str(), len(l.stack), l.pos, l.rune) //*/

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

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateComment: (stack=%v) %v\n", l.scope, lineno, colno, len(l.stack), st.node.str()) //*/

                        if 0 < len(l.stack) {
                                c := st.node
                                st = l.top()
                                st.node.children = append(st.node.children, c)
                        } else {
                                l.nodes = append(l.nodes, st.node) // append the comment node
                        }
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
                        if r := l.peek(); r == '=' {
                                l.get() // consume the '=' for ':='
                                l.top().code = int(nodeDefineSingleColoned)
                                l.step = l.stateDefine
                        } else {
                                l.top().node.end = l.backwardNonSpace(l.pos-1)
                                l.step = l.stateRule
                                //fmt.Fprintf(os.Stderr, "line head text: %v (%v)\n", l.top().node.str(), string(r))
                        }
                        break state_loop

                case l.rune == '.':
                        part := l.new(nodeNamePart)
                        part.pos = l.pos - 1
                        st := l.top()
                        st.node.children = append(st.node.children, part)

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

                default: l.escapeTextLine(l.top().node)
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
                if st.code == 0 { // skip spaces after '='
                        if !unicode.IsSpace(l.rune) {
                                st.node.pos, st.code = l.pos-1, 1
                        } else if l.rune != '\n' /* IsSpace */ {
                                st.node.pos = l.pos
                                continue
                        }
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

                default: l.escapeTextLine(l.top().node)
                }
        }
}

func (l *lex) escapeTextLine(t *node) {
        if l.rune != '\\' { return }

        // Escape: \\n \#
        if l.get() { // get the char right next to '\\'
                switch l.rune {
                case '#': fallthrough
                case '\n':
                        en := l.new(nodeEscape)
                        en.pos -= 2 // for the '\\\n', '\\#', etc.
                        if l.rune == '\n' {
                                /* FIXME: skip spaces after '\\\n' ?
                                for unicode.IsSpace(l.peek()) {
                                        l.get()
                                        en.end = l.pos
                                } */
                        }
                        t.children = append(t.children, en)
                }
        }
}

func (l *lex) stateRule() {
        r, t, n := l.peek(), nodeRuleSingleColoned, 1 // Assuming single colon.
        switch {
        case r == ':': // targets :: blah blah blah
                l.get() // drop the ':'
                if l.peek() == '=' {
                        l.get() // consume the '=' for '::='
                        l.top().code = int(nodeDefineDoubleColoned)
                        l.step = l.stateDefine
                        return
                }
                t, n = nodeRuleDoubleColoned, 2; fallthrough
        case r == '\n': fallthrough // targets :
        default: // targets : blah blah blah
                if r == '\n' {
                        //l.get() // drop the '\n'
                }

                targets := l.pop().node
                targets.kind = nodeTargets

                st := l.push(t, l.stateAppendNode, 0)
                st.node.children = []*node{ targets }
                st.node.pos -= n // for the ':' or '::'

                prerequisites := l.push(nodePrerequisites, l.stateRuleTextLine, 0).node
                st.node.children = append(st.node.children, prerequisites)

                /*
                lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                fmt.Fprintf(os.Stderr, "%v:%v:%v: stateRule: %v\n", l.scope, lineno, colno, st.node.children[0].str()) //*/
        }
}

func (l *lex) stateRuleTextLine() {
        st := l.top()
state_loop:
        for l.get() {
                if st.code == 0 && (l.rune == '\n' || !unicode.IsSpace(l.rune)) { // skip spaces after ':' or '::'
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

                case l.rune == '#':  fallthrough
                case l.rune == ';':  fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        st.node.end = l.pos
                        if l.rune != rune(0) {
                                st.node.end-- // exclude the '\n' or '#'
                        }

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateRuleTextLine: %v\n", l.scope, lineno, colno, st.node.str()) //*/

                        st = l.pop() // pop out the node

                        switch l.rune {
                        case ';':
                                st.node.end = l.backwardNonSpace(l.pos-1)
                                st = l.push(nodeAction, l.stateInlineAction, 0)
                                //st.node.pos-- // for the ';'
                        case '#':
                                st = l.push(nodeComment, l.stateComment, 0)
                                st.node.pos-- // for the '#'
                        case '\n':
                                if p := l.peek(); p == '\t' || p == '#' {
                                        st = l.push(nodeActions, l.stateTabbedActions, 0)
                                        //st.node.pos-- // for the '\t'
                                }
                        }
                        break state_loop

                default: l.escapeTextLine(l.top().node)
                }
        }
}

func (l *lex) stateInlineAction() {
        st := l.top()
state_loop:
        for l.get() {
                if st.code == 0 && !unicode.IsSpace(l.rune) { // skip spaces after ';'
                        st.node.pos, st.code = l.pos-1, 1
                }

                switch {
                case l.rune == '$':
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                //case l.rune == '#': fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        a := st.node
                        a.end = l.pos
                        if l.rune != rune(0) {
                                a.end-- // exclude the '\n' or '#'
                        }

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateInlineAction: %v\n", l.scope, lineno, colno, st.node.str()) //*/

                        st = l.pop() // pop out the node
                        st = l.top() // the rule node
                        st.node.children = append(st.node.children, a)

                        if l.peek() == '#' {
                                st = l.push(nodeComment, l.stateComment, 0)
                                st.node.pos-- // for the '#'
                        }
                        break state_loop

                default: l.escapeTextLine(l.top().node)
                }
        }
}

func (l *lex) stateTabbedActions() { // tab-indented action of a rule
        if st := l.top(); l.get() {
                switch {
                case l.rune == '\t':
                        st = l.push(nodeAction, l.stateAction, 0)
                        //st.node.pos-- // for the '\t'

                case l.rune == '#':
                        st = l.push(nodeComment, l.stateComment, 0)
                        st.node.pos-- // for the '#'

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateTabbedActions: %v (%v, stack=%v)\n", l.scope, lineno, colno, st.node.str(), l.top().node.kind, len(l.stack)) //*/

                default:
                        a := st.node
                        a.end = l.pos
                        if l.rune == '\n' {
                                a.end--
                        }

                        st = l.pop() // pop out the actions
                        st = l.top() // the rule node
                        st.node.children = append(st.node.children, a)

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateTabbedActions: %v (%v)\n", l.scope, lineno, colno, st.node.str(), l.top().node.kind) //*/
                }
        }
}

func (l *lex) stateAction() { // tab-indented action of a rule
        st := l.top()
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        a := st.node
                        a.end = l.pos
                        if l.rune != rune(0) {
                                a.end-- // exclude the '\n'
                        }

                        l.pop() // pop out the node

                        st = l.top()
                        st.node.children = append(st.node.children, a)

                        /*
                        lineno, colno := l.caculateLocationLineColumn(a.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateAction: %v (of %v)\n", l.scope, lineno, colno, a.str(), st.node.kind) //*/
                        break state_loop
                }
        }
}

func (l *lex) stateDollar() {
        if l.get() {
                switch {
                case l.rune == '(': l.push(nodeName, l.stateCallName, 0).delm = ')'
                case l.rune == '{': l.push(nodeName, l.stateCallName, 0).delm = '}'
                default:
                        name := l.new(nodeName)
                        name.pos = l.pos - 1 // include the single char
                        st := l.top() // nodeCall
                        st.node.children = append(st.node.children, name)
                        l.endCall(st)
                }
        }
}

func (l *lex) stateCallName() {
        st := l.top() // Must be a nodeName.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == ':' && st.code == 0:
                        prefix := l.new(nodeNamePrefix)
                        prefix.pos = l.pos - 1
                        st.node.children = append(st.node.children, prefix)
                        st.code++
                case l.rune == '.':
                        part := l.new(nodeNamePart)
                        part.pos = l.pos - 1
                        st.node.children = append(st.node.children, part)
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ' ': fallthrough
                case l.rune == delm:
                        name := st.node
                        name.end = l.pos - 1
                        l.pop()

                        st = l.top()
                        switch s := name.str(); s {
                        case "speak":
                                st.node.kind = nodeSpeak
                                if l.rune != delm {
                                        l.push(nodeArg, l.stateSpeakDialect, 0).delm = delm
                                } else {
                                        lineno, colno := l.getLineColumn()
                                        errorf("%v:%v:%v: unexpected delimiter\n", l.scope, lineno, colno)
                                }
                        default:
                                st.node.children = append(st.node.children, name)
                                switch l.rune {
                                case delm:
                                        l.endCall(st)
                                case ' ':
                                        l.push(nodeArg, l.stateCallArg, 0).delm = delm
                                }
                        }
                        break state_loop
                }
        }
}

func (l *lex) stateCallArg() {
        st := l.top() // Must be a nodeArg.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        arg := st.node
                        arg.end = l.pos - 1
                        l.pop()

                        st = l.top()
                        st.node.children = append(st.node.children, arg)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(nodeArg, l.stateCallArg, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) stateSpeakDialect() {
        st := l.top() // Must be a nodeArg.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        arg := st.node
                        arg.end = l.pos - 1
                        l.pop()

                        st = l.top()
                        st.node.children = append(st.node.children, arg)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(nodeArg, l.stateSpeakScript, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) stateSpeakScript() {
        st := l.top() // Must be a nodeArg.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        if st.code == 0 {
                                if l.get() {
                                        if l.rune != '\n' { // skip \\\n
                                                lineno, colno := l.getLineColumn()
                                                errorf("%v:%v:%v: bad escape \\%v in this context\n", l.scope, lineno, colno, string(l.rune))
                                        }
                                }
                        } else {
                                l.escapeTextLine(st.node)
                        }
                case l.rune == '-' && st.code == 0: /* skip */
                case l.rune != '-' && st.code == 0:
                        st.code, st.node.pos = 1, l.pos
                case l.rune == '\n' && st.code == 1 && l.peek() == '-':
                delimiter_loop:
                        for i, r, n := l.pos, rune(0), 1; i < len(l.s); i += n {
                                switch r, n = utf8.DecodeRune(l.s[i:]); r {
                                default: break delimiter_loop
                                case '-': /* skip */
                                case delm:
                                        script := st.node
                                        script.end = l.backwardNonSpace(l.pos)

                                        l.rune, l.pos, st = delm, i+1, l.pop()

                                        st = l.top() // the $(speak) node
                                        st.node.children = append(st.node.children, script)
                                        l.endCall(st)
                                        break state_loop
                                }
                        }
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        script := st.node
                        script.end = l.pos - 1

                        l.pop()

                        st = l.top() // the $(speak) node
                        st.node.children = append(st.node.children, script)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(nodeArg, l.stateSpeakScript, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) endCall(st *lexStack) {
        call := st.node
        call.end = l.pos

        l.pop() // pop out the current nodeCall

        // Append the call to it's parent.
        t := l.top().node
        t.children = append(t.children, call)
}

/*
stateDollar:
        st.node.children = []*node{ l.new(nodeName) }
        l.step = l.stateCallee 
*/
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
                        a := l.new(nodeArg)
                        i := len(st.node.children)-1
                        st.node.children[i].end = l.pos-1
                        st.node.children = append(st.node.children, a)

                case l.rune == '\\':
                        i := len(st.node.children)-1
                        if 0 <= i {
                                l.escapeTextLine(st.node.children[i])
                        }

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
        l.step, l.pos = l.stateGlobal, 0
        end := len(l.s); for l.pos < end { l.step() }

        var t *lexStack
        for 0 < len(l.stack) {
                t = l.top() // the current top state
                l.step() // Make extra step to give it a chance handling rune(0),
                if t == l.top() {
                        l.pop() // pop out the state if the top is still there
                }

        }
        return l.pos == end
}

type pendedBuild struct {
        m *Module
        p *Context
        args Items
}

// Context hold a parse context and the current module being processed.
type Context struct {
        lexingStack []*lex
        moduleStack []*Module

        l *lex // the current lexer
        m *Module // the current module being processed
        t *template // the current template being processed

        g *namespaceEmbed // the global namespace

        templates map[string]*template

        modules map[string]*Module
        moduleOrderList []*Module
        moduleBuildList []pendedBuild

        w *worker.Worker
}

func (ctx *Context) GetModules() map[string]*Module { return ctx.modules }
func (ctx *Context) GetModuleOrderList() []*Module { return ctx.moduleOrderList }
func (ctx *Context) GetModuleBuildList() []pendedBuild { return ctx.moduleBuildList }
func (ctx *Context) ResetModules() {
        ctx.modules = make(map[string]*Module, 8)
        ctx.moduleOrderList = []*Module{}
        ctx.moduleBuildList = []pendedBuild{}
}

func (ctx *Context) CurrentScope() string {
        return ctx.l.scope
}

func (ctx *Context) CurrentLocation() location {
        return ctx.l.location()
}

func (ctx *Context) CurrentModule() *Module {
        return ctx.m
}

func (ctx *Context) NewDeferWith(m *Module) func() {
        prev := ctx.m; ctx.m = m
        return func() { ctx.m = prev }
}

func (ctx *Context) With(m *Module, work func()) {
        revert := ctx.NewDeferWith(m); defer revert()
        work()
}

/*
func (ctx *Context) expand(loc location, str string) string {
        var buf bytes.Buffer
        var exp func(s []byte) (out string, l int)
        var getRune = func(s []byte) (r rune, l int) {
                if r, l = utf8.DecodeRune(s); r == utf8.RuneError || l <= 0 {
                        errorf("bad UTF8 encoding")
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
                var args Items
                var t bytes.Buffer
                if rr == 0 {
                        t.WriteRune(r)
                        out = ctx.call(loc, t.String(), args...).Expand(ctx)
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
                                args = append(args, stringitem(t.String())); t.Reset()
                        case '$':
                                if ss, ll := exp(s[rs:]); 0 < ll {
                                        t.WriteString(ss)
                                        s, l = s[rs+ll:], l + rs + ll
                                        continue
                                } else {
                                        errorf(string(s))
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
                                                args = append(args, stringitem(t.String()))
                                        } else {
                                                name = t.String()
                                        }
                                        t.Reset()
                                }
                                out, l = ctx.call(loc, name, args...).Expand(ctx), l + rs
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
                                errorf("bad variable")
                        } else {
                                s = s[ll:]
                                buf.WriteString(ss)
                        }
                } else {
                        buf.WriteRune(r)
                }
        }

        return buf.String()
} */

func (ctx *Context) CallWith(m *Module, name string, args ...Item) (is Items) {
        return ctx.callWith(ctx.l.location(), m, name, args...)
}

func (ctx *Context) Call(name string, args ...Item) (is Items) {
        return ctx.call(ctx.l.location(), name, args...)
}

func (ctx *Context) Set(name string, items ...Item) {
        hasPrefix, prefix, parts := ctx.expandNameString(name)
        ctx.setWithDetails(hasPrefix, prefix, parts, items...)
}

func (ctx *Context) setWithDetails(hasPrefix bool, prefix string, parts []string, items ...Item) {
        if ns := ctx.getNamespaceWithDetails(hasPrefix, prefix, parts); ns != nil {
                if m, n := ns.getDefineMap(), len(parts); m != nil && 0 < n {
                        var (
                                d *define
                                found bool
                                sym = parts[n-1]
                                loc = ctx.l.location()
                        )
                        if d, found = m[sym]; !found {
                                d = new(define)
                                m[sym] = d
                        }
                        if d.readonly {
                                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                                fmt.Printf("%v:%v:%v:warning: readonly '%s'\n", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                        } else {
                                d.name, d.value, d.loc = sym, items, loc
                        }
                }
        } else {
                var loc = ctx.l.location()
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                if hasPrefix {
                        errorf("%v:%v:%v: no namespace for '%s:%s'", ctx.l.scope, lineno, colno, prefix, strings.Join(parts, "."))
                } else {
                        errorf("%v:%v:%v: no namespace for '%s'", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                }
        }
}

func (ctx *Context) call(loc location, name string, args ...Item) (is Items) {
        hasPrefix, prefix, parts := ctx.expandNameString(name)
        is = ctx.callWithDetails(loc, hasPrefix, prefix, parts, args...)
        return
}

func (ctx *Context) callWithDetails(loc location, hasPrefix bool, prefix string, parts []string, args ...Item) (is Items) {
        n := len(parts)

        // Process special symbols and builtins first.
        if !hasPrefix && n == 1 {
                switch sym := parts[0]; sym {
                case "$":  is = append(is, stringitem("$"))
                case "me": // rename: $(me) -> $(me.name)
                        parts, n = append(parts, "name"), 2
                default:
                        if f, ok := builtins[sym]; ok && f != nil {
                                is = f(ctx, loc, args)
                                return
                        }
                }
        }

        if ns := ctx.getNamespaceWithDetails(hasPrefix, prefix, parts); ns != nil {
                if m := ns.getDefineMap(); m != nil && 0 < n {
                        sym := parts[n-1]
                        if d, ok := m[sym]; ok && d != nil {
                                is = d.value
                        }
                }
        } else {
                /*
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                if hasPrefix {
                        errorf("%v:%v:%v: no namespace for '%s:%s'", ctx.l.scope, lineno, colno, prefix, strings.Join(parts, "."))
                } else {
                        errorf("%v:%v:%v: no namespace for '%s'", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                } */
        }
        return
}

func (ctx *Context) callWith(loc location, m *Module, name string, args ...Item) (is Items) {
        o := ctx.m
        ctx.m = m
        is = ctx.call(loc, meDot+name, args...)
        ctx.m = o
        return
}

// getDefine returns a define for hierarchy names like `tool:m1.m2.var`, `m1.m2.var`, etc.
func (ctx *Context) getDefine(name string) (d *define, hasPrefix bool, prefix string, parts []string) {
        hasPrefix, prefix, parts = ctx.expandNameString(name)
        d = ctx.getDefineWithDetails(hasPrefix, prefix, parts)
        return
}
func (ctx *Context) getDefineWithDetails(hasPrefix bool, prefix string, parts []string) (d *define) {
        if ns := ctx.getNamespaceWithDetails(hasPrefix, prefix, parts); ns != nil {
                if m, n := ns.getDefineMap(), len(parts); m != nil && 0 < n {
                        d, _ = m[parts[n-1]]
                }
        }
        return
}

// getNamespaceAndDetails returns a namespace for hierarchy names like `tool:m1.m2.var`, `m1.m2.var`, etc.
func (ctx *Context) getNamespaceAndDetails(name string) (ns namespace, hasPrefix bool, prefix string, parts []string) {
        hasPrefix, prefix, parts = ctx.expandNameString(name)
        ns = ctx.getNamespaceWithDetails(hasPrefix, prefix, parts)
        return
}

func (ctx *Context) getNamespaceWithDetails(hasPrefix bool, prefix string, parts []string) (ns namespace) {
        num := len(parts)

        if hasPrefix {
                if s, ok := toolsets[prefix]; ok && s != nil {
                        ns = s.toolset.getNamespace()
                } else {
                        lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: undefined toolset prefix `%s'\n",
                                ctx.l.scope, lineno, colno, prefix)
                        return
                }
        }

        if num == 1 && ns == nil {
                ns = ctx.g
                return
        }

        lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
        for i, s := range parts[0:num-1] {
                if ns != nil {
                        ns = ns.getNamespace(s)
                } else if i == 0 {
                        switch s {
                        default:
                                if m, ok := ctx.modules[s]; !ok || m == nil {
                                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: '%s' is nil\n", ctx.l.scope, lineno, colno, s)
                                        //break loop_parts
                                } else {
                                        ns = m
                                }
                        case "me":
                                if ctx.m == nil {
                                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: 'me' is nil\n", ctx.l.scope, lineno, colno)
                                        //break loop_parts
                                } else {
                                        ns = ctx.m
                                }
                        case "~":
                                if ctx.m.Toolset == nil {
                                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: no bound toolset\n", ctx.l.scope, lineno, colno)
                                        //break loop_parts
                                } else {
                                        ns = ctx.m.Toolset.getNamespace()
                                }
                        }
                }
                if ns == nil {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: `%s' is undefined scope\n", ctx.l.scope, lineno, colno,
                                strings.Join(parts[0:i+1], "."))
                        break
                }
        }
        return
}

func (ctx *Context) multipart(n *node) (*bytes.Buffer, []int) {
        b, l, pos, parts := new(bytes.Buffer), n.l, n.pos, []int{ -1 }
        for _, c := range n.children {
                if pos < c.pos { b.Write(l.s[pos:c.pos]) }; pos = c.end
                switch c.kind {
                case nodeNamePrefix: b.WriteString(":"); parts[0] = b.Len()
                case nodeNamePart:   b.WriteString("."); parts = append(parts, b.Len())
                default:   b.WriteString(ctx.nodeItems(c).Expand(ctx))
                }
        }
        if pos < n.end {
                b.Write(l.s[pos:n.end])
        }
        return b, parts
}

func (ctx *Context) expandNameString(name string) (hasPrefix bool, prefix string, parts []string) {
        if i := strings.Index(name, ":"); 0 <= i {
                prefix, hasPrefix = name[0:i], true
                name = name[i+1:]
        }
        parts = strings.Split(name, ".")
        return
}

func (ctx *Context) expandNameNode(n *node) (scoped bool, name string, parts []string) {
        pos := 0
        b, i := ctx.multipart(n)
        if 0 <= i[0] {
                pos, scoped = i[0], true
                name = string(b.Bytes()[0:pos-1])
        }

        for _, n := range i[1:] {
                parts = append(parts, string(b.Bytes()[pos:n-1]))
                pos = n
        }
        parts = append(parts, string(b.Bytes()[pos:]))
        return
}

func (ctx *Context) speak(name string, scripts ...*node) (is Items) {
        if dialect, ok := dialects[name]; ok {
                for _, sn := range scripts {
                        is = append(is, dialect(ctx, sn)...)
                }
        } else if c, e := exec.LookPath(name); e == nil {
                var args []string
                for _, s := range scripts {
                        args = append(args, s.Expand(ctx))
                }
                out := new(bytes.Buffer)
                cmd := exec.Command(c, args...)
                cmd.Stdout = out
                if err := cmd.Run(); err != nil {
                        errorf("%v: %v", name, err)
                } else {
                        is = append(is, stringitem(out.String()))
                }
        } else {
                errorf("unknown dialect %v", name)
        }
        return
}

func (ctx *Context) nodeItems(n *node) (is Items) {
        switch n.kind {
        case nodeEscape:
                switch ctx.l.s[n.pos + 1] {
                case '\n': is = Items{ stringitem(" ") }
                case '#':  is = Items{ stringitem("#") }
                }

        case nodeCall:
                var args Items
                for _, an := range n.children[1:] {
                        args = args.Concat(ctx, ctx.nodeItems(an)...)
                }
                scoped, name, parts := ctx.expandNameNode(n.children[0])
                is = ctx.callWithDetails(n.loc(), scoped, name, parts, args...)

        case nodeSpeak:
                dialect := n.children[0].Expand(ctx)
                if 1 < len(n.children) {
                        is = ctx.speak(dialect, n.children[1:]...)
                } else {
                        is = ctx.speak(dialect)
                }

        case nodeName:          fallthrough
        case nodeArg:           fallthrough
        case nodeAction:        fallthrough
        case nodeDeferredText:  fallthrough
        case nodeTargets:       fallthrough
        case nodePrerequisites: fallthrough
        case nodeImmediateText:
                var (
                        s string
                        nc = len(n.children)
                )
                if 0 < nc {
                        b, _ := ctx.multipart(n)
                        s = b.String()
                } else {
                        s = n.str()
                }
                is = Items{ stringitem(s) }

        default:
                panic(fmt.Sprintf("fixme: %v: %v (%v children)\n", n.kind, n.str(), len(n.children)))
        }
        return
}

func (ctx *Context) ItemsStrings(a ...Item) (s []string) {
        for _, i := range a {
                s = append(s, i.Expand(ctx))
        }
        return
}

func (ctx *Context) processTempNode(n *node) bool {
        if n.kind == nodeImmediateText {
                for i, c := range n.children {
                        if c.kind == nodeCall {
                                switch s := c.children[0].str(); s {
                                case "post":
                                        if ctx.t.post != nil {
                                                errorf("already posted")
                                                return true
                                        }
                                        nn := &node{
                                                l:n.l, kind:n.kind, pos:n.pos,
                                                end:c.pos-1, children: n.children[0:i],
                                        }
                                        ctx.t.post = c
                                        ctx.t.declNodes = append(ctx.t.declNodes, nn)
                                        if i+1 < len(n.children) {
                                                n.pos, n.children = c.end, n.children[i+1:]
                                                ctx.processTempNode(n)
                                        }
                                        return true
                                        
                                case "commit":
                                        nn := &node{
                                                l:n.l, kind:n.kind, pos:n.pos,
                                                end:c.pos-1, children: n.children[0:i],
                                        }
                                        ctx.t.postNodes, ctx.t.commit = append(ctx.t.postNodes, nn), c
                                        n.children, n.pos = n.children[i:], c.end
                                        return false
                                }
                        }
                }
        }
        if ctx.t.post != nil {
                ctx.t.postNodes = append(ctx.t.postNodes, n)
        } else {
                ctx.t.declNodes = append(ctx.t.declNodes, n)
        }
        return true
}

func (ctx *Context) processNode(n *node) (err error) {
        if ctx.t != nil {
                if ctx.processTempNode(n) {
                        return
                }
        }

        switch n.kind {
        case nodeCall:
                if s := strings.TrimSpace(ctx.nodeItems(n).Expand(ctx)); s != "" {
                        lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: illigal: '%v'\n", ctx.l.scope, lineno, colno, s)
                }

        case nodeImmediateText:
                if s := strings.TrimSpace(ctx.nodeItems(n).Expand(ctx)); s != "" {
                        lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: syntax error: '%v'\n", ctx.l.scope, lineno, colno, s)
                }

        case nodeDefineQuestioned:
                scoped, name, parts := ctx.expandNameNode(n.children[0])
                if is := ctx.callWithDetails(n.loc(), scoped, name, parts); is.IsEmpty(ctx) {
                        ctx.setWithDetails(scoped, name, parts, n)
                }

        case nodeDefineDeferred:
                scoped, name, parts := ctx.expandNameNode(n.children[0])
                ctx.setWithDetails(scoped, name, parts, n)

        case nodeDefineSingleColoned: fallthrough
        case nodeDefineDoubleColoned:
                scoped, name, parts := ctx.expandNameNode(n.children[0])
                ctx.setWithDetails(scoped, name, parts, ctx.nodeItems(n.children[1])...)

        case nodeDefineAppend:
                scoped, name, parts := ctx.expandNameNode(n.children[0])
                if d := ctx.getDefineWithDetails(scoped, name, parts); d != nil {
                        d.value = append(d.value, n.children[1])
                } else {
                        value := ctx.nodeItems(n.children[1])
                        ctx.setWithDetails(scoped, name, parts, value...)
                }

        case nodeDefineNot:
                panic("'!=' not implemented")

        case nodeRuleSingleColoned: fallthrough
        case nodeRuleDoubleColoned:
                r := &rule{
                        targets:Split(ctx.nodeItems(n.children[0]).Expand(ctx)),
                        prerequisites:Split(ctx.nodeItems(n.children[1]).Expand(ctx)),
                        node:n,
                }
                if 2 < len(n.children) {
                        switch a := n.children[2]; a.kind {
                        case nodeActions:
                                for _, c := range n.children[2].children {
                                        r.actions = append(r.actions, c)
                                }
                        case nodeAction:
                                r.actions = append(r.actions, a)
                        }
                }

                // Map each target to the new rule.
                for _, s := range r.targets {
                        if ctx.m != nil {
                                r.ns = ctx.m.namespaceEmbed
                                ctx.m.rules[s] = r
                                if ctx.m.goal == nil {
                                        ctx.m.goal = r
                                }
                        } else {
                                r.ns = ctx.g
                                ctx.g.rules[s] = r
                                if ctx.g.goal == nil {
                                        ctx.g.goal = r
                                }
                        }
                }

                /*
                lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                fmt.Fprintf(os.Stderr, "%v:%v:%v: %v\n", ctx.l.scope, lineno, colno, n.kind) //*/

        default:
                panic(n.kind.String()+" not implemented")

        case nodeComment:
        }
        return
}

func (ctx *Context) parseBuffer() (err error) {
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
        ctx.lexingStack = append(ctx.lexingStack, ctx.l)
        defer func() {
                ctx.lexingStack = ctx.lexingStack[0:len(ctx.lexingStack)-1]

                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                lineno, colno := ctx.l.getLineColumn()
                                fmt.Printf("%v:%v:%v: %v\n", scope, lineno, colno, se)
                        } else {
                                panic(e)
                        }
                }
        }()

        ctx.l = &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0 }
        ctx.m = nil

        if err = ctx.parseBuffer(); err != nil {
                // ...
        }
        return
}

func (ctx *Context) include(fn string) (err error) {
        var (
                f *os.File
                s []byte
        )

        if f, err = os.Open(fn); err != nil {
                return
        }

        defer f.Close()

        if s, err = ioutil.ReadAll(f); err == nil {
                err = ctx.append(fn, s)
        }
        return
}

func NewContext(scope string, s []byte, vars map[string]string) (ctx *Context, err error) {
        ctx = &Context{
                templates: make(map[string]*template, 8),
                modules: make(map[string]*Module, 8),
                l: &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0 },
                g: &namespaceEmbed{
                        defines: make(map[string]*define, len(vars) + 16),
                        rules: make(map[string]*rule, 8),
                },
                w: worker.New(),
        }

        for k, v := range vars {
                ctx.Set(k, stringitem(v))
        }

        err = ctx.parseBuffer()
        return
}

func NewContextFromFile(fn string, vars map[string]string) (ctx *Context, err error) {
        s := []byte{} // TODO: needs init script
        if ctx, err = NewContext(fn, s, vars); err == nil && ctx != nil {
                err = ctx.include(fn)
        }
        return
}
