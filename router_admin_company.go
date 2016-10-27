// Router branch for /admin/company/ operations
package main

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
)

func adminCompaniesHandler(w http.ResponseWriter, r *http.Request,
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

	var cs []Company
	if dbConn.Where("name ~* ?", r.FormValue("keyword")).
		Order("id desc").Offset(int(pnum*psize)).Limit(int(psize)).
		Find(&cs).Error != nil {
		// Probably bad inputs
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
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
			"id":   company.ID,
			"name": company.Name,
		}
		for _, d := range ds {
			if d.DealState == DealStateOpen {
				ret["deal_id"] = d.ID
				break
			}
		}

		companies = append(companies, ret)
	}

	// Return all satisfied deals
	saveAdmin(w, r, ps, u, map[string]interface{}{"companies": companies})
}

func adminCompanyTagsHandler(w http.ResponseWriter, r *http.Request,
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

	var ts []Tag
	if dbConn.Where("name ~* ? or name_cn ~* ?", r.FormValue("keyword"),
		r.FormValue("keyword")).
		Order("id desc").Offset(int(pnum*psize)).Limit(int(psize)).
		Find(&ts).Error != nil {
		// Probably bad inputs
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	tags := []map[string]interface{}{}
	for _, tag := range ts {
		tags = append(tags, map[string]interface{}{
			"id":      tag.ID,
			"name":    tag.Name,
			"name_cn": tag.NameCn,
		})
	}

	// Return all satisfied deals
	saveAdmin(w, r, ps, u, map[string]interface{}{"tags": tags})
}

func adminCompanyIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
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

	// Process necessary information
	var funds []map[string]interface{}
	for _, f := range fundings {
		funds = append(funds, map[string]interface{}{
			"type":                      f.Type,
			"date":                      f.Date,
			"amount":                    f.Amount,
			"raised_to_date":            f.RaisedToDate,
			"status":                    f.Status,
			"conversion_price":          f.ConversionPrice,
			"pre_valuation":             f.PreValuation,
			"post_valuation":            f.PostValuation,
			"stage":                     f.Stage,
			"num_shares":                f.NumShares,
			"par_value":                 f.ParValue,
			"dividend_rate_percent":     f.DividendRatePercent,
			"original_issue_price":      f.OriginalIssuePrice,
			"liquidation":               f.Liquidation,
			"liquidation_pref_multiple": f.LiquidationPrefMultiple,
			"percent_owned":             f.PercentOwned,
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
	cuss := map[string][]map[string]interface{}{}
	for _, cu := range updates {
		cuc := map[string]interface{}{
			"title": cu.Title,
			"url":   cu.Url,
			"date":  cu.Date,
		}
		cus, ok := cuss[cu.Language]
		if ok {
			cus = append(cus, cuc)
		} else {
			cus = []map[string]interface{}{cuc}
		}
		cuss[cu.Language] = cus
	}

	saveAdmin(w, r, ps, u, map[string]interface{}{
		"name":            c.Name,
		"year_founded":    c.YearFounded,
		"state_founded":   c.StateFounded,
		"num_employees":   c.NumEmployees,
		"total_valuation": c.TotalValuation,
		"total_funding":   c.TotalFunding,
		"home_page":       c.HomePage,
		"hq":              c.Hq,
		"description":     c.Description,
		"description_cn":  c.DescriptionCn,
		"tags": fmt.Sprintf("%v,%v,%v",
			tags[0].ID, tags[1].ID, tags[2].ID),
		"investor_logo_pics":  c.InvestorLogoPics,
		"num_slides":          c.NumSlides,
		"video_url":           c.VideoUrl,
		"executives":          execs,
		"fundings":            funds,
		"updates":             cuss["en-US"],
		"updates_cn":          cuss["zh-CN"],
		"key_person":          c.KeyPerson,
		"growth_rate_percent": c.GrowthRatePercent,
		"size_multiple":       c.SizeMultiple,
		"investors":           c.Investors,
	})
}

func adminCompanyTagIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	tid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyTagUnknown, true, nil)
		return
	}

	var t Tag
	if dbConn.First(&t, tid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeCompanyTagUnknown, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, map[string]interface{}{
		"name":    t.Name,
		"name_cn": t.NameCn,
	})
}

func adminCompanyUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
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

	// Save all the non-relational fields
	name, ok := CheckField(true, r.FormValue("name"))
	if ok {
		c.Name = name
	}
	yearFounded, ok := CheckRange(true, r.FormValue("year_founded"),
		YearMax)
	if ok {
		c.YearFounded = yearFounded
	}
	stateFounded, ok := CheckField(true, r.FormValue("state_founded"))
	if ok {
		c.StateFounded = stateFounded
	}
	numEmployees, ok := CheckRange(true, r.FormValue("num_employees"),
		NumberMax)
	if ok {
		c.NumEmployees = numEmployees
	}
	totalValuation, tvok := CheckFloat(true, r.FormValue("total_valuation"),
		true)
	if tvok {
		c.TotalValuation = totalValuation
	}
	totalFunding, tfok := CheckFloat(true, r.FormValue("total_funding"),
		true)
	if tfok {
		c.TotalFunding = totalFunding
	}
	homePage, ok := CheckField(true, r.FormValue("home_page"))
	if ok {
		c.HomePage = homePage
	}
	hq, ok := CheckField(true, r.FormValue("hq"))
	if ok {
		c.Hq = hq
	}
	description, ok := CheckLength(true, r.FormValue("description"),
		1, StringBlockMax)
	if ok {
		c.Description = description
	}
	descriptionCn, ok := CheckLength(true, r.FormValue("description_cn"),
		1, StringBlockMax)
	if ok {
		c.DescriptionCn = descriptionCn
	}
	ilps, ok := CheckLength(true, r.FormValue("investor_logo_pics"),
		1, StringPlusMax)
	if ok {
		c.InvestorLogoPics = ilps
	}
	numSlides, ok := CheckRange(true, r.FormValue("num_slides"),
		NumberMax)
	if ok {
		c.NumSlides = numSlides
	}
	videoUrl, ok := CheckField(true, r.FormValue("video_url"))
	if ok {
		c.VideoUrl = videoUrl
	}
	keyPerson, ok := CheckField(true, r.FormValue("key_person"))
	if ok {
		c.KeyPerson = keyPerson
	}
	grp, ok := CheckFloat(true, r.FormValue("growth_rate_percent"), false)
	if ok {
		c.GrowthRatePercent = grp
	}
	sizeMultiple, ok := CheckFloat(true, r.FormValue("size_multiple"), true)
	if ok {
		c.SizeMultiple = sizeMultiple
	}
	investors, ok := CheckLength(true, r.FormValue("investors"),
		1, StringBlockMax)
	if ok {
		c.Investors = investors
	}

	// Process relational data first since some information have to be
	// cross-checked and saved
	tagss, ok := CheckLength(true, r.FormValue("tags"), 1, JsonMax)
	if ok {
		tss := strings.Split(tagss, ",")
		// Must be at least 3 to be valid
		if len(tss) >= 3 {
			var tags []Tag
			for i := 0; i < len(tss); i++ {
				tid, err := strconv.ParseUint(tss[i], 10, 64)
				if err != nil {
					continue
				}
				var tag Tag
				// Skip on error parsing
				if dbConn.First(&tag, tid).RecordNotFound() {
					continue
				}
				tags = append(tags, tag)
			}
			// Must read at least 3 tags
			if len(tags) >= 3 {
				if dbConn.Model(&c).Association("Tags").
					Replace(&tags).Error != nil {
					formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
					return
				}
			}
		}
	}

	execs, ok := CheckLength(true, r.FormValue("executives"), 1, JsonMax)
	if ok {
		var data []CompanyExecutive
		err := json.Unmarshal([]byte(execs), &data)
		// Ignore bad inputs
		if err == nil {
			// Full diff comparison would require more intricate protocols
			// which we do not have the needs (resources) to do right now
			// So we always do a full replace
			var executives []CompanyExecutive
			if dbConn.Model(&c).Related(&executives).Error == nil {
				for _, exec := range executives {
					dbConn.Delete(&exec)
				}
			}
			if dbConn.Model(&c).Association("CompanyExecutives").
				Replace(&data).Error != nil {
				formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
				return
			}
		}
	}

	funds, ok := CheckLength(true, r.FormValue("fundings"), 1, JsonMax)
	if ok {
		var data []Funding
		err := json.Unmarshal([]byte(funds), &data)
		// Ignore bad inputs
		if err == nil && len(data) > 0 {
			// Full diff comparison would require more intricate protocols
			// which we do not have the needs (resources) to do right now
			// So we always do a full replace
			var fundings []Funding
			if dbConn.Model(&c).Related(&fundings).Error == nil {
				for _, funding := range fundings {
					dbConn.Delete(&funding)
				}
			}
			if dbConn.Model(&c).Association("Fundings").
				Replace(&data).Error != nil {
				formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
				return
			}
			// Update linked company information
			if len(data) > 0 {
				c.TotalValuation = data[0].PostValuation
				c.TotalFunding = data[0].RaisedToDate
			}
		}
	}

	cus, cuok := CheckLength(true, r.FormValue("updates"), 1, JsonMax)
	cuscn, cucnok := CheckLength(true, r.FormValue("updates_cn"), 1, JsonMax)
	// Do both languages in one try
	if cuok || cucnok {
		var data []CompanyUpdate
		err := json.Unmarshal([]byte(cus), &data)
		// Ignore bad inputs
		if err == nil {
			// Fill in company id
			for i, _ := range data {
				data[i].Language = "en-US"
			}
		}
		var datacn []CompanyUpdate
		errcn := json.Unmarshal([]byte(cuscn), &datacn)
		// Ignore bad inputs
		if errcn == nil {
			// Fill in company id
			for i, _ := range datacn {
				datacn[i].Language = "zh-CN"
			}
		}

		if err == nil || errcn == nil {
			// Concatenate the two update parts
			data = append(data, datacn...)

			// Full diff comparison would require more intricate protocols
			// which we do not have the needs (resources) to do right now
			// So we always do a full replace
			var updates []CompanyUpdate
			if dbConn.Model(&c).Related(&updates).Error == nil {
				for _, update := range updates {
					// Replace same language, keep all others
					if (update.Language == "en-US" && err == nil) ||
						(update.Language == "zh-CN" && errcn == nil) {
						dbConn.Delete(&update)
					} else {
						data = append(data, update)
					}
				}
			}
			if dbConn.Model(&c).Association("CompanyUpdates").
				Replace(&data).Error != nil {
				formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
				return
			}
		}
	}

	// Save all changes
	if dbConn.Save(&c).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminCompanyAddHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	name, of := CheckFieldForm("", r, "name")
	yearFounded, of := CheckRangeForm(of, r, "year_founded",
		YearMax)
	stateFounded, of := CheckFieldForm(of, r, "state_founded")
	numEmployees, of := CheckRangeForm(of, r, "num_employees",
		NumberMax)
	totalValuation, of := CheckFloatForm(of, r, "total_valuation",
		true)
	totalFunding, of := CheckFloatForm(of, r, "total_funding",
		true)
	homePage, of := CheckFieldForm(of, r, "home_page")
	hq, of := CheckFieldForm(of, r, "hq")
	description, of := CheckLengthForm(of, r, "description",
		1, StringBlockMax)
	descriptionCn, of := CheckLengthForm(of, r, "description_cn",
		1, StringBlockMax)
	ilps, of := CheckLengthForm(of, r, "investor_logo_pics",
		1, StringPlusMax)
	numSlides, of := CheckRangeForm(of, r, "num_slides",
		NumberMax)
	videoUrl, of := CheckFieldForm(of, r, "video_url")
	keyPerson, of := CheckFieldForm(of, r, "key_person")
	grp, of := CheckFloatForm(of, r, "growth_rate_percent", false)
	sizeMultiple, of := CheckFloatForm(of, r, "size_multiple", true)
	investors, of := CheckLengthForm(of, r, "investors",
		1, StringBlockMax)
	tagss, of := CheckLengthForm(of, r, "tags", 1, JsonMax)
	execs, of := CheckLengthForm(of, r, "executives", 1, JsonMax)
	funds, of := CheckLengthForm(of, r, "fundings", 1, JsonMax)
	cus, of := CheckLengthForm(of, r, "updates", 1, JsonMax)
	cuscn, of := CheckLengthForm(of, r, "updates_cn", 1, JsonMax)

	// All must be passed in here, even if empty
	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	tss := strings.Split(tagss, ",")
	// Must be at least 3 to be valid
	if len(tss) < 3 {
		formatReturn(w, r, ps, ErrorCodeAdminCompanyTagsError, true, nil)
		return
	}
	var tags []Tag
	for i := 0; i < len(tss); i++ {
		tid, err := strconv.ParseUint(tss[i], 10, 64)
		if err != nil {
			continue
		}
		var tag Tag
		// Skip on error parsing
		if dbConn.First(&tag, tid).RecordNotFound() {
			continue
		}
		tags = append(tags, tag)
	}
	// Must read at least 3 tags
	if len(tags) < 3 {
		formatReturn(w, r, ps, ErrorCodeAdminCompanyTagsError, true, nil)
		return
	}

	// TODO: Check for strict inputs
	var dataExecs []CompanyExecutive
	if json.Unmarshal([]byte(execs), &dataExecs) != nil {
		formatReturn(w, r, ps, ErrorCodeAdminCompanyExecutivesError, true, nil)
		return
	}

	// TODO: Check for strict inputs
	var dataFunds []Funding
	if json.Unmarshal([]byte(funds), &dataFunds) != nil ||
		len(dataFunds) < 1 {
		formatReturn(w, r, ps, ErrorCodeAdminCompanyFundingsError, true, nil)
		return
	}
	// Update linked company information
	totalValuation = dataFunds[0].PostValuation
	totalFunding = dataFunds[0].RaisedToDate

	// TODO: Check for strict inputs
	// Do both languages in one try
	var dataUpdates, datacnUpdates []CompanyUpdate
	if json.Unmarshal([]byte(cus), &dataUpdates) != nil ||
		json.Unmarshal([]byte(cuscn), &datacnUpdates) != nil {
		formatReturn(w, r, ps, ErrorCodeAdminCompanyUpdatesError, true, nil)
		return
	}
	// Fill in language
	for i, _ := range dataUpdates {
		dataUpdates[i].Language = "en-US"
	}
	for i, _ := range datacnUpdates {
		datacnUpdates[i].Language = "zh-CN"
	}
	// Concatenate the two update parts
	dataUpdates = append(dataUpdates, datacnUpdates...)

	c := Company{
		Name:              name,
		YearFounded:       yearFounded,
		StateFounded:      stateFounded,
		NumEmployees:      numEmployees,
		TotalValuation:    totalValuation,
		TotalFunding:      totalFunding,
		HomePage:          homePage,
		Hq:                hq,
		Description:       description,
		DescriptionCn:     descriptionCn,
		Tags:              tags,
		InvestorLogoPics:  ilps,
		NumSlides:         numSlides,
		VideoUrl:          videoUrl,
		CompanyExecutives: dataExecs,
		Fundings:          dataFunds,
		CompanyUpdates:    dataUpdates,
		KeyPerson:         keyPerson,
		GrowthRatePercent: grp,
		SizeMultiple:      sizeMultiple,
		Investors:         investors,
	}

	// Due to strict id picking when first data were imported,
	// auto-incremented id is not available here, so we always insert
	// as one that's +1 from last
	var lc Company
	if dbConn.Unscoped().Last(&lc).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	c.ID = lc.ID + 1
	if dbConn.Create(&c).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, map[string]interface{}{"id": c.ID})
}

func adminCompanyLogoUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	cid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	if dbConn.First(&Company{}, cid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	// Parse and save logo file to fixed location
	err = saveFileUpload(r, "logo",
		path.Join(clientDir, "images",
			"company", fmt.Sprintf("%v", cid)), "logo.png")
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminCompanyBgUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	cid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	if dbConn.First(&Company{}, cid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	// Parse and save logo file to fixed location
	err = saveFileUpload(r, "bg",
		path.Join(clientDir, "images",
			"company", fmt.Sprintf("%v", cid)), "bg.jpg")
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminCompanySlideUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User, lang string) {
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

	sid, err := strconv.ParseUint(ps.ByName("slide_id"), 10, 64)
	if err != nil || sid >= c.NumSlides {
		formatReturn(w, r, ps, ErrorCodeCompanySlideUnknown, true, nil)
		return
	}

	// Parse and save logo file to fixed location
	err = saveFileUpload(r, "slide",
		path.Join(clientDir, companySecret, fmt.Sprintf("%v", cid),
			fmt.Sprintf("slide_%v", lang)),
		fmt.Sprintf("slide.%03d.png", sid))
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminCompanySlideEnUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	adminCompanySlideUpdateHandler(w, r, ps, u, "en")
}

func adminCompanySlideZhUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	adminCompanySlideUpdateHandler(w, r, ps, u, "zh")
}

func adminCompanyTagUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	tid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyTagUnknown, true, nil)
		return
	}

	var t Tag
	if dbConn.First(&t, tid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeCompanyTagUnknown, true, nil)
		return
	}

	name, ok := CheckField(true, r.FormValue("name"))
	if ok {
		t.Name = name
	}
	nameCn, ok := CheckField(true, r.FormValue("name_cn"))
	if ok {
		t.NameCn = nameCn
	}

	if dbConn.Save(&t).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, nil)
}

func adminCompanyTagAddHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	name, of := CheckFieldForm("", r, "name")
	nameCn, of := CheckFieldForm(of, r, "name_cn")

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	// Due to strict id picking when first data were imported,
	// auto-incremented id is not available here, so we always insert
	// as one that's +1 from last
	var lt Tag
	if dbConn.Unscoped().Last(&lt).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	t := Tag{Name: name, NameCn: nameCn}
	t.ID = lt.ID + 1
	if dbConn.Create(&t).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	saveAdmin(w, r, ps, u, map[string]interface{}{"id": t.ID})
}

func adminCompanyDeleteHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	cid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	if dbConn.Delete(&Company{}, cid).Error != nil {
		formatReturn(w, r, ps, ErrorCodeCompanyUnknown, true, nil)
		return
	}

	// Success
	saveAdmin(w, r, ps, u, nil)
}
