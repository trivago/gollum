// Copyright 2015 trivago GmbH
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

package contrib

// This is a stub file to enable registration of vendor specific plugins that
// are placed in sub folders of this folder.

import (
	//_ "github.com/trivago/gollum/contrib/native"  // plugins using cgo native bindings
	//_ "your/company/package"
	_ "github.com/trivago/gollum/contrib/trivago"
)

func init() {
}
