package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUInt(t *testing.T) {
	var testData = map[string]uint64{
		"12345":           12345,
		"123123123123123": 123123123123123,
		"1234567890":      1234567890,
	}

	for key, value := range testData {
		res, err := ParseUInt([]byte(key))
		if assert.NoError(t, err) {
			assert.Equal(t, value, res)
		}
	}
}

func TestParseInt(t *testing.T) {
	var testData = map[string]int64{
		"12345":            12345,
		"-12345":           -12345,
		"123123123123123":  123123123123123,
		"-123123123123123": -123123123123123,
		"1234567890":       1234567890,
	}

	for key, value := range testData {
		res, err := ParseInt([]byte(key))
		if assert.NoError(t, err) {
			assert.Equal(t, value, res)
		}
	}
}

func TestParseFloat(t *testing.T) {
	var testData = map[string]float64{
		"123.45":            123.45,
		"-12.345":           -12.345,
		"123123.123123123":  123123.123123123,
		"-12312.3123123123": -12312.3123123123,
		"1234567.890":       1234567.890,
	}

	for key, value := range testData {
		res, err := ParseFloat([]byte(key))
		if assert.NoError(t, err) {
			assert.Equal(t, value, res)
		}
	}
}
