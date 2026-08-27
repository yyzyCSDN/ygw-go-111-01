package record

import "github.com/cespare/xxhash/v2"

func Fingerprint(parts ...string) uint64 {
	h := xxhash.New()
	for _, part := range parts {
		_, _ = h.WriteString(part)
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}
