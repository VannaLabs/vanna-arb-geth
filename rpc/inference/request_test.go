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
	assert.Equal(t, nil, err)
	assert.Equal(t, result, "0.0013500629")
}

// Testing malformed Inference Parameters -> Should Fail
func TestMalformedInference(t *testing.T) {
	rc := NewRequestClient(5125)
	tx := InferenceTx{
		Hash:   "0x123",
		Model:  "QmXQpupTphRTeXJMEz3BCt9YUF6kikcqExxPdcVoL1BBhy",
		Params: "[[[3r.002, 0.005, 0.004056685]]",
		TxType: "inference",
	}
	result, err := rc.Emit(tx)
	assert.Equal(t, errors.New("Could not reach consensus"), err)
	assert.Equal(t, result, "INFERENCE ERROR")
}
