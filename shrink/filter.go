package shrink

import (
	"path/filepath"
	"regexp"

	. "github.com/icecream78/node_shrinker/walker"
)

type Filter struct {
	includeFileNames   map[string]struct{}
	shrunkFileExt      map[string]struct{}
	excludeNames       map[string]struct{}
	regExpIncludeNames []*regexp.Regexp
	regExpExcludeNames []*regexp.Regexp
}

func NewFilter(includeNames, excludeNames, includeExtenstions []string) *Filter {
	patternInclude, regularInclude := devidePatternsFromRegularNames(includeNames)
	patternExclude, regularExclude := devidePatternsFromRegularNames(excludeNames)

	// TODO: figure out how handle incoming errors
	compiledIncludeRegList, _ := compileRegExpList(patternInclude)
	compiledExcludeRegList, _ := compileRegExpList(patternExclude)

	return &Filter{
		includeFileNames: sliceToMap(regularInclude),
		shrunkFileExt:    sliceToMap(includeExtenstions),

		excludeNames:       sliceToMap(regularExclude),
		regExpIncludeNames: compiledIncludeRegList,
		regExpExcludeNames: compiledExcludeRegList,
	}
}

// Checks is provided file need to removed or not
func (f *Filter) Check(de FileInfoI) (bool, error) {
	if f.isIncludeName(de.Name()) {
		return true, nil
	}

	if f.isIncludeRegName(de.Name()) {
		return true, nil
	}

	if de.IsRegular() && f.isIncludeExt(de.Name()) {
		return true, nil
	}

	if f.isExcludeName(de.Name()) {
		return false, ExcludeError
	}

	if f.isExcludeRegName(de.Name()) {
		return false, ExcludeError
	}

	return false, NotProcessError
}

func (f *Filter) isExcludeName(name string) bool {
	_, exists := f.excludeNames[name]
	return exists
}

func (f *Filter) isExcludeRegName(name string) bool {
	for _, pattern := range f.regExpExcludeNames {
		matched := pattern.MatchString(name)
		if matched {
			return true
		}
	}

	return false
}

func (f *Filter) isIncludeRegName(name string) bool {
	for _, pattern := range f.regExpIncludeNames {
		matched := pattern.MatchString(name)
		if matched {
			return true
		}
	}
	return false
}

func (f *Filter) isIncludeName(name string) bool {
	_, exists := f.includeFileNames[name]
	return exists
}

func (f *Filter) isIncludeExt(name string) (exists bool) {
	ext := filepath.Ext(name)
	if ext == name { // for cases, when files starts with leading dot
		return
	}

	if _, exists = f.shrunkFileExt[ext]; exists {
		return
	}
	return
}
