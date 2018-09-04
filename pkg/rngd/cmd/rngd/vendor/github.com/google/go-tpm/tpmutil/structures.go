// Copyright (c) 2018, Google Inc. All rights reserved.
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

package tpmutil

// RawBytes is for Pack and RunCommand arguments that are already encoded.
// Compared to []byte, RawBytes will not be prepended with slice length during
// encoding.
type RawBytes []byte

// Tag is a command tag.
type Tag uint16

// Command is an identifier of a TPM command.
type Command uint32

// A commandHeader is the header for a TPM command.
type commandHeader struct {
	Tag  Tag
	Size uint32
	Cmd  Command
}

// ResponseCode is a response code returned by TPM.
type ResponseCode uint32

// RCSuccess is response code for successful command. Identical for TPM 1.2 and
// 2.0.
const RCSuccess ResponseCode = 0x000

// A responseHeader is a header for TPM responses.
type responseHeader struct {
	Tag  Tag
	Size uint32
	Res  ResponseCode
}

// A Handle is a reference to a TPM object.
type Handle uint32
