/*
 * This file is part of Chihaya.
 *
 * Chihaya is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Chihaya is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.
 */

package util

import (
	"bytes"
	"log"
	"strconv"
	"time"
)

func Bencode(data interface{}, buf *bytes.Buffer) {
	switch v := data.(type) {
	case string:
		buf.WriteString(strconv.Itoa(len(v)))
		buf.WriteRune(':')
		buf.WriteString(v)
	case int:
		buf.WriteRune('i')
		buf.WriteString(strconv.Itoa(v))
		buf.WriteRune('e')
	case uint:
		buf.WriteRune('i')
		buf.WriteString(strconv.FormatUint(uint64(v), 10))
		buf.WriteRune('e')
	case int64:
		buf.WriteRune('i')
		buf.WriteString(strconv.FormatInt(v, 10))
		buf.WriteRune('e')
	case uint64:
		buf.WriteRune('i')
		buf.WriteString(strconv.FormatUint(v, 10))
		buf.WriteRune('e')
	case time.Duration:
		// Assume seconds
		buf.WriteRune('i')
		buf.WriteString(strconv.FormatInt(int64(v/time.Second), 10))
		buf.WriteRune('e')
	case map[string]interface{}:
		buf.WriteRune('d')
		for key, val := range v {
			buf.WriteString(strconv.Itoa(len(key)))
			buf.WriteRune(':')
			buf.WriteString(key)
			Bencode(val, buf)
		}
		buf.WriteRune('e')
	case []string:
		buf.WriteRune('l')
		for _, val := range v {
			Bencode(val, buf)
		}
		buf.WriteRune('e')
	default:
		// Should handle []interface{} manually since Go can't do it implicitly (not currently necessary though)
		log.Printf("%T\n", v)
		panic("Tried to bencode an unsupported type!")
	}
}
