package txid

import (
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	lastseq atomic.Int64 // (millis << 14) | sequence (14 bits untuk 0-9999)
	buffxid = sync.Pool{
		New: func() any {
			// Pre-allocate 32 bytes (14 timestamp + 3 milis + max 6 POD_ID + 4 sequence)
			b := make([]byte, 0, 32)
			return &b
		},
	}
)

func Next(POD_ID ...int) string {
	now := time.Now()
	currentMillis := now.UnixMilli()

	var sequence int64

	for {
		oldVal := lastseq.Load()
		oldMillis := oldVal >> 14
		oldSeq := oldVal & 0x3FFF // 14 bits mask (max 16383, kita pakai 0-9999)

		if currentMillis == oldMillis {
			newSeq := (oldSeq + 1) % 10000 // 4 digit sequence (0-9999)
			newVal := (currentMillis << 14) | newSeq
			if lastseq.CompareAndSwap(oldVal, newVal) {
				sequence = newSeq
				break
			}
		} else {
			newVal := currentMillis << 14
			if lastseq.CompareAndSwap(oldVal, newVal) {
				sequence = 0
				break
			}
		}

		// Cegah CPU Spinning
		runtime.Gosched()
	}

	bp := buffxid.Get().(*[]byte)
	b := (*bp)[:0]

	// 14 Byte pertama: YYYYMMDDHHMMSS
	b = now.AppendFormat(b, "20060102150405")

	// Suffix: 7 digit angka (3 digit milis + 4 digit sequence)
	milis := int64(now.Nanosecond() / 1e6) // 0-999

	// 3 digit milis
	b = append(b, byte('0'+(milis/100)%10)) // Milis ratusan
	b = append(b, byte('0'+(milis/10)%10))  // Milis puluhan
	b = append(b, byte('0'+(milis)%10))     // Milis satuan

	// POD_ID (default "00" jika tidak ada)
	if len(POD_ID) > 0 {
		podStr := strconv.Itoa(POD_ID[0])
		b = append(b, []byte(podStr)...)
	} else {
		b = append(b, []byte("01")...)
	}

	// 4 digit sequence
	b = append(b, byte('0'+(sequence/1000)%10)) // Sequence ribuan
	b = append(b, byte('0'+(sequence/100)%10))  // Sequence ratusan
	b = append(b, byte('0'+(sequence/10)%10))   // Sequence puluhan
	b = append(b, byte('0'+(sequence)%10))      // Sequence satuan

	result := string(b)

	buffxid.Put(bp)
	return result
}
