package rabbit

import (
	"math/rand"
	"time"
)

func genRandomString(length int) string {
	src := rand.NewSource(time.Now().UnixNano())
	//nolint:gosec
	randomGen := rand.New(src)

	res := make([]byte, length)

	for i := 0; i < length; i++ {
		//nolint:gomnd
		res[i] = charset[randomGen.Intn(62)]
	}

	return string(res)
}

//nolint:gochecknoglobals
var charset = []byte{
	'0',
	'1',
	'2',
	'3',
	'4',
	'5',
	'6',
	'7',
	'8',
	'9',
	'a',
	'b',
	'c',
	'd',
	'e',
	'f',
	'g',
	'h',
	'i',
	'j',
	'k',
	'l',
	'm',
	'n',
	'o',
	'p',
	'q',
	'r',
	's',
	't',
	'u',
	'v',
	'w',
	'x',
	'y',
	'z',
	'A',
	'B',
	'C',
	'D',
	'E',
	'F',
	'G',
	'H',
	'I',
	'J',
	'K',
	'L',
	'M',
	'N',
	'O',
	'P',
	'Q',
	'R',
	'S',
	'T',
	'U',
	'V',
	'W',
	'X',
	'Y',
	'Z',
}
