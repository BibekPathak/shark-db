package sql

import (
	"fmt"
	"strconv"
)

type Parser struct {
	lexer *Lexer
	cur   Token
	peek  Token
}

func NewParser(input string) *Parser {
	p := &Parser{lexer: NewLexer(input)}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.cur = p.peek
	p.peek = p.lexer.NextToken()
}

func (p *Parser) Parse() (Node, error) {
	if p.cur.Type == TOKEN_EOF {
		return nil, fmt.Errorf("empty statement")
	}

	switch p.cur.Type {
	case TOKEN_SELECT:
		return p.parseSelect()
	case TOKEN_INSERT:
		return p.parseInsert()
	case TOKEN_UPDATE:
		return p.parseUpdate()
	case TOKEN_DELETE:
		return p.parseDelete()
	case TOKEN_CREATE:
		return p.parseCreate()
	case TOKEN_DROP:
		return p.parseDrop()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.cur.Value)
	}
}

func (p *Parser) parseSelect() (*SelectStmt, error) {
	stmt := &SelectStmt{}

	if p.cur.Type != TOKEN_SELECT {
		return nil, fmt.Errorf("expected SELECT")
	}
	p.nextToken()

	if p.cur.Type == TOKEN_MULT {
		stmt.Columns = []string{"*"}
		p.nextToken()
	} else {
		for {
			if p.cur.Type != TOKEN_IDENT && p.cur.Type != TOKEN_DOT {
				return nil, fmt.Errorf("expected column name")
			}
			col := p.cur.Value
			p.nextToken()
			if p.cur.Type == TOKEN_DOT {
				p.nextToken()
				if p.cur.Type != TOKEN_IDENT {
					return nil, fmt.Errorf("expected column name after DOT")
				}
				col = col + "." + p.cur.Value
				p.nextToken()
			}
			stmt.Columns = append(stmt.Columns, col)

			if p.cur.Type != TOKEN_COMMA {
				break
			}
			p.nextToken()
		}
	}

	if p.cur.Type != TOKEN_FROM {
		return nil, fmt.Errorf("expected FROM")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	if p.cur.Type == TOKEN_WHERE {
		p.nextToken()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		stmt.Where = expr
	}

	if p.cur.Type == TOKEN_ORDER {
		p.nextToken()
		if p.cur.Type != TOKEN_BY {
			return nil, fmt.Errorf("expected BY after ORDER")
		}
		p.nextToken()

		for {
			if p.cur.Type != TOKEN_IDENT {
				return nil, fmt.Errorf("expected column name in ORDER BY")
			}
			orderCol := OrderByCol{Column: p.cur.Value, Desc: false}
			p.nextToken()

			if p.cur.Type == TOKEN_DESC {
				orderCol.Desc = true
				p.nextToken()
			} else if p.cur.Type == TOKEN_ASC {
				p.nextToken()
			}

			stmt.OrderBy = append(stmt.OrderBy, orderCol)

			if p.cur.Type != TOKEN_COMMA {
				break
			}
			p.nextToken()
		}
	}

	if p.cur.Type == TOKEN_LIMIT {
		p.nextToken()
		if p.cur.Type == TOKEN_NUMBER {
			stmt.Limit, _ = strconv.Atoi(p.cur.Value)
			p.nextToken()
		}
	}

	return stmt, nil
}

func (p *Parser) parseInsert() (*InsertStmt, error) {
	stmt := &InsertStmt{}

	if p.cur.Type != TOKEN_INSERT {
		return nil, fmt.Errorf("expected INSERT")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_INTO {
		return nil, fmt.Errorf("expected INTO")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	if p.cur.Type != TOKEN_VALUES {
		return nil, fmt.Errorf("expected VALUES")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_LPAREN {
		return nil, fmt.Errorf("expected (")
	}
	p.nextToken()

	if p.cur.Type == TOKEN_STRING || p.cur.Type == TOKEN_NUMBER || p.cur.Type == TOKEN_IDENT {
		stmt.Key = p.cur.Value
		p.nextToken()
	} else {
		return nil, fmt.Errorf("expected key value")
	}

	if p.cur.Type == TOKEN_COMMA {
		p.nextToken()
	} else {
		return nil, fmt.Errorf("expected comma between key and value")
	}

	if p.cur.Type == TOKEN_STRING {
		stmt.Value = p.cur.Value
		p.nextToken()
	} else if p.cur.Type == TOKEN_IDENT {
		stmt.Value = p.cur.Value
		p.nextToken()
	} else {
		return nil, fmt.Errorf("expected value")
	}

	if p.cur.Type != TOKEN_RPAREN {
		return nil, fmt.Errorf("expected )")
	}
	p.nextToken()

	return stmt, nil
}

func (p *Parser) parseUpdate() (*UpdateStmt, error) {
	stmt := &UpdateStmt{}

	if p.cur.Type != TOKEN_UPDATE {
		return nil, fmt.Errorf("expected UPDATE")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	if p.cur.Type != TOKEN_SET {
		return nil, fmt.Errorf("expected SET")
	}
	p.nextToken()

	for {
		if p.cur.Type != TOKEN_IDENT {
			return nil, fmt.Errorf("expected column name")
		}
		col := p.cur.Value
		p.nextToken()

		if p.cur.Type != TOKEN_EQ {
			return nil, fmt.Errorf("expected =")
		}
		p.nextToken()

		val, err := p.parsePrimaryExpr()
		if err != nil {
			return nil, err
		}

		stmt.SetClauses = append(stmt.SetClauses, SetClause{Column: col, Value: val})

		if p.cur.Type != TOKEN_COMMA {
			break
		}
		p.nextToken()
	}

	if p.cur.Type == TOKEN_WHERE {
		p.nextToken()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		stmt.Where = expr
	}

	return stmt, nil
}

func (p *Parser) parseDelete() (*DeleteStmt, error) {
	stmt := &DeleteStmt{}

	if p.cur.Type != TOKEN_DELETE {
		return nil, fmt.Errorf("expected DELETE")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_FROM {
		return nil, fmt.Errorf("expected FROM")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	if p.cur.Type == TOKEN_WHERE {
		p.nextToken()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		stmt.Where = expr
	}

	return stmt, nil
}

func (p *Parser) parseCreate() (Node, error) {
	if p.cur.Type != TOKEN_CREATE {
		return nil, fmt.Errorf("expected CREATE")
	}
	p.nextToken()

	switch p.cur.Type {
	case TOKEN_TABLE:
		return p.parseCreateTable()
	case TOKEN_INDEX:
		return p.parseCreateIndex()
	default:
		return nil, fmt.Errorf("expected TABLE or INDEX")
	}
}

func (p *Parser) parseCreateTable() (*CreateTableStmt, error) {
	stmt := &CreateTableStmt{}

	if p.cur.Type != TOKEN_TABLE {
		return nil, fmt.Errorf("expected TABLE")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	if p.cur.Type != TOKEN_LPAREN {
		return nil, fmt.Errorf("expected (")
	}
	p.nextToken()

	for {
		if p.cur.Type == TOKEN_RPAREN {
			p.nextToken()
			break
		}

		if p.cur.Type != TOKEN_IDENT {
			return nil, fmt.Errorf("expected column name")
		}
		stmt.Columns = append(stmt.Columns, p.cur.Value)
		p.nextToken()

		if p.cur.Type == TOKEN_COMMA {
			p.nextToken()
		} else if p.cur.Type == TOKEN_RPAREN {
			p.nextToken()
			break
		} else {
			return nil, fmt.Errorf("expected , or )")
		}
	}

	return stmt, nil
}

func (p *Parser) parseCreateIndex() (*CreateIndexStmt, error) {
	stmt := &CreateIndexStmt{}

	if p.cur.Type != TOKEN_INDEX {
		return nil, fmt.Errorf("expected INDEX")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_ON {
		return nil, fmt.Errorf("expected ON")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	if p.cur.Type != TOKEN_LPAREN {
		return nil, fmt.Errorf("expected (")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected column name")
	}
	stmt.Column = p.cur.Value
	p.nextToken()

	if p.cur.Type != TOKEN_RPAREN {
		return nil, fmt.Errorf("expected )")
	}
	p.nextToken()

	return stmt, nil
}

func (p *Parser) parseDrop() (Node, error) {
	if p.cur.Type != TOKEN_DROP {
		return nil, fmt.Errorf("expected DROP")
	}
	p.nextToken()

	switch p.cur.Type {
	case TOKEN_TABLE:
		return p.parseDropTable()
	case TOKEN_INDEX:
		return p.parseDropIndex()
	default:
		return nil, fmt.Errorf("expected TABLE or INDEX")
	}
}

func (p *Parser) parseDropTable() (*DropTableStmt, error) {
	stmt := &DropTableStmt{}

	if p.cur.Type != TOKEN_TABLE {
		return nil, fmt.Errorf("expected TABLE")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	return stmt, nil
}

func (p *Parser) parseDropIndex() (*DropIndexStmt, error) {
	stmt := &DropIndexStmt{}

	if p.cur.Type != TOKEN_INDEX {
		return nil, fmt.Errorf("expected INDEX")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected table name")
	}
	stmt.Table = p.cur.Value
	p.nextToken()

	if p.cur.Type != TOKEN_LPAREN {
		return nil, fmt.Errorf("expected (")
	}
	p.nextToken()

	if p.cur.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("expected column name")
	}
	stmt.Column = p.cur.Value
	p.nextToken()

	if p.cur.Type != TOKEN_RPAREN {
		return nil, fmt.Errorf("expected )")
	}
	p.nextToken()

	return stmt, nil
}

func (p *Parser) parseExpr() (Expr, error) {
	return p.parseOrExpr()
}

func (p *Parser) parseOrExpr() (Expr, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}

	if p.cur.Type == TOKEN_OR {
		p.nextToken()
		right, err := p.parseOrExpr()
		if err != nil {
			return nil, err
		}
		return &LogicalExpr{Op: "OR", Left: left, Right: right}, nil
	}

	return left, nil
}

func (p *Parser) parseAndExpr() (Expr, error) {
	left, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	if p.cur.Type == TOKEN_AND {
		p.nextToken()
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		return &LogicalExpr{Op: "AND", Left: left, Right: right}, nil
	}

	return left, nil
}

func (p *Parser) parseNotExpr() (Expr, error) {
	if p.cur.Type == TOKEN_NOT {
		p.nextToken()
		expr, err := p.parseComparisonExpr()
		if err != nil {
			return nil, err
		}
		return &LogicalExpr{Op: "NOT", Left: expr}, nil
	}
	return p.parseComparisonExpr()
}

func (p *Parser) parseComparisonExpr() (Expr, error) {
	left, err := p.parsePrimaryExpr()
	if err != nil {
		return nil, err
	}

	switch p.cur.Type {
	case TOKEN_EQ, TOKEN_NE, TOKEN_LT, TOKEN_GT, TOKEN_LTE, TOKEN_GTE, TOKEN_LIKE:
		op := p.cur.Value
		p.nextToken()
		right, err := p.parsePrimaryExpr()
		if err != nil {
			return nil, err
		}
		return &ComparisonExpr{Op: op, Left: left, Right: right}, nil
	case TOKEN_IS:
		p.nextToken()
		if p.cur.Type == TOKEN_NULL {
			p.nextToken()
			return &ComparisonExpr{Op: "IS NULL", Left: left}, nil
		}
		return nil, fmt.Errorf("expected NULL after IS")
	}

	return left, nil
}

func (p *Parser) parsePrimaryExpr() (Expr, error) {
	switch p.cur.Type {
	case TOKEN_STRING:
		val := p.cur.Value
		p.nextToken()
		return &Literal{Value: val}, nil
	case TOKEN_NUMBER:
		val := p.cur.Value
		p.nextToken()
		return &Literal{Value: val}, nil
	case TOKEN_IDENT:
		col := p.cur.Value
		p.nextToken()
		if p.cur.Type == TOKEN_DOT {
			p.nextToken()
			if p.cur.Type != TOKEN_IDENT {
				return nil, fmt.Errorf("expected identifier after DOT")
			}
			col = col + "." + p.cur.Value
			p.nextToken()
		}
		return &ColumnRef{Column: col}, nil
	case TOKEN_LPAREN:
		p.nextToken()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != TOKEN_RPAREN {
			return nil, fmt.Errorf("expected )")
		}
		p.nextToken()
		return expr, nil
	}

	return nil, fmt.Errorf("unexpected token: %s", p.cur.Value)
}
