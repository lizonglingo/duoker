package config

import (
	"fmt"
	"github.com/ThreeKing2018/gocolor"
)

const (
	IpAmStorageFsPath = "/workplace/duoker/netconfig/subnet.json"
	NetStoragePath    = "/workplace/duoker/netconfig/network.json"
)

func Banner() string {
	return fmt.Sprintf("%s %s %s %s %s %s ",
		gocolor.SRedBG("welcome"),
		gocolor.SGreenBG("to"),
		gocolor.SYellowBG("use"),
		gocolor.SBlueBG("duoker"),
		"🖼🖼🖼",
		"❗❗")
}
