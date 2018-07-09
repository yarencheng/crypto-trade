package poloniex

import (
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/yarencheng/crypto-trade/go/exchange/poloniex/parser"
)

type TreeShapeListener struct {
	*parser.BaseJSONListener
}

func NewTreeShapeListener() *TreeShapeListener {
	return new(TreeShapeListener)
}

func (this *TreeShapeListener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	fmt.Printf("%v \n", ctx.GetText())
}

func fn() {

	input := antlr.NewInputStream(`[{"a":1},{"b":2}]`)

	// input, _ := antlr.NewFileStream(os.Args[1])
	lexer := parser.NewJSONLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewJSONParser(stream)
	p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.BuildParseTrees = true
	tree := p.Json()
	antlr.ParseTreeWalkerDefault.Walk(NewTreeShapeListener(), tree)
}
