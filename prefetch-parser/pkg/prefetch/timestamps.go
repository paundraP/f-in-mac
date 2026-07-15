package prefetch

import "time"

const filetimeUnixOffset = 11644473600

func FiletimeToTime(ft uint64) time.Time {
	totalSec := ft / 10000000
	rem := ft % 10000000
	return time.Unix(int64(totalSec)-filetimeUnixOffset, int64(rem*100)).UTC()
}

func FiletimeToTimes(fts []uint64) []time.Time {
	out := make([]time.Time, len(fts))
	for i, ft := range fts {
		out[i] = FiletimeToTime(ft)
	}
	return out
}
