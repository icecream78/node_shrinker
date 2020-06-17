package shrunk

func sliceToMap(sl []string) map[string]struct{} {
	m := make(map[string]struct{})
	for i := 0; i < len(sl); i++ {
		m[sl[i]] = struct{}{}
	}
	return m
}
