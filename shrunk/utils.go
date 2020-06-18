package shrunk

import "os"

func sliceToMap(sl []string) map[string]struct{} {
	m := make(map[string]struct{})
	for i := 0; i < len(sl); i++ {
		m[sl[i]] = struct{}{}
	}
	return m
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
