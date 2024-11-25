package types

import (
	"bytes"
	"crypto"
	"errors"
	"fmt"

	abhash "github.com/alphabill-org/alphabill-go-base/hash"
	"github.com/alphabill-org/alphabill-go-base/tree/mt"
	"github.com/alphabill-org/alphabill-go-base/types/hex"
	"github.com/alphabill-org/alphabill-go-base/util"
)

type (
	UnitStateProof struct {
		_                  struct{}       `cbor:",toarray"`
		Version            ABVersion      `json:"version"`
		UnitID             UnitID         `json:"unitId"`
		UnitValue          uint64         `json:"unitValue,string"`
		UnitLedgerHash     hex.Bytes      `json:"unitLedgerHash"`
		UnitTreeCert       *UnitTreeCert  `json:"unitTreeCert"`
		StateTreeCert      *StateTreeCert `json:"stateTreeCert"`
		UnicityCertificate TaggedCBOR     `json:"unicityCert"`
	}

	UnitTreeCert struct {
		_                     struct{}       `cbor:",toarray"`
		TransactionRecordHash hex.Bytes      `json:"txrHash"`  // t
		UnitDataHash          hex.Bytes      `json:"dataHash"` // s
		Path                  []*mt.PathItem `json:"path"`
	}

	StateTreeCert struct {
		_                 struct{}             `cbor:",toarray"`
		LeftSummaryHash   hex.Bytes            `json:"leftSummaryHash"`
		LeftSummaryValue  uint64               `json:"leftSummaryValue,string"`
		RightSummaryHash  hex.Bytes            `json:"rightSummaryHash"`
		RightSummaryValue uint64               `json:"rightSummaryValue,string"`
		Path              []*StateTreePathItem `json:"path"`
	}

	StateTreePathItem struct {
		_                   struct{}  `cbor:",toarray"`
		UnitID              UnitID    `json:"unitId"`       // (ι′)
		LogsHash            hex.Bytes `json:"logsHash"`     // (z)
		Value               uint64    `json:"value,string"` // (V)
		SiblingSummaryHash  hex.Bytes `json:"siblingSummaryHash"`
		SiblingSummaryValue uint64    `json:"siblingSummaryValue,string"`
	}

	StateUnitData struct {
		Data RawCBOR
	}

	UnitDataAndProof struct {
		_        struct{} `cbor:",toarray"`
		UnitData *StateUnitData
		Proof    *UnitStateProof
	}

	UnicityCertificateValidator interface {
		Validate(uc *UnicityCertificate) error
	}
)

func (u *UnitStateProof) getUCv1() (*UnicityCertificate, error) {
	if u == nil {
		return nil, errors.New("unit state proof is nil")
	}
	if u.UnicityCertificate == nil {
		return nil, ErrUnicityCertificateIsNil
	}
	uc := &UnicityCertificate{}
	err := Cbor.Unmarshal(u.UnicityCertificate, uc)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal unicity certificate: %w", err)
	}
	return uc, nil
}

func VerifyUnitStateProof(u *UnitStateProof, algorithm crypto.Hash, unitData *StateUnitData, ucv UnicityCertificateValidator) error {
	if u == nil {
		return errors.New("unit state proof is nil")
	}
	if u.UnitID == nil {
		return errors.New("unit ID is nil")
	}
	if u.UnitTreeCert == nil {
		return errors.New("unit tree cert is nil")
	}
	if u.StateTreeCert == nil {
		return errors.New("state tree cert is nil")
	}
	if u.UnicityCertificate == nil {
		return errors.New("unicity certificate is nil")
	}
	if unitData == nil {
		return errors.New("unit data is nil")
	}
	uc, err := u.getUCv1()
	if err != nil {
		return fmt.Errorf("failed to get unicity certificate: %w", err)
	}
	if err := ucv.Validate(uc); err != nil {
		return fmt.Errorf("invalid unicity certificate: %w", err)
	}
	hash, err := unitData.Hash(algorithm)
	if err != nil {
		return fmt.Errorf("failed to calculate unit data hash: %w", err)
	}
	if !bytes.Equal(u.UnitTreeCert.UnitDataHash, hash) {
		return errors.New("unit data hash does not match hash in unit tree")
	}
	ir := uc.InputRecord
	hash, summary, err := u.CalculateStateTreeOutput(algorithm)
	if err != nil {
		return fmt.Errorf("failed to calculate state tree output: %w", err)
	}
	if !bytes.Equal(util.Uint64ToBytes(summary), ir.SummaryValue) {
		return fmt.Errorf("invalid summary value: expected %X, got %X", ir.SummaryValue, util.Uint64ToBytes(summary))
	}
	if !bytes.Equal(hash, ir.Hash) {
		return fmt.Errorf("invalid state root hash: expected %X, got %X", ir.Hash, hash)
	}
	return nil
}

func (u *UnitStateProof) CalculateStateTreeOutput(algorithm crypto.Hash) ([]byte, uint64, error) {
	var z []byte
	if u.UnitTreeCert.TransactionRecordHash == nil {
		z = abhash.SumHashes(algorithm,
			u.UnitLedgerHash,
			u.UnitTreeCert.UnitDataHash,
		)
	} else {
		z = abhash.SumHashes(algorithm,
			abhash.SumHashes(algorithm, u.UnitLedgerHash, u.UnitTreeCert.TransactionRecordHash),
			u.UnitTreeCert.UnitDataHash,
		)
	}

	logRoot := mt.PlainTreeOutput(u.UnitTreeCert.Path, z, algorithm)
	id := u.UnitID
	sc := u.StateTreeCert
	v := u.UnitValue + sc.LeftSummaryValue + sc.RightSummaryValue
	h, err := computeHash(algorithm, id, logRoot, v, sc.LeftSummaryHash, sc.LeftSummaryValue, sc.RightSummaryHash, sc.RightSummaryValue)
	if err != nil {
		return nil, 0, err
	}
	for _, p := range sc.Path {
		vv := p.Value + v + p.SiblingSummaryValue
		if id.Compare(p.UnitID) == -1 {
			h, err = computeHash(algorithm, p.UnitID, p.LogsHash, vv, h, v, p.SiblingSummaryHash, p.SiblingSummaryValue)
		} else {
			h, err = computeHash(algorithm, p.UnitID, p.LogsHash, vv, p.SiblingSummaryHash, p.SiblingSummaryValue, h, v)
		}
		if err != nil {
			return nil, 0, err
		}
		v = vv
	}
	return h, v, nil
}

func (up *UnitDataAndProof) UnmarshalUnitData(v any) error {
	if up.UnitData == nil {
		return fmt.Errorf("unit data is nil")
	}
	return up.UnitData.UnmarshalData(v)
}

func (sd *StateUnitData) UnmarshalData(v any) error {
	if sd.Data == nil {
		return fmt.Errorf("state unit data is nil")
	}
	return Cbor.Unmarshal(sd.Data, v)
}

func (sd *StateUnitData) Hash(hashAlgo crypto.Hash) ([]byte, error) {
	hasher := abhash.New(hashAlgo.New())
	hasher.WriteRaw(sd.Data)
	return hasher.Sum()
}

func computeHash(algorithm crypto.Hash, id UnitID, logRoot []byte, summary uint64, leftHash []byte, leftSummary uint64, rightHash []byte, rightSummary uint64) ([]byte, error) {
	hasher := abhash.New(algorithm.New())
	hasher.Write(id)
	hasher.Write(logRoot)
	hasher.Write(summary)
	hasher.Write(leftHash)
	hasher.Write(leftSummary)
	hasher.Write(rightHash)
	hasher.Write(rightSummary)
	return hasher.Sum()
}

func (u *UnitStateProof) GetVersion() ABVersion {
	if u != nil && u.Version > 0 {
		return u.Version
	}
	return 1
}

func (u *UnitStateProof) MarshalCBOR() ([]byte, error) {
	type alias UnitStateProof
	if u.Version == 0 {
		u.Version = u.GetVersion()
	}
	return Cbor.MarshalTaggedValue(UnitStateProofTag, (*alias)(u))
}

func (u *UnitStateProof) UnmarshalCBOR(data []byte) error {
	type alias UnitStateProof
	if err := Cbor.UnmarshalTaggedValue(UnitStateProofTag, data, (*alias)(u)); err != nil {
		return err
	}
	return EnsureVersion(u, u.Version, 1)
}
