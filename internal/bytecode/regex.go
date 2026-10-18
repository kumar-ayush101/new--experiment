package bytecode

import "regexp"

// CompiledRegex wraps a compiled regular expression used by OpMatch/OpNMatch.
// Patterns are compiled once at compile time, not per record.
type CompiledRegex struct {
	Pattern string
	RE      *regexp.Regexp
}

// CompileRegex compiles a pattern into a CompiledRegex.
func CompileRegex(pattern string) (*CompiledRegex, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &CompiledRegex{Pattern: pattern, RE: re}, nil
}
