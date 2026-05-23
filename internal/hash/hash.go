package hash

import "hash/fnv"

func String64(value string) uint64 {
	hasher := fnv.New64a()

	_, _ = hasher.Write([]byte(value))

	return hasher.Sum64()
}

func Index(value string, size int) int {
	if size <= 0 {
		return 0
	}

	return int(String64(value) % uint64(size))
}
