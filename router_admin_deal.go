// Router branch for /admin/deal/ operations
package main

import (
	crand "crypto/rand"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

func adminDealsHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// Both fields are optional with defaults to 0
	pnum, _ := CheckRange(true, r.FormValue("page_number"), NumberMax)
	psize, _ := CheckRange(true, r.FormValue("page_size"), NumberMax)
	// Must be a valid page size
	psize, ok := PageSizes[psize]
	if !ok {
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	var ds []Deal
	kw := r.FormValue("keyword")
	// Fake a join with an existing company for user submitted deals
	if dbConn.Joins("join companies on (companies.id = deals.company_id " +
		"and companies.name ~* ?) or (companies.id = 1 " +
		"and deals.deal_state = ? and deals.name ~* ?)",
		kw, DealStateUserSubmitted, kw).
		Order("end_date desc, start_date desc").
		Offset(int(pnum * psize)).Limit(int(psize)).Find(&ds).Error != nil {
		// Probably bad inputs
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	deals := []map[string]interface{}{}
	for _, deal := range ds {
		var c Company
		var name string
		if dbConn.Model(&deal).Related(&c).Error != nil {
			name = deal.Name
		} else {
			name = c.Name
		}
		// Get current number of investors and shareholders
		var dshs []DealShareholder
		if dbConn.Model(&deal).Related(&dshs).Error != nil {
			continue
		}
		var dis []DealInvestor
		if dbConn.Model(&deal).Related(&dis).Error != nil {
			continue
		}
		deals = append(deals, map[string]interface{}{
			"id":            deal.ID,
			"company_id":    deal.CompanyID,
			"company_name":  name,
			"deal_state":    deal.DealState,
			"start_date":    unixTime(deal.StartDate),
			"end_date":      unixTime(deal.EndDate),
			"shares_amount": deal.SharesAmount,
			"shares_left":   deal.SharesLeft,
			"buys":          len(dis),
			"sells":         len(dshs),
		})
	}

	// Return all satisfied deals
	saveAdmin(w, r, ps, u, map[string]interface{}{"deals": deals})
}

func adminDealIdHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	var c Company
	var name string
	if dbConn.Model(&deal).Related(&c).Error != nil {
		name = deal.Name
	} else {
		name = c.Name
	}

	// Return all allowed information about deal
	saveAdmin(w, r, ps, u, map[string]interface{}{
		"company_id":        deal.CompanyID,
		"company_name":      name,
		"deal_state":        deal.DealState,
		"deal_special":      deal.DealSpecial,
		"start_date":        unixTime(deal.StartDate),
		"end_date":          unixTime(deal.EndDate),
		"fund_num":          deal.FundNum,
		"actual_price":      deal.ActualPrice,
		"actual_valuation":  deal.ActualValuation,
		"shares_amount":     deal.SharesAmount,
		"shares_left":       deal.SharesLeft,
		"shares_type":       deal.SharesType,
		"escrow_account":    deal.EscrowAccount,
		"escrow_account_cn": deal.EscrowAccountCn,
	})
}

func adminDealUpdateHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Change deal-specific states
	dealState, ok := CheckRange(true, r.FormValue("deal_state"),
		DealStateClosed)
	if ok {
		deal.DealState = dealState
	}
	dealSpecial, ok := CheckRange(true, r.FormValue("deal_special"),
		DealSpecialTwentyPercentOff)
	if ok {
		deal.DealSpecial = dealSpecial
	}
	startDate, ok := CheckRange(true, r.FormValue("start_date"),
		TimeMax)
	if ok {
		deal.StartDate = time.Unix(int64(startDate), 0)
	}
	endDate, ok := CheckRange(true, r.FormValue("end_date"),
		TimeMax)
	if ok {
		deal.EndDate = time.Unix(int64(endDate), 0)
	}
	fundNum, ok := CheckRange(true, r.FormValue("fund_num"),
		NumberMax)
	if ok {
		deal.FundNum = fundNum
	}
	actualPrice, ok := CheckFloat(true, r.FormValue("actual_price"),
		true)
	if ok {
		deal.ActualPrice = actualPrice
	}
	actualValuation, ok := CheckFloat(true, r.FormValue("actual_valuation"),
		true)
	if ok {
		deal.ActualValuation = actualValuation
	}
	sharesAmount, ok := CheckRange(true, r.FormValue("shares_amount"),
		NumberMax)
	if ok {
		deal.SharesAmount = sharesAmount
	}
	sharesLeft, ok := CheckRange(true, r.FormValue("shares_left"),
		NumberMax)
	if ok {
		deal.SharesLeft = sharesLeft
	}
	sharesType, ok := CheckRange(true, r.FormValue("shares_type"),
		DealSharesTypeCommon)
	if ok {
		deal.SharesType = sharesType
	}
	ea, ok := CheckField(true, r.FormValue("escrow_account"))
	if ok {
		deal.EscrowAccount = ea
	}
	eac, ok := CheckField(true, r.FormValue("escrow_account_cn"))
	if ok {
		deal.EscrowAccountCn = eac
	}

	// Save however many changed
	if dbConn.Save(&deal).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminDealAddHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// All fields must be present
	cid, of := CheckRangeForm("", r, "company_id", NumberMax)
	dealState, of := CheckRangeForm(of, r, "deal_state",
		DealStateClosed)
	dealSpecial, of := CheckRangeForm(of, r, "deal_special",
		DealSpecialTwentyPercentOff)
	startDate, of := CheckRangeForm(of, r, "start_date",
		TimeMax)
	endDate, of := CheckRangeForm(of, r, "end_date",
		TimeMax)
	fundNum, of := CheckRangeForm(of, r, "fund_num",
		NumberMax)
	actualPrice, of := CheckFloatForm(of, r, "actual_price",
		true)
	actualValuation, of := CheckFloatForm(of, r, "actual_valuation",
		true)
	sharesAmount, of := CheckRangeForm(of, r, "shares_amount",
		NumberMax)
	sharesLeft, of := CheckRangeForm(of, r, "shares_left",
		NumberMax)
	sharesType, of := CheckRangeForm(of, r, "shares_type",
		DealSharesTypeCommon)
	ea, ofea := CheckFieldForm("", r, "escrow_account")
	eac, ofeac := CheckFieldForm("", r, "escrow_account_cn")

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	var c Company
	if dbConn.First(&c, cid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	// Fill in default escrow account template for faster update
	if ofea != "" {
		ea = fmt.Sprintf(WireTransferInfos["en-US"], c.Name,
			formatRoman(fundNum))
	}
	if ofeac != "" {
		eac = fmt.Sprintf(WireTransferInfos["zh-CN"], c.Name,
			formatRoman(fundNum))
	}

	// Create new deal
	deal := Deal{
		CompanyID:       uint64(c.ID),
		DealState:       dealState,
		DealSpecial:     dealSpecial,
		StartDate:       time.Unix(int64(startDate), 0),
		EndDate:         time.Unix(int64(endDate), 0),
		FundNum:         fundNum,
		ActualPrice:     actualPrice,
		ActualValuation: actualValuation,
		SharesAmount:    sharesAmount,
		SharesLeft:      sharesLeft,
		SharesType:      sharesType,
		EscrowAccount:   ea,
		EscrowAccountCn: eac,
	}

	// Add to current company
	if dbConn.Model(&c).Association("Deals").Append(&deal).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, map[string]interface{}{"id": deal.ID})
}

func adminDealDeleteHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	if dbConn.Delete(&Deal{}, did).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Success
	saveAdmin(w, r, ps, u, nil)
}

func adminDealSellsHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Both fields are optional with defaults to 0
	pnum, _ := CheckRange(true, r.FormValue("page_number"), NumberMax)
	psize, _ := CheckRange(true, r.FormValue("page_size"), NumberMax)
	// Must be a valid page size
	psize, ok := PageSizes[psize]
	if !ok {
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	var dshs []DealShareholder
	if dbConn.Joins("join users on users.id = deal_shareholders.user_id " +
		"and users.full_name ~* ?", r.FormValue("keyword")).
		Offset(int(pnum * psize)).Limit(int(psize)).
		Find(&dshs, "deal_id = ?", did).Error != nil {
		// Probably bad inputs
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	sells := []map[string]interface{}{}
	for _, dsh := range dshs {
		var user User
		// Skip if problem
		if dbConn.Model(&dsh).Related(&user).Error != nil {
			continue
		}
		sells = append(sells, map[string]interface{}{
			"id":                     dsh.ID,
			"user_id":                user.ID,
			"user_name":              user.FullName,
			"deal_shareholder_state": dsh.DealShareholderState,
			"amount":                 dsh.SharesSellAmount})
	}

	// Return all satisfied sells
	saveAdmin(w, r, ps, u, map[string]interface{}{"sells": sells})
}

func adminDealSellHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	dsid, err := strconv.ParseUint(ps.ByName("sell_id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}

	// Check if deal shareholder is valid
	var dsh DealShareholder
	if dbConn.First(&dsh, dsid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}
	if dsh.DealID != uint64(deal.ID) {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}

	// Generate engagement letter tokens if user hasn't already
	if uint64(len(dsh.EngagementLetterToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", deal.ID),
			fmt.Sprintf("%v", dsh.UserID), "sell_engagement_letter")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			dsh.EngagementLetterToken = fmt.Sprintf("%x", b)
			// Ignore saving problems
			dbConn.Save(dsh)
		}
	}

	// Find related offer
	var offers []Offer
	if dbConn.Model(&deal).Related(&offers).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealSellNoSuchOffer, true, nil)
		return
	}

	// Make dummy structs so we always return
	var user User
	var b Bank
	var o Offer
	for _, offer := range offers {
		if offer.UserID != dsh.UserID {
			continue
		}

		// Continue even if no user information (because admin)
		dbConn.Model(&dsh).Related(&user)

		// Continue even if no bank information (because admin)
		dbConn.Model(&dsh).Related(&b)

		o = offer

		// If user does not have valid file tokens yet, generate them
		gened := false
		if uint64(len(o.ShareCertificateToken)) != TokenMinMax {
			pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
				fmt.Sprintf("%v", o.UserID), "share_certificate")
			if _, err := os.Stat(pifc); err == nil {
				// Conform naming with correct type
				o.ShareCertificateToken, o.ShareCertificateName =
					createFileTokenName("share_certificate",
						o.ShareCertificateType)
				gened = true
			}
		}
		if uint64(len(o.CompanyByLawsToken)) != TokenMinMax {
			pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
				fmt.Sprintf("%v", o.UserID), "company_by_laws")
			if _, err := os.Stat(pifc); err == nil {
				// Conform naming with correct type
				o.CompanyByLawsToken, o.CompanyByLawsName =
					createFileTokenName("company_by_laws",
						o.CompanyByLawsType)
				gened = true
			}
		}
		if uint64(len(o.ShareholderAgreementToken)) != TokenMinMax {
			pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
				fmt.Sprintf("%v", o.UserID), "shareholder_agreement")
			if _, err := os.Stat(pifc); err == nil {
				// Conform naming with correct type
				o.ShareholderAgreementToken, o.ShareholderAgreementName =
					createFileTokenName("shareholder_agreement",
						o.ShareholderAgreementType)
				gened = true
			}
		}
		if uint64(len(o.StockOptionPlanToken)) != TokenMinMax {
			pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
				fmt.Sprintf("%v", o.UserID), "stock_option_plan")
			if _, err := os.Stat(pifc); err == nil {
				// Conform naming with correct type
				o.StockOptionPlanToken, o.StockOptionPlanName =
					createFileTokenName("stock_option_plan",
						o.StockOptionPlanType)
				gened = true
			}
		}
		// Save if generated new tokens
		if gened {
			dbConn.Save(&o)
		}

		break
	}

	// Return all shareholder fields
	saveAdmin(w, r, ps, u, map[string]interface{}{
		"user_id":                        user.ID,
		"user_name":                      user.FullName,
		"deal_shareholder_state":         dsh.DealShareholderState,
		"own_type":                       o.OwnType,
		"vested":                         o.Vested,
		"restrictions":                   o.Restrictions,
		"shares_total_own":               o.SharesTotalOwn,
		"stock_type":                     o.StockType,
		"exercise_date":                  o.ExerciseDate,
		"exercise_price":                 o.ExercisePrice,
		"shares_to_sell":                 o.SharesToSell,
		"desire_price":                   o.DesirePrice,
		"share_certificate":              o.ShareCertificateToken,
		"share_certificate_filename":     o.ShareCertificateName,
		"company_by_laws":                o.CompanyByLawsToken,
		"company_by_laws_filename":       o.CompanyByLawsName,
		"shareholder_agreement":          o.ShareholderAgreementToken,
		"shareholder_agreement_filename": o.ShareholderAgreementName,
		"stock_option_plan":              o.StockOptionPlanToken,
		"stock_option_plan_filename":     o.StockOptionPlanName,
		"full_name":                      b.FullName,
		"nick_name":                      b.NickName,
		"routing_number":                 decField(b.RoutingNumberEncrypted),
		"account_number":                 decField(b.AccountNumberEncrypted),
		"account_type":                   b.AccountType,
		"engagement_letter":              dsh.EngagementLetterToken,
	})
}

// adminDealSellGetOffer checks a specific file related to selling
// and returns the offer or nil on error
func adminDealSellGetOffer(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) *Offer {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		return nil
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		return nil
	}

	dsid, err := strconv.ParseUint(ps.ByName("sell_id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return nil
	}

	// Check if deal shareholder is valid
	var dsh DealShareholder
	if dbConn.First(&dsh, dsid).RecordNotFound() {
		return nil
	}
	if dsh.DealID != uint64(deal.ID) {
		return nil
	}

	// Find related offer
	var offers []Offer
	if dbConn.Model(&deal).Related(&offers).Error != nil {
		return nil
	}

	// Find the offer to return or none
	for _, offer := range offers {
		if offer.UserID != dsh.UserID {
			continue
		}

		return &offer
	}

	return nil
}

func adminDealSellShareCertificateHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	o := adminDealSellGetOffer(w, r, ps)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "share_certificate")
		// Found offer, return file
		parseFileDownload(w, r, o.ShareCertificateType, pifc)
	}
}

// Although different domains, they check identical params
func adminDealSellShareCertificateTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealSellShareCertificateTokenHandler(w, r, ps)
}

func adminDealSellCompanyByLawsHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	o := adminDealSellGetOffer(w, r, ps)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "company_by_laws")
		// Found offer, return file
		parseFileDownload(w, r, o.CompanyByLawsType, pifc)
	}
}

// Although different domains, they check identical params
func adminDealSellCompanyByLawsTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealSellCompanyByLawsTokenHandler(w, r, ps)
}

func adminDealSellShareholderAgreementHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	o := adminDealSellGetOffer(w, r, ps)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "shareholder_agreement")
		// Found offer, return file
		parseFileDownload(w, r, o.ShareholderAgreementType, pifc)
	}
}

// Although different domains, they check identical params
func adminDealSellShareholderAgreementTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealSellShareholderAgreementTokenHandler(w, r, ps)
}

func adminDealSellStockOptionPlanHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	o := adminDealSellGetOffer(w, r, ps)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "stock_option_plan")
		// Found offer, return file
		parseFileDownload(w, r, o.StockOptionPlanType, pifc)
	}
}

// Although different domains, they check identical params
func adminDealSellStockOptionPlanTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealSellStockOptionPlanTokenHandler(w, r, ps)
}

// adminDealSellDocumentDownloadHandler returns the signed document pdf file
// if available during the seller stage
// "name" is the document name saved an
func adminDealSellDocumentDownloadHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, name string) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		return
	}

	dsid, err := strconv.ParseUint(ps.ByName("sell_id"), 10, 64)
	if err != nil {
		return
	}

	// Check if deal shareholder is valid
	var dsh DealShareholder
	if dbConn.First(&dsh, dsid).RecordNotFound() {
		return
	}
	if dsh.DealID != uint64(deal.ID) {
		return
	}

	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", dsh.DealID),
		fmt.Sprintf("%v", dsh.UserID), name)
	// Always in pdf format
	parseFileDownload(w, r, "application/pdf", pifc)
}

func adminDealSellEngagementLetterHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	adminDealSellDocumentDownloadHandler(w, r, ps, "sell_engagement_letter")
}

// Although different domains, they check identical params
func adminDealSellEngagementLetterTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealSellEngagementLetterTokenHandler(w, r, ps)
}

func adminDealSellUpdateHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	dsid, err := strconv.ParseUint(ps.ByName("sell_id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}

	// Check if deal shareholder is valid
	var dsh DealShareholder
	if dbConn.First(&dsh, dsid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}
	if dsh.DealID != uint64(deal.ID) {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}

	// Find related user
	var user User
	if dbConn.Model(&dsh).Related(&user).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealUserUnknown, true, nil)
		return
	}

	// Check state change
	dshState, ok := CheckRange(true,
		r.FormValue("deal_shareholder_state"), DealShareholderStateDealClosed)
	if ok {
		// Do not save here, wait for the aggregated update
		dsh.DealShareholderState = dshState
	}

	// Find related offer
	var offers []Offer
	if dbConn.Model(&deal).Related(&offers).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealSellNoSuchOffer, true, nil)
		return
	}

	var o *Offer
	newOffer := true
	for _, offer := range offers {
		if offer.UserID != dsh.UserID {
			continue
		}
		// Found existing, use that
		o = &offer
		newOffer = false
		break
	}
	// Create new if necessary
	if newOffer {
		// Fill in required fields
		o = &Offer{
			UserID:    dsh.UserID,
			CompanyID: deal.CompanyID,
			DealID:    uint64(deal.ID),
		}
	}

	// If no offer related updates, do not save anything
	offerUpdated := false

	// Selective stock type updates
	ownType, ok := CheckRange(true, r.FormValue("own_type"),
		OwnTypeSharesRsuOptions)
	if ok {
		o.OwnType = ownType
		offerUpdated = true
	}
	vested, ok := CheckRange(true, r.FormValue("vested"), NoYesMax)
	if ok {
		o.Vested = vested
		offerUpdated = true
	}
	restrictions, ok := CheckRange(true, r.FormValue("restrictions"),
		NoYesMax)
	if ok {
		o.Restrictions = restrictions
		offerUpdated = true
	}
	sharesTotalOwn, ok := CheckRange(true, r.FormValue("shares_total_own"),
		NumberMax)
	if ok {
		o.SharesTotalOwn = sharesTotalOwn
		offerUpdated = true
	}
	stockType, ok := CheckRange(true, r.FormValue("stock_type"),
		StockTypeOther)
	if ok {
		o.StockType = stockType
		offerUpdated = true
	}
	exerciseDate, ok := CheckDate(true, r.FormValue("exercise_date"))
	if ok {
		o.ExerciseDate = exerciseDate
		offerUpdated = true
	}
	exercisePrice, ok := CheckFloat(true, r.FormValue("exercise_price"), true)
	if ok {
		o.ExercisePrice = exercisePrice
		offerUpdated = true
	}
	sharesToSell, ok := CheckRange(true, r.FormValue("shares_to_sell"),
		NumberMax)
	if ok {
		o.SharesToSell = sharesToSell
		offerUpdated = true
		// Link the amount
		dsh.SharesSellAmount = uint64(float64(sharesToSell) * deal.ActualPrice)
	}
	desirePrice, ok := CheckFloat(true, r.FormValue("desire_price"), true)
	if ok {
		o.DesirePrice = desirePrice
		offerUpdated = true
	}

	// Process files (also selectively)
	dealDir := path.Join(fmt.Sprintf("%v", deal.ID),
		fmt.Sprintf("%v", user.ID))
	furl, _, ftype, err := parseFileUpload(r, "share_certificate",
		"deal", dealDir, "share_certificate")
	if err == nil {
		o.ShareCertificateDoc = furl
		o.ShareCertificateType = ftype
		// Conform naming with correct type
		o.ShareCertificateToken, o.ShareCertificateName =
			createFileTokenName("share_certificate", ftype)
		offerUpdated = true
	}
	furl, _, ftype, err = parseFileUpload(r, "company_by_laws",
		"deal", dealDir, "company_by_laws")
	if err == nil {
		o.CompanyByLawsDoc = furl
		o.CompanyByLawsType = ftype
		// Conform naming with correct type
		o.CompanyByLawsToken, o.CompanyByLawsName =
			createFileTokenName("company_by_laws", ftype)
		offerUpdated = true
	}
	furl, _, ftype, err = parseFileUpload(r, "shareholder_agreement",
		"deal", dealDir, "shareholder_agreement")
	if err == nil {
		o.ShareholderAgreementDoc = furl
		o.ShareholderAgreementType = ftype
		// Conform naming with correct type
		o.ShareholderAgreementToken, o.ShareholderAgreementName =
			createFileTokenName("shareholder_agreement", ftype)
		offerUpdated = true
	}
	furl, _, ftype, err = parseFileUpload(r, "stock_option_plan",
		"deal", dealDir, "stock_option_plan")
	if err == nil {
		o.StockOptionPlanDoc = furl
		o.StockOptionPlanType = ftype
		// Conform naming with correct type
		o.StockOptionPlanToken, o.StockOptionPlanName =
			createFileTokenName("stock_option_plan", ftype)
		offerUpdated = true
	}

	if offerUpdated && newOffer {
		// The following transaction should succeed together
		tx := dbConn.Begin()

		// Add to users first
		if tx.Model(&user).Association("Offers").Append(o).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Then to companies
		var c Company
		if tx.First(&c, deal.CompanyID).RecordNotFound() {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
		if tx.Model(&c).Association("Offers").Append(o).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Then to current deal
		if tx.Model(&deal).Association("Offers").Append(o).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Add to updates as well
		if tx.Save(&dsh).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Finalize
		tx.Commit()
	} else {
		// Not new offer
		if offerUpdated && dbConn.Save(o).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Save deal shareholder if necessary
		if dbConn.Save(&dsh).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	}

	// Save bank info - separate from above
	var b *Bank
	var ob Bank
	// Check existing bank, does not matter since we can create new
	if !dbConn.Model(&dsh).Related(&ob).RecordNotFound() {
		b = &ob
	}

	// Routing + account must be changed in pairs since we cannot
	// override the Bank struct information (only create new or reusing)
	routingNumber, newBank := CheckField(true, r.FormValue("routing_number"))
	accountNumber, newBank := CheckField(newBank,
		r.FormValue("account_number"))
	// Select fields to update
	fullName, fnok := CheckField(true, r.FormValue("full_name"))
	nickName, nnok := CheckField(true, r.FormValue("nick_name"))
	accountType, atok := CheckRange(true, r.FormValue("account_type"),
		AccountTypeSaving)

	// Start new routine
	updateBank := false
	if newBank {
		// Check if bank already exists
		var banks []Bank
		if dbConn.Model(&user).Related(&banks).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
		// Reset from original
		b = nil
		// One user can only set the same bank once
		for _, bank := range banks {
			if decField(bank.RoutingNumberEncrypted) == routingNumber &&
				decField(bank.AccountNumberEncrypted) == accountNumber {
				b = &bank
				break
			}
		}
		// Not in database, insert first
		if b == nil {
			// Defaults are OK even if no input from user
			b = &Bank{
				UserID:                 uint64(user.ID),
				FullName:               fullName,
				NickName:               nickName,
				AccountType:            accountType,
				RoutingNumberEncrypted: encField(routingNumber),
				AccountNumberEncrypted: encField(accountNumber)}
			if dbConn.Model(&user).Association("Banks").
				Append(b).Error != nil {
				formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
				return
			}
		} else {
			updateBank = true
		}

		// Create new link
		if dbConn.Model(b).Association("DealShareholders").
			Append(&dsh).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	} else if b != nil {
		updateBank = true
	}

	// Both new and old need to update info
	if updateBank {
		if fnok {
			b.FullName = fullName
		}
		if nnok {
			b.NickName = nickName
		}
		if atok {
			b.AccountType = accountType
		}
		if dbConn.Save(b).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminDealSellDeleteHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	dsid, err := strconv.ParseUint(ps.ByName("sell_id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}

	// Check if deal shareholder is valid
	var dsh DealShareholder
	if dbConn.First(&dsh, dsid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}
	if dsh.DealID != uint64(deal.ID) {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}

	// Also clears the previously unlinked
	if dbConn.Delete(Offer{}, "user_id = ? and deal_id = ?",
		dsh.UserID, dsh.DealID).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOfferUnknown, true, nil)
		return
	}
	if dbConn.Delete(&dsh).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealShareholderUnknown, true, nil)
		return
	}

	// Success
	saveAdmin(w, r, ps, u, nil)
}

func adminDealBuysHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Both fields are optional with defaults to 0
	pnum, _ := CheckRange(true, r.FormValue("page_number"), NumberMax)
	psize, _ := CheckRange(true, r.FormValue("page_size"), NumberMax)
	// Must be a valid page size
	psize, ok := PageSizes[psize]
	if !ok {
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	var dis []DealInvestor
	if dbConn.Joins("join users on users.id = deal_investors.user_id " +
		"and users.full_name ~* ?", r.FormValue("keyword")).
		Offset(int(pnum * psize)).Limit(int(psize)).
		Find(&dis, "deal_id = ?", did).Error != nil {
		// Probably bad inputs
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	buys := []map[string]interface{}{}
	for _, di := range dis {
		var user User
		// Skip if problem
		if dbConn.Model(&di).Related(&user).Error != nil {
			continue
		}
		buys = append(buys, map[string]interface{}{
			"id":                  di.ID,
			"user_id":             user.ID,
			"user_name":           user.FullName,
			"deal_investor_state": di.DealInvestorState,
			"amount":              di.SharesBuyAmount})
	}

	// Return all satisfied buys
	saveAdmin(w, r, ps, u, map[string]interface{}{"buys": buys})
}

func adminDealBuyHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	dbid, err := strconv.ParseUint(ps.ByName("buy_id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}

	// Check if deal investor is valid
	var di DealInvestor
	if dbConn.First(&di, dbid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}
	if di.DealID != uint64(deal.ID) {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}

	// Generate signed document tokens if user hasn't already
	gened := false
	if uint64(len(di.EngagementLetterToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", deal.ID),
			fmt.Sprintf("%v", di.UserID), "buy_engagement_letter")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.EngagementLetterToken = fmt.Sprintf("%x", b)
			gened = true
		}
	}
	if uint64(len(di.SummaryOfTermsToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", deal.ID),
			fmt.Sprintf("%v", di.UserID), "summary_of_terms")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.SummaryOfTermsToken = fmt.Sprintf("%x", b)
			gened = true
		}
	}
	if uint64(len(di.DePpmToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", deal.ID),
			fmt.Sprintf("%v", di.UserID), "de_ppm")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.DePpmToken = fmt.Sprintf("%x", b)
			gened = true
		}
	}
	if uint64(len(di.DeOperatingAgreementToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", deal.ID),
			fmt.Sprintf("%v", di.UserID), "de_operating_agreement")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.DeOperatingAgreementToken = fmt.Sprintf("%x", b)
			gened = true
		}
	}
	if uint64(len(di.DeSubscriptionAgreementToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", deal.ID),
			fmt.Sprintf("%v", di.UserID), "de_subscription_agreement")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.DeSubscriptionAgreementToken = fmt.Sprintf("%x", b)
			gened = true
		}
	}
	if gened {
		// Ignore saving problems
		dbConn.Save(di)
	}

	// Make dummy structs so we always return
	var user User
	dbConn.Model(&di).Related(&user)
	var b Bank
	dbConn.Model(&di).Related(&b)

	// Return all investor fields
	saveAdmin(w, r, ps, u, map[string]interface{}{
		"user_id":             user.ID,
		"user_name":           user.FullName,
		"deal_investor_state": di.DealInvestorState,
		"shares_buy_amount":   di.SharesBuyAmount,
		"full_name":           b.FullName,
		"nick_name":           b.NickName,
		"routing_number":      decField(b.RoutingNumberEncrypted),
		"account_number":      decField(b.AccountNumberEncrypted),
		"account_type":        b.AccountType,
		"engagement_letter":   di.EngagementLetterToken,
		"summary_terms":       di.SummaryOfTermsToken,
		"de_ppm":              di.DePpmToken,
		"de_operating":        di.DeOperatingAgreementToken,
		"de_subscription":     di.DeSubscriptionAgreementToken,
	})
}

// adminDealBuyDocumentDownloadHandler returns the signed document pdf file
// if available during the buyer stage
// "name" is the document name saved an
func adminDealBuyDocumentDownloadHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, name string) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		return
	}

	dbid, err := strconv.ParseUint(ps.ByName("buy_id"), 10, 64)
	if err != nil {
		return
	}

	// Check if deal investor is valid
	var di DealInvestor
	if dbConn.First(&di, dbid).RecordNotFound() {
		return
	}
	if di.DealID != uint64(deal.ID) {
		return
	}

	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", di.DealID),
		fmt.Sprintf("%v", di.UserID), name)
	// Always in pdf format
	parseFileDownload(w, r, "application/pdf", pifc)
}

func adminDealBuyEngagementLetterHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	adminDealBuyDocumentDownloadHandler(w, r, ps, "buy_engagement_letter")
}

// Although different domains, they check identical params
func adminDealBuyEngagementLetterTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyEngagementLetterTokenHandler(w, r, ps)
}

func adminDealBuySummaryTermsHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	adminDealBuyDocumentDownloadHandler(w, r, ps, "summary_of_terms")
}

// Although different domains, they check identical params
func adminDealBuySummaryTermsTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuySummaryTermsTokenHandler(w, r, ps)
}

func adminDealBuyDePpmHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	adminDealBuyDocumentDownloadHandler(w, r, ps, "de_ppm")
}

// Although different domains, they check identical params
func adminDealBuyDePpmTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDePpmTokenHandler(w, r, ps)
}

func adminDealBuyDeOperatingHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	adminDealBuyDocumentDownloadHandler(w, r, ps, "de_operating_agreement")
}

// Although different domains, they check identical params
func adminDealBuyDeOperatingTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDeOperatingTokenHandler(w, r, ps)
}

func adminDealBuyDeSubscriptionHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	adminDealBuyDocumentDownloadHandler(w, r, ps, "de_subscription_agreement")
}

// Although different domains, they check identical params
func adminDealBuyDeSubscriptionTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDeSubscriptionTokenHandler(w, r, ps)
}

func adminDealBuyUpdateHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	dbid, err := strconv.ParseUint(ps.ByName("buy_id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}

	// Check if deal investor is valid
	var di DealInvestor
	if dbConn.First(&di, dbid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}
	if di.DealID != uint64(deal.ID) {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}

	// Find related user
	var user User
	if dbConn.Model(&di).Related(&user).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealUserUnknown, true, nil)
		return
	}

	diState, ok := CheckRange(true, r.FormValue("deal_investor_state"),
		DealInvestorStateDealClosed)
	dwt := false
	if ok {
		dwt = diState != di.DealInvestorState &&
			diState == DealInvestorStateWaitingFundTransfer
		di.DealInvestorState = diState
	}
	sharesBuyAmount, ok := CheckRange(true,
		r.FormValue("shares_buy_amount"), NumberMax)
	if ok {
		di.SharesBuyAmount = sharesBuyAmount
	}

	if dbConn.Save(&di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	if dwt {
		// Notify user of the account information
		if user.LastLanguage != "en-US" &&
			user.LastLanguage != "zh-CN" {
			user.LastLanguage = "en-US"
		}
		author := EmailTexts[user.LastLanguage][EmailTextName]
		subject :=
			EmailTexts[user.
			LastLanguage][EmailTextSubjectWireTransferAccount]
		var c Company
		var name string
		if dbConn.Model(&deal).Related(&c).Error != nil {
			name = deal.Name
		} else {
			name = c.Name
		}
		body := fmt.Sprintf(
			EmailTexts[user.
			LastLanguage][EmailTextBodyWireTransferAccount],
			user.FullName, name, formatRoman(deal.FundNum))
		go sendMail(author, user.Email, user.FullName, subject, body)
	}

	// Save bank info - separate from above
	var b *Bank
	var ob Bank
	// Check existing bank, does not matter since we can create new
	if !dbConn.Model(&di).Related(&ob).RecordNotFound() {
		b = &ob
	}

	// Routing + account must be changed in pairs since we cannot
	// override the Bank struct information (only create new or reusing)
	routingNumber, newBank := CheckField(true, r.FormValue("routing_number"))
	accountNumber, newBank := CheckField(newBank,
		r.FormValue("account_number"))
	// Select fields to update
	fullName, fnok := CheckField(true, r.FormValue("full_name"))
	nickName, nnok := CheckField(true, r.FormValue("nick_name"))
	accountType, atok := CheckRange(true, r.FormValue("account_type"),
		AccountTypeSaving)

	// Start new routine
	updateBank := false
	if newBank {
		// Check if bank already exists
		var banks []Bank
		if dbConn.Model(&user).Related(&banks).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
		// Reset from original
		b = nil
		// One user can only set the same bank once
		for _, bank := range banks {
			if decField(bank.RoutingNumberEncrypted) == routingNumber &&
				decField(bank.AccountNumberEncrypted) == accountNumber {
				b = &bank
				break
			}
		}
		// Not in database, insert first
		if b == nil {
			// Defaults are OK even if no input from user
			b = &Bank{
				UserID:                 uint64(user.ID),
				FullName:               fullName,
				NickName:               nickName,
				AccountType:            accountType,
				RoutingNumberEncrypted: encField(routingNumber),
				AccountNumberEncrypted: encField(accountNumber)}
			if dbConn.Model(&user).Association("Banks").
				Append(b).Error != nil {
				formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
				return
			}
		} else {
			updateBank = true
		}

		// Create new link
		if dbConn.Model(b).Association("DealInvestors").
			Append(&di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	} else if b != nil {
		updateBank = true
	}

	// Both new and old need to update info
	if updateBank {
		if fnok {
			b.FullName = fullName
		}
		if nnok {
			b.NickName = nickName
		}
		if atok {
			b.AccountType = accountType
		}
		if dbConn.Save(b).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminDealBuyDeleteHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	// Check if deal is valid
	var deal Deal
	if dbConn.First(&deal, did).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}

	dbid, err := strconv.ParseUint(ps.ByName("buy_id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}

	// Check if deal investor is valid
	var di DealInvestor
	if dbConn.First(&di, dbid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}
	if di.DealID != uint64(deal.ID) {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}

	if dbConn.Delete(&di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealInvestorUnknown, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, nil)
}
