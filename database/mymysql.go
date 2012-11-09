package database

import (
	"github.com/ziutek/mymysql/mysql"
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
	}
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
	}
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
	}
	return 0
}

func (r *rowWrapper) Float64(nn int) float64 {
	switch data := r.r[nn].(type) {
	case float64:
		return data
	case float32:
		return float64(data)
	}
	return 0
}