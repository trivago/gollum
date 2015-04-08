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

package stream

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// Broadcast stream plugin
// Configuration example
//
//   - "stream.Broadcast":
//     Enable: true
//     Stream: "data"
//
// This stream does not define any options beside the standard ones.
// Messages are send to a all producers in the set of the producers listening
// to the given stream.
type Broadcast struct {
	core.StreamBase
}

func init() {
	shared.RuntimeType.Register(Broadcast{})
}
