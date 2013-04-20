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

package database

import (
	"chihaya/util"
	"encoding/base64"
	"strconv"
)

/*
 * For these, we assume that the caller already has a read lock on the record
 *
 * Buffers are used for efficient string concatenation
 * It may look ugly with all the explicit type conversions, but this tracker is about speed
 *
 * These functions take from the buffer pool but don't give back,
 * so it's expected that the buffers are returned in the flush functions
 */

func (db *Database) RecordTorrent(torrent *Torrent, deltaSnatch uint64) {
	tq := db.bufferPool.Take() // ~50 bytes per record max
	tq.WriteString("('")
	tq.WriteString(strconv.FormatUint(torrent.Id, 10))
	tq.WriteString("','")
	tq.WriteString(strconv.FormatUint(deltaSnatch, 10))
	tq.WriteString("','")
	tq.WriteString(strconv.FormatInt(int64(len(torrent.Seeders)), 10))
	tq.WriteString("','")
	tq.WriteString(strconv.FormatInt(int64(len(torrent.Leechers)), 10))
	tq.WriteString("','")
	tq.WriteString(strconv.FormatInt(torrent.LastAction, 10))
	tq.WriteString("')")

	db.torrentChannel <- tq
}

func (db *Database) RecordUser(user *User, rawDeltaUpload int64, rawDeltaDownload int64, deltaUpload int64, deltaDownload int64) {
	uq := db.bufferPool.Take() // ~60 bytes per record max
	uq.WriteString("('")
	uq.WriteString(strconv.FormatUint(user.Id, 10))
	uq.WriteString("','")
	uq.WriteString(strconv.FormatInt(deltaUpload, 10))
	uq.WriteString("','")
	uq.WriteString(strconv.FormatInt(deltaDownload, 10))
	uq.WriteString("','")
	uq.WriteString(strconv.FormatInt(rawDeltaDownload, 10))
	uq.WriteString("','")
	uq.WriteString(strconv.FormatInt(rawDeltaUpload, 10))
	uq.WriteString("')")

	db.userChannel <- uq
}

func (db *Database) RecordTransferHistory(peer *Peer, rawDeltaUpload int64, rawDeltaDownload int64, deltaTime int64, deltaSnatch uint64, active bool) {
	th := db.bufferPool.Take() // ~110 bytes per record max

	th.WriteString("('")
	th.WriteString(strconv.FormatUint(peer.UserId, 10))
	th.WriteString("','")
	th.WriteString(strconv.FormatUint(peer.TorrentId, 10))
	th.WriteString("','")
	th.WriteString(strconv.FormatInt(rawDeltaUpload, 10))
	th.WriteString("','")
	th.WriteString(strconv.FormatInt(rawDeltaDownload, 10))
	th.WriteString("','")
	th.WriteString(util.Btoa(peer.Seeding))
	th.WriteString("','")
	th.WriteString(strconv.FormatInt(peer.StartTime, 10))
	th.WriteString("','")
	th.WriteString(strconv.FormatInt(peer.LastAnnounce, 10))
	th.WriteString("','")
	th.WriteString(strconv.FormatInt(deltaTime, 10))
	th.WriteString("','")
	th.WriteString(util.Btoa(active))
	th.WriteString("','")
	th.WriteString(strconv.FormatUint(deltaSnatch, 10))
	th.WriteString("','")
	th.WriteString(strconv.FormatUint(peer.Left, 10))
	th.WriteString("')")

	db.transferHistoryChannel <- th
}

func (db *Database) RecordTransferIp(peer *Peer) {
	ti := db.bufferPool.Take() // ~95 bytes per record max

	ti.WriteString("('")
	ti.WriteString(strconv.FormatUint(peer.UserId, 10))
	ti.WriteString("','")
	ti.WriteString(strconv.FormatUint(peer.TorrentId, 10))
	ti.WriteString("','")
	ti.WriteString(base64.StdEncoding.EncodeToString([]byte(peer.Id))) // ~30 bytes
	ti.WriteString("','")
	ti.WriteString(strconv.FormatInt(peer.StartTime, 10))
	ti.WriteString("','")
	ti.WriteString(peer.Ip)
	ti.WriteString("','")
	ti.WriteString(strconv.FormatUint(uint64(peer.Port), 10))
	ti.WriteString("')")

	db.transferIpsChannel <- ti
}

func (db *Database) RecordSnatch(peer *Peer, now int64) {
	sn := db.bufferPool.Take() // ~36 bytes per record max

	sn.WriteString("('")
	sn.WriteString(strconv.FormatUint(peer.UserId, 10))
	sn.WriteString("','")
	sn.WriteString(strconv.FormatUint(peer.TorrentId, 10))
	sn.WriteString("','")
	sn.WriteString(strconv.FormatInt(now, 10))
	sn.WriteString("')")

	db.snatchChannel <- sn
}

func (db *Database) VerifyUsedSlots(user *User) {
	db.slotVerificationChannel <- user
}

func (db *Database) UnPrune(torrent *Torrent) {
	db.mainConn.mutex.Lock()
	db.mainConn.exec(db.unPruneTorrentStmt, torrent.Id)
	db.mainConn.mutex.Unlock()
}
