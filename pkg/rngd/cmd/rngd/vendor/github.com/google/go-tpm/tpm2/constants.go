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

package tpm2

import (
	"crypto/elliptic"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/google/go-tpm/tpmutil"
)

func init() {
	tpmutil.UseTPM20LengthPrefixSize()
}

// Algorithm represents a TPM_ALG_ID value.
type Algorithm uint16

// IsNull returns true if a is AlgNull or zero (unset).
func (a Algorithm) IsNull() bool {
	return a == AlgNull || a == AlgUnknown
}

// UsesCount returns true if a signature algorithm uses count value.
func (a Algorithm) UsesCount() bool {
	return a == AlgECDAA
}

// Supported Algorithms.
const (
	AlgUnknown   Algorithm = 0x0000
	AlgRSA       Algorithm = 0x0001
	AlgSHA1      Algorithm = 0x0004
	AlgAES       Algorithm = 0x0006
	AlgKeyedHash Algorithm = 0x0008
	AlgSHA256    Algorithm = 0x000B
	AlgSHA384    Algorithm = 0x000C
	AlgSHA512    Algorithm = 0x000D
	AlgNull      Algorithm = 0x0010
	AlgRSASSA    Algorithm = 0x0014
	AlgRSAES     Algorithm = 0x0015
	AlgRSAPSS    Algorithm = 0x0016
	AlgOAEP      Algorithm = 0x0017
	AlgECDSA     Algorithm = 0x0018
	AlgECDH      Algorithm = 0x0019
	AlgECDAA     Algorithm = 0x001A
	AlgKDF2      Algorithm = 0x0021
	AlgECC       Algorithm = 0x0023
	AlgCTR       Algorithm = 0x0040
	AlgOFB       Algorithm = 0x0041
	AlgCBC       Algorithm = 0x0042
	AlgCFB       Algorithm = 0x0043
	AlgECB       Algorithm = 0x0044
)

// SessionType defines the type of session created in StartAuthSession.
type SessionType uint8

// Supported session types.
const (
	SessionHMAC   SessionType = 0x00
	SessionPolicy SessionType = 0x01
	SessionTrial  SessionType = 0x03
)

// SessionAttributes represents an attribute of a session.
type SessionAttributes byte

const (
	AttrContinueSession SessionAttributes = 1 << iota
	AttrAuditExclusive
	AttrAuditReset
	_ // bit 3 reserved
	_ // bit 4 reserved
	AttrDecrypt
	AttrEcrypt
	AttrAudit
)

// KeyProp is a bitmask used in Attributes field of key templates. Individual
// flags should be OR-ed to form a full mask.
type KeyProp uint32

// Key properties.
const (
	FlagFixedTPM            KeyProp = 0x00000002
	FlagFixedParent         KeyProp = 0x00000010
	FlagSensitiveDataOrigin KeyProp = 0x00000020
	FlagUserWithAuth        KeyProp = 0x00000040
	FlagAdminWithPolicy     KeyProp = 0x00000080
	FlagNoDA                KeyProp = 0x00000400
	FlagRestricted          KeyProp = 0x00010000
	FlagDecrypt             KeyProp = 0x00020000
	FlagSign                KeyProp = 0x00040000

	FlagSealDefault   = FlagFixedTPM | FlagFixedParent
	FlagSignerDefault = FlagSign | FlagRestricted | FlagFixedTPM |
		FlagFixedParent | FlagSensitiveDataOrigin | FlagUserWithAuth
	FlagStorageDefault = FlagDecrypt | FlagRestricted | FlagFixedTPM |
		FlagFixedParent | FlagSensitiveDataOrigin | FlagUserWithAuth
)

// Reserved Handles.
const (
	HandleOwner tpmutil.Handle = 0x40000001 + iota
	HandleRevoke
	HandleTransport
	HandleOperator
	HandleAdmin
	HandleEK
	HandleNull
	HandleUnassigned
	HandlePasswordSession
	HandleLockout
	HandleEndorsement
	HandlePlatform
)

// Capability identifies some TPM property or state type.
type Capability uint32

// TPM Capabilies.
const (
	CapabilityAlgs Capability = iota
	CapabilityHandles
	CapabilityCommands
	CapabilityPPCommands
	CapabilityAuditCommands
	CapabilityPCRs
	CapabilityTPMProperties
	CapabilityPCRProperties
	CapabilityECCCurves
	CapabilityAuthPolicies
)

// TPM Structure Tags. Tags are used to disambiguate structures, similar to Alg
// values: tag value defines what kind of data lives in a nested field.
const (
	TagNull          tpmutil.Tag = 0x8000
	TagNoSessions    tpmutil.Tag = 0x8001
	TagSessions      tpmutil.Tag = 0x8002
	TagAttestCertify tpmutil.Tag = 0x8017
	TagHashCheck     tpmutil.Tag = 0x8024
)

// StartupType instructs the TPM on how to handle its state during Shutdown or
// Startup.
type StartupType uint16

// Startup types
const (
	StartupClear StartupType = iota
	StartupState
)

// EllipticCurve identifies specific EC curves.
type EllipticCurve uint16

// ECC curves supported by TPM 2.0 spec.
const (
	CurveNISTP192 = EllipticCurve(iota + 1)
	CurveNISTP224
	CurveNISTP256
	CurveNISTP384
	CurveNISTP521

	CurveBNP256 = EllipticCurve(iota + 10)
	CurveBNP638

	CurveSM2P256 = EllipticCurve(0x0020)
)

var toGoCurve = map[EllipticCurve]elliptic.Curve{
	CurveNISTP224: elliptic.P224(),
	CurveNISTP256: elliptic.P256(),
	CurveNISTP384: elliptic.P384(),
	CurveNISTP521: elliptic.P521(),
}

// Supported TPM operations.
const (
	cmdEvictControl       tpmutil.Command = 0x00000120
	cmdUndefineSpace      tpmutil.Command = 0x00000122
	cmdClockSet           tpmutil.Command = 0x00000128
	cmdDefineSpace        tpmutil.Command = 0x0000012A
	cmdPCRAllocate        tpmutil.Command = 0x0000012B
	cmdCreatePrimary      tpmutil.Command = 0x00000131
	cmdIncrementNVCounter tpmutil.Command = 0x00000134
	cmdWriteNV            tpmutil.Command = 0x00000137
	cmdPCREvent           tpmutil.Command = 0x0000013C
	cmdStartup            tpmutil.Command = 0x00000144
	cmdShutdown           tpmutil.Command = 0x00000145
	cmdStirRandom         tpmutil.Command = 0x00000146
	cmdActivateCredential tpmutil.Command = 0x00000147
	cmdCertify            tpmutil.Command = 0x00000148
	cmdReadNV             tpmutil.Command = 0x0000014E
	cmdCreate             tpmutil.Command = 0x00000153
	cmdLoad               tpmutil.Command = 0x00000157
	cmdQuote              tpmutil.Command = 0x00000158
	cmdSign               tpmutil.Command = 0x0000015D
	cmdUnseal             tpmutil.Command = 0x0000015E
	cmdContextLoad        tpmutil.Command = 0x00000161
	cmdContextSave        tpmutil.Command = 0x00000162
	cmdFlushContext       tpmutil.Command = 0x00000165
	cmdLoadExternal       tpmutil.Command = 0x00000167
	cmdMakeCredential     tpmutil.Command = 0x00000168
	cmdReadPublicNV       tpmutil.Command = 0x00000169
	cmdReadPublic         tpmutil.Command = 0x00000173
	cmdStartAuthSession   tpmutil.Command = 0x00000176
	cmdGetCapability      tpmutil.Command = 0x0000017A
	cmdGetRandom          tpmutil.Command = 0x0000017B
	cmdHash               tpmutil.Command = 0x0000017D
	cmdPCRRead            tpmutil.Command = 0x0000017E
	cmdPolicyPCR          tpmutil.Command = 0x0000017F
	cmdReadClock          tpmutil.Command = 0x00000181
	cmdPCRExtend          tpmutil.Command = 0x00000182
	cmdPolicyGetDigest    tpmutil.Command = 0x00000189
	cmdPolicyPassword     tpmutil.Command = 0x0000018C
)

// Regular TPM 2.0 devices use 24-bit mask (3 bytes) for PCR selection.
const sizeOfPCRSelect = 3

// digestSize returns the size of a digest for hashing algorithm alg, or 0 if
// it's not recognized.
func digestSize(alg Algorithm) int {
	switch alg {
	case AlgSHA1:
		return sha1.Size
	case AlgSHA256:
		return sha256.Size
	case AlgSHA512:
		return sha512.Size
	default:
		return 0
	}
}

const defaultRSAExponent = 1<<16 + 1

var hashConstructors = map[Algorithm]func() hash.Hash{
	AlgSHA1:   sha1.New,
	AlgSHA256: sha256.New,
	AlgSHA384: sha512.New384,
	AlgSHA512: sha512.New,
}
