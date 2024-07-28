package model

import (
	"strings"
	"time"
)

type Message struct {
	Hostname  string `json:"hostname"`
	IpAddress string `json:"ipaddress"`
	serial    string `json:"serial" binding:"optional"`

	//Make following json field optional with default 0
	FailCount int `json:"failcount" binding:"optional"`

	Timestamp time.Time `json:"-"`
}

func (m *Message) parseSerial() string {
	return strings.Join(strings.Split(m.Hostname, "-")[1:], "-")
}

func (m *Message) GetSerial() string {
	if m.serial == "" {
		m.serial = m.parseSerial()
	}

	return m.serial
}
