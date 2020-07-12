package shrunk

import (
	"fmt"
	"os"
	"regexp"
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
	if os.IsNotExist(err) {
		return false
	}
	return true
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

func compileRegExpList(regExpList []string) []*regexp.Regexp {
	regList := make([]*regexp.Regexp, 0)
	for i := 0; i < len(regExpList); i++ {
		cmp, err := regexp.Compile(regExpList[i])
		if err != nil {
			// TODO: write more proper errro handling
			fmt.Println("skiping")
		}
		regList = append(regList, cmp)
	}
	return regList
}
