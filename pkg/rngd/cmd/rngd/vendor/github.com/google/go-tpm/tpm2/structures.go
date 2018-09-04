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
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/go-tpm/tpmutil"
)

// NVPublic contains the public area of an NV index.
type NVPublic struct {
	NVIndex    tpmutil.Handle
	NameAlg    Algorithm
	Attributes KeyProp
	AuthPolicy []byte
	DataSize   uint16
}

type tpmsSensitiveCreate struct {
	UserAuth []byte
	Data     []byte
}

// PCRSelection contains a slice of PCR indexes and a hash algorithm used in
// them.
type PCRSelection struct {
	Hash Algorithm
	PCRs []int
}

type tpmsPCRSelection struct {
	Hash Algorithm
	Size byte
	PCRs tpmutil.RawBytes
}

// Public contains the public area of an object.
type Public struct {
	Type       Algorithm
	NameAlg    Algorithm
	Attributes KeyProp
	AuthPolicy []byte

	// If Type is AlgKeyedHash, then do not set these.
	// Otherwise, only one of the Parameters fields should be set. When encoding/decoding,
	// one will be picked based on Type.
	RSAParameters *RSAParams
	ECCParameters *ECCParams
}

// Encode serializes a Public structure in TPM wire format.
func (p Public) Encode() ([]byte, error) {
	head, err := tpmutil.Pack(p.Type, p.NameAlg, p.Attributes, p.AuthPolicy)
	if err != nil {
		return nil, fmt.Errorf("encoding Type, NameAlg, Attributes, AuthPolicy: %v", err)
	}
	var params []byte
	switch p.Type {
	case AlgRSA:
		params, err = p.RSAParameters.encode()
	case AlgKeyedHash:
		// We only support "keyedHash" objects for the purposes of
		// creating "Sealed Data Blobs".
		var unique uint16
		params, err = tpmutil.Pack(AlgNull, unique)
	case AlgECC:
		params, err = p.ECCParameters.encode()
	default:
		err = fmt.Errorf("unsupported type in TPMT_PUBLIC: %v", p.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("encoding RSAParameters, ECCParameters or KeyedHash: %v", err)
	}
	return concat(head, params)
}

func decodePublic(in *bytes.Buffer) (Public, error) {
	var pub Public
	var err error
	if err = tpmutil.UnpackBuf(in, &pub.Type, &pub.NameAlg, &pub.Attributes, &pub.AuthPolicy); err != nil {
		return pub, fmt.Errorf("decoding TPMT_PUBLIC: %v", err)
	}

	switch pub.Type {
	case AlgRSA:
		pub.RSAParameters, err = decodeRSAParams(in)
	case AlgECC:
		pub.ECCParameters, err = decodeECCParams(in)
	default:
		err = fmt.Errorf("unsupported type in TPMT_PUBLIC: %v", pub.Type)
	}
	return pub, err
}

// RSAParams represents parameters of an RSA key pair.
//
// Symmetric and Sign may be nil, depending on key Attributes in Public.
//
// One of Modulus and ModulusRaw must always be non-nil. Modulus takes
// precedence. ModulusRaw is used for key templates where the field named
// "unique" must be a byte array of all zeroes.
type RSAParams struct {
	Symmetric  *SymScheme
	Sign       *SigScheme
	KeyBits    uint16
	Exponent   uint32
	ModulusRaw []byte
	Modulus    *big.Int
}

func (p *RSAParams) encode() ([]byte, error) {
	if p == nil {
		return nil, nil
	}
	sym, err := p.Symmetric.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding Symmetric: %v", err)
	}
	sig, err := p.Sign.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding Sign: %v", err)
	}
	rest, err := tpmutil.Pack(p.KeyBits, p.Exponent)
	if err != nil {
		return nil, fmt.Errorf("encoding KeyBits, Exponent: %v", err)
	}

	if p.Modulus == nil && len(p.ModulusRaw) == 0 {
		return nil, errors.New("RSAParams.Modulus or RSAParams.ModulusRaw must be set")
	}
	if p.Modulus != nil && len(p.ModulusRaw) > 0 {
		return nil, errors.New("both RSAParams.Modulus and RSAParams.ModulusRaw can't be set")
	}
	mod := p.ModulusRaw
	if p.Modulus != nil {
		mod = p.Modulus.Bytes()
	}
	unique, err := tpmutil.Pack(mod)
	if err != nil {
		return nil, fmt.Errorf("encoding Modulus: %v", err)
	}

	return concat(sym, sig, rest, unique)
}

func decodeRSAParams(in *bytes.Buffer) (*RSAParams, error) {
	var params RSAParams
	var err error

	if params.Symmetric, err = decodeSymScheme(in); err != nil {
		return nil, fmt.Errorf("decoding Symmetric: %v", err)
	}
	if params.Sign, err = decodeSigScheme(in); err != nil {
		return nil, fmt.Errorf("decoding Sign: %v", err)
	}
	var modBytes []byte
	if err := tpmutil.UnpackBuf(in, &params.KeyBits, &params.Exponent, &modBytes); err != nil {
		return nil, fmt.Errorf("decoding KeyBits, Exponent, Modulus: %v", err)
	}
	if params.Exponent == 0 {
		params.Exponent = defaultRSAExponent
	}
	params.Modulus = new(big.Int).SetBytes(modBytes)
	return &params, nil
}

// ECCParams represents parameters of an ECC key pair.
//
// Symmetric, Sign and KDF may be nil, depending on key Attributes in Public.
type ECCParams struct {
	Symmetric *SymScheme
	Sign      *SigScheme
	CurveID   EllipticCurve
	KDF       *KDFScheme
	Point     ECPoint
}

// ECPoint represents a ECC coordinates for a point.
type ECPoint struct {
	X, Y *big.Int
}

func (p *ECCParams) encode() ([]byte, error) {
	if p == nil {
		return nil, nil
	}
	sym, err := p.Symmetric.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding Symmetric: %v", err)
	}
	sig, err := p.Sign.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding Sign: %v", err)
	}
	curve, err := tpmutil.Pack(p.CurveID)
	if err != nil {
		return nil, fmt.Errorf("encoding CurveID: %v", err)
	}
	kdf, err := p.KDF.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding KDF: %v", err)
	}
	point, err := tpmutil.Pack(p.Point.X.Bytes(), p.Point.Y.Bytes())
	if err != nil {
		return nil, fmt.Errorf("encoding Point: %v", err)
	}
	return concat(sym, sig, curve, kdf, point)
}

func decodeECCParams(in *bytes.Buffer) (*ECCParams, error) {
	var params ECCParams
	var err error

	if params.Symmetric, err = decodeSymScheme(in); err != nil {
		return nil, fmt.Errorf("decoding Symmetric: %v", err)
	}
	if params.Sign, err = decodeSigScheme(in); err != nil {
		return nil, fmt.Errorf("decoding Sign: %v", err)
	}
	if err := tpmutil.UnpackBuf(in, &params.CurveID); err != nil {
		return nil, fmt.Errorf("decoding CurveID: %v", err)
	}
	if params.KDF, err = decodeKDFScheme(in); err != nil {
		return nil, fmt.Errorf("decoding KDF: %v", err)
	}
	var x, y []byte
	if err := tpmutil.UnpackBuf(in, &x, &y); err != nil {
		return nil, fmt.Errorf("decoding Point: %v", err)
	}
	params.Point.X = new(big.Int).SetBytes(x)
	params.Point.Y = new(big.Int).SetBytes(y)
	return &params, nil
}

// SymScheme represents a symmetric encryption scheme.
type SymScheme struct {
	Alg     Algorithm
	KeyBits uint16
	Mode    Algorithm
}

func (s *SymScheme) encode() ([]byte, error) {
	if s == nil || s.Alg.IsNull() {
		return tpmutil.Pack(AlgNull)
	}
	return tpmutil.Pack(s.Alg, s.KeyBits, s.Mode)
}

func decodeSymScheme(in *bytes.Buffer) (*SymScheme, error) {
	var scheme SymScheme
	if err := tpmutil.UnpackBuf(in, &scheme.Alg); err != nil {
		return nil, fmt.Errorf("decoding Alg: %v", err)
	}
	if scheme.Alg == AlgNull {
		return nil, nil
	}
	if err := tpmutil.UnpackBuf(in, &scheme.KeyBits, &scheme.Mode); err != nil {
		return nil, fmt.Errorf("decoding KeyBits, Mode: %v", err)
	}
	return &scheme, nil
}

// SigScheme represents a signing scheme.
type SigScheme struct {
	Alg   Algorithm
	Hash  Algorithm
	Count uint32
}

func (s *SigScheme) encode() ([]byte, error) {
	if s == nil || s.Alg.IsNull() {
		return tpmutil.Pack(AlgNull)
	}
	if s.Alg.UsesCount() {
		return tpmutil.Pack(s.Alg, s.Hash, s.Count)
	}
	return tpmutil.Pack(s.Alg, s.Hash)
}

func decodeSigScheme(in *bytes.Buffer) (*SigScheme, error) {
	var scheme SigScheme
	if err := tpmutil.UnpackBuf(in, &scheme.Alg); err != nil {
		return nil, fmt.Errorf("decoding Alg: %v", err)
	}
	if scheme.Alg == AlgNull {
		return nil, nil
	}
	if err := tpmutil.UnpackBuf(in, &scheme.Hash); err != nil {
		return nil, fmt.Errorf("decoding Hash: %v", err)
	}
	if scheme.Alg.UsesCount() {
		if err := tpmutil.UnpackBuf(in, &scheme.Count); err != nil {
			return nil, fmt.Errorf("decoding Count: %v", err)
		}
	}
	return &scheme, nil
}

// KDFScheme represents a KDF (Key Derivation Function) scheme.
type KDFScheme struct {
	Alg  Algorithm
	Hash Algorithm
}

func (s *KDFScheme) encode() ([]byte, error) {
	if s == nil || s.Alg.IsNull() {
		return tpmutil.Pack(AlgNull)
	}
	return tpmutil.Pack(s.Alg, s.Hash)
}

func decodeKDFScheme(in *bytes.Buffer) (*KDFScheme, error) {
	var scheme KDFScheme
	if err := tpmutil.UnpackBuf(in, &scheme.Alg); err != nil {
		return nil, fmt.Errorf("decoding Alg: %v", err)
	}
	if scheme.Alg == AlgNull {
		return nil, nil
	}
	if err := tpmutil.UnpackBuf(in, &scheme.Hash); err != nil {
		return nil, fmt.Errorf("decoding Hash: %v", err)
	}
	return &scheme, nil
}

// Signature combines all possible signatures from RSA and ECC keys. Only one
// of RSA or ECC will be populated.
type Signature struct {
	Alg Algorithm
	RSA *SignatureRSA
	ECC *SignatureECC
}

func decodeSignature(in *bytes.Buffer) (*Signature, error) {
	var sig Signature
	if err := tpmutil.UnpackBuf(in, &sig.Alg); err != nil {
		return nil, fmt.Errorf("decoding Alg: %v", err)
	}
	switch sig.Alg {
	case AlgRSASSA:
		sig.RSA = new(SignatureRSA)
		if err := tpmutil.UnpackBuf(in, sig.RSA); err != nil {
			return nil, fmt.Errorf("decoding RSA: %v", err)
		}
	case AlgECDSA:
		sig.ECC = new(SignatureECC)
		var r, s []byte
		if err := tpmutil.UnpackBuf(in, &sig.ECC.HashAlg, &r, &s); err != nil {
			return nil, fmt.Errorf("decoding ECC: %v", err)
		}
		sig.ECC.R = big.NewInt(0).SetBytes(r)
		sig.ECC.S = big.NewInt(0).SetBytes(s)
	default:
		return nil, fmt.Errorf("unsupported signature algorithm 0x%x", sig.Alg)
	}
	return &sig, nil
}

// SignatureRSA is an RSA-specific signature value.
type SignatureRSA struct {
	HashAlg   Algorithm
	Signature []byte
}

// SignatureECC is an ECC-specific signature value.
type SignatureECC struct {
	HashAlg Algorithm
	R       *big.Int
	S       *big.Int
}

// Private contains private section of a TPM key.
type Private struct {
	Type      Algorithm
	AuthValue []byte
	SeedValue []byte
	Sensitive []byte
}

// Encode serializes a Private structure in TPM wire format.
func (p Private) Encode() ([]byte, error) {
	if p.Type.IsNull() {
		return nil, nil
	}
	return tpmutil.Pack(p)
}

type tpmtSigScheme struct {
	Scheme Algorithm
	Hash   Algorithm
}

// AttestationData contains data attested by TPM commands (like Certify).
type AttestationData struct {
	Magic               uint32
	Type                tpmutil.Tag
	QualifiedSigner     Name
	ExtraData           []byte
	ClockInfo           ClockInfo
	FirmwareVersion     uint64
	AttestedCertifyInfo *CertifyInfo
}

// DecodeAttestationData decode a TPMS_ATTEST message. No error is returned if
// the input has extra trailing data.
func DecodeAttestationData(in []byte) (*AttestationData, error) {
	buf := bytes.NewBuffer(in)

	var ad AttestationData
	if err := tpmutil.UnpackBuf(buf, &ad.Magic, &ad.Type); err != nil {
		return nil, fmt.Errorf("decoding Magic/Type: %v", err)
	}
	n, err := decodeName(buf)
	if err != nil {
		return nil, fmt.Errorf("decoding QualifiedSigner: %v", err)
	}
	ad.QualifiedSigner = *n
	if err := tpmutil.UnpackBuf(buf, &ad.ExtraData, &ad.ClockInfo, &ad.FirmwareVersion); err != nil {
		return nil, fmt.Errorf("decoding ExtraData/ClockInfo/FirmwareVersion: %v", err)
	}

	// The spec specifies several other types of attestation data. We only need
	// parsing of Certify attestation data for now. If you need support for
	// other attestation types, add them here.
	if ad.Type != TagAttestCertify {
		return nil, fmt.Errorf("only Certify attestation structure is supported, got type 0x%x", ad.Type)
	}
	if ad.AttestedCertifyInfo, err = decodeCertifyInfo(buf); err != nil {
		return nil, fmt.Errorf("decoding AttestedCertifyInfo: %v", err)
	}
	return &ad, nil
}

// Encode serializes an AttestationData structure in TPM wire format.
func (ad AttestationData) Encode() ([]byte, error) {
	if ad.Type != TagAttestCertify {
		return nil, fmt.Errorf("only Certify attestation structure is supported, got type 0x%x", ad.Type)
	}
	head, err := tpmutil.Pack(ad.Magic, ad.Type)
	if err != nil {
		return nil, fmt.Errorf("encoding Magic, Type: %v", err)
	}
	signer, err := ad.QualifiedSigner.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding QualifiedSigner: %v", err)
	}
	tail, err := tpmutil.Pack(ad.ExtraData, ad.ClockInfo, ad.FirmwareVersion)
	if err != nil {
		return nil, fmt.Errorf("encoding ExtraData, ClockInfo, FirmwareVersion: %v", err)
	}
	info, err := ad.AttestedCertifyInfo.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding AttestedCertifyInfo: %v", err)
	}
	return concat(head, signer, tail, info)
}

// CertifyInfo contains Certify-specific data for TPMS_ATTEST.
type CertifyInfo struct {
	Name          Name
	QualifiedName Name
}

func decodeCertifyInfo(in *bytes.Buffer) (*CertifyInfo, error) {
	var ci CertifyInfo

	n, err := decodeName(in)
	if err != nil {
		return nil, fmt.Errorf("decoding Name: %v", err)
	}
	ci.Name = *n

	n, err = decodeName(in)
	if err != nil {
		return nil, fmt.Errorf("decoding QualifiedName: %v", err)
	}
	ci.QualifiedName = *n

	return &ci, nil
}

func (ci CertifyInfo) encode() ([]byte, error) {
	n, err := ci.Name.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding Name: %v", err)
	}
	qn, err := ci.QualifiedName.encode()
	if err != nil {
		return nil, fmt.Errorf("encoding QualifiedName: %v", err)
	}
	return concat(n, qn)
}

// Name contains a name for TPM entities. Only one of Handle/Digest should be
// set.
type Name struct {
	Handle *tpmutil.Handle
	Digest *HashValue
}

func decodeName(in *bytes.Buffer) (*Name, error) {
	var nameBuf []byte
	if err := tpmutil.UnpackBuf(in, &nameBuf); err != nil {
		return nil, err
	}

	name := new(Name)
	switch len(nameBuf) {
	case 0:
		// No name is present.
	case 4:
		name.Handle = new(tpmutil.Handle)
		if err := tpmutil.UnpackBuf(bytes.NewBuffer(nameBuf), name.Handle); err != nil {
			return nil, fmt.Errorf("decoding Handle: %v", err)
		}
	default:
		var err error
		name.Digest, err = decodeHashValue(bytes.NewBuffer(nameBuf))
		if err != nil {
			return nil, fmt.Errorf("decoding Digest: %v", err)
		}
	}
	return name, nil
}

func (n Name) encode() ([]byte, error) {
	var buf []byte
	var err error
	switch {
	case n.Handle != nil:
		if buf, err = tpmutil.Pack(*n.Handle); err != nil {
			return nil, fmt.Errorf("encoding Handle: %v", err)
		}
	case n.Digest != nil:
		if buf, err = n.Digest.encode(); err != nil {
			return nil, fmt.Errorf("encoding Digest: %v", err)
		}
	default:
		// Name is empty, which is valid.
	}
	return tpmutil.Pack(buf)
}

// MatchesPublic compares Digest in Name against given Public structure. Note:
// this only works for regular Names, not Qualified Names.
func (n Name) MatchesPublic(p Public) (bool, error) {
	buf, err := p.Encode()
	if err != nil {
		return false, err
	}
	if n.Digest == nil {
		return false, errors.New("Name doesn't have a Digest, can't compare to Public")
	}
	hfn, ok := hashConstructors[n.Digest.Alg]
	if !ok {
		return false, fmt.Errorf("Name hash algorithm 0x%x not supported", n.Digest.Alg)
	}

	h := hfn()
	h.Write(buf)
	digest := h.Sum(nil)

	return bytes.Equal(digest, n.Digest.Value), nil
}

// HashValue is an algorithm-specific hash value.
type HashValue struct {
	Alg   Algorithm
	Value []byte
}

func decodeHashValue(in *bytes.Buffer) (*HashValue, error) {
	var hv HashValue
	if err := tpmutil.UnpackBuf(in, &hv.Alg); err != nil {
		return nil, fmt.Errorf("decoding Alg: %v", err)
	}
	hfn, ok := hashConstructors[hv.Alg]
	if !ok {
		return nil, fmt.Errorf("unsupported hash algorithm type 0x%x", hv.Alg)
	}
	hv.Value = make([]byte, hfn().Size())
	if _, err := in.Read(hv.Value); err != nil {
		return nil, fmt.Errorf("decoding Value: %v", err)
	}
	return &hv, nil
}

func (hv HashValue) encode() ([]byte, error) {
	return tpmutil.Pack(hv.Alg, tpmutil.RawBytes(hv.Value))
}

// ClockInfo contains TPM state info included in AttestationData.
type ClockInfo struct {
	Clock        uint64
	ResetCount   uint32
	RestartCount uint32
	Safe         byte
}
