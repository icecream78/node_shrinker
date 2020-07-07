package shrunk

import (
	"os"
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

func devidePatternsFromRegularNames(input []string) (patterns []string, regular []string) {
	patterns = make([]string, 0)
	regular = make([]string, 0)
	for _, in := range input {
		isPatterns := false
		for _, c := range in {
			switch c {
			case '*':
			case '?':
			case '[':
			case ']':
			case '\\':
			case '_':
			case '-':
			case '^':
			case '$':
				isPatterns = true
				patterns = append(patterns, in)
				break
			}
		}
		if !isPatterns {
			regular = append(regular, in)
		}
	}
	return
}
