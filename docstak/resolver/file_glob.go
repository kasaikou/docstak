package resolver

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
)

type FileGlobConfig struct {
	Rootdir    string   `json:"root"`
	Rules      []string `json:"rules"`
	IgnoreRule []string `json:"ignores"`
}

func ResolveFileGlob(config FileGlobConfig) ([]string, error) {

	fileSystem := os.DirFS(config.Rootdir)
	candidates := []string{}
	for i := range config.Rules {
		matched, err := doublestar.Glob(fileSystem, config.Rules[i], doublestar.WithFilesOnly())
		if err != nil {
			return nil, err
		}

		candidates = append(candidates, matched...)
	}

	slices.Sort(candidates)
	candidates = slices.Compact(candidates)

	results := make([]string, 0, len(candidates))
	for i := range candidates {
		matched := true
		for j := range config.IgnoreRule {
			ignored, err := doublestar.PathMatch(config.IgnoreRule[j], candidates[i])
			if err != nil {
				return nil, err
			} else if ignored {
				matched = false
				break
			}
		}

		if matched {
			results = append(results, candidates[i])
		}
	}

	return results, nil
}

func ResolveFileGlobFullpath(config FileGlobConfig) ([]string, error) {
	results, err := ResolveFileGlob(config)
	if err != nil {
		return results, err
	}
	for i := range results {
		results[i] = filepath.Join(config.Rootdir, results[i])
	}

	return results, nil
}
