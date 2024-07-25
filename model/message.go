package model

import "time"

type Message struct {
	Hostname  string `json:"hostname"`
	IpAddress string `json:"ipaddress"`
	Serial    string `json:"serial"`

	Timestamp time.Time `json:"-"`
}
