package main

import (
	"alidayu"
	"fmt"
	"math/rand"
	"time"
)

var s1 = rand.NewSource(time.Now().UTC().UnixNano())
var r1 = rand.New(s1)

func send(mobile string, code string) {
	alidayu.AppKey = aliDayuAppKey
	alidayu.AppSecret = aliDayuAppSecret
	alidayu.UseHTTP = true
	success, resp := alidayu.SendSMS(mobile, "源投信息", "SMS_16035023",
		fmt.Sprintf(`{"product":"MarketX","code":"%v","timeout":"15分钟"}`, code))
	fmt.Println("Success:", success)
	fmt.Println(resp)
}

func generateCode() string {
	return fmt.Sprintf("%v%v%v%v", r1.Intn(10), r1.Intn(10), r1.Intn(10), r1.Intn(10))
}
