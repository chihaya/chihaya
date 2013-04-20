// This file is part of Chihaya.
//
// Chihaya is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Chihaya is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.

// Package util implements some convenient functionality reused throughout
// Chihaya.
package util

// MinInt returns the smaller of the two integers provided.
func MinInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns the larger of the two integers provided.
func MaxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

// Btoa converts a boolean value into the string form "1" or "0".
func Btoa(a bool) string {
	if a {
		return "1"
	}
	return "0"
}
