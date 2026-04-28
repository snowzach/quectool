package quectool

import (
	"time"
)

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	s := time.Duration(d).String()
	b := make([]byte, 0, len(s)+2)
	b = append(b, '"')
	b = append(b, s...)
	b = append(b, '"')
	return b, nil
}
