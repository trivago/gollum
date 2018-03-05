// Copyright 2015-2018 trivago N.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"fmt"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/tgo/tstrings"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// TargetFile is a struct for file producer target files
type TargetFile struct {
	dir               string
	name              string
	ext               string
	originalPath      string
	folderPermissions os.FileMode
}

// NewTargetFile returns a new TargetFile instance
func NewTargetFile(fileDir, fileName, fileExt string, permissions os.FileMode) TargetFile {
	return TargetFile{
		fileDir,
		fileName,
		fileExt,
		fmt.Sprintf("%s/%s%s", fileDir, fileName, fileExt),
		permissions,
	}
}

// GetOriginalPath returns the base path from instantiation
func (streamFile *TargetFile) GetOriginalPath() string {
	return streamFile.originalPath
}

// GetFinalPath returns the final file path with possible rotations
func (streamFile *TargetFile) GetFinalPath(rotate components.RotateConfig) string {
	return fmt.Sprintf("%s/%s", streamFile.dir, streamFile.GetFinalName(rotate))
}

// GetFinalName returns the final file name with possible rotations
func (streamFile *TargetFile) GetFinalName(rotate components.RotateConfig) string {
	var logFileName string

	// Generate the log filename based on rotation, existing files, etc.
	if !rotate.Enabled {
		logFileName = fmt.Sprintf("%s%s", streamFile.name, streamFile.ext)
	} else {
		timestamp := time.Now().Format(rotate.Timestamp)
		signature := fmt.Sprintf("%s_%s", streamFile.name, timestamp)
		maxSuffix := uint64(0)

		files, _ := ioutil.ReadDir(streamFile.dir)
		for _, f := range files {
			if strings.HasPrefix(f.Name(), signature) {
				// Special case.
				// If there is no extension, counter stays at 0
				// If there is an extension (and no count), parsing the "." will yield a counter of 0
				// If there is a count, parsing it will work as intended
				counter := uint64(0)
				if len(f.Name()) > len(signature) {
					counter, _ = tstrings.Btoi([]byte(f.Name()[len(signature)+1:]))
				}

				if maxSuffix <= counter {
					maxSuffix = counter + 1
				}
			}
		}

		if maxSuffix == 0 {
			logFileName = fmt.Sprintf("%s%s", signature, streamFile.ext)
		} else {
			formatString := "%s_%d%s"
			if rotate.ZeroPad > 0 {
				formatString = fmt.Sprintf("%%s_%%0%dd%%s", rotate.ZeroPad)
			}
			logFileName = fmt.Sprintf(formatString, signature, int(maxSuffix), streamFile.ext)
		}
	}

	return logFileName
}

// GetDir create file directory if it not exists and returns the dir name
func (streamFile *TargetFile) GetDir() (string, error) {
	// Assure path is existing
	if err := os.MkdirAll(streamFile.dir, streamFile.folderPermissions); err != nil {
		return "", fmt.Errorf("Failed to create %s because of %s", streamFile.dir, err.Error())
	}

	return streamFile.dir, nil
}

// GetSymlinkPath returns a symlink path for the current file
func (streamFile *TargetFile) GetSymlinkPath() string {
	return fmt.Sprintf("%s/%s_current%s", streamFile.dir, streamFile.name, streamFile.ext)
}

func (streamFile *TargetFile) String() string {
	return streamFile.GetOriginalPath()
}
