// Router branch for /wechat/ operations
package main

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func wechatAddHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params) {
	// All params must be present
	openID, of := CheckLengthForm("", r, "openid",
		OpenIdMinMax, OpenIdMinMax)
	unionID, of := CheckLengthForm("", r, "unionid",
		OpenIdMinMax, OpenIdMinMax)
	citizenType, of := CheckRangeForm(of, r, "citizen_type",
		CitizenTypeOther)
	overseasBank, of := CheckRangeForm(of, r, "overseas_bank",
		NoYesMax)
	investmentAmount, of := CheckRangeForm(of, r, "investment_amount",
		InvestmentAmountMore100K)

	if of != "" {
		formatReturnJson(w, r, ps, ErrorFmtCodeBadArgument, of, LoginNone, nil)
		return
	}

	if !dbConn.First(&Wechat{}, "union_id = ?", unionID).RecordNotFound() {
		formatReturnJson(w, r, ps, ErrorCodeWechatExists, "", LoginNone, nil)
		return
	}

	// Parse photo id
	furl, fname, ftype, err := parseFileUpload(r, "photo_id", "wechat",
		unionID, "photo_id")
	if err != nil {
		formatReturnJson(w, r, ps, ErrorCodeUploadIdError, "", LoginNone, nil)
		return
	}

	// Save link in case we change storage types in the future
	wc := Wechat{
		WechatState:      WechatStateCreated,
		OpenID:           openID,
		UnionID:          unionID,
		PhotoIDPic:       furl,
		PhotoIDName:      fname,
		PhotoIDType:      ftype,
		CitizenType:      citizenType,
		OverseasBank:     overseasBank,
		InvestmentAmount: investmentAmount,
	}
	if dbConn.Create(&wc).Error != nil {
		formatReturnJson(w, r, ps, ErrorCodeWechatError, "", LoginNone, nil)
		return
	}

	// All good
	formatReturnJson(w, r, ps, ErrorCodeNone, "", LoginNone,
		map[string]interface{}{"state": wc.WechatState})
}

func wechatIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params) {
	unionId, ok := CheckLength(true, ps.ByName("id"),
		OpenIdMinMax, OpenIdMinMax)

	if !ok {
		formatReturnJson(w, r, ps, ErrorCodeIdError, "", LoginNone, nil)
		return
	}

	// Check if exists
	var wc Wechat
	if dbConn.First(&wc, "union_id = ?", unionId).RecordNotFound() {
		formatReturnJson(w, r, ps, ErrorCodeWechatError, "", LoginNone, nil)
		return
	}

	// All good
	formatReturnJson(w, r, ps, ErrorCodeNone, "", LoginNone,
		map[string]interface{}{"state": wc.WechatState})
}
