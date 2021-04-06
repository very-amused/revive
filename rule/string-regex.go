package rule

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"

	"github.com/mgechev/revive/lint"
)

// StringRegexRule lints strings and/or comments according to a set of regular expressions given as Arguments
type StringRegexRule struct{}

// Apply applies the rule to the given file.
func (r *StringRegexRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	var failures []lint.Failure

	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}

	w := lintStringRegexRule{onFailure: onFailure}
	w.parseArguments(arguments)
	ast.Walk(w, file.AST)

	return failures
}

func (r *StringRegexRule) Name() string {
	return "string-regex"
}

type lintStringRegexRule struct {
	onFailure func(lint.Failure)

	rules []stringRegexSubrule
}

type stringRegexSubrule struct {
	Regexp       *regexp.Regexp
	ErrorMessage *string
}

func (w lintStringRegexRule) Visit(node ast.Node) ast.Visitor {
	// First, check if node is a string literal
	lit, ok := node.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return w
	}

	// Unquote the string
	unquoted := lit.Value[1 : len(lit.Value)-1]
	w.lintMessage(unquoted, node)

	return w
}

func (w *lintStringRegexRule) parseArguments(arguments lint.Arguments) {
	for i, argument := range arguments {
		regex, errorMessage := w.parseArgument(argument, i)
		w.rules = append(w.rules, stringRegexSubrule{
			Regexp:       regex,
			ErrorMessage: errorMessage,
		})
	}
}

func (w lintStringRegexRule) parseArgument(argument interface{}, argNum int) (regex *regexp.Regexp, errorMessage *string) {
	g, ok := argument.([]interface{}) // Cast to generic slice first
	if !ok {
		panic(fmt.Sprintf("unable to parse argument %d", argNum))
	}
	var rule []string
	for i, obj := range g {
		val, ok := obj.(string)
		if !ok {
			panic(fmt.Sprintf("unable to parse argument %d, option %d", argNum, i))
		}
		rule = append(rule, val)
	}
	// Strip / characters from the beginning and end of rule[0] before compiling
	regex, err := regexp.Compile(rule[0][1 : len(rule[0])-1])
	if err != nil {
		panic(fmt.Sprintf("unable to compile %s as regexp (argument %d, option %d)", rule[0], argNum, 0))
	}

	// Parse custom error message if provided
	errorMessage = nil
	if len(rule) == 2 {
		errorMessage = &rule[1]
	}
	return regex, errorMessage
}

func (w lintStringRegexRule) lintMessage(s string, node ast.Node) {
	for _, rule := range w.rules {
		// Fail if the string doesn't match the user's regex
		if rule.Regexp.Match([]byte(s)) {
			continue
		}
		var failure string
		if rule.ErrorMessage != nil {
			failure = fmt.Sprintf("string literal doesn't match user defined regex (%s)", *rule.ErrorMessage)
		} else {
			failure = fmt.Sprintf("string literal doesn't match user defined regex /%s/", rule.Regexp.String())
		}
		w.onFailure(lint.Failure{
			Confidence: 1,
			Failure:    failure,
			Node:       node})
	}
}
