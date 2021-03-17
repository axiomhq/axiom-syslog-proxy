package parser

import (
	"strconv"
)

const (
	ascii0  = 48
	ascii9  = 57
	asciif0 = 48.0
)

// A NumError records a failed conversion.
type NumError struct {
	Func string // the failing function (ParseBool, ParseInt, ParseUint, ParseFloat)
	Num  string // the input
	Err  error  // the reason the conversion failed (e.g. ErrRange, ErrSyntax, etc.)
}

func (e *NumError) Error() string {
	return "util." + e.Func + ": " + "parsing " + strconv.Quote(e.Num) + ": " + e.Err.Error()
}

// ParseUInt is similar to strconv.ParseUint, but operates on []byte which can save a string allocation
// and is therefore faster
func ParseUInt(str []byte) (uint64, error) {
	var result uint64
	for _, char := range str {
		if char >= ascii0 && char <= ascii9 {
			result = result*10 + (uint64(char) - ascii0)
		} else {
			return 0, &NumError{Func: "ParseUInt", Num: string(str), Err: strconv.ErrSyntax}
		}
	}

	return result, nil
}

// ParseInt is similar to strconv.ParseInt, but operates on []byte which can save a string allocation
// and is therefore faster
func ParseInt(str []byte) (int64, error) {
	var result int64
	isPositive := true
	for i, char := range str {
		switch {
		case char == '-' && i == 0:
			isPositive = false
		case char == '+' && i == 0:
			isPositive = true
		case char < ascii0 || char > ascii9:
			return 0, &NumError{Func: "ParseInt", Num: string(str), Err: strconv.ErrSyntax}
		case isPositive:
			result = result*10 + (int64(char) - ascii0)
		case !isPositive:
			result = result*10 - (int64(char) - ascii0)
		}
	}

	return result, nil
}

// ParseFloat is similar to strconv.ParseFloat, but operates on []byte which can save a string allocation
// and is therefore faster
// Note that it currently doesn't support exponents (scientific format of floats)
func ParseFloat(str []byte) (float64, error) {
	var lResult float64
	var rResult float64
	isPositive := true
	floatingPointIndex := -1
	for i := 0; i < len(str) && floatingPointIndex < 0; i++ {
		char := str[i]
		switch {
		case char == '-' && i == 0:
			isPositive = false
		case char == '+' && i == 0:
			isPositive = true
		case char == '.':
			floatingPointIndex = i
		case char < ascii0 || char > ascii9:
			return 0.0, &NumError{Func: "ParseFloat", Num: string(str), Err: strconv.ErrSyntax}
		case isPositive:
			lResult = lResult*10 + (float64(char) - asciif0)
		case !isPositive:
			lResult = lResult*10 - (float64(char) - asciif0)
		}
	}

	if floatingPointIndex < 0 {
		return lResult, nil
	}
	// golang has no reverse range, or iterators, for shame
	for i := len(str) - 1; i > floatingPointIndex; i-- {
		char := str[i]
		switch {
		case char < ascii0 || char > ascii9:
			return 0.0, &NumError{Func: "ParseFloat", Num: string(str), Err: strconv.ErrSyntax}
		default:
			rResult += (float64(char) - asciif0)
			rResult *= 0.1
		}
	}

	if isPositive {
		lResult += rResult
	} else {
		lResult -= rResult
	}

	return lResult, nil
}
