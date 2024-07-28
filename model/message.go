package model

import "time"

type Message struct {
	Hostname  string `json:"hostname"`
	IpAddress string `json:"ipaddress"`
	Serial    string `json:"serial"`

	//Make following json field optional with default 0
	FailCount int `json:"failcount" binding:"optional"`

	Timestamp time.Time `json:"-"`
}
