package sql

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_IDENT
	TOKEN_STRING
	TOKEN_NUMBER
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_COMMA
	TOKEN_SEMICOLON
	TOKEN_EQ
	TOKEN_NE
	TOKEN_LT
	TOKEN_GT
	TOKEN_LTE
	TOKEN_GTE
	TOKEN_PLUS
	TOKEN_MINUS
	TOKEN_MULT
	TOKEN_DIV
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT
	TOKEN_LIKE
	TOKEN_IS
	TOKEN_NULL
	TOKEN_SELECT
	TOKEN_FROM
	TOKEN_WHERE
	TOKEN_ORDER
	TOKEN_BY
	TOKEN_ASC
	TOKEN_DESC
	TOKEN_LIMIT
	TOKEN_INSERT
	TOKEN_INTO
	TOKEN_VALUES
	TOKEN_UPDATE
	TOKEN_SET
	TOKEN_DELETE
	TOKEN_CREATE
	TOKEN_TABLE
	TOKEN_INDEX
	TOKEN_DROP
	TOKEN_ON
	TOKEN_DOT
	TOKEN_AS
)

type Token struct {
	Type    TokenType
	Value   string
	Pos     int
}

var keywords = map[string]TokenType{
	"SELECT":  TOKEN_SELECT,
	"FROM":   TOKEN_FROM,
	"WHERE":  TOKEN_WHERE,
	"ORDER":  TOKEN_ORDER,
	"BY":     TOKEN_BY,
	"ASC":    TOKEN_ASC,
	"DESC":   TOKEN_DESC,
	"LIMIT":  TOKEN_LIMIT,
	"AND":    TOKEN_AND,
	"OR":     TOKEN_OR,
	"NOT":    TOKEN_NOT,
	"LIKE":   TOKEN_LIKE,
	"IS":     TOKEN_IS,
	"NULL":   TOKEN_NULL,
	"INSERT": TOKEN_INSERT,
	"INTO":   TOKEN_INTO,
	"VALUES": TOKEN_VALUES,
	"UPDATE": TOKEN_UPDATE,
	"SET":    TOKEN_SET,
	"DELETE": TOKEN_DELETE,
	"CREATE": TOKEN_CREATE,
	"TABLE":  TOKEN_TABLE,
	"INDEX":  TOKEN_INDEX,
	"DROP":   TOKEN_DROP,
	"ON":     TOKEN_ON,
	"AS":     TOKEN_AS,
}

type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input, pos: 0}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TOKEN_EOF, Pos: l.pos}
	}

	c := l.input[l.pos]

	if c == '.' {
		l.pos++
		return Token{Type: TOKEN_DOT, Value: ".", Pos: l.pos}
	}

	if c == '(' {
		l.pos++
		return Token{Type: TOKEN_LPAREN, Value: "(", Pos: l.pos}
	}

	if c == ')' {
		l.pos++
		return Token{Type: TOKEN_RPAREN, Value: ")", Pos: l.pos}
	}

	if c == ',' {
		l.pos++
		return Token{Type: TOKEN_COMMA, Value: ",", Pos: l.pos}
	}

	if c == ';' {
		l.pos++
		return Token{Type: TOKEN_SEMICOLON, Value: ";", Pos: l.pos}
	}

	if c == '=' {
		l.pos++
		if l.peek() == '=' {
			l.pos++
			return Token{Type: TOKEN_EQ, Value: "==", Pos: l.pos}
		}
		return Token{Type: TOKEN_EQ, Value: "=", Pos: l.pos}
	}

	if c == '!' {
		l.pos++
		if l.peek() == '=' {
			l.pos++
			return Token{Type: TOKEN_NE, Value: "!=", Pos: l.pos}
		}
	}

	if c == '<' {
		l.pos++
		if l.peek() == '=' {
			l.pos++
			return Token{Type: TOKEN_LTE, Value: "<=", Pos: l.pos}
		}
		return Token{Type: TOKEN_LT, Value: "<", Pos: l.pos}
	}

	if c == '>' {
		l.pos++
		if l.peek() == '=' {
			l.pos++
			return Token{Type: TOKEN_GTE, Value: ">=", Pos: l.pos}
		}
		return Token{Type: TOKEN_GT, Value: ">", Pos: l.pos}
	}

	if c == '+' {
		l.pos++
		return Token{Type: TOKEN_PLUS, Value: "+", Pos: l.pos}
	}

	if c == '-' {
		l.pos++
		return Token{Type: TOKEN_MINUS, Value: "-", Pos: l.pos}
	}

	if c == '*' {
		l.pos++
		return Token{Type: TOKEN_MULT, Value: "*", Pos: l.pos}
	}

	if c == '/' {
		l.pos++
		return Token{Type: TOKEN_DIV, Value: "/", Pos: l.pos}
	}

	if c == '\'' {
		return l.readString()
	}

	if unicode.IsDigit(rune(c)) {
		return l.readNumber()
	}

	if unicode.IsLetter(rune(c)) || c == '_' {
		return l.readIdent()
	}

	l.pos++
	return Token{Type: TOKEN_EOF, Value: string(c), Pos: l.pos}
}

func (l *Lexer) peek() byte {
	if l.pos < len(l.input) {
		return l.input[l.pos]
	}
	return 0
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

func (l *Lexer) readString() Token {
	l.pos++
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '\'' {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.pos++
		}
		l.pos++
	}
	value := l.input[start:l.pos]
	if l.pos < len(l.input) {
		l.pos++
	}
	return Token{Type: TOKEN_STRING, Value: value, Pos: l.pos}
}

func (l *Lexer) readNumber() Token {
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '.') {
		l.pos++
	}
	return Token{Type: TOKEN_NUMBER, Value: l.input[start:l.pos], Pos: l.pos}
}

func (l *Lexer) readIdent() Token {
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '_' || l.input[l.pos] == '.') {
		l.pos++
	}
	value := l.input[start:l.pos]
	upper := strings.ToUpper(value)
	if kt, ok := keywords[upper]; ok {
		return Token{Type: kt, Value: value, Pos: l.pos}
	}
	return Token{Type: TOKEN_IDENT, Value: value, Pos: l.pos}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	l.pos = 0
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TOKEN_EOF {
			break
		}
	}
	return tokens, nil
}

func TokenTypeName(t TokenType) string {
	switch t {
	case TOKEN_EOF:
		return "EOF"
	case TOKEN_IDENT:
		return "IDENT"
	case TOKEN_STRING:
		return "STRING"
	case TOKEN_NUMBER:
		return "NUMBER"
	case TOKEN_LPAREN:
		return "LPAREN"
	case TOKEN_RPAREN:
		return "RPAREN"
	case TOKEN_COMMA:
		return "COMMA"
	case TOKEN_SEMICOLON:
		return "SEMICOLON"
	case TOKEN_EQ:
		return "EQ"
	case TOKEN_NE:
		return "NE"
	case TOKEN_LT:
		return "LT"
	case TOKEN_GT:
		return "GT"
	case TOKEN_LTE:
		return "LTE"
	case TOKEN_GTE:
		return "GTE"
	case TOKEN_PLUS:
		return "PLUS"
	case TOKEN_MINUS:
		return "MINUS"
	case TOKEN_MULT:
		return "MULT"
	case TOKEN_DIV:
		return "DIV"
	case TOKEN_AND:
		return "AND"
	case TOKEN_OR:
		return "OR"
	case TOKEN_NOT:
		return "NOT"
	case TOKEN_LIKE:
		return "LIKE"
	case TOKEN_IS:
		return "IS"
	case TOKEN_NULL:
		return "NULL"
	case TOKEN_SELECT:
		return "SELECT"
	case TOKEN_FROM:
		return "FROM"
	case TOKEN_WHERE:
		return "WHERE"
	case TOKEN_ORDER:
		return "ORDER"
	case TOKEN_BY:
		return "BY"
	case TOKEN_ASC:
		return "ASC"
	case TOKEN_DESC:
		return "DESC"
	case TOKEN_LIMIT:
		return "LIMIT"
	case TOKEN_INSERT:
		return "INSERT"
	case TOKEN_INTO:
		return "INTO"
	case TOKEN_VALUES:
		return "VALUES"
	case TOKEN_UPDATE:
		return "UPDATE"
	case TOKEN_SET:
		return "SET"
	case TOKEN_DELETE:
		return "DELETE"
	case TOKEN_CREATE:
		return "CREATE"
	case TOKEN_TABLE:
		return "TABLE"
	case TOKEN_INDEX:
		return "INDEX"
	case TOKEN_DROP:
		return "DROP"
	case TOKEN_ON:
		return "ON"
	case TOKEN_DOT:
		return "DOT"
	case TOKEN_AS:
		return "AS"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}
