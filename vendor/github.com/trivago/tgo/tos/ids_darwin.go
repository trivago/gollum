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

// +build darwin

package tos

const (
	// NobodyUid represents the user id of the user nobody
	NobodyUid = 0xFFFFFFFE
	// NobodyUid represents the group id of the group nobody
	NobodyGid = 0xFFFFFFFE
	// RootUid represents the user id of the root user
	RootUid = 0
	// RootUid represents the group id of the root group
	RootGid = 0
)
