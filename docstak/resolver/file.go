/*
Copyright 2024 Kasai Kou

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolver

import (
	"os"
	"path/filepath"
)

func resolveFileWithBasename(dir, name string) (filename string, exist bool) {
	filename = filepath.Join(dir, name)
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		return filename, true
	}

	dir = filepath.Dir(dir)
	_, err = os.Stat(dir)
	if dir[len(dir)-1] != filepath.Separator && !os.IsNotExist(err) {
		return resolveFileWithBasename(dir, name)
	} else {
		return "", false
	}
}

// If startDir is empty string, resolve from current working directory.
func ResolveFileWithBasename(startDir, name string) (filename string, exist bool) {

	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			panic("cannot get working directory: " + err.Error())
		}

		startDir = wd
	}

	if !filepath.IsAbs(startDir) {
		panic("startDir argument must be absolute path")
	}

	return resolveFileWithBasename(startDir, name)
}
