package model

import "time"

type Message struct {
	Hostname  string `json:"hostname"`
	IpAddress string `json:"ipaddress"`

	Timestamp time.Time `json:"-"`
}
