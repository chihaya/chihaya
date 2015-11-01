package udp

import (
	"bytes"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
	"testing"
)

func TestParseQuery(t *testing.T) {
	passkey := "abc123"
	srv := NewServer(&config.DefaultConfig, nil)
	ann := &models.Announce{}
	buf := &bytes.Buffer{}
	for i := 0; i < 98; i++ {
		buf.WriteByte(0x1)
	}

	//now add optional parameters
	buf.WriteByte(0x2)  //type
	buf.WriteByte(0x10) //length=16
	buf.WriteString("/?passkey=")
	buf.WriteString(passkey)

	err := srv.handleOptionalParameters(buf.Bytes(), ann)
	if err != nil {
		t.Fatalf("Error while parsing optional parameters: %s", err)
	}

	if ann.Passkey != passkey {
		t.Fatalf("Parsed passkey: %s != %s", ann.Passkey, passkey)
	}
}

func TestParseMultipleQuery(t *testing.T) {
	passkey := "abc123"
	srv := NewServer(&config.DefaultConfig, nil)
	ann := &models.Announce{}
	buf := &bytes.Buffer{}
	for i := 0; i < 98; i++ {
		buf.WriteByte(0x1)
	}

	//now add optional parameters
	buf.WriteByte(0x2) //type
	buf.WriteByte(0xA) //length=10
	buf.WriteString("/?passkey=")
	buf.WriteByte(0x2) //type
	buf.WriteByte(0x6) //length=6
	buf.WriteString(passkey)

	err := srv.handleOptionalParameters(buf.Bytes(), ann)
	if err != nil {
		t.Fatalf("Error while handling optional parameters: %s", err)
	}

	if ann.Passkey != passkey {
		t.Fatalf("Parsed passkey: %s != %s", ann.Passkey, passkey)
	}
}

func TestURLEncoded(t *testing.T) {
	passkey := "abc%20123" //has an URLEncoded space in there
	srv := NewServer(&config.DefaultConfig, nil)
	ann := &models.Announce{}
	buf := &bytes.Buffer{}
	for i := 0; i < 98; i++ {
		buf.WriteByte(0x1)
	}

	//now add optional parameters
	buf.WriteByte(0x2) //type
	buf.WriteByte(0xA) //length=10
	buf.WriteString("/?passkey=")
	buf.WriteByte(0x2) //type
	buf.WriteByte(0x9) //length=9
	buf.WriteString(passkey)

	err := srv.handleOptionalParameters(buf.Bytes(), ann)
	if err != nil {
		t.Fatalf("Error while handling optional parameters: %s", err)
	}

	if ann.Passkey != "abc 123" {
		t.Fatalf("Parsed passkey: %s != %s", ann.Passkey, passkey)
	}
}
