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

package treflect

import (
	"fmt"
	"reflect"
)

// GetMissingMethods checks if a given object implements all methods of a
// given interface. It returns the interface coverage [0..1] as well as an array
// of error messages. If the interface is correctly implemented the coverage is
// 1 and the error message array is empty.
func GetMissingMethods(objType reflect.Type, ifaceType reflect.Type) (float32, []string) {
	missing := []string{}
	if objType.Implements(ifaceType) {
		return 1.0, missing
	}

	methodCount := ifaceType.NumMethod()
	for mIdx := 0; mIdx < methodCount; mIdx++ {
		ifaceMethod := ifaceType.Method(mIdx)
		objMethod, exists := objType.MethodByName(ifaceMethod.Name)
		signatureMismatch := false

		switch {
		case !exists:
			missing = append(missing, fmt.Sprintf("Missing: \"%s\" %v", ifaceMethod.Name, ifaceMethod.Type))
			continue // ### continue, error found ###

		case ifaceMethod.Type.NumOut() != objMethod.Type.NumOut():
			signatureMismatch = true

		case ifaceMethod.Type.NumIn()+1 != objMethod.Type.NumIn():
			signatureMismatch = true

		default:
			for oIdx := 0; !signatureMismatch && oIdx < ifaceMethod.Type.NumOut(); oIdx++ {
				signatureMismatch = ifaceMethod.Type.Out(oIdx) != objMethod.Type.Out(oIdx)
			}
			for iIdx := 0; !signatureMismatch && iIdx < ifaceMethod.Type.NumIn(); iIdx++ {
				signatureMismatch = ifaceMethod.Type.In(iIdx) != objMethod.Type.In(iIdx+1)
			}
		}

		if signatureMismatch {
			missing = append(missing, fmt.Sprintf("Invalid: \"%s\" %v is not %v", ifaceMethod.Name, objMethod.Type, ifaceMethod.Type))
		}
	}

	return float32(methodCount-len(missing)) / float32(methodCount), missing
}
