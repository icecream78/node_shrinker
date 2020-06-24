package shrunk

import (
	"fmt"
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
	_, err := osManager.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func testCall(dsdas OsI) {
	fmt.Println("dsadas")
}
