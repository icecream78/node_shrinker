package shrink

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func sliceToMap(sl ...[]string) map[string]struct{} {
	m := make(map[string]struct{})
	for i := 0; i < len(sl); i++ {
		tmp := sl[i]
		for j := 0; j < len(tmp); j++ {
			m[tmp[j]] = struct{}{}
		}
	}
	return m
}

func pathExists(path string) bool {
	_, err := fsManager.Stat(path, false)
	return !os.IsNotExist(err)
}

func isStringPattern(input string) bool {
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '*', '?', '[', ']', '\\', '_', '-', '^', '$':
			return true
		}
	}
	return false
}

func isComplexExternsion(input string) bool {
	splitted := strings.Split(input, ".")

	return len(splitted) > 2
}

func devidePatternsFromRegularNames(input []string) (patterns []string, regular []string) {
	patterns = make([]string, 0)
	regular = make([]string, 0)
	for _, in := range input {
		isPattern := isStringPattern(in)
		if isPattern {
			patterns = append(patterns, in)
		} else {
			regular = append(regular, in)
		}
	}
	return
}

func compileRegExpList(regExpList []string) ([]*regexp.Regexp, error) {
	regList := make([]*regexp.Regexp, 0)
	for i := 0; i < len(regExpList); i++ {
		cmp, err := regexp.Compile(regExpList[i])
		if err != nil {
			return nil, fmt.Errorf("Error compile regular expression: %s", regExpList[i])
		}
		regList = append(regList, cmp)
	}
	return regList, nil
}

func filterExtensionsList(regExpList []string) ([]*regexp.Regexp, []string) {
	patterns := make([]*regexp.Regexp, 0)
	regular := make([]string, 0)
	for _, in := range regExpList {
		isPattern := isComplexExternsion(in)
		if isPattern {
			fixedRegExp := fmt.Sprintf("*%s", in)
			cmp, err := regexp.Compile(fixedRegExp)

			if err != nil {
				// TODO: say error here
				continue
			}
			patterns = append(patterns, cmp)
		} else {
			regular = append(regular, in)
		}
	}
	return patterns, regular
}
