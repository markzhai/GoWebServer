package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

var s1 = rand.NewSource(time.Now().UTC().UnixNano())
var r1 = rand.New(s1)

func send(mobile string, code string) {
	fmt.Printf("Now you have %g problems.", math.Sqrt(7))
	AppKey = "23463881"
	AppSecret = "7ef326a11df885f88788682016fdd8a2"
	UseHTTP = true
	success, resp := SendSMS("13564298181", "身份验证", "SMS_16035023",
		`{"product":"MarketX","code":"1234","timeout":"15分钟"}`)
	fmt.Println("Success:", success)
	fmt.Println(resp)
}

func generateCode() string {
	return fmt.Sprintf("%v%v%v%v", r1.Intn(10), r1.Intn(10), r1.Intn(10), r1.Intn(10))
}
