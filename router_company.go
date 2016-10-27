// Router branch for /company/ operations
package main

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"sort"
	"strconv"
)

func companiesHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params) {
	// Must start with at least a keyword
	kw, ok := CheckField(true, r.FormValue("keyword"))
	if !ok {
		formatReturnJson(w, r, ps, ErrorCodeKeywordError, "", LoginNone, nil)
		return
	}

	var cs []Company
	if dbConn.Where("name ~* ?", kw).Order("id desc").Limit(100).
		Find(&cs).Error != nil {
		// Probably bad inputs
		formatReturnJson(w, r, ps, ErrorCodeKeywordError, "", LoginNone, nil)
		return
	}

	companies := []map[string]interface{}{}
	for _, company := range cs {
		var ds []Deal
		// Skip if problem
		if dbConn.Model(&company).Related(&ds).Error != nil {
			continue
		}
		// Check if there is any open deal, should be one at a time
		ret := map[string]interface{}{
			"id":           company.ID,
			"name":         company.Name,
			"deal_state":   DealStateClosed,
			"deal_special": DealSpecialNone,
		}
		for _, d := range ds {
			// One state per company at a time
			if d.DealState != DealStateClosed {
				ret["deal_state"] = d.DealState
				ret["deal_special"] = d.DealSpecial
				break
			}
		}

		companies = append(companies, ret)
	}

	formatReturnJson(w, r, ps, ErrorCodeNone, "", LoginNone,
		map[string]interface{}{"companies": companies})
}

func companyIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// Make sure we don't read more information than user could
	// Special debug flag does it anyway
	if u.UserState < UserStateActive {
		formatReturn(w, r, ps, ErrorCodeUserNotOnboard, true, nil)
		return
	}

	cid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	var c Company
	if dbConn.First(&c, cid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	// Reading information should not fail unless data is corrupt
	var tags []Tag
	if dbConn.Model(&c).Related(&tags, "Tags").Error != nil ||
		len(tags) < 3 {
		formatReturn(w, r, ps, ErrorCodeCompanyInfoError, true, nil)
		return
	}
	var fundings []Funding
	if dbConn.Model(&c).Related(&fundings).Error != nil ||
		len(fundings) < 1 {
		formatReturn(w, r, ps, ErrorCodeCompanyInfoError, true, nil)
		return
	}
	sort.Sort(FundingSort(fundings))
	var executives []CompanyExecutive
	if dbConn.Model(&c).Related(&executives).Error != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyInfoError, true, nil)
		return
	}
	sort.Sort(CompanyExecutiveSort(executives))
	var updates []CompanyUpdate
	if dbConn.Model(&c).Related(&updates).Error != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyInfoError, true, nil)
		return
	}
	sort.Sort(CompanyUpdateSort(updates))

	// Process language-specific returns
	reqLang := ps[len(ps)-2].Value
	var desc, tagss string
	if reqLang == "zh-CN" {
		desc = c.DescriptionCn
		tagss = tags[0].NameCn + "、" + tags[1].NameCn + "、" +
			tags[2].NameCn
	} else {
		desc = c.Description
		tagss = tags[0].Name + ", " + tags[1].Name + ", " +
			tags[2].Name
	}

	// Process necessary information
	var funds []map[string]interface{}
	for _, f := range fundings {
		funds = append(funds, map[string]interface{}{
			"type":             f.Type,
			"date":             f.Date,
			"amount":           f.Amount,
			"raised_to_date":   f.RaisedToDate,
			"status":           f.Status,
			"conversion_price": f.ConversionPrice,
		})
	}
	var execs []map[string]interface{}
	for _, e := range executives {
		execs = append(execs, map[string]interface{}{
			"name":   e.Name,
			"role":   e.Role,
			"office": e.Office,
		})
	}
	var cus []map[string]interface{}
	for _, cu := range updates {
		if cu.Language != reqLang {
			continue
		}
		cus = append(cus, map[string]interface{}{
			"title": cu.Title,
			"url":   cu.Url,
			"date":  cu.Date,
		})
	}

	ret := map[string]interface{}{
		"name":               c.Name,
		"year_founded":       c.YearFounded,
		"state_founded":      c.StateFounded,
		"num_employees":      c.NumEmployees,
		"total_valuation":    c.TotalValuation,
		"total_funding":      c.TotalFunding,
		"home_page":          c.HomePage,
		"hq":                 c.Hq,
		"description":        desc,
		"tags":               tagss,
		"investor_logo_pics": c.InvestorLogoPics,
		"num_slides":         c.NumSlides,
		"video_url":          c.VideoUrl,
		"executives":         execs,
		"fundings":           funds,
		"updates":            cus,
		"secret":             companySecret,
	}
	// Check if there is any open deal, should be one at a time
	var ds []Deal
	if dbConn.Model(&c).Related(&ds).Error != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyInfoError, true, nil)
		return
	}
	for _, d := range ds {
		if d.DealState == DealStateOpen {
			// Only return if deal is live
			deal := map[string]interface{}{
				"id":               d.ID,
				"deal_state":       d.DealState,
				"deal_special":     d.DealSpecial,
				"start_date":       unixTime(d.StartDate),
				"end_date":         unixTime(d.EndDate),
				"fund_num":         d.FundNum,
				"actual_price":     d.ActualPrice,
				"actual_valuation": d.ActualValuation,
				"shares_amount":    d.SharesAmount,
				"shares_left":      d.SharesLeft,
				"shares_type":      d.SharesType,
			}
			// Add on user specific information if available
			var dshs []DealShareholder
			dbConn.Model(&d).Related(&dshs)
			var dis []DealInvestor
			dbConn.Model(&d).Related(&dis)
			// Default to none
			deal["deal_user_state"] = DealUserStateNone
			for _, dsh := range dshs {
				if dsh.UserID == uint64(u.ID) {
					deal["deal_user_state"] = DealUserStateShareholder
					deal["deal_shareholder_state"] = dsh.DealShareholderState
					break
				}
			}
			for _, di := range dis {
				if di.UserID == uint64(u.ID) {
					deal["deal_user_state"] = DealUserStateInvestor
					deal["deal_investor_state"] = di.DealInvestorState
					break
				}
			}
			ret["deal"] = deal
			break
		}
	}

	// Read all fine, construct results
	saveLogin(w, r, ps, false, u, ret)
}
