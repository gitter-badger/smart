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

type define struct {
        name string
        node []interface{} // *node, *flatstr, string
        readonly bool
        loc location
}

type rule struct {
        targets, prerequisites []string
        actions []interface{} // *node, string
        node *node
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
                nodeName:                       "call-name",
                nodeNamePrefix:                 "call-name-prefix",
                nodeNamePart:                   "call-name-part",
                nodeArg:                        "call-arg",
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
        }
)

func (k nodeType) String() string {
        return nodeTypeNames[int(k)]
}

type location struct {
        offset, end int // (node.pos, node.end)
}

// flatstr is a expanded string with a location
type flatstr struct {
        s string
        l location
}

type node struct {
        l *lex
        kind nodeType
        children []*node
        pos, end int
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

                /*
        case l.rune == '\n':
                l.lineno, l.prevColno, l.colno = l.lineno+1, l.colno, 0
        case l.runeLen > 1:
                l.colno += 2
        default:
                l.colno ++ */
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
                        l.endCall(st, 0)
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
                        st.node.children = append(st.node.children, name)
                        switch l.rune {
                        case delm:
                                l.endCall(st, 1)
                        case ' ':
                                l.push(nodeArg, l.stateCallArg, 0).delm = delm
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
                                l.endCall(st, 1)
                        } else {
                                l.push(nodeArg, l.stateCallArg, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) endCall(st *lexStack, off int) {
        call := st.node
        call.end = l.pos //- off

        /*
        lineno, colno := l.caculateLocationLineColumn(call.loc())
        fmt.Fprintf(os.Stderr, "%v:%v:%v: %v (of %v)\n", l.scope, lineno, colno, call.str(), st.node.kind) //*/

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
        args []string
}

// Context hold a parse context and the current module being processed.
type Context struct {
        lexingStack []*lex
        moduleStack []*Module

        l *lex // the current lexer
        m *Module // the current module being processed

        // variables in the context
        defines map[string]*define

        // rules in the context
        rules map[string]*rule

        modules map[string]*Module
        moduleOrderList []*Module
        moduleBuildList []pendedBuild
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
                var args []string
                var t bytes.Buffer
                if rr == 0 {
                        t.WriteRune(r)
                        out = ctx.call(loc, t.String(), args...)
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
                                                args = append(args, t.String())
                                        } else {
                                                name = t.String()
                                        }
                                        t.Reset()
                                }
                                out, l = ctx.call(loc, name, args...), l + rs
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
}

func (ctx *Context) CallWith(m *Module, name string, args ...string) string {
        return ctx.callWith(ctx.l.location(), m, name, args...)
}

func (ctx *Context) Call(name string, args ...string) string {
        return ctx.call(ctx.l.location(), name, args...)
}

func (ctx *Context) Set(name string, items ...interface{}) {
        if i := strings.Index(name, ":"); 0 <= i {
                ctx.setScoped(name[0:i], strings.Split(name[i+1:], "."), items...)
        } else {
                ctx.setMultipart(strings.Split(name, "."), items...)
        }
}

func (ctx *Context) call(loc location, name string, args ...string) string {
        if i := strings.Index(name, ":"); 0 <= i {
                return ctx.callScoped(loc, name[0:i], strings.Split(name[i+1:], "."), args...)
        }
        return ctx.callMultipart(loc, strings.Split(name, "."), args...)
}

func (ctx *Context) callScoped(loc location, name string, parts []string, args ...string) (s string) {
        if ts, ok := toolsets[name]; ok && ts != nil {
                s = ts.toolset.Call(ctx, parts, args...)
        } else {
                errorf("'%v:%v' is undefined", name, strings.Join(parts, "."))
        }
        return
}

func (ctx *Context) callMultipart(loc location, parts []string, args ...string) string {
        num := len(parts)
        vars, name := ctx.defines, parts[num-1]

        if num == 1 { // single part name
                switch {
                default:
                        if f, ok := builtins[name]; ok {
                                // Expand all arguments.
                                //fmt.Printf("%v: %v\n", name, args)
                                //for i := range args { args[i] = ctx.expand(loc, args[i]) }
                                //fmt.Printf("%v: %v\n", name, args)
                                return f(ctx, loc, args)
                        } else if *flagW {
                                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                                fmt.Printf("%v:%v:%v:warning: `%v' undefined\n", ctx.l.scope, lineno, colno, name)
                        }
                case name == "$": return "$";
                case name == "call":
                        if 0 < len(args) {
                                return ctx.call(loc, args[0], args[1:]...)
                        }
                        return ""
                case name == "me":
                        if ctx.m != nil {
                                return ctx.m.GetName(ctx) //return ctx.m.Get(ctx, "dir")
                        }
                        return ""
                }
        } else {
                /*
                var m *Module
                for i, s := range parts[0:num-1] {
                        if i == 0 {
                                switch {
                                case s == "me": m = ctx.m
                                default:     m, _ = ctx.modules[s]
                                }
                        } else {
                                m, _ = m.Children[s]
                        }
                        if m == nil {
                                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                                fmt.Printf("%v:%v:%v:warning: `%s' undefined\n", ctx.l.scope, lineno, colno,
                                        strings.Join(parts[0:i+1], "."))
                                break
                        }
                } */
                if m := ctx.getModuleMultipartName(parts); m != nil {
                        vars = m.defines
                }
        }

        if vars != nil {
                if d, ok := vars[name]; ok && d != nil {
                        return ctx.getDefineValue(d)
                }
        }

        return ""
}

func (ctx *Context) getModuleMultipartName(parts []string) (m *Module) {
        num := len(parts)
        for i, s := range parts[0:num-1] {
                if i == 0 {
                        switch {
                        case s == "me": m = ctx.m
                        default:     m, _ = ctx.modules[s]
                        }
                } else {
                        m, _ = m.Children[s]
                }
                if m == nil {
                        lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
                        fmt.Printf("%v:%v:%v:warning: `%s' undefined\n", ctx.l.scope, lineno, colno,
                                strings.Join(parts[0:i+1], "."))
                        break
                }
        }
        return
}

func (ctx *Context) getDefineValue(d *define) string {
        b, f0, fn := new(bytes.Buffer), "%s", " %s"
        for n, i := range d.node {
                f := f0[0:]
                if 0 < n { f = fn[0:] }
                /*
                switch t := i.(type) {
                case string:   fmt.Fprintf(b, f, t)
                case *flatstr: fmt.Fprintf(b, f, t.s)
                case *node:    fmt.Fprintf(b, f, ctx.expandNode(t.children[1]))
                default: errorf("unsupported '%v'", t)
                } */
                fmt.Fprintf(b, f, ctx.ItemString(i))
        }
        return b.String()
}

func (ctx *Context) ItemString(i interface{}) (s string) {
        switch t := i.(type) {
        case string:   s = t
        case *flatstr: s = t.s
        case *node:
                if nodeDefineDeferred <= t.kind && t.kind <= nodeDefineAppend {
                        s = ctx.expandNode(t.children[1])
                } else {
                        s = ctx.expandNode(t)
                }
        default:
                errorf("unsupported '%v'", t)
        }
        return
}

func (ctx *Context) ItemsStrings(a ...interface{}) (s []string) {
        for _, i := range a {
                s = append(s, ctx.ItemString(i))
        }
        return
}

func (ctx *Context) callWith(loc location, m *Module, name string, args ...string) (s string) {
        o := ctx.m
        ctx.m = m
        s = ctx.call(loc, meDot+name, args...)
        ctx.m = o
        return
}

func (ctx *Context) getMultipart(parts []string) (v *define) {
        num := len(parts)
        vars, name := ctx.defines, parts[num-1]

        if 1 < num {
                if m := ctx.getModuleMultipartName(parts); m != nil {
                        vars = m.defines
                } else {
                        vars = nil
                }
        }

        if vars != nil {
                v, _ = vars[name]
        }
        return
}

func (ctx *Context) setScoped(name string, parts []string, items...interface{}) {
        if ts, ok := toolsets[name]; ok && ts != nil {
                ts.toolset.Set(ctx, parts, items...)
        } else {
                errorf("'%v:%v' is undefined", name, strings.Join(parts, "."))
        }
}

func (ctx *Context) setMultipart(parts []string, a ...interface{}) (v *define) {
        loc, num := ctx.l.location(), len(parts)
        vars, name := ctx.defines, parts[num-1]

        if 1 < num {
                if m := ctx.getModuleMultipartName(parts); m != nil {
                        vars = m.defines
                } else {
                        vars = nil
                }
        }

        if vars == nil {
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                fmt.Printf("%v:%v:%v:warning: undefined scope '%s'\n", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                return
        }

        var ok bool
        if v, ok = vars[name]; !ok {
                v = &define{}
                vars[name] = v
        }

        if v.readonly {
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                fmt.Printf("%v:%v:%v:warning: readonly '%s'\n", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                return
        }
        
        v.name, v.node, v.loc = name, a, loc
        return
}

func (ctx *Context) _set(name string, a ...interface{}) (v *define) {
        loc := ctx.l.location()

        //fmt.Printf("set: %v, %v\n", name, a)

        if name == "me" {
                errorf("%v: `me' is readonly", loc)
                return
        }

        vars := ctx.defines
        if strings.HasPrefix(name, meDot) {
                if ctx.m == nil {
                        /* lineno, colno := ctx.l.getLineColumn()
                        fmt.Printf("%v:%v:%v:warning: no module defined for '%v'\n",
                                ctx.l.scope, lineno, colno, name) */
                        return
                }
                vars, name = ctx.m.defines, strings.TrimPrefix(name, meDot)
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
        
        v.name, v.node, v.loc = name, a, ctx.l.location()
        return
}

func (ctx *Context) multipart(n *node) (*bytes.Buffer, []int) {
        b, l, pos, parts := new(bytes.Buffer), n.l, n.pos, []int{ -1 }
        for _, c := range n.children {
                if pos < c.pos { b.Write(l.s[pos:c.pos]) }; pos = c.end
                switch c.kind {
                case nodeNamePrefix: b.WriteString(":"); parts[0] = b.Len()
                case nodeNamePart:   b.WriteString("."); parts = append(parts, b.Len())
                default:   b.WriteString(ctx.expandNode(c))
                }
        }
        if pos < n.end {
                b.Write(l.s[pos:n.end])
        }
        return b, parts
}

func (ctx *Context) expandName(n *node) (parts []string, scoped bool, name string) {
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

func (ctx *Context) expandNode(n *node) string {
        nc := len(n.children)

        switch n.kind {
        case nodeEscape:
                switch ctx.l.s[n.pos + 1] {
                case '\n': return " "
                case '#':  return "#"
                }
                return ""

        case nodeCall:
                var parts, args []string
                for _, an := range n.children[1:] {
                        args = append(args, ctx.expandNode(an))
                }

                /*
                callScoped, pos := false, 0
                b, i := ctx.multipart(n.children[0])
                if 0 <= i[0] {
                        pos, callScoped = i[0], true
                }

                for _, n := range i[1:] {
                        parts = append(parts, string(b.Bytes()[pos:n-1]))
                        pos = n
                }
                parts = append(parts, string(b.Bytes()[pos:])) */
                parts, callScoped, name := ctx.expandName(n.children[0])
                if callScoped {
                        //name := string(b.Bytes()[0:i[0]-1])
                        return ctx.callScoped(n.loc(), name, parts, args...)
                } else {
                        return ctx.callMultipart(n.loc(), parts, args...)
                }

        case nodeName:          fallthrough
        case nodeArg:           fallthrough
        case nodeDeferredText:  fallthrough
        case nodeTargets:       fallthrough
        case nodePrerequisites: fallthrough
        case nodeImmediateText:
                if 0 < nc {
                        b, _ := ctx.multipart(n)
                        return b.String()
                } else {
                        return n.str()
                }
        default:
                panic(fmt.Sprintf("fixme: %v: %v (%v)\n", n.kind, n.str(), nc))
        }

        return ""
}

func (ctx *Context) processNode(n *node) (err error) {
        /*
        lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
        fmt.Fprintf(os.Stderr, "%v:%v:%v: '%v'\n", ctx.l.scope, lineno, colno, n.kind) //*/

        switch n.kind {
        case nodeCall:
                if s := strings.TrimSpace(ctx.expandNode(n)); s != "" {
                        lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: illigal: '%v'\n",
                                ctx.l.scope, lineno, colno, s)
                }

        case nodeImmediateText:
                if s := strings.TrimSpace(ctx.expandNode(n)); s != "" {
                        lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: syntax error: '%v'\n",
                                ctx.l.scope, lineno, colno, s)
                }

        case nodeDefineQuestioned:
                parts, scoped, name := ctx.expandName(n.children[0])
                if scoped {
                        if ctx.callScoped(n.loc(), name, parts) == "" {
                                ctx.setScoped(name, parts, n)
                        }
                } else if ctx.callMultipart(n.loc(), parts) == "" {
                        ctx.setMultipart(parts, n)
                }

        case nodeDefineDeferred:
                parts, scoped, name := ctx.expandName(n.children[0])
                if scoped {
                        ctx.setScoped(name, parts, n)
                } else {
                        ctx.setMultipart(parts, n)
                }

        case nodeDefineSingleColoned: fallthrough
        case nodeDefineDoubleColoned:
                parts, scoped, name := ctx.expandName(n.children[0])
                str := ctx.expandNode(n.children[1])
                if scoped {
                        ctx.setScoped(name, parts, str)
                } else {
                        ctx.setMultipart(parts, str)
                }

        case nodeDefineAppend:
                parts, scoped, name := ctx.expandName(n.children[0])
                if scoped {
                        ctx.setScoped(name, parts, n) // FIXME: append instead of replace
                } else {
                        if d := ctx.getMultipart(parts); d != nil {
                                deferred := true
                                if 0 < len(d.node) {
                                        _, deferred = d.node[0].(*node)
                                }

                                if deferred {
                                        d.node = append(d.node, n)
                                } else {
                                        vc := n.children[1]
                                        v := ctx.expandNode(vc)
                                        d.node = append(d.node, &flatstr{ v, vc.loc() })
                                }
                        } else {
                                ctx.setMultipart(parts, n)
                        }
                }

        case nodeDefineNot:
                panic("'!=' not implemented")

        case nodeRuleSingleColoned: fallthrough
        case nodeRuleDoubleColoned:
                r := &rule{
                        targets:Split(ctx.expandNode(n.children[0])),
                        prerequisites:Split(ctx.expandNode(n.children[1])),
                        node:n,
                }
                if 2 < len(n.children) {
                        for c, _ := range n.children[2].children {
                                r.actions = append(r.actions, c)
                        }
                }

                // Map each target to the new rule.
                for _, s := range r.targets {
                        if ctx.m != nil {
                                ctx.m.rules[s] = r
                        } else {
                                ctx.rules[s] = r
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

        //fmt.Printf("_parse: %v, %v, %v\n", len(ctx.lexingStack), len(ctx.l.nodes), len(ctx.defines))

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

        //fmt.Printf("include: %v\n", fn)

        s, err = ioutil.ReadAll(f)
        if err == nil {
                err = ctx.append(fn, s)
        }

        return
}

func NewContext(scope string, s []byte, vars map[string]string) (ctx *Context, err error) {
        ctx = &Context{
                l: &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0 },
                defines: make(map[string]*define, len(vars) + 16),
                modules: make(map[string]*Module, 8),
                rules: make(map[string]*rule, 8),
        }

        for k, v := range vars {
                ctx.Set(k, v)
        }

        err = ctx._parse()
        return
}

func NewContextFromFile(fn string, vars map[string]string) (ctx *Context, err error) {
        s := []byte{} // TODO: needs init script
        if ctx, err = NewContext(fn, s, vars); err == nil && ctx != nil {
                err = ctx.include(fn)
        }
        return
}
