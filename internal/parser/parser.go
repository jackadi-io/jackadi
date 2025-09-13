package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/shlex"
)

type Arguments struct {
	Positional []any
	Options    map[string]any
}

func NewArguments() Arguments {
	args := Arguments{}
	args.Options = make(map[string]any)
	return args
}

// ParseArgs extract positional args and optional args from a list of arguments.
//
// Key value are following the pattern: key=value or key="value".
func ParseArgs(args []string) (Arguments, error) {
	a := NewArguments()
	re := regexp.MustCompile("^(?P<key>[a-zA-Z09_]+)=(?P<value>.+)$")

	for _, arg := range args {
		groups := re.FindAllStringSubmatch(arg, -1)
		if groups == nil {
			if len(a.Options) > 0 {
				return Arguments{}, errors.New("positional arguments cannot be after key values")
			}
			a.Positional = append(a.Positional, arg)
			continue
		}

		value, err := shlex.Split(strconv.Quote(groups[0][2]))
		if err != nil {
			return Arguments{}, fmt.Errorf("failed to process arguments: %w", err)
		}
		a.Options[groups[0][1]] = strings.Join(value, " ")
	}

	return a, nil
}
