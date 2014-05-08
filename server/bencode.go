package server

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

func writeBencoded(w io.Writer, data interface{}) {
	switch v := data.(type) {
	case string:
		str := fmt.Sprintf("%s:%s", strconv.Itoa(len(v)), v)
		io.WriteString(w, str)

	case int:
		str := fmt.Sprintf("i%se", strconv.Itoa(v))
		io.WriteString(w, str)

	case uint:
		str := fmt.Sprintf("i%se", strconv.FormatUint(uint64(v), 10))
		io.WriteString(w, str)

	case int64:
		str := fmt.Sprintf("i%se", strconv.FormatInt(v, 10))
		io.WriteString(w, str)

	case uint64:
		str := fmt.Sprintf("i%se", strconv.FormatUint(v, 10))
		io.WriteString(w, str)

	case time.Duration: // Assume seconds
		str := fmt.Sprintf("i%se", strconv.FormatInt(int64(v/time.Second), 10))
		io.WriteString(w, str)

	case map[string]interface{}:
		io.WriteString(w, "d")
		for key, val := range v {
			str := fmt.Sprintf("%s:%s", strconv.Itoa(len(key)), key)
			io.WriteString(w, str)
			writeBencoded(w, val)
		}
		io.WriteString(w, "e")

	case []string:
		io.WriteString(w, "l")
		for _, val := range v {
			writeBencoded(w, val)
		}
		io.WriteString(w, "e")

	default:
		// Although not currently necessary,
		// should handle []interface{} manually; Go can't do it implicitly
		panic("tried to bencode an unsupported type!")
	}
}
