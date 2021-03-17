// +build gofuzz

package parser

func Fuzz(data []byte) int {
	msg, _ := parseLine(data)

	if msg != nil {
		return 1
	}

	return 0
}
