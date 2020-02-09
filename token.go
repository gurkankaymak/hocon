package hocon

type TokenType int
const (
	TokenTypeStart TokenType = iota
	TokenTypeEnd
	TokenTypeComma
	TokenTypeEquals
	TokenTypeColon
	TokenTypeOpenCurly
	TokenTypeCloseCurly
	TokenTypeOpenSquare
	TokenTypeCloseSquare
	TokenTypeValue
	TokenTypeNewLine
	TokenTypeUnquotedText
	TokenTypeIgnoredWhitespace
	TokenTypeSubstitution
	TokenTypeProblem
	TokenTypeComment
	TokenTypePlusEquals
)

type Token struct {
	tokenType TokenType
	value string
}
