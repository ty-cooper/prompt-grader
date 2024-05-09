package cmd

import (
	"encoding/base64"
	"math"
)

func Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func Base64Decode(str string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}
	return string(data), err
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
