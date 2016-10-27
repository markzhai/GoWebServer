// Router branch for /deal/ operations
package main

import (
	crand "crypto/rand"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Primitives for docusign document checking
var (
	sellLock = &sync.Mutex{}
	sellChecks = map[uint64]bool{}
	buyLock = &sync.Mutex{}
	buyChecks = map[uint64]bool{}
)

func dealsHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// Make sure we don't read more information than user could
	// Special debug flag does it anyway
	if u.UserState < UserStateActive {
		formatReturn(w, r, ps, ErrorCodeUserNotOnboard, true, nil)
		return
	}

	// Do not blame user for our db mistakes :'(
	var openDeals []Deal
	dbConn.Find(&openDeals, "deal_state = ?", DealStateOpen)
	var previewDeals []Deal
	dbConn.Find(&previewDeals, "deal_state = ?", DealStatePreview)
	var closedDeals []Deal
	dbConn.Find(&closedDeals, "deal_state = ?", DealStateClosed)

	// Read language once
	reqLang := ps[len(ps) - 2].Value
	var processDeals = func(deals []Deal) []map[string]interface{} {
		ds := []map[string]interface{}{}
		for _, d := range deals {
			var c Company
			// Skip if problem
			if dbConn.Model(&d).Related(&c).Error != nil {
				continue
			}
			var tags []Tag
			if dbConn.Model(&c).Related(&tags, "Tags").Error != nil ||
				len(tags) < 3 {
				continue
			}
			var fundings []Funding
			if dbConn.Model(&c).Related(&fundings).Error != nil ||
				len(fundings) < 1 {
				continue
			}
			// Read deal user state
			var dshs []DealShareholder
			if dbConn.Model(&d).Related(&dshs).Error != nil {
				continue
			}
			var dis []DealInvestor
			if dbConn.Model(&d).Related(&dis).Error != nil {
				continue
			}
			dus := DealUserStateNone
			for _, dsh := range dshs {
				if dsh.UserID == uint64(u.ID) {
					dus = DealUserStateShareholder
					break
				}
			}
			for _, di := range dis {
				if di.UserID == uint64(u.ID) {
					dus = DealUserStateInvestor
					break
				}
			}
			// Ready to construct the results
			var tagss string
			if reqLang == "zh-CN" {
				tagss = tags[0].NameCn + "、" + tags[1].NameCn + "、" +
					tags[2].NameCn
			} else {
				tagss = tags[0].Name + ", " + tags[1].Name + ", " +
					tags[2].Name
			}
			ds = append(ds, map[string]interface{}{
				"id":               d.ID,
				"company_id":       d.CompanyID,
				"deal_user_state":  dus,
				"deal_special":     d.DealSpecial,
				"start_date":       unixTime(d.StartDate),
				"end_date":         unixTime(d.EndDate),
				"fund_num":         d.FundNum,
				"actual_price":     d.ActualPrice,
				"actual_valuation": d.ActualValuation,
				"shares_amount":    d.SharesAmount,
				"shares_left":      d.SharesLeft,
				"shares_type":      d.SharesType,
				"tags":             tagss,
				"name":             c.Name,
			})
		}
		return ds
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"open_deals":    processDeals(openDeals),
			"preview_deals": processDeals(previewDeals),
			"closed_deals":  processDeals(closedDeals)})
}

// dealCheckState makes sure the preliminary state is set
// Sell processes should have param sell=true
// If get param is true, do not fail if deal ops have not started,
// instead return the second part as "" to indicate
func dealCheckState(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User, sell bool,
ds int64, get bool, silent bool, open bool) (*Deal, interface{}, bool) {
	// Validate user state
	if (sell && u.UserState < UserStateActiveId) ||
		(!sell && u.UserState < UserStateActiveAccredId) {
		if !silent {
			formatReturn(w, r, ps, ErrorCodeDealError, true, nil)
		}
		return nil, nil, false
	}

	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		if !silent {
			formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		}
		return nil, nil, false
	}

	// Deal structure must be validated
	var d Deal
	if dbConn.First(&d, did).RecordNotFound() {
		if !silent {
			formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		}
		return nil, nil, false
	}
	// Deal must be live
	if open && d.DealState != DealStateOpen {
		if !silent {
			formatReturn(w, r, ps, ErrorCodeDealNotLive, true, nil)
		}
		return nil, nil, false
	}

	var dshs []DealShareholder
	if dbConn.Model(&d).Related(&dshs).Error != nil {
		if !silent {
			formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		}
		return nil, nil, false
	}
	var dis []DealInvestor
	if dbConn.Model(&d).Related(&dis).Error != nil {
		if !silent {
			formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		}
		return nil, nil, false
	}

	// Nothing has been created, so we want to make sure this user
	// has not done anything yet
	if ds < 0 {
		for _, dsh := range dshs {
			if dsh.UserID == uint64(u.ID) {
				if !silent {
					formatReturn(w, r, ps, ErrorCodeDealAlreadySelling,
						true, nil)
				}
				return nil, nil, false
			}
		}
		for _, di := range dis {
			if di.UserID == uint64(u.ID) {
				if !silent {
					formatReturn(w, r, ps, ErrorCodeDealAlreadyBuying,
						true, nil)
				}
				return nil, nil, false
			}
		}
		return &d, nil, true
	}

	// Now check if the x (sell or buy) state requirement is met
	if sell {
		for _, dsh := range dshs {
			if dsh.UserID == uint64(u.ID) {
				if dsh.DealShareholderState < uint64(ds) {
					if !silent {
						formatReturn(w, r, ps, ErrorCodeDealUserStateError,
							true, nil)
					}
					return nil, nil, false
				}
				return &d, &dsh, true
			}
		}
	} else {
		for _, di := range dis {
			if di.UserID == uint64(u.ID) {
				if di.DealInvestorState < uint64(ds) {
					if !silent {
						formatReturn(w, r, ps, ErrorCodeDealUserStateError,
							true, nil)
					}
					return nil, nil, false
				}
				return &d, &di, true
			}
		}
	}

	// Something is wrong, not found
	// Get operations are special, return "success"
	if get {
		return &d, "", false
	}

	if !silent {
		formatReturn(w, r, ps, ErrorCodeDealUserStateError, true, nil)
	}
	return nil, nil, false
}

func dealSellEngagementLetterSignHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	d, dshu, ok := dealCheckState(w, r, ps, u, true,
		DealShareholderStateEngagementStarted, true, false, true)
	var dsh *DealShareholder
	if !ok {
		// Already returned message
		if dshu == nil {
			return
		}
		// Create anew
	} else {
		// Existing shareholder
		dsh, ok = dshu.(*DealShareholder)
		if !ok {
			// Bug
			formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
			return
		}
	}

	// If starting anew or link expired request new
	uid := fmt.Sprintf("%v", u.ID)
	var envId string
	var envTime, envCheck time.Time
	if dsh == nil || dsh.EngagementLetterSignId == "" ||
		dsh.EngagementLetterSignExpire.Before(time.Now()) {
		var c Company
		if dbConn.Model(d).Related(&c).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignError, true, nil)
			return
		}
		address := u.Address1
		if u.Address2 != "" {
			address += " " + u.Address2
		}
		reqLang := ps[len(ps) - 2].Value
		eid, et, err := serverDocusign.CreateEnvelopeWithTemplate(
			docusignSellEngagementLetter,
			"client", uid, SignTexts[reqLang][SignSellEngagementLetter],
			u.Email, u.FullName,
			map[string]interface{}{
				"client_name": u.FullName,
				"address":     address,
				"shares_type": SharesTypeTexts["en-US"][d.SharesType],
				"company":     c.Name,
			})
		if err != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignEnvelopeError,
				true, nil)
			return
		}
		envId = eid
		envTime = et
		envCheck = time.Time{}
	} else {
		envId = dsh.EngagementLetterSignId
		envTime = dsh.EngagementLetterSignExpire
		envCheck = dsh.EngagementLetterSignCheck
	}

	// Now create embedded url
	url, err := serverDocusign.CreateEmbeddedRecipientUrl(r.Host,
		envId, uid, u.Email, u.FullName)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealDocusignRecipientError,
			true, nil)
		return
	}

	// If new, create the deal shareholder struct, add to users
	if dsh == nil {
		// The following transaction should succeed together
		tx := dbConn.Begin()

		// Create the deal shareholder struct, add to users
		dsh = &DealShareholder{
			UserID:                     uint64(u.ID),
			DealID:                     uint64(d.ID),
			DealShareholderState:       DealShareholderStateEngagementStarted,
			EngagementLetterSignId:     envId,
			EngagementLetterSignUrl:    url,
			EngagementLetterSignExpire: envTime,
			EngagementLetterSignCheck:  envCheck,
		}
		if tx.Model(u).Association("DealShareholders").Append(
			dsh).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Then add to deal
		if tx.Model(d).Association("DealShareholders").Append(
			dsh).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Finalize
		tx.Commit()
	} else {
		// If existing, update struct
		dsh.EngagementLetterSignId = envId
		dsh.EngagementLetterSignUrl = url
		dsh.EngagementLetterSignExpire = envTime
		dsh.EngagementLetterSignCheck = envCheck
		if dbConn.Save(dsh).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_shareholder_state": dsh.DealShareholderState,
			"docusign_sign_url":      dsh.EngagementLetterSignUrl,
			"docusign_sign_id":       dsh.EngagementLetterSignId,
		})
}

func dealSellEngagementLetterCheckHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	// Start docusign checking lock asap
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}
	sellLock.Lock()
	sellChecks[did] = true
	sellLock.Unlock()
	defer func() {
		sellLock.Lock()
		sellChecks[did] = false
		sellLock.Unlock()
	}()

	d, dshu, ok := dealCheckState(w, r, ps, u, true,
		DealShareholderStateEngagementStarted, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	dsh, ok := dshu.(*DealShareholder)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// Must have started signing and respects checking limit
	if dsh.EngagementLetterSignId == "" ||
		dsh.EngagementLetterSignExpire.Before(time.Now()) ||
		dsh.EngagementLetterSignCheck.After(time.Now()) {
		formatReturn(w, r, ps, ErrorCodeDealDocusignSignError,
			true, nil)
		return
	}

	// Check if document is completed
	eid := dsh.EngagementLetterSignId
	completed, terminal, err := serverDocusign.GetEnvelopeStatus(eid)
	if err != nil || !terminal {
		dsh.EngagementLetterSignCheck = time.Now().Add(docusignStatusDelay)
	} else {
		// Terminated, reset values
		dsh.EngagementLetterSignId = ""
		dsh.EngagementLetterSignUrl = ""
		dsh.EngagementLetterSignExpire = time.Time{}
		dsh.EngagementLetterSignCheck = time.Time{}
	}
	// At terminal state, reset current envelope
	if err != nil || !terminal || !completed {
		if dbConn.Save(dsh).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignStatusError,
				true, nil)
		}
		return
	}

	// Download file and save
	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
		fmt.Sprintf("%v", u.ID), "sell_engagement_letter")
	err = serverDocusign.DownloadEnvelopeDocument(eid, pifc)
	if err != nil {
		// Must save state on leave
		if dbConn.Save(dsh).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignDownloadError,
				true, nil)
		}
		return
	}

	// Create token for view
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	dsh.EngagementLetterToken = fmt.Sprintf("%x", b)
	dsh.DealShareholderState = DealShareholderStateEngagementLetterSigned
	if dbConn.Save(dsh).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_shareholder_state": dsh.DealShareholderState,
			"engagement_letter":      dsh.EngagementLetterToken})
}

func dealSellNewOfferHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, dshu, ok := dealCheckState(w, r, ps, u, true,
		DealShareholderStateEngagementLetterSigned, false, false, false)
	if !ok {
		// Already returned message
		return
	}
	dsh, ok := dshu.(*DealShareholder)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}
	// Either live or submitting user offer
	if d.DealState != DealStateOpen &&
		d.DealState != DealStateUserSubmitted {
		formatReturn(w, r, ps, ErrorCodeDealNotLiveOrSubmitted, true, nil)
		return
	}

	rets := map[string]interface{}{}
	if dsh.DealShareholderState < DealShareholderStateOfferCreated {
		// All required arguments must be present to create a new offer
		ownType, of := CheckRangeForm("", r, "own_type",
			OwnTypeSharesRsuOptions)
		vested, of := CheckRangeForm(of, r, "vested", NoYesMax)
		restrictions, of := CheckRangeForm(of, r, "restrictions",
			NoYesMax)
		sharesTotalOwn, of := CheckRangeForm(of, r, "shares_total_own",
			NumberMax)
		stockType, of := CheckRangeForm(of, r, "stock_type",
			StockTypeOther)
		exerciseDate, of := CheckDateForm(of, r, "exercise_date")
		exercisePrice, of := CheckFloatForm(of, r, "exercise_price",
			true)
		sharesToSell, of := CheckRangeForm(of, r, "shares_to_sell",
			NumberMax)
		desirePrice, of := CheckFloatForm(of, r, "desire_price", true)

		if of != "" {
			formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
			return
		}

		// Create a new offer with the most basic fields
		o := &Offer{
			UserID:         uint64(u.ID),
			CompanyID:      d.CompanyID,
			DealID:         uint64(d.ID),
			OwnType:        ownType,
			Vested:         vested,
			Restrictions:   restrictions,
			SharesTotalOwn: sharesTotalOwn,
			StockType:      stockType,
			ExerciseDate:   exerciseDate,
			ExercisePrice:  exercisePrice,
			SharesToSell:   sharesToSell,
			DesirePrice:    desirePrice,
		}

		// The following transaction should succeed together
		tx := dbConn.Begin()

		// Add to users first
		if tx.Model(u).Association("Offers").Append(o).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Then to companies if existing
		if d.DealState != DealStateUserSubmitted {
			var c Company
			if tx.First(&c, d.CompanyID).RecordNotFound() {
				tx.Rollback()
				formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
				return
			}
			if tx.Model(&c).Association("Offers").Append(o).Error != nil {
				tx.Rollback()
				formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
				return
			}
		}

		// Then to current deal
		if tx.Model(d).Association("Offers").Append(o).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Starting state
		dsh.DealShareholderState = DealShareholderStateOfferCreated
		dsh.SharesSellAmount = uint64(float64(sharesToSell) * d.ActualPrice)
		if tx.Save(dsh).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Finalize
		tx.Commit()
	} else if dsh.DealShareholderState < DealShareholderStateOfferCompleted {
		// The offer could be in "incomplete" state, waiting for more files
		var offers []Offer
		if dbConn.Model(d).Related(&offers).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealSellNoSuchOffer, true, nil)
			return
		}

		var o *Offer
		for _, offer := range offers {
			if offer.UserID == dsh.UserID {
				o = &offer
				break
			}
		}
		if o == nil {
			formatReturn(w, r, ps, ErrorCodeDealSellNoSuchOffer, true, nil)
			return
		}

		// Can only upload after offer is soft "created"
		dealDir := path.Join(fmt.Sprintf("%v", d.ID), fmt.Sprintf("%v", u.ID))
		furl, _, ftype, err := parseFileUpload(r, "share_certificate",
			"deal", dealDir, "share_certificate")
		if err == nil {
			o.ShareCertificateDoc = furl
			o.ShareCertificateType = ftype
			// Conform naming with correct type
			o.ShareCertificateToken, o.ShareCertificateName =
				createFileTokenName("share_certificate", ftype)
			rets["share_certificate"] = o.ShareCertificateToken
			rets["share_certificate_filename"] = o.ShareCertificateName
		}
		furl, _, ftype, err = parseFileUpload(r, "company_by_laws",
			"deal", dealDir, "company_by_laws")
		if err == nil {
			o.CompanyByLawsDoc = furl
			o.CompanyByLawsType = ftype
			// Conform naming with correct type
			o.CompanyByLawsToken, o.CompanyByLawsName =
				createFileTokenName("company_by_laws", ftype)
			rets["company_by_laws"] = o.CompanyByLawsToken
			rets["company_by_laws_filename"] = o.CompanyByLawsName
		}
		furl, _, ftype, err = parseFileUpload(r, "shareholder_agreement",
			"deal", dealDir, "shareholder_agreement")
		if err == nil {
			o.ShareholderAgreementDoc = furl
			o.ShareholderAgreementType = ftype
			// Conform naming with correct type
			o.ShareholderAgreementToken, o.ShareholderAgreementName =
				createFileTokenName("shareholder_agreement", ftype)
			rets["shareholder_agreement"] = o.ShareholderAgreementToken
			rets["shareholder_agreement_filename"] = o.ShareholderAgreementName
		}
		furl, _, ftype, err = parseFileUpload(r, "stock_option_plan",
			"deal", dealDir, "stock_option_plan")
		if err == nil {
			o.StockOptionPlanDoc = furl
			o.StockOptionPlanType = ftype
			// Conform naming with correct type
			o.StockOptionPlanToken, o.StockOptionPlanName =
				createFileTokenName("stock_option_plan", ftype)
			rets["stock_option_plan"] = o.StockOptionPlanToken
			rets["stock_option_plan_filename"] = o.StockOptionPlanName
		}

		// Save state anyway
		if dbConn.Save(o).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Check if all files are present
		if o.ShareCertificateName != "" && o.CompanyByLawsName != "" &&
			o.ShareholderAgreementName != "" && o.StockOptionPlanName != "" {
			dsh.DealShareholderState = DealShareholderStateOfferCompleted
			if dbConn.Save(dsh).Error != nil {
				formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
				return
			}
		}
	}

	rets["deal_shareholder_state"] = dsh.DealShareholderState
	saveLogin(w, r, ps, false, u, rets)
}

func dealNewSellNewOfferHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	cname, of := CheckFieldForm("", r, "company_name")

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	// The following transaction should succeed together
	tx := dbConn.Begin()

	// If there is a similarly-matching deal, then use it
	var deal Deal
	var d *Deal
	if tx.First(&deal, "deal_state = ? and lower(name) = ?",
		DealStateUserSubmitted, strings.ToLower(cname)).RecordNotFound() {
		// Create a new deal
		d = &Deal{
			DealState: DealStateUserSubmitted,
			Name:      cname,
		}
		if tx.Create(d).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	} else {
		d = &deal
		var dshs []DealShareholder
		if tx.Model(d).Related(&dshs).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
		for _, dsh := range dshs {
			if dsh.UserID == uint64(u.ID) {
				tx.Rollback()
				formatReturn(w, r, ps, ErrorCodeDealAlreadyOffering,
					true, nil)
				return
			}
		}
	}

	// Appear to be "signed", skipping due to dummy deal
	dsh := &DealShareholder{
		UserID:               uint64(u.ID),
		DealID:               uint64(d.ID),
		DealShareholderState: DealShareholderStateEngagementLetterSigned,
	}
	if tx.Model(u).Association("DealShareholders").Append(dsh).Error != nil {
		tx.Rollback()
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	// Then add to deal
	if tx.Model(d).Association("DealShareholders").Append(dsh).Error != nil {
		tx.Rollback()
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	// Finalize
	tx.Commit()

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_shareholder_state": dsh.DealShareholderState,
			"deal_id":                d.ID,
		})
}

func dealSellHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, dshu, ok := dealCheckState(w, r, ps, u, true,
		DealShareholderStateEngagementStarted, true, false, false)
	if !ok {
		if dshu == "" {
			saveLogin(w, r, ps, false, u,
				map[string]interface{}{"deal_shareholder_state": -1})
		}
		// Already returned message
		return
	}
	dsh, ok := dshu.(*DealShareholder)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}
	// Either live or submitting user offer
	if d.DealState != DealStateOpen &&
		d.DealState != DealStateUserSubmitted {
		formatReturn(w, r, ps, ErrorCodeDealNotLiveOrSubmitted, true, nil)
		return
	}

	// Construct necessary args and append optionals
	sellLock.Lock()
	// By default not locking
	dc := DocusignCheckingNo
	if sellChecks[uint64(d.ID)] {
		dc = DocusignCheckingYes
	}
	sellLock.Unlock()
	var c Company
	var name string
	if dbConn.Model(d).Related(&c).Error != nil {
		name = d.Name
	} else {
		name = c.Name
	}
	args := map[string]interface{}{
		"deal_shareholder_state": dsh.DealShareholderState,
		"docusign_checking":      dc,
		"company_name":           name,
	}
	// Do not waste resources when docusign checking is pending
	if dc == DocusignCheckingYes {
		saveLogin(w, r, ps, false, u, args)
		return
	}

	// Check if engagement letter has been signed
	if dsh.DealShareholderState >= DealShareholderStateEngagementLetterSigned {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
			fmt.Sprintf("%v", u.ID), "sell_engagement_letter")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			dsh.EngagementLetterToken = fmt.Sprintf("%x", b)
			// Ignore saving problems
			if dbConn.Save(dsh).Error == nil {
				args["engagement_letter"] = dsh.EngagementLetterToken
			}
		}
	}

	// Return just state if no offer has been created
	if dsh.DealShareholderState < DealShareholderStateOfferCreated {
		saveLogin(w, r, ps, false, u, args)
		return
	}

	// Find related offer
	var offers []Offer
	if dbConn.Model(d).Related(&offers).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealSellNoSuchOffer, true, nil)
		return
	}

	for _, o := range offers {
		if o.UserID != dsh.UserID {
			continue
		}

		args["own_type"] = o.OwnType
		args["vested"] = o.Vested
		args["restrictions"] = o.Restrictions
		args["shares_total_own"] = o.SharesTotalOwn
		args["stock_type"] = o.StockType
		args["exercise_date"] = o.ExerciseDate
		args["exercise_price"] = o.ExercisePrice
		args["shares_to_sell"] = o.SharesToSell
		args["desire_price"] = o.DesirePrice

		// Return files if available
		files := map[string]interface{}{}
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "share_certificate")
		if _, err := os.Stat(pifc); err == nil {
			o.ShareCertificateToken, _ =
				createFileTokenName("share_certificate", "")
			files["share_certificate"] = o.ShareCertificateToken
			files["share_certificate_filename"] = o.ShareCertificateName
		}
		pifc = path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "company_by_laws")
		if _, err := os.Stat(pifc); err == nil {
			o.CompanyByLawsToken, _ =
				createFileTokenName("company_by_laws", "")
			files["company_by_laws"] = o.CompanyByLawsToken
			files["company_by_laws_filename"] = o.CompanyByLawsName
		}
		pifc = path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "shareholder_agreement")
		if _, err := os.Stat(pifc); err == nil {
			o.ShareholderAgreementToken, _ =
				createFileTokenName("shareholder_agreement", "")
			files["shareholder_agreement"] = o.ShareholderAgreementToken
			files["shareholder_agreement_filename"] = o.ShareholderAgreementName
		}
		pifc = path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "stock_option_plan")
		if _, err := os.Stat(pifc); err == nil {
			o.StockOptionPlanToken, _ =
				createFileTokenName("stock_option_plan", "")
			files["stock_option_plan"] = o.StockOptionPlanToken
			files["stock_option_plan_filename"] = o.StockOptionPlanName
		}
		// Check and save
		if len(files) > 0 && dbConn.Save(&o).Error == nil {
			for k, v := range files {
				args[k] = v
			}
		}

		// Return bank information if filled
		if dsh.DealShareholderState >= DealShareholderStateBankInfoSubmitted {
			var b Bank
			if dbConn.Model(dsh).Related(&b).Error != nil {
				formatReturn(w, r, ps, ErrorCodeDealNoBankInfo, true, nil)
				return
			}
			args["full_name"] = b.FullName
			args["nick_name"] = b.NickName
			args["routing_number"] = decField(b.RoutingNumberEncrypted)
			args["account_number"] = decField(b.AccountNumberEncrypted)
			args["account_type"] = b.AccountType
		}

		saveLogin(w, r, ps, false, u, args)
		return
	}

	formatReturn(w, r, ps, ErrorCodeDealSellNoSuchOffer, true, nil)
}

// dealSellGetOffer checks a specific file related to selling and returns
// the offer or nil on error
func dealSellGetOffer(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) *Offer {
	d, dshu, ok := dealCheckState(w, r, ps, u, true,
		DealShareholderStateOfferCreated, true, true, false)
	if !ok {
		return nil
	}
	dsh, ok := dshu.(*DealShareholder)
	if !ok {
		return nil
	}
	// Either live or submitting user offer
	if d.DealState != DealStateOpen &&
		d.DealState != DealStateUserSubmitted {
		return nil
	}

	// Find related offer
	var offers []Offer
	if dbConn.Model(d).Related(&offers).Error != nil {
		return nil
	}

	for _, o := range offers {
		if o.UserID != dsh.UserID {
			continue
		}

		return &o
	}

	return nil
}

// dealSellTokenGetOffer checks a specific file related to selling and returns
// the offer or nil on error, without permission checking but relies on
// a previously-generated token
// tiname is the token index name in the offer
func dealSellTokenGetOffer(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, tiname string) *Offer {
	// Check token for offer
	token, ok := CheckLength(true, ps.ByName("token"),
		TokenMinMax, TokenMinMax)
	if !ok {
		return nil
	}

	// Check token validity
	var offer Offer
	if dbConn.First(&offer, tiname + " = ?", token).RecordNotFound() {
		return nil
	}

	return &offer
}

func dealSellShareCertificateHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	o := dealSellGetOffer(w, r, ps, u)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "share_certificate")
		// Found offer, return file
		parseFileDownload(w, r, o.ShareCertificateType, pifc)
	}
}

func dealSellShareCertificateTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	o := dealSellTokenGetOffer(w, r, ps, "share_certificate_token")
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "share_certificate")
		// Found offer, return file
		parseFileDownload(w, r, o.ShareCertificateType, pifc)
	}
}

func dealSellCompanyByLawsHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	o := dealSellGetOffer(w, r, ps, u)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "company_by_laws")
		// Found offer, return file
		parseFileDownload(w, r, o.CompanyByLawsType, pifc)
	}
}

func dealSellCompanyByLawsTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	o := dealSellTokenGetOffer(w, r, ps, "company_by_laws_token")
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "company_by_laws")
		// Found offer, return file
		parseFileDownload(w, r, o.CompanyByLawsType, pifc)
	}
}

func dealSellShareholderAgreementHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	o := dealSellGetOffer(w, r, ps, u)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "shareholder_agreement")
		// Found offer, return file
		parseFileDownload(w, r, o.ShareholderAgreementType, pifc)
	}
}

func dealSellShareholderAgreementTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	o := dealSellTokenGetOffer(w, r, ps, "shareholder_agreement_token")
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "shareholder_agreement")
		// Found offer, return file
		parseFileDownload(w, r, o.ShareholderAgreementType, pifc)
	}
}

func dealSellStockOptionPlanHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	o := dealSellGetOffer(w, r, ps, u)
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", u.ID), "stock_option_plan")
		// Found offer, return file
		parseFileDownload(w, r, o.StockOptionPlanType, pifc)
	}
}

func dealSellStockOptionPlanTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	o := dealSellTokenGetOffer(w, r, ps, "stock_option_plan_token")
	if o != nil {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", o.DealID),
			fmt.Sprintf("%v", o.UserID), "stock_option_plan")
		// Found offer, return file
		parseFileDownload(w, r, o.StockOptionPlanType, pifc)
	}
}

// dealSellDocumentDownloadHandler checks the minimum state "ms" to download
// a signed document and returns the pdf file if available during
// the seller stage; it also has the option "u == nil" to check for the token
// based approach with similar ways for download
// "name" is the document name saved and "tiname" is the token index name
// to search for (token only)
func dealSellDocumentDownloadHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User, ms uint64,
name, tiname string) {
	var dsh *DealShareholder
	if u != nil {
		_, dshu, ok := dealCheckState(w, r, ps, u, true, int64(ms),
			false, true, true)
		if !ok {
			return
		}
		dsh, ok = dshu.(*DealShareholder)
		if !ok {
			return
		}
	} else {
		// Check token for deal shareholder
		token, ok := CheckLength(true, ps.ByName("token"),
			TokenMinMax, TokenMinMax)
		if !ok {
			return
		}
		var dsho DealShareholder
		if dbConn.First(&dsho, tiname + " = ?", token).RecordNotFound() {
			return
		}
		dsh = &dsho
		if dsh.DealShareholderState < ms {
			return
		}
	}

	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", dsh.DealID),
		fmt.Sprintf("%v", dsh.UserID), name)
	// Always in pdf format
	parseFileDownload(w, r, "application/pdf", pifc)
}

func dealSellEngagementLetterHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	dealSellDocumentDownloadHandler(w, r, ps, u,
		DealShareholderStateEngagementLetterSigned,
		"sell_engagement_letter", "")
}

func dealSellEngagementLetterTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealSellDocumentDownloadHandler(w, r, ps, nil,
		DealShareholderStateEngagementLetterSigned,
		"sell_engagement_letter", "engagement_letter_token")
}

func dealSellBankInfoHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	_, dshu, ok := dealCheckState(w, r, ps, u, true,
		DealShareholderStateAdminApprovedOffer, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	dsh, ok := dshu.(*DealShareholder)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	fullName, of := CheckFieldForm("", r, "full_name")
	nickName, of := CheckFieldForm(of, r, "nick_name")
	routingNumber, of := CheckFieldForm(of, r, "routing_number")
	accountNumber, of := CheckFieldForm(of, r, "account_number")
	accountType, of := CheckRangeForm(of, r, "account_type",
		AccountTypeSaving)

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	// Check if bank already exists
	var banks []Bank
	if dbConn.Model(u).Related(&banks).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}
	var b *Bank
	// One user can only set the same bank once
	for _, bank := range banks {
		if decField(bank.RoutingNumberEncrypted) == routingNumber &&
			decField(bank.AccountNumberEncrypted) == accountNumber {
			b = &bank
			break
		}
	}
	if b == nil {
		b = &Bank{
			UserID:                 uint64(u.ID),
			FullName:               fullName,
			NickName:               nickName,
			RoutingNumberEncrypted: encField(routingNumber),
			AccountNumberEncrypted: encField(accountNumber),
			AccountType:            accountType}
		if dbConn.Model(u).Association("Banks").Append(b).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	}

	// Save bank information
	if dbConn.Model(b).Association("DealShareholders").Append(dsh).Error !=
		nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	dsh.DealShareholderState = DealShareholderStateBankInfoSubmitted
	if dbConn.Save(dsh).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_shareholder_state": dsh.DealShareholderState})
}

func dealBuyEngagementLetterSignHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateEngagementStarted, true, false, true)
	var di *DealInvestor
	if !ok {
		// Already returned message
		if diu == nil {
			return
		}
		// Create anew
	} else {
		// Existing investor
		di, ok = diu.(*DealInvestor)
		if !ok {
			// Bug
			formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
			return
		}
	}

	// If starting anew or link expired request new
	uid := fmt.Sprintf("%v", u.ID)
	var envId string
	var envTime, envCheck time.Time
	if di == nil || di.EngagementLetterSignId == "" ||
		di.EngagementLetterSignExpire.Before(time.Now()) {
		var c Company
		if dbConn.Model(d).Related(&c).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignError, true, nil)
			return
		}
		address := u.Address1
		if u.Address2 != "" {
			address += " " + u.Address2
		}
		reqLang := ps[len(ps) - 2].Value
		eid, et, err := serverDocusign.CreateEnvelopeWithTemplate(
			docusignBuyEngagementLetter,
			"client", uid, SignTexts[reqLang][SignBuyEngagementLetter],
			u.Email, u.FullName,
			map[string]interface{}{
				"client_name": u.FullName,
				"address":     address,
				"shares_type": SharesTypeTexts["en-US"][d.SharesType],
				"company":     c.Name,
			})
		if err != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignEnvelopeError,
				true, nil)
			return
		}
		envId = eid
		envTime = et
		envCheck = time.Time{}
	} else {
		envId = di.EngagementLetterSignId
		envTime = di.EngagementLetterSignExpire
		envCheck = di.EngagementLetterSignCheck
	}

	// Now create embedded url
	url, err := serverDocusign.CreateEmbeddedRecipientUrl(r.Host,
		envId, uid, u.Email, u.FullName)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealDocusignRecipientError,
			true, nil)
		return
	}

	// If new, create the deal investor struct, add to users
	if di == nil {
		// The following transaction should succeed together
		tx := dbConn.Begin()

		// Create the deal investor struct, add to users
		di = &DealInvestor{
			UserID:                     uint64(u.ID),
			DealID:                     uint64(d.ID),
			DealInvestorState:          DealInvestorStateEngagementStarted,
			EngagementLetterSignId:     envId,
			EngagementLetterSignUrl:    url,
			EngagementLetterSignExpire: envTime,
			EngagementLetterSignCheck:  envCheck,
		}
		if tx.Model(u).Association("DealInvestors").Append(di).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Then add to deal
		if tx.Model(d).Association("DealInvestors").Append(di).Error != nil {
			tx.Rollback()
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}

		// Finalize
		tx.Commit()
	} else {
		// If existing, update struct
		di.EngagementLetterSignId = envId
		di.EngagementLetterSignUrl = url
		di.EngagementLetterSignExpire = envTime
		di.EngagementLetterSignCheck = envCheck
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"docusign_sign_url":   di.EngagementLetterSignUrl,
			"docusign_sign_id":    di.EngagementLetterSignId,
		})
}

func dealBuyEngagementLetterCheckHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	// Start docusign checking lock asap
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}
	buyLock.Lock()
	buyChecks[did] = true
	buyLock.Unlock()
	defer func() {
		buyLock.Lock()
		buyChecks[did] = false
		buyLock.Unlock()
	}()

	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateEngagementStarted, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// Must have started signing and respects checking limit
	if di.EngagementLetterSignId == "" ||
		di.EngagementLetterSignExpire.Before(time.Now()) ||
		di.EngagementLetterSignCheck.After(time.Now()) {
		formatReturn(w, r, ps, ErrorCodeDealDocusignSignError,
			true, nil)
		return
	}

	// Check if document is completed
	eid := di.EngagementLetterSignId
	completed, terminal, err := serverDocusign.GetEnvelopeStatus(eid)
	if err != nil || !terminal {
		di.EngagementLetterSignCheck = time.Now().Add(docusignStatusDelay)
	} else {
		// Terminated, reset values
		di.EngagementLetterSignId = ""
		di.EngagementLetterSignUrl = ""
		di.EngagementLetterSignExpire = time.Time{}
		di.EngagementLetterSignCheck = time.Time{}
	}
	// At terminal state, reset current envelope
	if err != nil || !terminal || !completed {
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignStatusError,
				true, nil)
		}
		return
	}

	// Download file and save
	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
		fmt.Sprintf("%v", u.ID), "buy_engagement_letter")
	err = serverDocusign.DownloadEnvelopeDocument(eid, pifc)
	if err != nil {
		// Must save state on leave
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignDownloadError,
				true, nil)
		}
		return
	}

	// Create token for view
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	di.EngagementLetterToken = fmt.Sprintf("%x", b)
	di.DealInvestorState = DealInvestorStateEngagementLetterSigned
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"engagement_letter":   di.EngagementLetterToken})
}

func dealBuyNewInterestHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateEngagementLetterSigned, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	sharesBuyAmount, of := CheckRangeForm("", r,
		"shares_buy_amount", NumberMax)

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	// Check to make sure we have enough shares
	if sharesBuyAmount > d.SharesLeft {
		formatReturn(w, r, ps, ErrorCodeDealNotEnoughShares, true, nil)
		return
	}

	di.SharesBuyAmount = sharesBuyAmount
	di.DealInvestorState = DealInvestorStateInterestSubmitted
	// No other changes
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{"deal_investor_state": di.DealInvestorState})
}

func dealBuyHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateEngagementStarted, true, false, true)
	if !ok {
		if diu == "" {
			saveLogin(w, r, ps, false, u,
				map[string]interface{}{"deal_investor_state": -1})
		}
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// Construct necessary args and append optionals
	buyLock.Lock()
	// By default not locking
	dc := DocusignCheckingNo
	if buyChecks[uint64(d.ID)] {
		dc = DocusignCheckingYes
	}
	buyLock.Unlock()
	var c Company
	var name string
	if dbConn.Model(d).Related(&c).Error != nil {
		name = d.Name
	} else {
		name = c.Name
	}
	args := map[string]interface{}{
		"deal_investor_state": di.DealInvestorState,
		"docusign_checking":   dc,
		"company_name":        name,
	}
	// Do not waste resources when docusign checking is pending
	if dc == DocusignCheckingYes {
		saveLogin(w, r, ps, false, u, args)
		return
	}

	files := map[string]interface{}{}
	// Check if engagement letter has been signed
	if di.DealInvestorState >= DealInvestorStateEngagementLetterSigned {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
			fmt.Sprintf("%v", u.ID), "buy_engagement_letter")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.EngagementLetterToken = fmt.Sprintf("%x", b)
			files["engagement_letter"] = di.EngagementLetterToken
		}
	}

	if di.DealInvestorState >= DealInvestorStateInterestSubmitted {
		args["shares_buy_amount"] = di.SharesBuyAmount
	}

	// Check if summary of terms has been signed
	if di.DealInvestorState >= DealInvestorStateSummaryTermsSigned {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
			fmt.Sprintf("%v", u.ID), "summary_of_terms")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.SummaryOfTermsToken = fmt.Sprintf("%x", b)
			files["summary_terms"] = di.SummaryOfTermsToken
		}
	}

	// Return bank information if filled
	if di.DealInvestorState >= DealInvestorStateBankInfoSubmitted {
		var b Bank
		if dbConn.Model(di).Related(&b).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealNoBankInfo, true, nil)
			return
		}
		args["full_name"] = b.FullName
		args["nick_name"] = b.NickName
		args["routing_number"] = decField(b.RoutingNumberEncrypted)
		args["account_number"] = decField(b.AccountNumberEncrypted)
		args["account_type"] = b.AccountType
	}

	// Check if de ppm has been signed
	if di.DealInvestorState >= DealInvestorStateDePpmSigned {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
			fmt.Sprintf("%v", u.ID), "de_ppm")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.DePpmToken = fmt.Sprintf("%x", b)
			files["de_ppm"] = di.DePpmToken
		}
	}

	// Check if de operating agreement has been signed
	if di.DealInvestorState >= DealInvestorStateDeOperatingAgreementSigned {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
			fmt.Sprintf("%v", u.ID), "de_operating_agreement")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.DeOperatingAgreementToken = fmt.Sprintf("%x", b)
			files["de_operating"] = di.DeOperatingAgreementToken
		}
	}

	// Check if de subscription agreement has been signed
	if di.DealInvestorState >= DealInvestorStateDeSubscriptionAgreementSigned {
		pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
			fmt.Sprintf("%v", u.ID), "de_subscription_agreement")
		if _, err := os.Stat(pifc); err == nil {
			b := make([]byte, TokenMinMax / 2)
			crand.Read(b)
			di.DeSubscriptionAgreementToken = fmt.Sprintf("%x", b)
			files["de_subscription"] = di.DeSubscriptionAgreementToken
		}
	}

	// Check if wire information can be displayed
	if di.DealInvestorState >= DealInvestorStateWaitingFundTransfer {
		reqLang := ps[len(ps) - 2].Value
		if reqLang == "zh-CN" {
			args["wire"] = d.EscrowAccountCn
		} else {
			args["wire"] = d.EscrowAccount
		}
	}

	// Check and save
	if len(files) > 0 && dbConn.Save(di).Error == nil {
		for k, v := range files {
			args[k] = v
		}
	}

	saveLogin(w, r, ps, false, u, args)
}

// dealBuyDocumentDownloadHandler checks the minimum state "ms" to download
// a signed document and returns the pdf file if available during
// the buyer stage; it also has the option "u == nil" to check for the token
// based approach with similar ways for download
// "name" is the document name saved and "tiname" is the token index name
// to search for (token only)
func dealBuyDocumentDownloadHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User, ms uint64,
name, tiname string) {
	var di *DealInvestor
	if u != nil {
		_, diu, ok := dealCheckState(w, r, ps, u, false, int64(ms),
			false, true, true)
		if !ok {
			return
		}
		di, ok = diu.(*DealInvestor)
		if !ok {
			return
		}
	} else {
		// Check token for deal investor
		token, ok := CheckLength(true, ps.ByName("token"),
			TokenMinMax, TokenMinMax)
		if !ok {
			return
		}
		var dio DealInvestor
		if dbConn.First(&dio, tiname + " = ?", token).RecordNotFound() {
			return
		}
		di = &dio
		if di.DealInvestorState < ms {
			return
		}
	}

	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", di.DealID),
		fmt.Sprintf("%v", di.UserID), name)
	// Always in pdf format
	parseFileDownload(w, r, "application/pdf", pifc)
}

func dealBuyEngagementLetterHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	dealBuyDocumentDownloadHandler(w, r, ps, u,
		DealInvestorStateEngagementLetterSigned,
		"buy_engagement_letter", "")
}

func dealBuyEngagementLetterTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDocumentDownloadHandler(w, r, ps, nil,
		DealInvestorStateEngagementLetterSigned,
		"buy_engagement_letter", "engagement_letter_token")
}

func dealBuySummaryTermsHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	dealBuyDocumentDownloadHandler(w, r, ps, u,
		DealInvestorStateSummaryTermsSigned,
		"summary_of_terms", "")
}

func dealBuySummaryTermsTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDocumentDownloadHandler(w, r, ps, nil,
		DealInvestorStateSummaryTermsSigned,
		"summary_of_terms", "summary_of_terms_token")
}

func dealBuyDePpmHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	dealBuyDocumentDownloadHandler(w, r, ps, u,
		DealInvestorStateDePpmSigned,
		"de_ppm", "")
}

func dealBuyDePpmTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDocumentDownloadHandler(w, r, ps, nil,
		DealInvestorStateDePpmSigned,
		"de_ppm", "de_ppm_token")
}

func dealBuyDeOperatingHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	dealBuyDocumentDownloadHandler(w, r, ps, u,
		DealInvestorStateDeOperatingAgreementSigned,
		"de_operating_agreement", "")
}

func dealBuyDeOperatingTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDocumentDownloadHandler(w, r, ps, nil,
		DealInvestorStateDeOperatingAgreementSigned,
		"de_operating_agreement", "de_operating_agreement_token")
}

func dealBuyDeSubscriptionHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params, u *User) {
	dealBuyDocumentDownloadHandler(w, r, ps, u,
		DealInvestorStateDeSubscriptionAgreementSigned,
		"de_subscription_agreement", "")
}

func dealBuyDeSubscriptionTokenHandler(w http.ResponseWriter,
r *http.Request, ps httprouter.Params) {
	dealBuyDocumentDownloadHandler(w, r, ps, nil,
		DealInvestorStateDeSubscriptionAgreementSigned,
		"de_subscription_agreement", "de_subscription_agreement_token")
}

func dealBuySummaryTermsSignHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateInterestSubmitted, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// If starting anew or link expired request new
	uid := fmt.Sprintf("%v", u.ID)
	var envId string
	var envTime, envCheck time.Time
	if di == nil || di.SummaryOfTermsSignId == "" ||
		di.SummaryOfTermsSignExpire.Before(time.Now()) {
		var c Company
		if dbConn.Model(d).Related(&c).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignError, true, nil)
			return
		}
		reqLang := ps[len(ps) - 2].Value
		eid, et, err := serverDocusign.CreateEnvelopeWithTemplate(
			docusignBuySummaryOfTerms,
			"client", uid, SignTexts[reqLang][SignBuySummaryOfTerms],
			u.Email, u.FullName,
			map[string]interface{}{
				"client_name":  u.FullName,
				"amount":       di.SharesBuyAmount,
				"amount_all":   d.SharesAmount,
				"shares_type":  SharesTypeTexts["en-US"][d.SharesType],
				"company":      c.Name,
				"closing_date": d.EndDate.Format("01/02/2006"),
			})
		if err != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignEnvelopeError,
				true, nil)
			return
		}
		envId = eid
		envTime = et
		envCheck = time.Time{}
	} else {
		envId = di.SummaryOfTermsSignId
		envTime = di.SummaryOfTermsSignExpire
		envCheck = di.SummaryOfTermsSignCheck
	}

	// Now create embedded url
	url, err := serverDocusign.CreateEmbeddedRecipientUrl(r.Host,
		envId, uid, u.Email, u.FullName)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealDocusignRecipientError,
			true, nil)
		return
	}

	// Update signing struct
	di.SummaryOfTermsSignId = envId
	di.SummaryOfTermsSignUrl = url
	di.SummaryOfTermsSignExpire = envTime
	di.SummaryOfTermsSignCheck = envCheck
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"docusign_sign_url":   di.SummaryOfTermsSignUrl,
			"docusign_sign_id":    di.SummaryOfTermsSignId,
		})
}

func dealBuySummaryTermsCheckHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// Start docusign checking lock asap
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}
	buyLock.Lock()
	buyChecks[did] = true
	buyLock.Unlock()
	defer func() {
		buyLock.Lock()
		buyChecks[did] = false
		buyLock.Unlock()
	}()

	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateInterestSubmitted, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// Must have started signing and respects checking limit
	if di.SummaryOfTermsSignId == "" ||
		di.SummaryOfTermsSignExpire.Before(time.Now()) ||
		di.SummaryOfTermsSignCheck.After(time.Now()) {
		formatReturn(w, r, ps, ErrorCodeDealDocusignSignError,
			true, nil)
		return
	}

	// Check if document is completed
	eid := di.SummaryOfTermsSignId
	completed, terminal, err := serverDocusign.GetEnvelopeStatus(eid)
	if err != nil || !terminal {
		di.SummaryOfTermsSignCheck = time.Now().Add(docusignStatusDelay)
	} else {
		// Terminated, reset values
		di.SummaryOfTermsSignId = ""
		di.SummaryOfTermsSignUrl = ""
		di.SummaryOfTermsSignExpire = time.Time{}
		di.SummaryOfTermsSignCheck = time.Time{}
	}
	// At terminal state, reset current envelope
	if err != nil || !terminal || !completed {
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignStatusError,
				true, nil)
		}
		return
	}

	// Download file and save
	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
		fmt.Sprintf("%v", u.ID), "summary_of_terms")
	err = serverDocusign.DownloadEnvelopeDocument(eid, pifc)
	if err != nil {
		// Must save state on leave
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignDownloadError,
				true, nil)
		}
		return
	}

	// Create token for view
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	di.SummaryOfTermsToken = fmt.Sprintf("%x", b)
	di.DealInvestorState = DealInvestorStateSummaryTermsSigned
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"summary_terms":       di.SummaryOfTermsToken})
}

func dealBuyBankInfoHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	_, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateAdminApprovedInterest, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	fullName, of := CheckFieldForm("", r, "full_name")
	nickName, of := CheckFieldForm(of, r, "nick_name")
	routingNumber, of := CheckFieldForm(of, r, "routing_number")
	accountNumber, of := CheckFieldForm(of, r, "account_number")
	accountType, of := CheckRangeForm(of, r, "account_type",
		AccountTypeSaving)

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	// Check if bank already exists
	var banks []Bank
	if dbConn.Model(u).Related(&banks).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}
	var b *Bank
	// One user can only set the same bank once
	for _, bank := range banks {
		if decField(bank.RoutingNumberEncrypted) == routingNumber &&
			decField(bank.AccountNumberEncrypted) == accountNumber {
			b = &bank
			break
		}
	}
	if b == nil {
		b = &Bank{
			UserID:                 uint64(u.ID),
			FullName:               fullName,
			NickName:               nickName,
			RoutingNumberEncrypted: encField(routingNumber),
			AccountNumberEncrypted: encField(accountNumber),
			AccountType:            accountType}
		if dbConn.Model(u).Association("Banks").Append(b).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
			return
		}
	}

	// Save bank information
	if dbConn.Model(b).Association("DealInvestors").Append(di).Error !=
		nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	di.DealInvestorState = DealInvestorStateBankInfoSubmitted
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{"deal_investor_state": di.DealInvestorState})
}

func dealBuyDePpmSignHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateBankVerified, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// If starting anew or link expired request new
	uid := fmt.Sprintf("%v", u.ID)
	var envId string
	var envTime, envCheck time.Time
	if di == nil || di.DePpmSignId == "" ||
		di.DePpmSignExpire.Before(time.Now()) {
		var c Company
		if dbConn.Model(d).Related(&c).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignError, true, nil)
			return
		}
		var fundings []Funding
		if dbConn.Model(&c).Related(&fundings).Error != nil ||
			len(fundings) < 1 {
			formatReturn(w, r, ps, ErrorCodeDealDocusignError, true, nil)
			return
		}
		reqLang := ps[len(ps) - 2].Value
		eid, et, err := serverDocusign.CreateEnvelopeWithTemplate(
			docusignBuyDePpm,
			"client", uid, SignTexts[reqLang][SignBuyDePpm],
			u.Email, u.FullName,
			map[string]interface{}{
				"fund_name": fmt.Sprintf("MarketX %v Fund %v, LLC", c.Name,
					formatRoman(d.FundNum)),
				"actual_price":        d.ActualPrice,
				"acutal_valuation":    formatMoney(d.ActualValuation),
				"last_price":          fundings[0].ConversionPrice,
				"last_valuation":      formatMoney(fundings[0].PostValuation),
				"last_valuation_date": fundings[0].Date,
				"shares_type":         SharesTypeTexts["en-US"][d.SharesType],
			})
		if err != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignEnvelopeError,
				true, nil)
			return
		}
		envId = eid
		envTime = et
		envCheck = time.Time{}
	} else {
		envId = di.DePpmSignId
		envTime = di.DePpmSignExpire
		envCheck = di.DePpmSignCheck
	}

	// Now create embedded url
	url, err := serverDocusign.CreateEmbeddedRecipientUrl(r.Host,
		envId, uid, u.Email, u.FullName)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealDocusignRecipientError,
			true, nil)
		return
	}

	// Update signing struct
	di.DePpmSignId = envId
	di.DePpmSignUrl = url
	di.DePpmSignExpire = envTime
	di.DePpmSignCheck = envCheck
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"docusign_sign_url":   di.DePpmSignUrl,
			"docusign_sign_id":    di.DePpmSignId,
		})
}

func dealBuyDePpmCheckHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// Start docusign checking lock asap
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}
	buyLock.Lock()
	buyChecks[did] = true
	buyLock.Unlock()
	defer func() {
		buyLock.Lock()
		buyChecks[did] = false
		buyLock.Unlock()
	}()

	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateBankVerified, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// Must have started signing and respects checking limit
	if di.DePpmSignId == "" ||
		di.DePpmSignExpire.Before(time.Now()) ||
		di.DePpmSignCheck.After(time.Now()) {
		formatReturn(w, r, ps, ErrorCodeDealDocusignSignError,
			true, nil)
		return
	}

	// Check if document is completed
	eid := di.DePpmSignId
	completed, terminal, err := serverDocusign.GetEnvelopeStatus(eid)
	if err != nil || !terminal {
		di.DePpmSignCheck = time.Now().Add(docusignStatusDelay)
	} else {
		// Terminated, reset values
		di.DePpmSignId = ""
		di.DePpmSignUrl = ""
		di.DePpmSignExpire = time.Time{}
		di.DePpmSignCheck = time.Time{}
	}
	// At terminal state, reset current envelope
	if err != nil || !terminal || !completed {
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignStatusError,
				true, nil)
		}
		return
	}

	// Download file and save
	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
		fmt.Sprintf("%v", u.ID), "de_ppm")
	err = serverDocusign.DownloadEnvelopeDocument(eid, pifc)
	if err != nil {
		// Must save state on leave
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignDownloadError,
				true, nil)
		}
		return
	}

	// Create token for view
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	di.DePpmToken = fmt.Sprintf("%x", b)
	di.DealInvestorState = DealInvestorStateDePpmSigned
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"de_ppm":              di.DePpmToken})
}

func dealBuyDeOperatingSignHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateDePpmSigned, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// If starting anew or link expired request new
	uid := fmt.Sprintf("%v", u.ID)
	var envId string
	var envTime, envCheck time.Time
	if di == nil || di.DeOperatingAgreementSignId == "" ||
		di.DeOperatingAgreementSignExpire.Before(time.Now()) {
		var c Company
		if dbConn.Model(d).Related(&c).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignError, true, nil)
			return
		}
		reqLang := ps[len(ps) - 2].Value
		eid, et, err := serverDocusign.CreateEnvelopeWithTemplate(
			docusignBuyDeOperatingAgreement,
			"client", uid, SignTexts[reqLang][SignBuyDeOperatingAgreement],
			u.Email, u.FullName,
			map[string]interface{}{
				"client_name": u.FullName,
				"fund_name": fmt.Sprintf("MarketX %v Fund %v, LLC", c.Name,
					formatRoman(d.FundNum)),
				"company":       c.Name,
				"company_state": c.StateFounded,
				"shares_type":   SharesTypeTexts["en-US"][d.SharesType],
			})
		if err != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignEnvelopeError,
				true, nil)
			return
		}
		envId = eid
		envTime = et
		envCheck = time.Time{}
	} else {
		envId = di.DeOperatingAgreementSignId
		envTime = di.DeOperatingAgreementSignExpire
		envCheck = di.DeOperatingAgreementSignCheck
	}

	// Now create embedded url
	url, err := serverDocusign.CreateEmbeddedRecipientUrl(r.Host,
		envId, uid, u.Email, u.FullName)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealDocusignRecipientError,
			true, nil)
		return
	}

	// Update signing struct
	di.DeOperatingAgreementSignId = envId
	di.DeOperatingAgreementSignUrl = url
	di.DeOperatingAgreementSignExpire = envTime
	di.DeOperatingAgreementSignCheck = envCheck
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"docusign_sign_url":   di.DeOperatingAgreementSignUrl,
			"docusign_sign_id":    di.DeOperatingAgreementSignId,
		})
}

func dealBuyDeOperatingCheckHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// Start docusign checking lock asap
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}
	buyLock.Lock()
	buyChecks[did] = true
	buyLock.Unlock()
	defer func() {
		buyLock.Lock()
		buyChecks[did] = false
		buyLock.Unlock()
	}()

	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateDePpmSigned, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// Must have started signing and respects checking limit
	if di.DeOperatingAgreementSignId == "" ||
		di.DeOperatingAgreementSignExpire.Before(time.Now()) ||
		di.DeOperatingAgreementSignCheck.After(time.Now()) {
		formatReturn(w, r, ps, ErrorCodeDealDocusignSignError,
			true, nil)
		return
	}

	// Check if document is completed
	eid := di.DeOperatingAgreementSignId
	completed, terminal, err := serverDocusign.GetEnvelopeStatus(eid)
	if err != nil || !terminal {
		di.DeOperatingAgreementSignCheck = time.Now().Add(docusignStatusDelay)
	} else {
		// Terminated, reset values
		di.DeOperatingAgreementSignId = ""
		di.DeOperatingAgreementSignUrl = ""
		di.DeOperatingAgreementSignExpire = time.Time{}
		di.DeOperatingAgreementSignCheck = time.Time{}
	}
	// At terminal state, reset current envelope
	if err != nil || !terminal || !completed {
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignStatusError,
				true, nil)
		}
		return
	}

	// Download file and save
	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
		fmt.Sprintf("%v", u.ID), "de_operating_agreement")
	err = serverDocusign.DownloadEnvelopeDocument(eid, pifc)
	if err != nil {
		// Must save state on leave
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignDownloadError,
				true, nil)
		}
		return
	}

	// Create token for view
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	di.DeOperatingAgreementToken = fmt.Sprintf("%x", b)
	di.DealInvestorState = DealInvestorStateDeOperatingAgreementSigned
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"de_operating":        di.DeOperatingAgreementToken})
}

func dealBuyDeSubscriptionSignHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateDeOperatingAgreementSigned, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// If starting anew or link expired request new
	uid := fmt.Sprintf("%v", u.ID)
	var envId string
	var envTime, envCheck time.Time
	if di == nil || di.DeSubscriptionAgreementSignId == "" ||
		di.DeSubscriptionAgreementSignExpire.Before(time.Now()) {
		var c Company
		if dbConn.Model(d).Related(&c).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignError, true, nil)
			return
		}
		address := u.Address1
		if u.Address2 != "" {
			address += " " + u.Address2
		}
		// Calculate amounts
		amountp := di.SharesBuyAmount
		amounte := uint64(float64(amountp) * 0.05)
		amountt := amountp + amounte
		reqLang := ps[len(ps) - 2].Value
		eid, et, err := serverDocusign.CreateEnvelopeWithTemplate(
			docusignBuyDeSubscriptionAgreement,
			"client", uid, SignTexts[reqLang][SignBuyDeSubscriptionAgreement],
			u.Email, u.FullName,
			map[string]interface{}{
				"client_name": u.FullName,
				"fund_name": fmt.Sprintf("MarketX %v Fund %v, LLC", c.Name,
					formatRoman(d.FundNum)),
				"amount_principal": amountp,
				"amount_expense":   amounte,
				"amount_total":     amountt,
				"ssn":              decField(u.SsnEncrypted),
				"address":          address,
				"phone_number":     u.PhoneNumber,
				"email":            u.Email,
			})
		if err != nil {
			formatReturn(w, r, ps, ErrorCodeDealDocusignEnvelopeError,
				true, nil)
			return
		}
		envId = eid
		envTime = et
		envCheck = time.Time{}
	} else {
		envId = di.DeSubscriptionAgreementSignId
		envTime = di.DeSubscriptionAgreementSignExpire
		envCheck = di.DeSubscriptionAgreementSignCheck
	}

	// Now create embedded url
	url, err := serverDocusign.CreateEmbeddedRecipientUrl(r.Host,
		envId, uid, u.Email, u.FullName)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealDocusignRecipientError,
			true, nil)
		return
	}

	// Update signing struct
	di.DeSubscriptionAgreementSignId = envId
	di.DeSubscriptionAgreementSignUrl = url
	di.DeSubscriptionAgreementSignExpire = envTime
	di.DeSubscriptionAgreementSignCheck = envCheck
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"docusign_sign_url":   di.DeSubscriptionAgreementSignUrl,
			"docusign_sign_id":    di.DeSubscriptionAgreementSignId,
		})
}

func dealBuyDeSubscriptionCheckHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// Start docusign checking lock asap
	did, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeDealUnknown, true, nil)
		return
	}
	buyLock.Lock()
	buyChecks[did] = true
	buyLock.Unlock()
	defer func() {
		buyLock.Lock()
		buyChecks[did] = false
		buyLock.Unlock()
	}()

	d, diu, ok := dealCheckState(w, r, ps, u, false,
		DealInvestorStateDeOperatingAgreementSigned, false, false, true)
	if !ok {
		// Already returned message
		return
	}
	di, ok := diu.(*DealInvestor)
	if !ok {
		// Bug
		formatReturn(w, r, ps, ErrorCodeServerInternal, true, nil)
		return
	}

	// Must have started signing and respects checking limit
	if di.DeSubscriptionAgreementSignId == "" ||
		di.DeSubscriptionAgreementSignExpire.Before(time.Now()) ||
		di.DeSubscriptionAgreementSignCheck.After(time.Now()) {
		formatReturn(w, r, ps, ErrorCodeDealDocusignSignError,
			true, nil)
		return
	}

	// Check if document is completed
	eid := di.DeSubscriptionAgreementSignId
	completed, terminal, err := serverDocusign.GetEnvelopeStatus(eid)
	if err != nil || !terminal {
		di.DeSubscriptionAgreementSignCheck =
			time.Now().Add(docusignStatusDelay)
	} else {
		// Terminated, reset values
		di.DeSubscriptionAgreementSignId = ""
		di.DeSubscriptionAgreementSignUrl = ""
		di.DeSubscriptionAgreementSignExpire = time.Time{}
		di.DeSubscriptionAgreementSignCheck = time.Time{}
	}
	// At terminal state, reset current envelope
	if err != nil || !terminal || !completed {
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignStatusError,
				true, nil)
		}
		return
	}

	// Download file and save
	pifc := path.Join(dataDir, "deal", fmt.Sprintf("%v", d.ID),
		fmt.Sprintf("%v", u.ID), "de_subscription_agreement")
	err = serverDocusign.DownloadEnvelopeDocument(eid, pifc)
	if err != nil {
		// Must save state on leave
		if dbConn.Save(di).Error != nil {
			formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		} else {
			formatReturn(w, r, ps, ErrorCodeDealDocusignDownloadError,
				true, nil)
		}
		return
	}

	// Create token for view
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	di.DeSubscriptionAgreementToken = fmt.Sprintf("%x", b)
	di.DealInvestorState = DealInvestorStateDeSubscriptionAgreementSigned
	if dbConn.Save(di).Error != nil {
		formatReturn(w, r, ps, ErrorCodeDealOperationError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{
			"deal_investor_state": di.DealInvestorState,
			"de_subscription":     di.DeSubscriptionAgreementToken})
}
