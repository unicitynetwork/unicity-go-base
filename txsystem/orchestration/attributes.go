package orchestration

import "github.com/alphabill-org/alphabill-go-base/types"

const (
	PartitionTypeID    types.PartitionTypeID = 4
	DefaultPartitionID types.PartitionID     = 4

	TransactionTypeAddVAR uint16 = 1
)

type (
	AddVarAttributes struct {
		_   struct{} `cbor:",toarray"`
		Var ValidatorAssignmentRecord
	}

	ValidatorAssignmentRecord struct {
		_                      struct{} `cbor:",toarray"`
		EpochNumber            uint64
		EpochSwitchRoundNumber uint64 // root chain round number
		ValidatorAssignment    ValidatorAssignment
	}

	ValidatorAssignment struct {
		_          struct{} `cbor:",toarray"`
		Validators []ValidatorInfo
		QuorumSize uint64 // total amount of staked Alpha required to reach consensus
	}

	ValidatorInfo struct {
		_           struct{} `cbor:",toarray"`
		ValidatorID []byte   // validator public key used to sign validation messages
		Stake       uint64   // total amount of staked Alpha by the validator
	}
)
