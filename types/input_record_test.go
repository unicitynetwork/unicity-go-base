package types

import (
	"crypto/sha256"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	abhash "github.com/alphabill-org/alphabill-go-base/hash"
	test "github.com/alphabill-org/alphabill-go-base/testutils"
)

var ir = &InputRecord{
	Version:         1,
	PreviousHash:    []byte{0, 0, 1},
	Hash:            []byte{0, 0, 2},
	BlockHash:       []byte{0, 0, 3},
	SummaryValue:    []byte{0, 0, 4},
	ETHash:          []byte{0, 0, 5},
	RoundNumber:     1,
	Epoch:           0,
	SumOfEarnedFees: 20,
}

func TestInputRecord_IsValid(t *testing.T) {
	randomHash := test.RandomBytes(32)
	validIR := InputRecord{
		Version:      1,
		PreviousHash: randomHash,
		Hash:         randomHash,
		BlockHash:    nil,
		SummaryValue: randomHash,
		RoundNumber:  1,
		Timestamp:    NewTimestamp(),
	}
	require.NoError(t, validIR.IsValid())

	t.Run("summary value hash is nil", func(t *testing.T) {
		testIR := validIR
		testIR.SummaryValue = nil
		require.ErrorIs(t, ErrSummaryValueIsNil, testIR.IsValid())
	})

	t.Run("state changes, but block hash is nil", func(t *testing.T) {
		testIR := &InputRecord{
			Version:         1,
			PreviousHash:    randomHash,
			Hash:            []byte{1, 2, 3},
			BlockHash:       nil,
			SummaryValue:    []byte{2, 3, 4},
			SumOfEarnedFees: 1,
			RoundNumber:     1,
			Timestamp:       NewTimestamp(),
		}
		require.EqualError(t, testIR.IsValid(), "block hash is nil but state hash changed")
	})

	t.Run("state does not change, but block hash is not nil", func(t *testing.T) {
		testIR := &InputRecord{
			Version:         1,
			PreviousHash:    randomHash,
			Hash:            randomHash,
			BlockHash:       []byte{1, 2, 3},
			SummaryValue:    []byte{2, 3, 4},
			SumOfEarnedFees: 1,
			RoundNumber:     1,
			Timestamp:       NewTimestamp(),
		}
		require.EqualError(t, testIR.IsValid(), "state hash didn't change but block hash is not nil")
	})

	t.Run("timestamp unassigned", func(t *testing.T) {
		testIR := validIR
		testIR.Timestamp = 0
		require.EqualError(t, testIR.IsValid(), `timestamp is unassigned`)
	})

	t.Run("version unassigned", func(t *testing.T) {
		testIR := validIR
		testIR.Version = 0
		require.EqualError(t, testIR.IsValid(), `invalid version (type *types.InputRecord)`)
	})

	t.Run("unmarshal CBOR - ok", func(t *testing.T) {
		irBytes, err := validIR.MarshalCBOR()
		require.NoError(t, err)
		require.NotNil(t, irBytes)

		ir2 := &InputRecord{}
		require.NoError(t, ir2.UnmarshalCBOR(irBytes))
		require.Equal(t, validIR, *ir2)
	})

	t.Run("unmarshal CBOR - invalid version", func(t *testing.T) {
		validIR.Version = 2
		irBytes, err := validIR.MarshalCBOR()
		require.NoError(t, err)
		require.NotNil(t, irBytes)

		ir2 := &InputRecord{}
		require.ErrorContains(t, ir2.UnmarshalCBOR(irBytes), "invalid version (type *types.InputRecord), expected 1, got 2")
	})
}

func TestInputRecord_IsNil(t *testing.T) {
	var ir *InputRecord
	require.ErrorIs(t, ir.IsValid(), ErrInputRecordIsNil)
}

func TestInputRecord_AddToHasher(t *testing.T) {
	ir = &InputRecord{
		Version:         1,
		PreviousHash:    []byte{0, 0, 1},
		Hash:            []byte{0, 0, 2},
		BlockHash:       []byte{0, 0, 3},
		SummaryValue:    []byte{0, 0, 4},
		RoundNumber:     1,
		Epoch:           0,
		Timestamp:       1731504540,
		SumOfEarnedFees: 20,
		ETHash:          []byte{0, 0, 5},
	}
	hasher := sha256.New()
	abhasher := abhash.New(hasher)
	ir.AddToHasher(abhasher)
	hash := hasher.Sum(nil)

	expectedHash := []byte{0x65, 0x33, 0x67, 0xb2, 0xf2, 0xff, 0x9d, 0xa5, 0x2, 0x86, 0x2, 0x65, 0x46, 0xf6, 0x62,
		0x77, 0x89, 0x83, 0x10, 0x63, 0x60, 0x6b, 0x23, 0x60, 0xf2, 0x16, 0x61, 0x5a, 0x60, 0x16, 0x1, 0xbf}
	require.Equal(t, expectedHash, hash)
}

func Test_EqualIR(t *testing.T) {
	var irA = &InputRecord{
		PreviousHash:    []byte{1, 1, 1},
		Hash:            []byte{2, 2, 2},
		BlockHash:       []byte{3, 3, 3},
		SummaryValue:    []byte{4, 4, 4},
		RoundNumber:     2,
		SumOfEarnedFees: 33,
	}
	t.Run("equal", func(t *testing.T) {
		irB := &InputRecord{
			PreviousHash:    []byte{1, 1, 1},
			Hash:            []byte{2, 2, 2},
			BlockHash:       []byte{3, 3, 3},
			SummaryValue:    []byte{4, 4, 4},
			RoundNumber:     2,
			SumOfEarnedFees: 33,
		}
		require.True(t, isEqualIR(t, irA, irB))
	})
	t.Run("Previous hash not equal", func(t *testing.T) {
		irB := &InputRecord{
			PreviousHash:    []byte{1, 1},
			Hash:            []byte{2, 2, 2},
			BlockHash:       []byte{3, 3, 3},
			SummaryValue:    []byte{4, 4, 4},
			RoundNumber:     2,
			SumOfEarnedFees: 33,
		}
		require.False(t, isEqualIR(t, irA, irB))
	})
	t.Run("Hash not equal", func(t *testing.T) {
		irB := &InputRecord{
			PreviousHash:    []byte{1, 1, 1},
			Hash:            []byte{2, 2, 2, 3},
			BlockHash:       []byte{3, 3, 3},
			SummaryValue:    []byte{4, 4, 4},
			RoundNumber:     2,
			SumOfEarnedFees: 33,
		}
		require.False(t, isEqualIR(t, irA, irB))
	})
	t.Run("Block hash not equal", func(t *testing.T) {
		irB := &InputRecord{
			PreviousHash:    []byte{1, 1, 1},
			Hash:            []byte{2, 2, 2},
			BlockHash:       nil,
			SummaryValue:    []byte{4, 4, 4},
			RoundNumber:     2,
			SumOfEarnedFees: 33,
		}
		require.False(t, isEqualIR(t, irA, irB))
	})
	t.Run("Summary value not equal", func(t *testing.T) {
		irB := &InputRecord{
			PreviousHash:    []byte{1, 1, 1},
			Hash:            []byte{2, 2, 2},
			BlockHash:       []byte{3, 3, 3},
			SummaryValue:    []byte{},
			RoundNumber:     2,
			SumOfEarnedFees: 33,
		}
		require.False(t, isEqualIR(t, irA, irB))
	})
	t.Run("RoundNumber not equal", func(t *testing.T) {
		irB := &InputRecord{
			PreviousHash:    []byte{1, 1, 1},
			Hash:            []byte{2, 2, 2},
			BlockHash:       []byte{3, 3, 3},
			SummaryValue:    []byte{4, 4, 4},
			RoundNumber:     1,
			SumOfEarnedFees: 33,
		}
		require.False(t, isEqualIR(t, irA, irB))
	})
	t.Run("SumOfEarnedFees not equal", func(t *testing.T) {
		irB := &InputRecord{
			PreviousHash:    []byte{1, 1, 1},
			Hash:            []byte{2, 2, 2},
			BlockHash:       []byte{3, 3, 3},
			SummaryValue:    []byte{4, 4, 4},
			RoundNumber:     2,
			SumOfEarnedFees: 1,
		}
		require.False(t, isEqualIR(t, irA, irB))
	})
}

func Test_AssertEqualIR(t *testing.T) {
	var irA = InputRecord{
		PreviousHash:    []byte{1, 1, 1},
		Hash:            []byte{2, 2, 2},
		BlockHash:       []byte{3, 3, 3},
		SummaryValue:    []byte{4, 4, 4},
		Timestamp:       20241113,
		Epoch:           1,
		RoundNumber:     2,
		SumOfEarnedFees: 33,
	}

	t.Run("equal", func(t *testing.T) {
		irB := irA
		require.NoError(t, AssertEqualIR(&irA, &irB))
	})

	t.Run("Previous hash not equal", func(t *testing.T) {
		irB := irA
		irB.PreviousHash = []byte{1, 1}
		require.EqualError(t, AssertEqualIR(&irA, &irB), "previous state hash is different: 010101 vs 0101")
	})

	t.Run("Hash not equal", func(t *testing.T) {
		irB := irA
		irB.Hash = []byte{2, 2, 2, 3}
		require.EqualError(t, AssertEqualIR(&irA, &irB), "state hash is different: 020202 vs 02020203")
	})

	t.Run("Block hash not equal", func(t *testing.T) {
		irB := irA
		irB.BlockHash = nil
		require.EqualError(t, AssertEqualIR(&irA, &irB), "block hash is different: 030303 vs ")
	})

	t.Run("Summary value not equal", func(t *testing.T) {
		irB := irA
		irB.SummaryValue = []byte{}
		require.EqualError(t, AssertEqualIR(&irA, &irB), "summary value is different: [4 4 4] vs []")
	})

	t.Run("RoundNumber not equal", func(t *testing.T) {
		irB := irA
		irB.RoundNumber = 1
		require.EqualError(t, AssertEqualIR(&irA, &irB), "round number is different: 2 vs 1")
	})

	t.Run("SumOfEarnedFees not equal", func(t *testing.T) {
		irB := irA
		irB.SumOfEarnedFees = 1
		require.EqualError(t, AssertEqualIR(&irA, &irB), "sum of fees is different: 33 vs 1")
	})

	t.Run("Timestamp not equal", func(t *testing.T) {
		irB := irA
		irB.Timestamp = 0
		require.EqualError(t, AssertEqualIR(&irA, &irB), "timestamp is different: 20241113 vs 0")
	})

	t.Run("Epoch not equal", func(t *testing.T) {
		irB := irA
		irB.Epoch = 10
		require.EqualError(t, AssertEqualIR(&irA, &irB), "epoch is different: 1 vs 10")
	})
}

func TestInputRecord_NewRepeatUC(t *testing.T) {
	repeatUC := ir.NewRepeatIR()
	require.NotNil(t, repeatUC)
	require.True(t, isEqualIR(t, ir, repeatUC))
	require.True(t, reflect.DeepEqual(ir, repeatUC))
	ir.RoundNumber++
	require.False(t, isEqualIR(t, ir, repeatUC))
}

func isEqualIR(t *testing.T, ir1, ir2 *InputRecord) bool {
	b, err := EqualIR(ir1, ir2)
	require.NoError(t, err)
	return b
}

func TestStringer(t *testing.T) {
	var testIR *InputRecord = nil
	require.Equal(t, "input record is nil", testIR.String())
	testIR = &InputRecord{
		PreviousHash:    []byte{1, 1, 1},
		Hash:            []byte{2, 2, 2},
		BlockHash:       []byte{3, 3, 3},
		ETHash:          []byte{4, 4, 4},
		SummaryValue:    []byte{5, 5, 5},
		RoundNumber:     2,
		SumOfEarnedFees: 33,
	}
	require.Equal(t, "H: 020202 H': 010101 Bh: 030303 round: 2 epoch: 0 fees: 33 ETh: 040404 summary: 050505", testIR.String())
}
