// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bittorrent

import "testing"

func TestClientID(t *testing.T) {
	var clientTable = []struct{ peerID, clientID string }{
		{"-AZ3034-6wfG2wk6wWLc", "AZ3034"},
		{"-AZ3042-6ozMq5q6Q3NX", "AZ3042"},
		{"-BS5820-oy4La2MWGEFj", "BS5820"},
		{"-AR6360-6oZyyMWoOOBe", "AR6360"},
		{"-AG2083-s1hiF8vGAAg0", "AG2083"},
		{"-AG3003-lEl2Mm4NEO4n", "AG3003"},
		{"-MR1100-00HS~T7*65rm", "MR1100"},
		{"-LK0140-ATIV~nbEQAMr", "LK0140"},
		{"-KT2210-347143496631", "KT2210"},
		{"-TR0960-6ep6svaa61r4", "TR0960"},
		{"-XX1150-dv220cotgj4d", "XX1150"},
		{"-AZ2504-192gwethivju", "AZ2504"},
		{"-KT4310-3L4UvarKuqIu", "KT4310"},
		{"-AZ2060-0xJQ02d4309O", "AZ2060"},
		{"-BD0300-2nkdf08Jd890", "BD0300"},
		{"-A~0010-a9mn9DFkj39J", "A~0010"},
		{"-UT2300-MNu93JKnm930", "UT2300"},
		{"-UT2300-KT4310KT4301", "UT2300"},

		{"T03A0----f089kjsdf6e", "T03A0-"},
		{"S58B-----nKl34GoNb75", "S58B--"},
		{"M4-4-0--9aa757Efd5Bl", "M4-4-0"},

		{"AZ2500BTeYUzyabAfo6U", "AZ2500"}, // BitTyrant
		{"exbc0JdSklm834kj9Udf", "exbc0J"}, // Old BitComet
		{"FUTB0L84j542mVc84jkd", "FUTB0L"}, // Alt BitComet
		{"XBT054d-8602Jn83NnF9", "XBT054"}, // XBT
		{"OP1011affbecbfabeefb", "OP1011"}, // Opera
		{"-ML2.7.2-kgjjfkd9762", "ML2.7."}, // MLDonkey
		{"-BOWA0C-SDLFJWEIORNM", "BOWA0C"}, // Bits on Wheels
		{"Q1-0-0--dsn34DFn9083", "Q1-0-0"}, // Queen Bee
		{"Q1-10-0-Yoiumn39BDfO", "Q1-10-"}, // Queen Bee Alt
		{"346------SDFknl33408", "346---"}, // TorreTopia
		{"QVOD0054ABFFEDCCDEDB", "QVOD00"}, // Qvod

		{"", ""},
		{"-", ""},
		{"12345", ""},
		{"-12345", ""},
		{"123456", "123456"},
		{"-123456", "123456"},
	}

	for _, tt := range clientTable {
		if parsedID := NewClientID(tt.peerID); parsedID != ClientID(tt.clientID) {
			t.Error("Incorrectly parsed peer ID", tt.peerID, "as", parsedID)
		}
	}
}
