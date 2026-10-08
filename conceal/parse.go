package conceal

import (
	"errors"
	"fmt"
	"strings"
)

type ruleBuilder func(val string) (Rule, error)

var ruleBuilders = map[string]ruleBuilder{
	"b":  buildBytesRule,
	"r":  buildRandRule,
	"rd": buildRandDigitsRule,
	"rc": buildRandCharRule,
	"t":  buildTimestampRule,
	"dz": buildDataSizeRule,
	"d":  buildDataRule,
}

func ParseRules(spec string) (Rules, error) {
	var (
		rules Rules
		errs  []error
	)

	for i := 0; i < len(spec); {
		start := strings.IndexByte(spec[i:], '<')
		if start == -1 {
			break
		}
		start += i

		end := strings.IndexByte(spec[start:], '>')
		if end == -1 {
			return nil, errors.New("missing enclosing >")
		}
		end += start

		key, val, _ := strings.Cut(spec[start+1:end], " ")
		i = end + 1

		builder, ok := ruleBuilders[key]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown tag <%s>", key))
			continue
		}

		rule, err := builder(val)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to build <%s>: %w", key, err))
			continue
		}

		rules = append(rules, rule)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return rules, nil
}
