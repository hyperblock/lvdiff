package vgcfg

// PEG, Parsing Expression Grammer file.
// The tool peg is required to generate the parse from PEG
//   go get github.com/pointlander/peg

//go:generate peg -switch -inline vgcfg.peg

import (
	"fmt"
	"strconv"
)

type baseParser struct {
	root          *Group
	curgroup      *Group
	varname       string
	varvalues     []interface{}
	islist        bool
	internalError error
}

func (p *baseParser) Prepare() {
	p.root = newGroup("__root__", nil)
	p.curgroup = p.root
	p.varname = ""
	p.varvalues = make([]interface{}, 0, 10)
	p.islist = false
}

func (p *baseParser) Root() *Group {
	return p.root
}

func (p *baseParser) SetVarName(text string) {
	if len(p.varname) != 0 {
		p.internalError = fmt.Errorf("variable name already exists: %q", p.varname)
		return
	}

	p.varname = text
}

func (p *baseParser) AddStringValue(text string) {
	p.varvalues = append(p.varvalues, text)
}

func (p *baseParser) AddIntegerValue(text string) {
	v, err := strconv.ParseInt(text, 0, 64)
	if err != nil {
		p.internalError = fmt.Errorf("can not parse integer from string %q error %q", text, err)
		return
	}

	p.varvalues = append(p.varvalues, v)
}

func (p *baseParser) AddVariable() {
	if len(p.varname) == 0 {
		p.internalError = fmt.Errorf("can not add variable: no name")
		return
	}

	if p.islist {
		p.curgroup.addVar(p.varname, p.varvalues)
	} else {
		if len(p.varvalues) != 1 {
			p.internalError = fmt.Errorf("can not add variable: multiple values but not list")
			return
		}

		p.curgroup.addVar(p.varname, p.varvalues[0])
	}

	p.varname = ""
	p.varvalues = make([]interface{}, 0, 10)
	p.islist = false
}

func (p *baseParser) SetIsList(v bool) {
	p.islist = v
}

func (p *baseParser) BeginGroup() {
	if len(p.varname) == 0 {
		p.internalError = fmt.Errorf("can not begin group: no name")
		return
	}

	subgroup := newGroup(p.varname, p.curgroup)
	p.curgroup.addSubGroup(subgroup)
	p.curgroup = subgroup
	p.varname = ""
}

func (p *baseParser) EndGroup() {
	if p.curgroup.Parent() == nil {
		p.internalError = fmt.Errorf("parent group of %q is nil", p.curgroup.Name())
		return
	}

	p.curgroup = p.curgroup.Parent()
}

func (p *baseParser) PrintComment(text string) {
	// ignore
}
