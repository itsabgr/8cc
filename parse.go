package main

import (
	"errors"
	"fmt"
)

const MAX_ARGS = 6

var globals []*Ast
var fparams []*Ast
var locals []*Ast
var labelseq = 0

var ctype_int = &Ctype{CTYPE_INT, nil, 0}
var ctype_char = &Ctype{CTYPE_CHAR, nil, 0}

func ast_uop(typ int, ctype *Ctype, operand *Ast) *Ast {
	r := &Ast{}
	r.typ = typ
	r.ctype = ctype
	r.unary.operand = operand
	return r
}

func ast_binop(typ int, left *Ast, right *Ast) *Ast {
	r := &Ast{}
	r.typ = typ
	r.ctype = result_type(byte(typ), left.ctype, right.ctype)
	if typ != '=' && convert_array(left.ctype).typ != CTYPE_PTR &&
		convert_array(right.ctype).typ == CTYPE_PTR {
		r.binop.left = right
		r.binop.right = left
	} else {
		r.binop.left = left
		r.binop.right = right
	}
	return r
}

func ast_int(val int) *Ast {
	r := &Ast{}
	r.typ = AST_LITERAL
	r.ctype = ctype_int
	r.ival = val
	return r
}

func ast_char(c byte) *Ast {
	r := &Ast{}
	r.typ = AST_LITERAL
	r.ctype = ctype_char
	r.c = c
	return r
}

func make_label() Cstring {
	seq := labelseq
	labelseq++
	s := fmt.Sprintf(".L%d", seq)
	return NewCstringFromLiteral(s)
}

func ast_lvar(ctype *Ctype, name Cstring) *Ast {
	r := &Ast{}
	r.typ = AST_LVAR
	r.ctype = ctype
	r.variable.lname = name
	if locals != nil {
		locals = append(locals, r)
	}
	return r
}

func ast_gvar(ctype *Ctype, name Cstring, filelocal bool) *Ast {
	r := &Ast{}
	r.typ = AST_GVAR
	r.ctype = ctype
	r.gvar.gname = name
	if filelocal {
		r.gvar.glabel = make_label()
	} else {
		r.gvar.glabel = name
	}
	globals = append(globals, r)
	return r
}

func ast_string(str Cstring) *Ast {
	r := &Ast{}
	r.typ = AST_STRING
	r.ctype = make_array_type(ctype_char, strlen(str)+1)
	r.str.val = str
	r.str.slabel = make_label()
	globals = append(globals, r)
	return r
}

func ast_funcall(ctype *Ctype, fname Cstring, args []*Ast) *Ast {
	r := &Ast{}
	r.typ = AST_FUNCALL
	r.ctype = ctype
	r.fnc.fname = fname
	r.fnc.args = args
	return r
}

func ast_func(rettype *Ctype, fname Cstring, params []*Ast, locals []*Ast, body Block) *Ast {
	r := &Ast{}
	r.typ = AST_FUNC
	r.ctype = rettype
	r.fnc.fname = fname
	r.fnc.params = params
	r.fnc.locals = locals
	r.fnc.body = body
	return r
}

func ast_decl(variable *Ast, init *Ast) *Ast {
	r := &Ast{}
	r.typ = AST_DECL
	r.ctype = nil
	r.decl.declvar = variable
	r.decl.declinit = init
	return r
}

func ast_array_init(arrayinit []*Ast) *Ast {
	r := &Ast{}
	r.typ = AST_ARRAY_INIT
	r.ctype = nil
	r.array_initializer.arrayinit = arrayinit
	return r
}

func ast_if(cond *Ast, then Block, els Block) *Ast {
	r := &Ast{}
	r.typ = AST_IF
	r.ctype = nil
	r._if.cond = cond
	r._if.then = then
	r._if.els = els
	return r
}

func ast_for(init *Ast, cond *Ast, step *Ast, body Block) *Ast {
	r := &Ast{}
	r.typ = AST_FOR
	r.ctype = nil
	r._for.init = init
	r._for.cond = cond
	r._for.step = step
	r._for.body = body
	return r
}

func ast_return(retval *Ast) *Ast {
	r := &Ast{}
	r.typ = AST_RETURN
	r.ctype = nil
	r._return.retval = retval
	return r
}

func make_ptr_type(ctype *Ctype) *Ctype {
	r := &Ctype{}
	r.typ = CTYPE_PTR
	r.ptr = ctype
	return r
}

func make_array_type(ctype *Ctype, size int) *Ctype {
	r := &Ctype{}
	r.typ = CTYPE_ARRAY
	r.ptr = ctype
	r.size = size
	return r
}

func find_var(name Cstring) *Ast {
	for _, v := range fparams {
		if strcmp(name, v.variable.lname) == 0 {
			return v
		}
	}

	for _, v := range locals {
		if strcmp(name, v.variable.lname) == 0 {
			return v
		}
	}

	for _, v := range globals {
		if strcmp(name, v.variable.lname) == 0 {
			return v
		}
	}

	return nil
}

func ensure_lvalue(ast *Ast) {
	switch ast.typ {
	case AST_LVAR, AST_GVAR,AST_DEREF:
		return
	}
	_error("lvalue expected, but got %s", ast)
	return
}

func is_right_assoc(tok *Token) bool {
	return tok.v.punct == '='
}

func priority(tok *Token) int {
	switch tok.v.punct {
	case '=':
		return 1
	case PUNCT_EQ:
		return 2
	case '<','>':
		return 3
	case '+','-':
		return 4
	case '*', '/':
		return 5
	default:
		return -1
	}
}

func read_func_args(fname Cstring) *Ast {
	var args []*Ast
	for {
		tok := read_token()
		if is_punct(tok, ')') {
			break
		}
		unget_token(tok)
		args = append(args, read_expr(0))
		tok = read_token()
		if is_punct(tok, ')') {
			break
		}
		if !is_punct(tok, ',') {
			_error("Unexpected token: '%s'", tok)
		}
	}
	if MAX_ARGS < len(args) {
		_error("Too many arguments: %s", fname)
	}
	return ast_funcall(ctype_int, fname, args)
}

func read_ident_or_func(name Cstring) *Ast {
	ch := read_token()
	if is_punct(ch, '(') {
		return read_func_args(name)
	}
	unget_token(ch)

	v := find_var(name)
	if v == nil {
		_error("Undefined varaible: %s", name)
	}
	return v
}

func read_prim() *Ast {
	tk := read_token()
	if tk == nil {
		return nil
	}
	switch tk.typ {
	case TTYPE_IDENT:
		return read_ident_or_func(tk.v.sval)
	case TTYPE_INT:
		return ast_int(tk.v.ival)
	case TTYPE_CHAR:
		return ast_char(tk.v.c)
	case TTYPE_STRING:
		return ast_string(tk.v.sval)
	case TTYPE_PUNCT:
		_error("unexpected character: '%c'", tk.v.punct)
	default:
		_error("Don't know how to handle '%d'", tk.typ)
	}

	return nil
}

func result_type_int(op byte, a *Ctype, b *Ctype) (*Ctype, error) {
	if a.typ > b.typ {
		b, a = a, b
	}

	default_err := errors.New("")
	if b.typ == CTYPE_PTR {
		if op == '=' {
			return a, nil
		}
		if op != '+' && op != '-' {
			return nil, default_err
		}
		if a.typ != CTYPE_INT {
			return nil, default_err
		}
		return b, nil
	}

	switch a.typ {
	case CTYPE_VOID:
		return nil, default_err
	case CTYPE_INT:
		fallthrough
	case CTYPE_CHAR:
		switch b.typ {
		case CTYPE_INT:
			fallthrough
		case CTYPE_CHAR:
			return ctype_int, nil
		case CTYPE_ARRAY:
			fallthrough
		case CTYPE_PTR:
			return b, nil
		}
		_error("internal error")
	case CTYPE_ARRAY:
		return nil, default_err
	default:
		_error("internal error")
	}

	return nil, default_err
}

func read_subscript_expr(ast *Ast) *Ast {
	sub := read_expr(0)
	expect(']')
	t := ast_binop('+', ast, sub)
	return ast_uop(AST_DEREF, t.ctype.ptr, t)
}

func read_postfix_expr() *Ast {
	r := read_prim()
	for  {
		tok := read_token()
		if tok == nil {
			return r
		}
		if is_punct(tok, '[') {
			r = read_subscript_expr(r)
		} else if is_punct(tok, PUNCT_INC) || is_punct(tok, PUNCT_DEC)  {
			r = ast_uop(tok.v.punct, r.ctype, r)
			return r
		} else {
			unget_token(tok)
			return r
		}
	}
}

func convert_array(ctype *Ctype) *Ctype {
	if ctype.typ != CTYPE_ARRAY {
		return ctype
	}
	return make_ptr_type(ctype.ptr)
}

func result_type(op byte, a *Ctype, b *Ctype) *Ctype {
	ret, err := result_type_int(op, convert_array(a), convert_array(b))
	if err != nil {
		_error("incompatible operands: %c: <%s> and <%s>",
			op, a, b)
	}
	return ret
}

func read_unary_expr() *Ast {
	tok := read_token()
	if tok.typ != TTYPE_PUNCT {
		unget_token(tok)
		return read_postfix_expr()
	}
	if is_punct(tok, '(') {
		r := read_expr(0)
		expect(')')
		return r
	}
	if is_punct(tok, '&') {
		operand := read_unary_expr()
		ensure_lvalue(operand)
		return ast_uop(AST_ADDR, make_ptr_type(operand.ctype), operand)
	}
	if is_punct(tok, '*') {
		operand := read_unary_expr()
		ctype := convert_array(operand.ctype) // looks no need to call convert_array.
		if ctype.typ != CTYPE_PTR {
			_error("pointer type expected, but got %", ctype)
		}
		return ast_uop(AST_DEREF, operand.ctype.ptr, operand)
	}
	unget_token(tok)
	return read_prim()
}

func read_expr(prec int) *Ast {
	ast := read_unary_expr()
	if ast == nil {
		return nil
	}
	for {
		tok := read_token()
		if tok.typ != TTYPE_PUNCT {
			unget_token(tok)
			return ast
		}
		prec2 := priority(tok)
		if prec2 < 0 || prec2 < prec {
			unget_token(tok)
			return ast
		}

		if is_punct(tok, '=') {
			ensure_lvalue(ast)
		}
		var prec_incr int
		if is_right_assoc(tok) {
			prec_incr = 0
		} else {
			prec_incr = 1
		}
		rest := read_expr(prec2 + prec_incr)
		ast = ast_binop(tok.v.punct, ast, rest)
	}
	return ast
}

func get_ctype(tok *Token) *Ctype {
	if tok == nil {
		return nil
	}
	if tok.typ != TTYPE_IDENT {
		return nil
	}

	if strcmp(tok.v.sval, NewCstringFromLiteral("int")) == 0 {
		return ctype_int
	}
	if strcmp(tok.v.sval, NewCstringFromLiteral("char")) == 0 {
		return ctype_char
	}

	return nil
}

func is_type_keyword(tok *Token) bool {
	return get_ctype(tok) != nil
}

func expect(punct byte) {
	tok := read_token()
	if !is_punct(tok, int(punct)) {
		_error("'%c' expected but got %s", punct, tok)
	}
}

func read_decl_array_init_int(ctype *Ctype) *Ast {
	tok := read_token()
	if ctype.ptr.typ == CTYPE_CHAR && tok.typ == TTYPE_STRING {
		return ast_string(tok.v.sval)
	}

	if !is_punct(tok, '{') {
		_error("Expected an initializer list, but got %s", tok)
	}
	var initlist []*Ast
	for {
		tok := read_token()
		if is_punct(tok, '}') {
			break
		}
		unget_token(tok)
		init := read_expr(0)
		initlist = append(initlist, init)
		result_type('=', init.ctype, ctype.ptr)
		tok = read_token()
		if !is_punct(tok, ',') {
			unget_token(tok)
		}
	}

	return ast_array_init(initlist)
}

func read_decl_spec() *Ctype {
	tok := read_token()
	ctype := get_ctype(tok)
	if ctype == nil {
		_error("Type expected, but got %s", tok)
	}
	for {
		tok = read_token()
		if !is_punct(tok, '*') {
			unget_token(tok)
			return ctype
		}
		// pointer
		ctype = make_ptr_type(ctype)
	}
	return ctype
}

func read_decl_array_init(v *Ast) *Ast {
	var init *Ast
	if v.ctype.typ == CTYPE_ARRAY {
		init = read_decl_array_init_int(v.ctype)
		var length int
		if init.typ == AST_STRING {
			length = strlen(init.str.val) + 1
		} else {
			length = len(init.array_initializer.arrayinit)
		}
		if v.ctype.size == -1 {
			v.ctype.size = length
		} else if v.ctype.size != length {
			_error("Invalid array initializer: expected %d items but got %d",
				v.ctype.size, length)
		}
	} else {
		init = read_expr(0)
	}
	expect(';')
	return ast_decl(v, init)
}

func read_array_dimensions_int() *Ctype {
	tok := read_token()
	if !is_punct(tok, '[') {
		unget_token(tok)
		return nil;
	}
	dim := -1
	tok = peek_token()
	if !is_punct(tok, ']') {
		size := read_expr(0)
		if size.typ != AST_LITERAL || size.ctype.typ != CTYPE_INT {
			_error("Integer expected, but got %s", size)
		}
		dim = size.ival
	}
	expect(']')
	sub := read_array_dimensions_int()
	if sub != nil {
		if sub.size == -1 && dim == -1 {
			_error("Array size is not specified")
		}
		return make_array_type(sub, dim)
	}

	return  make_array_type(nil, dim)
}

func read_array_dimensions(basetype *Ctype) *Ctype {
	ctype := read_array_dimensions_int()
	if ctype == nil {
		return basetype
	}
	p := ctype
	for ; p.ptr != nil; p = p.ptr {}
	p.ptr = basetype
	return ctype
}

func read_decl() *Ast {
	ctype := read_decl_spec()
	varname := read_token()
	if varname.typ != TTYPE_IDENT {
		_error("Identifier expected, but got %s", varname)
	}
	ctype = read_array_dimensions(ctype)
	variable := ast_lvar(ctype, varname.v.sval)
	tok := read_token()
	if is_punct(tok,'=') {
		return read_decl_array_init(variable);
	}
	unget_token(tok)
	expect(';')
	return ast_decl(variable, nil)
}

func read_if_stmt() *Ast {
	expect('(')
	cond := read_expr(0)
	expect(')')
	expect('{')
	then := read_block()
	tok := read_token()
	if tok == nil || tok.typ != TTYPE_IDENT || strcmp(tok.v.sval, NewCstringFromLiteral("else")) != 0 {
		unget_token(tok)
		return ast_if(cond, then, nil)
	}
	expect('{')
	els := read_block()
	return ast_if(cond, then, els)
}

func read_opt_decl_or_stmt() *Ast {
	tok := read_token()
	if is_punct(tok, ';') {
		return nil
	}
	unget_token(tok)
	return read_decl_or_stmt()
}

func read_opt_expr() *Ast {
	tok := read_token()
	if is_punct(tok, ';') {
		return nil
	}
	unget_token(tok)
	r := read_expr(0)
	expect(';')
	return r
}

func read_for_stmt() *Ast {
	expect('(')
	init := read_opt_decl_or_stmt()
	cond := read_opt_expr()
	var step *Ast
	if is_punct(peek_token(), ')') {
		step = nil
	} else {
		step = read_expr(0)
	}
	expect(')')
	expect('{')
	body := read_block()
	return ast_for(init, cond, step, body)
}

func read_return_stmt() *Ast {
	retval := read_expr(0)
	expect(';')
	return ast_return(retval)
}

func is_ident(tok *Token, s string) bool {
	return tok.typ == TTYPE_IDENT && strcmp(tok.v.sval, NewCstringFromLiteral(s)) == 0
}

func read_stmt() *Ast {
	tok := read_token()
	if is_ident(tok, "if") {
		return read_if_stmt()
	}
	if is_ident(tok, "for") {
		return read_for_stmt();
	}
	if is_ident(tok, "return") {
		return read_return_stmt();
	}
	unget_token(tok)
	r := read_expr(0)
	expect(';')
	return r
}

func read_decl_or_stmt() *Ast {
	tok := peek_token()
	if tok == nil {
		return nil
	}

	if is_type_keyword(tok) {
		return read_decl()
	} else {
		return read_stmt()
	}
}

func read_block() Block {
	var r []*Ast

	for {
		stmt := read_decl_or_stmt()
		if stmt != nil {
			r = append(r, stmt)
		}
		if stmt == nil {
			break
		}
		tok := read_token()
		if is_punct(tok, '}') {
			break
		}
		unget_token(tok)
	}

	return Block(r)
}

func read_params() []*Ast {
	var params []*Ast
	pt := read_token()
	if is_punct(pt, ')') {
		return nil
	}
	unget_token(pt)
	for {
		ctype := read_decl_spec()
		pname := read_token()
		if pname.typ != TTYPE_IDENT {
			_error("Identifier expected, but got %s", pname)
		}
		ctype = read_array_dimensions(ctype)
		if ctype.typ == CTYPE_ARRAY {
			ctype = make_ptr_type(ctype.ptr)
		}
		params = append(params, ast_lvar(ctype, pname.v.sval))
		tok := read_token()
		if is_punct(tok, ')') {
			return params
		}
		if !is_punct(tok, ',') {
			_error("Comma expected, but got %s", tok)
		}
	}
	return params // this is never reached
}

func read_func_decl() *Ast {
	tok := peek_token()
	if tok == nil {
		return nil
	}
	rettype := read_decl_spec()
	fname := read_token()
	if fname.typ != TTYPE_IDENT {
		_error("Function name expected, but got %s", fname)
	}
	expect('(')
	fparams = read_params()
	expect('{')
	locals = make([]*Ast, 0)
	body := read_block()
	r := ast_func(rettype, fname.v.sval, fparams, locals, body)
	locals = nil
	fparams = nil
	return r
}

func read_func_list() []*Ast {
	var func_list []*Ast

	for {
		fnc := read_func_decl()
		if fnc == nil {
			return func_list
		}
		func_list = append(func_list, fnc)
	}

	return nil
}

func (ctype *Ctype) String() string {
	switch ctype.typ {
	case CTYPE_VOID:
		return "void"
	case CTYPE_INT:
		return "int"
	case CTYPE_CHAR:
		return "char"
	case CTYPE_PTR:
		return fmt.Sprintf("*%s", ctype.ptr)
	case CTYPE_ARRAY:
		return fmt.Sprintf("[%d]%s", ctype.size, ctype.ptr)
	default:
		_error("Unknown ctype: %d", ctype)
	}

	return ""
}

type Block []*Ast

func (block Block) String() string {
	s := "{"
	for _, v := range block {
		s += v.String()
		s += ";"
	}
	s += "}"
	return s
}

func (ast *Ast) String() string {
	if ast == nil {
		return "(null)"
	}
	switch ast.typ {
	case AST_LITERAL:
		switch ast.ctype.typ {
		case CTYPE_INT:
			return fmt.Sprintf("%d", ast.ival)
		case CTYPE_CHAR:
			return fmt.Sprintf("'%c'", ast.c)
		default:
			_error("internal error")
			return ""
		}
	case AST_STRING:
		return fmt.Sprintf("\"%s\"", quote_cstring(ast.str.val))
	case AST_LVAR:
		return fmt.Sprintf("%s", ast.variable.lname)
	case AST_GVAR:
		return fmt.Sprintf("%s", ast.gvar.gname)
	case AST_FUNCALL:
		s := fmt.Sprintf("(%s)%s(", ast.ctype, ast.fnc.fname)
		for i, v := range ast.fnc.args {
			s += v.String()
			if i < len(ast.fnc.args)-1 {
				s += ","
			}
		}
		s += ")"
		return s
	case AST_FUNC:
		s := fmt.Sprintf("(%s)%s(",
			ast.ctype,
			ast.fnc.fname)
		for i, p := range ast.fnc.params {
			s += fmt.Sprintf("%s %s", p.ctype, p)
			if i < (len(ast.fnc.params) - 1) {
				s += ","
			}
		}
		s += fmt.Sprintf(")%s",
			ast.fnc.body)
		return s
	case AST_DECL:
		s := fmt.Sprintf("(decl %s %s",
			ast.decl.declvar.ctype,
			ast.decl.declvar.variable.lname)
		if ast.decl.declinit != nil {
			s += fmt.Sprintf(" %s", ast.decl.declinit)
		}
		s += ")"
		return s
	case AST_ARRAY_INIT:
		s := "{"
		for i, v := range ast.array_initializer.arrayinit {
			s += v.String()
			if i != len(ast.array_initializer.arrayinit)-1 {
				s += ","
			}
		}
		s += "}"
		return s
	case AST_ADDR:
		return fmt.Sprintf("(& %s)", ast.unary.operand)
	case AST_DEREF:
		return fmt.Sprintf("(* %s)", ast.unary.operand)
	case AST_IF:
		s := fmt.Sprintf("(if %s %s",
			ast._if.cond,
			ast._if.then)
		if ast._if.els != nil {
			s += fmt.Sprintf(" %s", ast._if.els)
		}
		s += ")"
		return s
	case AST_FOR:
		s := fmt.Sprintf("(for %s %s %s %s)",
			ast._for.init,
			ast._for.cond,
			ast._for.step,
			ast._for.body)
		return s
	case AST_RETURN:
		return fmt.Sprintf("(return %s)", ast._return.retval)
	case PUNCT_INC:
		return fmt.Sprintf("(%s ++)", ast.unary.operand)
	case PUNCT_DEC:
		return fmt.Sprintf("(%s --)", ast.unary.operand)
	default:
		left := ast.binop.left
		right := ast.binop.right
		var s string
		if ast.typ == PUNCT_EQ {
			s += "(== "
		} else {
			s += fmt.Sprintf("(%c ", ast.typ)
		}
		s += fmt.Sprintf("%s %s)", left, right)
		return s
	}
}
