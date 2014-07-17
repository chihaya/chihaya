// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package models

import (
	"testing"
)

type PeerClientPair struct {
	announce Announce
	clientID string
}

var TestClients = []PeerClientPair{
	{Announce{PeerID: "-AZ3034-6wfG2wk6wWLc"}, "AZ3034"},
	{Announce{PeerID: "-AZ3042-6ozMq5q6Q3NX"}, "AZ3042"},
	{Announce{PeerID: "-BS5820-oy4La2MWGEFj"}, "BS5820"},
	{Announce{PeerID: "-AR6360-6oZyyMWoOOBe"}, "AR6360"},
	{Announce{PeerID: "-AG2083-s1hiF8vGAAg0"}, "AG2083"},
	{Announce{PeerID: "-AG3003-lEl2Mm4NEO4n"}, "AG3003"},
	{Announce{PeerID: "-MR1100-00HS~T7*65rm"}, "MR1100"},
	{Announce{PeerID: "-LK0140-ATIV~nbEQAMr"}, "LK0140"},
	{Announce{PeerID: "-KT2210-347143496631"}, "KT2210"},
	{Announce{PeerID: "-TR0960-6ep6svaa61r4"}, "TR0960"},
	{Announce{PeerID: "-XX1150-dv220cotgj4d"}, "XX1150"},
	{Announce{PeerID: "-AZ2504-192gwethivju"}, "AZ2504"},
	{Announce{PeerID: "-KT4310-3L4UvarKuqIu"}, "KT4310"},
	{Announce{PeerID: "-AZ2060-0xJQ02d4309O"}, "AZ2060"},
	{Announce{PeerID: "-BD0300-2nkdf08Jd890"}, "BD0300"},
	{Announce{PeerID: "-A~0010-a9mn9DFkj39J"}, "A~0010"},
	{Announce{PeerID: "-UT2300-MNu93JKnm930"}, "UT2300"},
	{Announce{PeerID: "-UT2300-KT4310KT4301"}, "UT2300"},

	{Announce{PeerID: "T03A0----f089kjsdf6e"}, "T03A0-"},
	{Announce{PeerID: "S58B-----nKl34GoNb75"}, "S58B--"},
	{Announce{PeerID: "M4-4-0--9aa757Efd5Bl"}, "M4-4-0"},

	{Announce{PeerID: "AZ2500BTeYUzyabAfo6U"}, "AZ2500"}, // BitTyrant
	{Announce{PeerID: "exbc0JdSklm834kj9Udf"}, "exbc0J"}, // Old BitComet
	{Announce{PeerID: "FUTB0L84j542mVc84jkd"}, "FUTB0L"}, // Alt BitComet
	{Announce{PeerID: "XBT054d-8602Jn83NnF9"}, "XBT054"}, // XBT
	{Announce{PeerID: "OP1011affbecbfabeefb"}, "OP1011"}, // Opera
	{Announce{PeerID: "-ML2.7.2-kgjjfkd9762"}, "ML2.7."}, // MLDonkey
	{Announce{PeerID: "-BOWA0C-SDLFJWEIORNM"}, "BOWA0C"}, // Bits on Wheels
	{Announce{PeerID: "Q1-0-0--dsn34DFn9083"}, "Q1-0-0"}, // Queen Bee
	{Announce{PeerID: "Q1-10-0-Yoiumn39BDfO"}, "Q1-10-"}, // Queen Bee Alt
	{Announce{PeerID: "346------SDFknl33408"}, "346---"}, // TorreTopia
	{Announce{PeerID: "QVOD0054ABFFEDCCDEDB"}, "QVOD00"}, // Qvod

	{Announce{PeerID: ""}, ""},
	{Announce{PeerID: "-"}, ""},
	{Announce{PeerID: "12345"}, ""},
	{Announce{PeerID: "-12345"}, ""},
	{Announce{PeerID: "123456"}, "123456"},
	{Announce{PeerID: "-123456"}, "123456"},
}

func TestClientID(t *testing.T) {
	for _, pair := range TestClients {
		if parsedID := pair.announce.ClientID(); parsedID != pair.clientID {
			t.Error("Incorrectly parsed peer ID", pair.announce.PeerID, "as", parsedID)
		}
	}
}
