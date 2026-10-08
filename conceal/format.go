package conceal

import (
	"errors"
	"strings"
)

func buildNumFormat(str string) (NumFormat, error) {
	str = strings.ToLower(str)

	switch str {
	case "be":
		return NumFormatBE, nil
	case "le":
		return NumFormatLE, nil
	case "ascii":
		return NumFormatAscii, nil
	case "hex":
		return NumFormatHex, nil
	}
	return NumFormatBE, errors.New("wrong format")
}

type NumFormat int

const (
	NumFormatBE NumFormat = iota
	NumFormatLE
	NumFormatAscii
	NumFormatHex
)

func (f NumFormat) Spec() string {
	switch f {
	case NumFormatBE:
		return "be"
	case NumFormatLE:
		return "le"
	case NumFormatAscii:
		return "ascii"
	case NumFormatHex:
		return "hex"
	}
	return ""
}
