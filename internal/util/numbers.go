package util

import (
	"log"
	"strconv"
)

func MustAtoi64(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return v
}
