// Router branch for /admin/user/ operations
package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"path"
	"strconv"
)

func adminUsersHandler(w http.ResponseWriter, r *http.Request,
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

	var us []User
	if dbConn.Where("full_name ~* ?", r.FormValue("keyword")).
		Order("id desc").Offset(int(pnum*psize)).Limit(int(psize)).
		Find(&us).Error != nil {
		// Probably bad inputs
		formatReturn(w, r, ps, ErrorCodeAdminPageError, true, nil)
		return
	}

	users := []map[string]interface{}{}
	for _, user := range us {
		users = append(users, map[string]interface{}{
			"id":           user.ID,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"email":        user.Email,
			"email_token":  user.EmailToken,
			"role_type":    user.RoleType,
			"phone_number": user.PhoneNumber,
			"user_state":   user.UserState,
			"citizen_type": user.CitizenType,
			"created":      user.CreatedAt.Unix(),
		})
	}

	// Return all satisfied users
	saveAdmin(w, r, ps, u, map[string]interface{}{"users": users})
}

func adminUserIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	uid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	// Check if user is valid
	var user User
	if dbConn.First(&user, uid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	// If user does not have a valid photo id token yet, generate it
	if uint64(len(user.PhotoIDToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "user", fmt.Sprintf("%v", user.ID),
			"photo_id")
		if _, err := os.Stat(pifc); err == nil {
			user.PhotoIDToken, user.PhotoIDName =
				createFileTokenName("photo_id", user.PhotoIDType)
			// Ignore saving errors just don't return
			dbConn.Save(user)
		}
	}

	// If user does not have a valid photo id token yet, generate it
	if uint64(len(user.BusinessCardToken)) != TokenMinMax {
		pifc := path.Join(dataDir, "user", fmt.Sprintf("%v", user.ID),
			"business_card")
		if _, err := os.Stat(pifc); err == nil {
			user.BusinessCardToken, user.BusinessCardName =
				createFileTokenName("business_card", user.BusinessCardType)
			// Ignore saving errors just don't return
			dbConn.Save(user)
		}
	}

	// Return all allowed information about user
	saveAdmin(w, r, ps, u, map[string]interface{}{
		"user_state":                  user.UserState,
		"user_level":                  user.UserLevel,
		"role_type":                   user.RoleType,
		"first_name":                  user.FirstName,
		"last_name":                   user.LastName,
		"email":                       user.Email,
		"dob":                         user.Dob,
		"phone_number":                user.PhoneNumber,
		"address1":                    user.Address1,
		"address2":                    user.Address2,
		"city":                        user.City,
		"state":                       user.State,
		"zip":                         user.Zip,
		"country":                     user.Country,
		"citizen_type":                user.CitizenType,
		"ssn":                         decField(user.SsnEncrypted),
		"photo_id":                    user.PhotoIDToken,
		"photo_id_filename":           user.PhotoIDName,
		"business_card":               user.BusinessCardToken,
		"business_card_filename":      user.BusinessCardName,
		"employment_type":             user.EmploymentType,
		"employer":                    user.Employer,
		"occupation":                  user.Occupation,
		"public_company_policy_maker": user.PublicCompanyPolicyMaker,
		"employed_by_broker_dealer":   user.EmployedByBrokerDealer,
		"risk_tolerance":              user.RiskTolerance,
		"marital_status":              user.MaritalStatus,
		"household_income":            user.HouseholdIncome,
		"household_networth":          user.HouseholdNetworth,
		"invest_portfolio_total":      user.InvestPortfolioTotal,
		"invest_exp_total":            user.InvestExpTotal,
		"invest_real_estate_portion":  user.InvestRealEstatePortion,
		"invest_convert_cash_ninety_days_portion": user.
			InvestConvertCashNinetyDaysPortion,
		"invest_alternative_portion":   user.InvestAlternativePortion,
		"invest_private_company":       user.InvestPrivateCompany,
		"invest_portfolio_horizon":     user.InvestPortfolioHorizon,
		"education_level":              user.EducationLevel,
		"invest_knowledge":             user.InvestKnowledge,
		"work_with_financial_advisors": user.WorkWithFinancialAdvisors,
		"investor_type":                user.InvestorType,
		"investor_situation":           user.InvestorSituation,
	})
}

func adminUserUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	uid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	// Check if user is valid
	var user User
	if dbConn.First(&user, uid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	// Admin has full power over state changing even if illogical
	_, ok := userKycFieldCheck(r, &user, false, false)
	if !ok {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	// Change user-specific states
	userLevel, ok := CheckRange(true, r.FormValue("user_level"),
		UserLevelAdmin)
	if ok {
		user.UserLevel = userLevel
	}
	roleType, ok := CheckRange(true, r.FormValue("role_type"),
		RoleTypeInvestor)
	if ok {
		user.RoleType = roleType
	}
	userState, ok := CheckRange(true, r.FormValue("user_state"),
		UserStateBanned)
	ua := false
	if ok {
		// Strict user state checking after dashboard
		// to make sure admin does not do anything out of ordinary
		// In the future this should vary with user level, i.e.
		// the level of admin priviledge
		ua = userState != user.UserState &&
			userState >= UserStateActive &&
			userState <= UserStateActiveAccredId
		if ua &&
			((user.CitizenType == CitizenTypeOther &&
				((user.RoleType == RoleTypeInvestor &&
					userState != UserStateActiveAccredId) ||
					(user.RoleType == RoleTypeShareholder &&
						userState != UserStateActiveId))) ||
				(user.CitizenType != CitizenTypeOther &&
					((user.RoleType == RoleTypeInvestor &&
						userState != UserStateActiveAccred) ||
						(user.RoleType == RoleTypeShareholder &&
							userState != UserStateActive)))) {
			formatReturn(w, r, ps, ErrorCodeAdminUserStateError,
				true, nil)
			return
		}
		user.UserState = userState
	}

	// Modify other existing fields
	firstName, fok := CheckField(true, r.FormValue("first_name"))
	if fok {
		user.FirstName = firstName
	}
	lastName, lok := CheckField(true, r.FormValue("last_name"))
	if lok {
		user.LastName = lastName
	}
	// Update full name if either first or last name changed
	if fok || lok {
		user.FullName = NameConventions[user.CitizenType](user.FirstName,
			user.LastName)
	}

	investorType, ok := CheckRange(true, r.FormValue("investor_type"),
		InvestorTypeAdvisor)
	if ok {
		user.InvestorType = investorType
	}
	is := InvestorSituationUSBusinessIndividualAbove
	if user.CitizenType == CitizenTypeOther {
		is = InvestorSituationCNBusinessIndividualAbove
	}
	investorSituation, ok := CheckRange(true,
		r.FormValue("investor_situation"), uint64(is))
	if ok {
		user.InvestorSituation = investorSituation
	}

	// Save however many changed
	if dbConn.Save(&user).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	// Send approval notification if necessary
	if ua {
		// Notify user of the account information
		if user.LastLanguage != "en-US" &&
			user.LastLanguage != "zh-CN" {
			user.LastLanguage = "en-US"
		}

		var subject, body string
		if user.RoleType == RoleTypeInvestor {
			subject =
				EmailTexts[user.
					LastLanguage][EmailTextSubjectInvestorApproved]
			body = fmt.Sprintf(
				EmailTexts[user.
					LastLanguage][EmailTextBodyInvestorApproved],
				user.FullName, fmt.Sprintf(emailInvestorLink, serverDomain))
		} else if user.RoleType == RoleTypeShareholder {
			subject =
				EmailTexts[user.
					LastLanguage][EmailTextSubjectShareholderApproved]
			body = fmt.Sprintf(
				EmailTexts[user.
					LastLanguage][EmailTextBodyShareholderApproved],
				user.FullName, fmt.Sprintf(emailShareholderLink, serverDomain))
		}

		author := EmailTexts[user.LastLanguage][EmailTextName]
		go sendMail(author, user.Email, user.FullName, subject, body)
	}

	// Make sure we refresh current state
	if u.ID == user.ID {
		u = &user
	}
	saveAdmin(w, r, ps, u, nil)
}

func adminUserPhotoIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	uid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		return
	}

	// Check if user is valid
	var user User
	if dbConn.First(&user, uid).RecordNotFound() {
		return
	}

	pifc := path.Join(dataDir, "user", fmt.Sprintf("%v", user.ID), "photo_id")
	parseFileDownload(w, r, user.PhotoIDType, pifc)
}

func adminUserPhotoIdTokenHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params) {
	token, ok := CheckLength(true, ps.ByName("token"),
		TokenMinMax, TokenMinMax)
	if !ok {
		return
	}

	// Check token validity
	var user User
	if dbConn.First(&user, "photo_id_token = ?", token).RecordNotFound() {
		return
	}

	pifc := path.Join(dataDir, "user", fmt.Sprintf("%v", user.ID), "photo_id")
	parseFileDownload(w, r, user.PhotoIDType, pifc)
}

func adminUserPhotoIdUploadHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	uid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	// Check if user is valid
	var user User
	if dbConn.First(&user, uid).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	// Parse photo id
	furl, _, ftype, err := parseFileUpload(r, "photo_id", "user",
		user.ID, "photo_id")
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}
	user.PhotoIDPic = furl
	user.PhotoIDType = ftype
	user.PhotoIDToken, user.PhotoIDName =
		createFileTokenName("photo_id", ftype)

	if dbConn.Save(&user).Error != nil {
		formatReturn(w, r, ps, ErrorCodeAdminError, true, nil)
		return
	}

	// Make sure we refresh current state
	if u.ID == user.ID {
		u = &user
	}
	saveAdmin(w, r, ps, u, nil)
}

func adminUserDeleteHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	uid, err := strconv.ParseUint(ps.ByName("id"), 10, 64)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	if dbConn.Delete(&User{}, uid).Error != nil {
		formatReturn(w, r, ps, ErrorCodeUserUnknown, true, nil)
		return
	}

	// Success
	saveAdmin(w, r, ps, u, nil)
}
