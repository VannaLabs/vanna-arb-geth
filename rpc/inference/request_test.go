package inference

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Basic inference test on a spread quoting regression model
func TestInference(t *testing.T) {
	rc := NewRequestClient(5125)
	tx := InferenceTx{
		Hash:   "0x123456789",
		Model:  "QmXQpupTphRTeXJMEz3BCt9YUF6kikcqExxPdcVoL1BBhy",
		Params: "[[0.002, 0.005, 0.004056685]]",
		TxType: "inference",
	}
	result, err := rc.Emit(tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, "0.0013500629", result)
}

func TestZKInference(t *testing.T) {
	rc := NewRequestClient(5125)
	tx := InferenceTx{
		Hash:   "0x123456789",
		Model:  "QmbbzDwqSxZSgkz1EbsNHp2mb67rYeUYHYWJ4wECE24S7A",
		Params: "[[0.73286679744720459, 0.4486280083656311]]",
		TxType: "zkinference",
	}
	result, err := rc.Emit(tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, "4.625", result)
}

// Testing malformed Inference Parameters -> Should Fail
func TestMalformedInference(t *testing.T) {
	rc := NewRequestClient(5125)
	tx := InferenceTx{
		Hash:   "0x123",
		Model:  "QmXQpupTphRTeXJMEz3BCt9YUF6kikcqExxPdcVoL1BBhy",
		Params: "[[[3r.002, 0.005, 0.004056685]]",
		TxType: "zkinference",
	}
	result, err := rc.Emit(tx)
	assert.Equal(t, errors.New("Could not reach inference consensus"), err)
	assert.Equal(t, "INFERENCE ERROR", result)
}
