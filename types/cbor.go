package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/alphabill-org/alphabill-go-base/types/hex"
	"github.com/fxamacker/cbor/v2"
)

type (
	RawCBOR    []byte
	TaggedCBOR = RawCBOR

	cborHandler struct {
		encMode cbor.EncMode
	}
)

var (
	Cbor = cborHandler{}

	cborNil = []byte{0xf6}
)

/*
Set Core Deterministic Encoding as standard. See <https://www.rfc-editor.org/rfc/rfc8949.html#name-deterministically-encoded-c>.
*/
func (c *cborHandler) cborEncoder() (cbor.EncMode, error) {
	if c.encMode != nil {
		return c.encMode, nil
	}
	encMode, err := cbor.CoreDetEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	c.encMode = encMode
	return encMode, nil
}

func (c cborHandler) Marshal(v any) ([]byte, error) {
	enc, err := c.cborEncoder()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(v)
}

func (c cborHandler) MarshalTagged(tag ABTag, arr ...interface{}) ([]byte, error) {
	data, err := c.Marshal(arr)
	if err != nil {
		return nil, err
	}
	return c.Marshal(cbor.RawTag{
		Number:  tag,
		Content: data,
	})
}

func (c cborHandler) MarshalTaggedValue(tag ABTag, v any) ([]byte, error) {
	data, err := c.Marshal(v)
	if err != nil {
		return nil, err
	}
	return c.Marshal(cbor.RawTag{
		Number:  tag,
		Content: data,
	})
}

func (c cborHandler) Unmarshal(data []byte, v any) error {
	return cbor.Unmarshal(data, v)
}

func (c cborHandler) UnmarshalTagged(data []byte) (ABTag, []interface{}, error) {
	var raw cbor.RawTag
	if err := c.Unmarshal(data, &raw); err != nil {
		return 0, nil, err
	}
	arr := make([]interface{}, 0)
	if err := c.Unmarshal(raw.Content, &arr); err != nil {
		return 0, nil, err
	}
	return raw.Number, arr, nil
}

func (c cborHandler) UnmarshalTaggedValue(tag ABTag, data []byte, v any) error {
	var raw cbor.RawTag
	if err := c.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Number != tag {
		return fmt.Errorf("unexpected tag: %d, expected: %d", raw.Number, tag)
	}

	if err := c.Unmarshal(raw.Content, v); err != nil {
		return err
	}
	// check if v is of Versioned interface
	if ver, ok := v.(Versioned); ok {
		if ver.GetVersion() == 0 {
			return errors.New("version number cannot be zero")
		}
	}
	return nil
}

func (c cborHandler) GetEncoder(w io.Writer) (*cbor.Encoder, error) {
	enc, err := c.cborEncoder()
	if err != nil {
		return nil, err
	}
	return enc.NewEncoder(w), nil
}

func (c cborHandler) Encode(w io.Writer, v any) error {
	enc, err := c.GetEncoder(w)
	if err != nil {
		return err
	}
	return enc.Encode(v)
}

func (c cborHandler) GetDecoder(r io.Reader) *cbor.Decoder {
	return cbor.NewDecoder(r)
}

func (c cborHandler) Decode(r io.Reader, v any) error {
	return c.GetDecoder(r).Decode(v)
}

// MarshalCBOR returns r or CBOR nil if r is empty.
func (r RawCBOR) MarshalCBOR() ([]byte, error) {
	if len(r) == 0 {
		return cborNil, nil
	}
	return r, nil
}

// UnmarshalCBOR copies data into r unless it's CBOR "nil marker" - in that
// case r is set to empty slice.
func (r *RawCBOR) UnmarshalCBOR(data []byte) error {
	if r == nil {
		return errors.New("UnmarshalCBOR on nil pointer")
	}
	if bytes.Equal(data, cborNil) {
		*r = (*r)[0:0]
	} else {
		*r = append((*r)[0:0], data...)
	}
	return nil
}

func (r RawCBOR) MarshalText() ([]byte, error) {
	return hex.Encode(r), nil
}

func (r *RawCBOR) UnmarshalText(src []byte) error {
	res, err := hex.Decode(src)
	if err == nil {
		*r = res
	}
	return err
}
