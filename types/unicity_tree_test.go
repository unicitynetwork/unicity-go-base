package types

import (
	"crypto"
	"testing"

	abhash "github.com/alphabill-org/alphabill-go-base/hash"
	"github.com/stretchr/testify/require"

	"github.com/alphabill-org/alphabill-go-base/tree/imt"
)

func TestNewUnicityTree(t *testing.T) {
	unicityTree, err := NewUnicityTree(crypto.SHA256, []*UnicityTreeData{
		{
			Partition:     1,
			ShardTreeRoot: []byte{9, 9, 9, 9},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, unicityTree)
}

func TestGetCertificate_Ok(t *testing.T) {
	key1 := PartitionID(1)
	key2 := PartitionID(2)
	data := []*UnicityTreeData{
		{
			Partition:     key2,
			ShardTreeRoot: []byte{2, 2, 2, 2},
		},
		{
			Partition:     key1,
			ShardTreeRoot: []byte{1, 1, 1, 1},
		},
	}
	unicityTree, err := NewUnicityTree(crypto.SHA256, data)
	require.NoError(t, err)
	cert, err := unicityTree.Certificate(key1)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Equal(t, key1, cert.Partition)

	// restore first hash step. the data slice is now sorted so [0]==key1
	h := crypto.SHA256.New()
	data[0].AddToHasher(abhash.New(h))
	hashSteps := []*imt.PathItem{imt.NewPathItem(data[0].Partition.Bytes(), h.Sum(nil))}
	for _, hashStep := range cert.HashSteps {
		hashSteps = append(hashSteps, hashStep.ToIMTPathItem())
	}

	root, err := imt.IndexTreeOutput(hashSteps, key1.Bytes(), crypto.SHA256)
	require.NoError(t, err)
	require.Equal(t, unicityTree.RootHash(), root)
	// PartitionID 0 is not part of UT
	cert, err = unicityTree.Certificate(PartitionID(0))
	require.EqualError(t, err, "certificate for partition 00000000 not found")
	require.Nil(t, cert)
}

func TestGetCertificate_InvalidKey(t *testing.T) {
	unicityTree, err := NewUnicityTree(crypto.SHA256, []*UnicityTreeData{
		{
			Partition:     0x01020301,
			ShardTreeRoot: []byte{9, 9, 9, 9},
		},
	})
	require.NoError(t, err)
	cert, err := unicityTree.Certificate(0x0102)

	require.Nil(t, cert)
	require.EqualError(t, err, "certificate for partition 00000102 not found")
}

func TestGetCertificate_KeyNotFound(t *testing.T) {
	unicityTree, err := NewUnicityTree(crypto.SHA256, []*UnicityTreeData{
		{
			Partition:     0x01020301,
			ShardTreeRoot: []byte{9, 9, 9, 9},
		},
	})
	require.NoError(t, err)
	cert, err := unicityTree.Certificate(1)
	require.Nil(t, cert)
	require.EqualError(t, err, "certificate for partition 00000001 not found")
}
