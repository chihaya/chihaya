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

package database

import (
	"github.com/ziutek/mymysql/mysql"
	"log"
)

/*
 * Some efficient type conversions (that don't use reflection) for reading mymysql rows
 */

type rowWrapper struct {
	r mysql.Row
}

func (r *rowWrapper) Str(nn int) string {
	return r.r.Str(nn)
}

func (r *rowWrapper) Int64(nn int) int64 {
	switch data := r.r[nn].(type) {
	case int64:
		return data
	case int32:
		return int64(data)
	case int16:
		return int64(data)
	case int8:
		return int64(data)
	case int:
		return int64(data)
	case uint64:
		return int64(data)
	case uint32:
		return int64(data)
	case uint16:
		return int64(data)
	case uint8:
		return int64(data)
	case uint:
		return int64(data)
	}
	log.Panicf("i64 %T", r.r[nn])
	return 0
}

func (r *rowWrapper) Uint64(nn int) uint64 {
	switch data := r.r[nn].(type) {
	case uint64:
		return data
	case uint32:
		return uint64(data)
	case uint16:
		return uint64(data)
	case uint8:
		return uint64(data)
	case uint:
		return uint64(data)
	case int64:
		return uint64(data)
	case int32:
		return uint64(data)
	case int16:
		return uint64(data)
	case int8:
		return uint64(data)
	case int:
		return uint64(data)
	}
	log.Panicf("ui64 %T", r.r[nn])
	return 0
}

func (r *rowWrapper) Uint(nn int) uint {
	switch data := r.r[nn].(type) {
	case uint64:
		return uint(data)
	case uint32:
		return uint(data)
	case uint16:
		return uint(data)
	case uint8:
		return uint(data)
	case uint:
		return data
	case int64:
		return uint(data)
	case int32:
		return uint(data)
	case int16:
		return uint(data)
	case int8:
		return uint(data)
	case int:
		return uint(data)
	}
	log.Panicf("ui %T", r.r[nn])
	return 0
}

func (r *rowWrapper) Float64(nn int) float64 {
	switch data := r.r[nn].(type) {
	case float64:
		return data
	case float32:
		return float64(data)
	}
	log.Panicf("f64 %T", r.r[nn])
	return 0
}
