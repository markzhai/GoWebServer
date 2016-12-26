package main

import (
	"alidayu"
	"fmt"
	"math/rand"
	"time"
)

var s1 = rand.NewSource(time.Now().UTC().UnixNano())
var r1 = rand.New(s1)

func send(mobile string, code string) (bool, string) {
	alidayu.AppKey = aliDayuAppKey
	alidayu.AppSecret = aliDayuAppSecret
	alidayu.UseHTTP = !useSsl
	success, resp := alidayu.SendSMS(mobile, "源投金融", "SMS_26070193",
		fmt.Sprintf(`{"product":"MarketX","code":"%v","timeout":"15分钟"}`, code))

	if serverLog != nil {
		serverLog.Println("Success:", success)
		serverLog.Println(resp)
	} else {
		fmt.Println("Success:", success)
		fmt.Println(resp)
	}

	return success, resp
}

func generateCode() string {
	return fmt.Sprintf("%v%v%v%v", r1.Intn(10), r1.Intn(10), r1.Intn(10), r1.Intn(10))
}
