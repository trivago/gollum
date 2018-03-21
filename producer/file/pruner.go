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
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/tgo/tio"
	"os"
	"time"
)

// Pruner file producer pruning component
//
// Parameters
//
// - Prune/Count: this value removes old logfiles upon rotate so that only the given
// number of logfiles remain. Logfiles are located by the name defined by "File"
// and are pruned by date (followed by name). Set this value to "0" to disable pruning by count.
// By default this parameter is set to "0".
//
// - Prune/AfterHours: This value removes old logfiles that are older than a given number
// of hours. Set this value to "0" to disable pruning by lifetime.
// By default this parameter is set to "0".
//
// - Prune/TotalSizeMB: This value removes old logfiles upon rotate so that only the
// given number of MBs are used by logfiles. Logfiles are located by the name
// defined by "File" and are pruned by date (followed by name).
// Set this value to "0" to disable pruning by file size.
// By default this parameter is set to "0".
//
type Pruner struct {
	pruneCount int   `config:"Prune/Count" default:"0"`
	pruneHours int   `config:"Prune/AfterHours" default:"0"`
	pruneSize  int64 `config:"Prune/TotalSizeMB" default:"0" metric:"mb"`
	rotate     components.RotateConfig
	Logger     logrus.FieldLogger // Logger need to set
}

// Configure initializes this object with values from a plugin config.
func (pruner *Pruner) Configure(conf core.PluginConfigReader) {
	if pruner.pruneSize > 0 && pruner.rotate.SizeByte > 0 {
		pruner.pruneSize -= pruner.rotate.SizeByte >> 20
		if pruner.pruneSize <= 0 {
			pruner.pruneCount = 1
			pruner.pruneSize = 0
		}
	}
}

// Prune starts prune methods by hours, by count and by size
func (pruner *Pruner) Prune(baseFilePath string) {
	if pruner.pruneHours > 0 {
		pruner.pruneByHour(baseFilePath, pruner.pruneHours)
	}
	if pruner.pruneCount > 0 {
		pruner.pruneByCount(baseFilePath, pruner.pruneCount)
	}
	if pruner.pruneSize > 0 {
		pruner.pruneToSize(baseFilePath, pruner.pruneSize)
	}
}

func (pruner *Pruner) pruneByHour(baseFilePath string, hours int) {
	baseDir, baseName, _ := tio.SplitPath(baseFilePath)

	files, err := tio.ListFilesByDateMatching(baseDir, baseName+".*")
	if err != nil {
		pruner.Logger.Error("Error pruning files: ", err)
		return // ### return, error ###
	}

	pruneDate := time.Now().Add(time.Duration(-hours) * time.Hour)

	for i := 0; i < len(files) && files[i].ModTime().Before(pruneDate); i++ {
		filePath := fmt.Sprintf("%s/%s", baseDir, files[i].Name())
		if err := os.Remove(filePath); err != nil {
			pruner.Logger.Errorf("Failed to prune \"%s\": %s", filePath, err.Error())
		} else {
			pruner.Logger.Infof("Pruned \"%s\"", filePath)
		}
	}
}

func (pruner *Pruner) pruneByCount(baseFilePath string, count int) {
	baseDir, baseName, _ := tio.SplitPath(baseFilePath)

	files, err := tio.ListFilesByDateMatching(baseDir, baseName+".*")
	if err != nil {
		pruner.Logger.Error("Error pruning files: ", err)
		return // ### return, error ###
	}

	numFilesToPrune := len(files) - count
	if numFilesToPrune < 1 {
		return // ## return, nothing to prune ###
	}

	for i := 0; i < numFilesToPrune; i++ {
		filePath := fmt.Sprintf("%s/%s", baseDir, files[i].Name())
		if err := os.Remove(filePath); err != nil {
			pruner.Logger.Errorf("Failed to prune \"%s\": %s", filePath, err.Error())
		} else {
			pruner.Logger.Infof("Pruned \"%s\"", filePath)
		}
	}
}

func (pruner *Pruner) pruneToSize(baseFilePath string, maxSize int64) {
	baseDir, baseName, _ := tio.SplitPath(baseFilePath)

	files, err := tio.ListFilesByDateMatching(baseDir, baseName+".*")
	if err != nil {
		pruner.Logger.Error("Error pruning files: ", err)
		return // ### return, error ###
	}

	totalSize := int64(0)
	for _, file := range files {
		totalSize += file.Size()
	}

	for _, file := range files {
		if totalSize <= maxSize {
			return // ### return, done ###
		}
		filePath := fmt.Sprintf("%s/%s", baseDir, file.Name())
		if err := os.Remove(filePath); err != nil {
			pruner.Logger.Errorf("Failed to prune \"%s\": %s", filePath, err.Error())
		} else {
			pruner.Logger.Infof("Pruned \"%s\"", filePath)
			totalSize -= file.Size()
		}
	}
}
