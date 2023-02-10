package luhn

import (
	"strconv"
)

func Valid(s string) bool {
	var digRev []int // each char of string reversed, converted to int, removed spaces
	ss := []rune(s)

	for i := len(ss) - 1; i >= 0; i-- {
		if ss[i] == ' ' {
			continue
		}
		n, err := strconv.Atoi(string(ss[i]))
		if err != nil {
			return false // invalid digit encountered
		}
		digRev = append(digRev, n)
	}

	if len(digRev) < 2 {
		return false
	}

	var sum int
	for i, n := range digRev {
		if i%2 != 0 {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
	}
	return sum%10 == 0
}
