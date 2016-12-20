// Copyright 2015-2016 trivago GmbH
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

package shared

import (
	"os"
	"sort"
	"testing"
	"time"
)

func TestMath(t *testing.T) {
	expect := NewExpect(t)

	expect.Equal(1, MaxI(1, 0))
	expect.Equal(0, MinI(0, 1))

	expect.Equal(1, MaxI(-1, 1))
	expect.Equal(-1, MinI(-1, 1))
}

func TestItoLen(t *testing.T) {
	expect := NewExpect(t)

	expect.Equal(1, ItoLen(0))
	expect.Equal(1, ItoLen(1))
	expect.Equal(2, ItoLen(10))
	expect.Equal(2, ItoLen(33))
	expect.Equal(3, ItoLen(100))
	expect.Equal(3, ItoLen(555))
}

func TestItob(t *testing.T) {
	expect := NewExpect(t)
	buffer := [3]byte{'1', '1', '1'}

	Itob(0, buffer[:])
	expect.Equal("011", string(buffer[:]))

	Itob(123, buffer[:])
	expect.Equal("123", string(buffer[:]))

	Itob(45, buffer[:])
	expect.Equal("45", string(buffer[:2]))
}

func TestItobe(t *testing.T) {
	expect := NewExpect(t)
	buffer := [3]byte{'1', '1', '1'}

	Itobe(0, buffer[:])
	expect.Equal("110", string(buffer[:]))

	Itobe(123, buffer[:])
	expect.Equal("123", string(buffer[:]))

	Itobe(45, buffer[:])
	expect.Equal("45", string(buffer[1:]))
}

func TestBtoi(t *testing.T) {
	expect := NewExpect(t)

	result, length := Btoi([]byte("0"))
	expect.Equal(0, int(result))
	expect.Equal(1, length)

	result, length = Btoi([]byte("test"))
	expect.Equal(0, int(result))
	expect.Equal(0, length)

	result, length = Btoi([]byte("10"))
	expect.Equal(10, int(result))
	expect.Equal(2, length)

	result, length = Btoi([]byte("10x"))
	expect.Equal(10, int(result))
	expect.Equal(2, length)

	result, length = Btoi([]byte("33"))
	expect.Equal(33, int(result))
	expect.Equal(2, length)

	result, length = Btoi([]byte("100"))
	expect.Equal(100, int(result))
	expect.Equal(3, length)

	result, length = Btoi([]byte("333"))
	expect.Equal(333, int(result))
	expect.Equal(3, length)
}

func TestListFilesByDateMatching(t *testing.T) {
	expect := NewExpect(t)

	files, err := ListFilesByDateMatching(".", "\\.go$")
	expect.NoError(err)
	expect.Greater(len(files), 1)

	lastTime := int64(0)
	diffCount := 0

	for _, file := range files {
		if expect.Leq(lastTime, file.ModTime().UnixNano()) {
			if lastTime != file.ModTime().UnixNano() {
				diffCount++
			}
		}
		lastTime = file.ModTime().UnixNano()
	}

	expect.Greater(diffCount, 0)
}

type fileInfoMock struct {
	os.FileInfo
	name string
	mod  time.Time
}

func (info fileInfoMock) Name() string {
	return info.name
}

func (info fileInfoMock) Size() int64 {
	return 0
}

func (info fileInfoMock) Mode() os.FileMode {
	return os.FileMode(0)
}

func (info fileInfoMock) ModTime() time.Time {
	return info.mod
}

func (info fileInfoMock) IsDir() bool {
	return false
}

func (info fileInfoMock) Sys() interface{} {
	return nil
}

func TestSplitPath(t *testing.T) {
	expect := NewExpect(t)

	dir, name, ext := SplitPath("a/b")

	expect.Equal("a", dir)
	expect.Equal("b", name)
	expect.Equal("", ext)

	dir, name, ext = SplitPath("a/b.c")

	expect.Equal("a", dir)
	expect.Equal("b", name)
	expect.Equal(".c", ext)

	dir, name, ext = SplitPath("b")

	expect.Equal(".", dir)
	expect.Equal("b", name)
	expect.Equal("", ext)
}

func TestFilesByDate(t *testing.T) {
	expect := NewExpect(t)

	testData := FilesByDate{
		fileInfoMock{name: "log1", mod: time.Unix(1, 0)},
		fileInfoMock{name: "log3", mod: time.Unix(0, 0)},
		fileInfoMock{name: "log2", mod: time.Unix(0, 0)},
		fileInfoMock{name: "log4", mod: time.Unix(2, 0)},
		fileInfoMock{name: "alog5", mod: time.Unix(0, 0)},
	}

	sort.Sort(testData)

	expect.Equal("alog5", testData[0].Name())
	expect.Equal("log2", testData[1].Name())
	expect.Equal("log3", testData[2].Name())
	expect.Equal("log1", testData[3].Name())
	expect.Equal("log4", testData[4].Name())
}

func TestIndexN(t *testing.T) {
	expect := NewExpect(t)

	testString := "a.b.c.d"
	expect.Equal(-1, IndexN(testString, ".", 4))
	expect.Equal(-1, IndexN(testString, ".", 0))
	expect.Equal(1, IndexN(testString, ".", 1))
	expect.Equal(3, IndexN(testString, ".", 2))
	expect.Equal(5, IndexN(testString, ".", 3))

	expect.Equal(-1, LastIndexN(testString, ".", 4))
	expect.Equal(-1, LastIndexN(testString, ".", 0))
	expect.Equal(5, LastIndexN(testString, ".", 1))
	expect.Equal(3, LastIndexN(testString, ".", 2))
	expect.Equal(1, LastIndexN(testString, ".", 3))
}

func TestIsJSON(t *testing.T) {
	expect := NewExpect(t)
	testObj := "{\"object\": true}"
	result, _ := IsJSON([]byte(testObj))
	expect.True(result)
}
