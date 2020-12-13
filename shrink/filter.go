package shrink

import (
	"path/filepath"
	"regexp"

	. "github.com/icecream78/node_shrinker/walker"
)

type Filter struct {
	shrunkDirNames     map[string]struct{}
	shrunkFileNames    map[string]struct{}
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
		shrunkDirNames:     sliceToMap(DefaultRemoveDirNames, regularInclude),
		shrunkFileNames:    sliceToMap(DefaultRemoveFileNames, regularInclude),
		shrunkFileExt:      sliceToMap(DefaultRemoveFileExt, includeExtenstions),
		excludeNames:       sliceToMap(regularExclude),
		regExpIncludeNames: compiledIncludeRegList,
		regExpExcludeNames: compiledExcludeRegList,
	}
}

// Checks is provided file need to removed or not
func (f *Filter) Check(de FileInfoI) (bool, error) {
	if f.isExcludeName(de.Name()) {
		return false, ExcludeError
	}

	if f.isIncludeName(de.Name()) {
		return true, nil
	} else if de.IsDir() && f.isDirToRemove(de.Name()) {
		return true, SkipDirError
	} else if de.IsRegular() && f.isFileToRemove(de.Name()) {
		return true, nil
	}
	return false, NotProcessError
}

func (f *Filter) isExcludeName(name string) bool {
	_, exists := f.excludeNames[name]
	if exists {
		return true
	}

	for _, pattern := range f.regExpExcludeNames {
		matched := pattern.MatchString(name)
		if matched {
			return true
		}
	}

	return false
}

func (f *Filter) isIncludeName(name string) bool {
	for _, pattern := range f.regExpIncludeNames {
		matched := pattern.MatchString(name)
		if matched {
			return true
		}
	}
	return false
}

func (f *Filter) isDirToRemove(name string) bool {
	_, exists := f.shrunkDirNames[name]
	return exists
}

func (f *Filter) isFileToRemove(name string) (exists bool) {
	if _, exists = f.shrunkFileNames[name]; exists {
		return
	}
	ext := filepath.Ext(name)
	if ext == name { // for cases, when files starts with leading dot
		return
	}
	if _, exists = f.shrunkFileExt[ext]; exists {
		return
	}
	return
}
