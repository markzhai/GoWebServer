// Router branch for /user/ operations
package main

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
)

func userNdaHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// State must be after confirmed and accred if investor
	if (u.RoleType == RoleTypeShareholder && u.UserState < UserStateConfirmed) ||
		(u.RoleType == RoleTypeInvestor && u.UserState < UserStateAccred) {
		formatReturn(w, r, ps, ErrorCodeNdaError, true, nil)
		return
	}

	// Advance user state if necessary
	if u.UserState < UserStateNdaAgreed {
		u.UserState = UserStateNdaAgreed
		if dbConn.Save(u).Error != nil {
			formatReturn(w, r, ps, ErrorCodeNdaError, true, nil)
			return
		}
	}

	saveLogin(w, r, ps, false, u, nil)
}

// userInvestorFieldCheck takes the common checking of investor-related fields
// and returns whether all fields are accounted for
// Caller has to do its own state and permission checking
func userInvestorFieldCheck(r *http.Request, u *User) string {
	allof := ""
	employmentType, of := CheckRangeForm("", r, "employment_type",
		EmploymentTypeUnemployed)
	if of == "" {
		u.EmploymentType = employmentType
	} else if allof == "" {
		allof = of
	}
	employer, of := CheckFieldForm("", r, "employer")
	if of == "" {
		u.Employer = employer
	} else if allof == "" {
		allof = of
	}
	occupation, of := CheckFieldForm("", r, "occupation")
	if of == "" {
		u.Occupation = occupation
	} else if allof == "" {
		allof = of
	}
	pcpm, of := CheckRangeForm("", r,
		"public_company_policy_maker", NoYesMax)
	if of == "" {
		u.PublicCompanyPolicyMaker = pcpm
	} else if allof == "" {
		allof = of
	}
	ebbd, of := CheckRangeForm("", r, "employed_by_broker_dealer",
		NoYesMax)
	if of == "" {
		u.EmployedByBrokerDealer = ebbd
	} else if allof == "" {
		allof = of
	}
	riskTolerance, of := CheckRangeForm("", r, "risk_tolerance",
		RiskToleranceAggressive)
	if of == "" {
		u.RiskTolerance = riskTolerance
	} else if allof == "" {
		allof = of
	}
	maritalStatus, of := CheckRangeForm("", r, "marital_status",
		MaritalStatusMarried)
	if of == "" {
		u.MaritalStatus = maritalStatus
	} else if allof == "" {
		allof = of
	}
	householdIncome, of := CheckRangeForm("", r,
		"household_income", HouseholdIncomeMore300K)
	if of == "" {
		u.HouseholdIncome = householdIncome
	} else if allof == "" {
		allof = of
	}
	householdNetworth, of := CheckRangeForm("", r,
		"household_networth", HouseholdNetworthMore5M)
	if of == "" {
		u.HouseholdNetworth = householdNetworth
	} else if allof == "" {
		allof = of
	}
	ipt, of := CheckRangeForm("", r, "invest_portfolio_total",
		NumberMax)
	if of == "" {
		u.InvestPortfolioTotal = ipt
	} else if allof == "" {
		allof = of
	}
	iet, of := CheckRangeForm("", r, "invest_exp_total",
		NumberMax)
	if of == "" {
		u.InvestExpTotal = iet
	} else if allof == "" {
		allof = of
	}
	irep, of := CheckRangeForm("", r,
		"invest_real_estate_portion", PercentMax)
	if of == "" {
		u.InvestRealEstatePortion = irep
	} else if allof == "" {
		allof = of
	}
	iccndp, of := CheckRangeForm("", r,
		"invest_convert_cash_ninety_days_portion",
		PercentMax)
	if of == "" {
		u.InvestConvertCashNinetyDaysPortion = iccndp
	} else if allof == "" {
		allof = of
	}
	iap, of := CheckRangeForm("", r, "invest_alternative_portion",
		PercentMax)
	if of == "" {
		u.InvestAlternativePortion = iap
	} else if allof == "" {
		allof = of
	}
	ipc, of := CheckRangeForm("", r, "invest_private_company",
		NoYesMax)
	if of == "" {
		u.InvestPrivateCompany = ipc
	} else if allof == "" {
		allof = of
	}
	iph, of := CheckRangeForm("", r, "invest_portfolio_horizon",
		InvestPortfolioHorizonMore10)
	if of == "" {
		u.InvestPortfolioHorizon = iph
	} else if allof == "" {
		allof = of
	}
	educationLevel, of := CheckRangeForm("", r, "education_level",
		EducationLevelGraduate)
	if of == "" {
		u.EducationLevel = educationLevel
	} else if allof == "" {
		allof = of
	}
	investKnowledge, of := CheckRangeForm("", r,
		"invest_knowledge", InvestKnowledgeExpert)
	if of == "" {
		u.InvestKnowledge = investKnowledge
	} else if allof == "" {
		allof = of
	}
	wwfa, of := CheckRangeForm("", r,
		"work_with_financial_advisors", NoYesMax)
	if of == "" {
		u.WorkWithFinancialAdvisors = wwfa
	} else if allof == "" {
		allof = of
	}

	return allof
}

// userKycFieldCheck takes the common checking of the kyc-related fields
// and returns whether all fields are accounted for
// This is a nice helper to various parts of updating functions
// Returns a tuple (invalid parameter, has error)
func userKycFieldCheck(r *http.Request, u *User,
	ctForce bool, ctImmutable bool) (string, bool) {
	// We save as many valid fields as possible
	allof := ""

	dob, of := CheckDateForm("", r, "dob")
	if of == "" {
		u.Dob = dob
	} else if allof == "" {
		allof = of
	}

	phoneNumber, of := CheckLengthForm("", r, "phone_number", PhoneMin, PhoneMax)
	if of == "" {
		u.PhoneNumber = phoneNumber
	} else if allof == "" {
		allof = of
	}

	address1, of := CheckFieldForm("", r, "address1")
	if of == "" {
		u.Address1 = address1
	} else if allof == "" {
		allof = of
	}

	// optional
	address2, of := CheckFieldForm("", r, "address2")
	if of == "" {
		u.Address2 = address2
	}
	firstName, of := CheckFieldForm("", r, "first_name")
	if of == "" {
		u.FirstName = firstName
	}
	lastName, of := CheckFieldForm("", r, "last_name")
	if of == "" {
		u.LastName = lastName
	}
	u.FullName = NameConventions[u.CitizenType](firstName, lastName)

	city, of := CheckFieldForm("", r, "city")
	if of == "" {
		u.City = city
	} else if allof == "" {
		allof = of
	}
	zip, of := CheckLengthForm("", r, "zip", ZipMin, ZipMax)
	if of == "" {
		u.Zip = zip
	} else if allof == "" {
		allof = of
	}
	country, of := CheckLengthForm("", r, "country",
		CountryMinMax, CountryMinMax)
	if of == "" {
		u.Country = country
	} // country is optional
	citizenType, of := CheckRangeForm("", r, "citizen_type",
		CitizenTypeOther)
	// Special field controlled by ctImmutable and ctForce
	if of == "" {
		// Citizen type cannot be changed
		if ctImmutable {
			return "", false
		}
		u.CitizenType = citizenType
	} else {
		// Must have citizen type
		if ctForce {
			return "", false
		}
		citizenType = u.CitizenType
	}
	// Only US residents have SSNs
	if citizenType != CitizenTypeOther {
		ssn, of := CheckSSNForm("", r, "ssn")
		if of == "" {
			u.SsnEncrypted = encField(ssn)
		} else if allof == "" {
			allof = of
		}
	}

	state, of := CheckFieldForm("", r, "state")
	// Check US state strictly
	if citizenType != CitizenTypeOther {
		state = formatUsState(state)
		if state == "" {
			of = "state"
		}
	}
	if of == "" {
		u.State = state
	} else if allof == "" {
		allof = of
	}

	password, of := CheckLengthForm("", r, "password", PasswordMinMax, PasswordMinMax)
	if of == "" {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password),
			bcrypt.DefaultCost)
		// Return unknown register error here
		if err != nil {
			of = err.Error()
			allof = of
		} else {
			u.PasswordHash = string(passwordHash)
		}
	} else if allof == "" {
		allof = of
	}

	if u.CitizenType == CitizenTypeOther {
		idCardNumber, of := CheckIdCardForm(of, r, "id_card_number")
		if of == "" {
			u.IDCardNumber = idCardNumber
		} else if allof == "" {
			//formatReturn(w, r, ps, ErrorCodeIdCardInvalid, false, nil)
			allof = of
		}
    weixin, of := CheckFieldForm(of, r, "weixin")
    if of == "" {
      u.Weixin = weixin
    } else if allof == "" {
      allof = of
    }
    employer, of := CheckFieldForm(of, r, "employer")
    if of == "" {
      u.Employer = employer
    } else if allof == "" {
      allof = of
    }
    occupation, of := CheckFieldForm("", r, "occupation")
    if of == "" {
      u.Occupation = occupation
    } else if allof == "" {
      allof = of
    }
	}

	// Check common investor fields if we are at an investor stage
	// or shareholder with enough power to switch
	if (u.RoleType == RoleTypeInvestor && u.UserState >= UserStateAccred) ||
		(u.RoleType == RoleTypeShareholder && u.UserState >= UserStateActive) {
		// Must check for side effects (partial edits)
		invof := userInvestorFieldCheck(r, u)
		if allof == "" {
			allof = invof
		}
	}

	return allof, true
}

const registerConfirmEmailTemplate = `Hello MarketX,
<br>
<p>The following person has requested access to our www.themarketx.com site:</p>
<p style="line-height:1.38;margin-top:0pt;margin-bottom:0pt">- First Name: %v</p>
<p style="line-height:1.38;margin-top:0pt;margin-bottom:0pt">- Last Name: %v</p>
<p style="line-height:1.38;margin-top:0pt;margin-bottom:0pt">- Email: %v</p>
<p>- Phone Number: %v</p>
<p>Please review their request and take the appropriate action next.</p>
<p style="line-height:1.38;margin-top:0pt;margin-bottom:0pt">Thank you!</p>
<p style="line-height:1.38;margin-top:0pt;margin-bottom:0pt">MarketX</p>
`

func userKycHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// State must be after nda
	if u.UserState < UserStateNdaAgreed {
		formatReturn(w, r, ps, ErrorCodeKycError, true, nil)
		return
	}

	// CitizenType cannot be changed!!! Only admin should do that
	allof, ok := userKycFieldCheck(r, u, false, true)
	if !ok {
		formatReturn(w, r, ps, ErrorCodeKycError, true, nil)
		return
	}

	// Advance state based on citizen type
	if allof == "" {
		if u.CitizenType == CitizenTypeOther &&
			u.UserState < UserStateKycWaitingId {
			u.UserState = UserStateKycWaitingId
		} else if u.CitizenType != CitizenTypeOther &&
			u.UserState < UserStateKycWaitingQuestions {
			u.UserState = UserStateKycWaitingQuestions
		}
	}

	// Save information even on error
	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeKycError, true, nil)
		return
	}

	if allof != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, allof, true, nil)
		return
	}

	message := fmt.Sprintf(registerConfirmEmailTemplate, u.FirstName, u.LastName, u.Email, u.PhoneNumber)

	go sendMail(supportEmailUsername, supportEmail, supportEmailUsername, "New Request Access on themarketx.com", message)
	go sendMail(supportEmailUsername, "han.lai@themarketx.com", supportEmailUsername, "New Request Access on themarketx.com", message)
	saveLogin(w, r, ps, false, u, nil)
}

func userKycQuestionsHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// State must be after saving kyc information
	if u.UserState < UserStateKycWaitingQuestions ||
		u.UserState == UserStateKycFailed {
		formatReturn(w, r, ps, ErrorCodeKycGetQuestions, true, nil)
		return
	}

	// TODO: Should probably enforce a timer for any third party services
	// instead of relying on the http request timeout
	// Since we are already in a goroutine having nothing else to do
	// it's not wise to "go" again
	// NB: On development we use the dummy user to quickly show the
	// demo without going through the real pain
	var kycUser *User
	if environment != "production" {
		// Make sure we don't contaminate the dummy user
		cloneUser := DummyUser
		kycUser = &cloneUser
	} else {
		kycUser = u
	}

	// Step 1: setup investor record
	err := serverTransact.CreateInvestorRecord(kycUser)
	if err != nil {
		ec := ErrorCodeKycRecordError
		// Ineligible for kyc (hard fail, should be more specific)
		if err == TransactErrorCall {
			ec = ErrorCodeKycHardFail
		}
		formatReturn(w, r, ps, ec, true, nil)
		return
	}

	// Step 2: setup investor account: get questions!
	ques, err := serverTransact.CreateInvestorAccount(kycUser)
	if err != nil {
		ec := ErrorCodeKycAccountError
		// Ineligible for kyc (hard fail, should be more specific)
		if err == TransactErrorCall {
			ec = ErrorCodeKycHardFail
		}
		formatReturn(w, r, ps, ec, true, nil)
		return
	}

	// Save information
	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeKycGetQuestions, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u,
		map[string]interface{}{"questions": ques})
}

func userKycCheckHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// State must be after saving kyc information
	if u.UserState < UserStateKycWaitingQuestions ||
		u.UserState == UserStateKycFailed {
		formatReturn(w, r, ps, ErrorCodeKycError, true, nil)
		return
	}

	// 5 question/answer pairs at most
	// Must not skip numbering
	ans := map[string]string{}

	type1, ok := CheckField(true, r.FormValue("type1"))
	ans1, ok := CheckField(ok, r.FormValue("ans1"))
	// No question is available is an error, if at least one,
	// then there is at least a list
	if !ok {
		formatReturn(w, r, ps, ErrorCodeKycCheckError, true, nil)
		return
	} else {
		ans["type1"] = type1
		ans["ans1"] = ans1
	}
	type2, ok := CheckField(ok, r.FormValue("type2"))
	ans2, ok := CheckField(ok, r.FormValue("ans2"))
	if ok {
		ans["type2"] = type2
		ans["ans2"] = ans2
	}
	type3, ok := CheckField(ok, r.FormValue("type3"))
	ans3, ok := CheckField(ok, r.FormValue("ans3"))
	if ok {
		ans["type3"] = type3
		ans["ans3"] = ans3
	}
	type4, ok := CheckField(ok, r.FormValue("type4"))
	ans4, ok := CheckField(ok, r.FormValue("ans4"))
	if ok {
		ans["type4"] = type4
		ans["ans4"] = ans4
	}
	type5, ok := CheckField(ok, r.FormValue("type5"))
	ans5, ok := CheckField(ok, r.FormValue("ans5"))
	if ok {
		ans["type5"] = type5
		ans["ans5"] = ans5
	}

	var fail ErrorCode = ErrorCodeNone
	// Check KYC questions
	err := serverTransact.KycStatus(u, ans)
	if err != nil {
		fail = ErrorCodeKycCheckQuestions
	}

	// Jump state on success
	if fail == ErrorCodeNone {
		// Shareholder goes to active
		if u.RoleType == RoleTypeShareholder &&
			u.UserState < UserStateActive {
			u.UserState = UserStateActive
		}
		// Investor goes to active + passed accred
		if u.RoleType == RoleTypeInvestor &&
			u.UserState < UserStateActiveAccred {
			u.UserState = UserStateActiveAccred
		}
	}

	// Even if failed states can be updated and returned
	if dbConn.Save(u).Error != nil {
		fail = ErrorCodeKycCheckError
	}

	if fail != ErrorCodeNone {
		// Normally we don't return user_state and user_level but since
		// kycstatus can be "failed but changed state" so we need to
		// return more information here
		formatReturn(w, r, ps, fail, true,
			map[string]interface{}{
				"user_state": u.UserState,
				"user_level": u.UserLevel,
				"role_type":  u.RoleType,
				"attempts":   u.TransactApiKycAttempts})
		return
	}

	// The only one OK return which advances user state
	saveLogin(w, r, ps, false, u, nil)
}

func userUploadIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// State must be after kyc basic info for non-US
	// and after active for US
	if (u.CitizenType == CitizenTypeOther &&
		u.UserState < UserStateKycWaitingId) ||
		(u.CitizenType != CitizenTypeOther &&
			u.UserState < UserStateActive) {
		formatReturn(w, r, ps, ErrorCodeUploadIdError, true, nil)
		return
	}

	// Parse photo id
	furl, _, ftype, err := parseFileUpload(r, "photo_id", "user",
		u.ID, "photo_id")
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeUploadIdError, true, nil)
		return
	}
	u.PhotoIDPic = furl
	u.PhotoIDType = ftype
	u.PhotoIDToken, u.PhotoIDName = createFileTokenName("photo_id", ftype)

	// Check and set correct state
	if u.CitizenType == CitizenTypeOther {
		if u.UserState < UserStateKycWaitingApproval {
			u.UserState = UserStateKycWaitingApproval
		}
	} else {
		// Do exact state transitions, all other states don't need
		// further changes (simply updating id)
		if u.UserState == UserStateActive {
			u.UserState = UserStateActiveId
		} else if u.UserState == UserStateActiveAccred {
			u.UserState = UserStateActiveAccredId
		}
	}

	// Only change state if changed
	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeUploadIdError, true, nil)
		return
	}

	// Return the good link relative path
	saveLogin(w, r, ps, false, u, nil)
}

func userUploadBusinessCardHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// Parse photo id
	furl, _, ftype, err := parseFileUpload(r, "business_card", "user",
		u.ID, "business_card")
	if err != nil {
		formatReturn(w, r, ps, ErrorCodeUploadIdError, true, nil)
		return
	}
	u.BusinessCardPic = furl
	u.BusinessCardType = ftype
	u.BusinessCardToken, u.BusinessCardName = createFileTokenName("photo_id", ftype)

	// Only change state if changed
	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeUploadIdError, true, nil)
		return
	}

	// Return the good link relative path
	saveLogin(w, r, ps, false, u, nil)
}

func userSelfAccredHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// Only support investor sign up which is before nda
	if u.RoleType != RoleTypeInvestor ||
		u.UserState < UserStateConfirmed {
		formatReturn(w, r, ps, ErrorCodeSelfAccredError, true, nil)
		return
	}

	// Both must be present since this is a short form
	investorType, ok := CheckRange(true, r.FormValue("investor_type"),
		InvestorTypeAdvisor)
	is := InvestorSituationUSBusinessIndividualAbove
	if u.CitizenType == CitizenTypeOther {
		is = InvestorSituationCNBusinessIndividualAbove
	}
	investorSituation, ok := CheckRange(ok,
		r.FormValue("investor_situation"), uint64(is))

	if !ok {
		formatReturn(w, r, ps, ErrorCodeSelfAccredError, true, nil)
		return
	}

	u.InvestorType = investorType
	u.InvestorSituation = investorSituation
	// Change state if first time
	if u.UserState < UserStateAccred {
		u.UserState = UserStateAccred
	}

	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeSelfAccredError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u, nil)
}

func userSelfAccredSwitchHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// Only support shareholder investing switch, which is after active
	if u.RoleType != RoleTypeShareholder ||
		u.UserState < UserStateActive {
		formatReturn(w, r, ps, ErrorCodeSelfAccredSwitchError, true, nil)
		return
	}

	// Both must be present since this is a short form
	investorType, of := CheckRangeForm("", r, "investor_type",
		InvestorTypeAdvisor)
	is := InvestorSituationUSBusinessIndividualAbove
	if u.CitizenType == CitizenTypeOther {
		is = InvestorSituationCNBusinessIndividualAbove
	}
	investorSituation, of := CheckRangeForm(of, r,
		"investor_situation", uint64(is))
	// Check other fields if the above passed
	if of == "" {
		of = userInvestorFieldCheck(r, u)
	}

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	u.InvestorType = investorType
	u.InvestorSituation = investorSituation
	// Other fields saved within userInvestorFieldCheck call

	// Now check based on state transitions
	if u.UserState == UserStateActive {
		u.UserState = UserStateActiveAccred
	} else if u.UserState == UserStateActiveId {
		u.UserState = UserStateActiveAccredId
	} else if u.UserState == UserStateConfirmed {
		u.UserState = UserStateAccred
	}

	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeSelfAccredSwitchError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u, nil)
}

func userHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// State must be after confirmed
	if u.UserState < UserStateConfirmed {
		formatReturn(w, r, ps, ErrorCodeUserNotOnboard, true, nil)
		return
	}

	// Return info based on user state process steps
	// The very basic information
	info := map[string]interface{}{
		"first_name":   u.FirstName,
		"last_name":    u.LastName,
		"email":        u.Email,
		"phone_number": u.PhoneNumber,
		"role_type":    u.RoleType, // overlap, but ok
		"citizen_type": u.CitizenType,
	}

	// Investor-specific questions
	if u.UserState >= UserStateAccred && u.RoleType == RoleTypeInvestor {
		info["investor_type"] = u.InvestorType
		info["investor_situation"] = u.InvestorSituation
	}

	// Kyc questions
	if u.UserState >= UserStateKycWaitingQuestions {
		if u.CitizenType != CitizenTypeOther &&
			u.UserState == UserStateKycWaitingQuestions {
			// Return initialized value if have not been called before
			if u.TransactApiKycAttempts == 0 {
				info["attempts"] = 3
			} else {
				info["attempts"] = u.TransactApiKycAttempts
			}
		}
		info["dob"] = u.Dob
		info["phone_number"] = u.PhoneNumber
		info["address1"] = u.Address1
		info["address2"] = u.Address2
		info["city"] = u.City
		info["state"] = u.State
		info["zip"] = u.Zip
		info["country"] = u.Country
		if u.CitizenType != CitizenTypeOther {
			info["ssn"] = decField(u.SsnEncrypted)
		}
		// Return investor info or switched-investor info
		if u.RoleType == RoleTypeInvestor ||
			u.UserState >= UserStateActiveAccred {
			info["employment_type"] = u.EmploymentType
			info["employer"] = u.Employer
			info["occupation"] = u.Occupation
			info["public_company_policy_maker"] = u.PublicCompanyPolicyMaker
			info["employed_by_broker_dealer"] = u.EmployedByBrokerDealer
			info["risk_tolerance"] = u.RiskTolerance
			info["marital_status"] = u.MaritalStatus
			info["household_income"] = u.HouseholdIncome
			info["household_networth"] = u.HouseholdNetworth
			info["invest_portfolio_total"] = u.InvestPortfolioTotal
			info["invest_exp_total"] = u.InvestExpTotal
			info["invest_real_estate_portion"] = u.InvestRealEstatePortion
			info["invest_convert_cash_ninety_days_portion"] =
				u.InvestConvertCashNinetyDaysPortion
			info["invest_alternative_portion"] = u.InvestAlternativePortion
			info["invest_private_company"] = u.InvestPrivateCompany
			info["invest_portfolio_horizon"] = u.InvestPortfolioHorizon
			info["education_level"] = u.EducationLevel
			info["invest_knowledge"] = u.InvestKnowledge
			info["work_with_financial_advisors"] = u.WorkWithFinancialAdvisors
			info["investor_type"] = u.InvestorType           // overlap, but ok
			info["investor_situation"] = u.InvestorSituation // overlap, but ok
		}
	}

	// Return user photo if uploaded
	if u.UserState >= UserStateKycWaitingApproval {
		pifc := path.Join(dataDir, "user", fmt.Sprintf("%v", u.ID),
			"photo_id")
		if _, err := os.Stat(pifc); err == nil {
			u.PhotoIDToken, _ = createFileTokenName("photo_id", "")
			// Ignore saving errors just don't return
			if dbConn.Save(u).Error == nil {
				info["photo_id"] = u.PhotoIDToken
				info["photo_id_filename"] = u.PhotoIDName
			}
		}
	}

	// Return the number of buys/sells for easier front-end display
	if u.UserState >= UserStateActive {
		var dis []DealInvestor
		if dbConn.Model(u).Related(&dis).Error == nil {
			info["buys"] = len(dis)
			var buysa uint64
			for _, di := range dis {
				buysa += di.SharesBuyAmount
			}
			info["buys_amount"] = buysa
		}
		var dshs []DealShareholder
		if dbConn.Model(u).Related(&dshs).Error == nil {
			info["sells"] = len(dshs)
			var sellsa uint64
			for _, dsh := range dshs {
				sellsa += dsh.SharesSellAmount
			}
			info["sells_amount"] = sellsa
		}
		var ods []Deal
		if dbConn.Find(&ods, "deal_state = ?", DealStateOpen).Error == nil {
			info["open_deals"] = len(ods)
		}
	}

	saveLogin(w, r, ps, false, u, info)
}

func userUpdateHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	// State must be active for respective resident types
	if (u.CitizenType == CitizenTypeOther &&
		u.UserState < UserStateActiveId) ||
		(u.CitizenType != CitizenTypeOther &&
			u.UserState < UserStateActive) {
		formatReturn(w, r, ps, ErrorCodeUserUpdateError, true, nil)
		return
	}

	// CitizenType cannot be changed!!! Only admin should do that
	_, ok := userKycFieldCheck(r, u, false, true)
	if !ok {
		formatReturn(w, r, ps, ErrorCodeUserUpdateError, true, nil)
		return
	}

	// Modify other existing fields
	firstName, fok := CheckField(true, r.FormValue("first_name"))
	if fok {
		u.FirstName = firstName
	}
	lastName, lok := CheckField(true, r.FormValue("last_name"))
	if lok {
		u.LastName = lastName
	}
	// Update full name if either first or last name changed
	if fok || lok {
		u.FullName = NameConventions[u.CitizenType](u.FirstName, u.LastName)
	}

	// Only update if already done self accred
	if (u.RoleType == RoleTypeShareholder &&
		u.UserState >= UserStateActiveAccred) ||
		(u.RoleType == RoleTypeInvestor &&
			u.UserState >= UserStateAccred) {
		investorType, ok := CheckRange(true, r.FormValue("investor_type"),
			InvestorTypeAdvisor)
		if ok {
			u.InvestorType = investorType
		}
		is := InvestorSituationUSBusinessIndividualAbove
		if u.CitizenType == CitizenTypeOther {
			is = InvestorSituationCNBusinessIndividualAbove
		}
		investorSituation, ok := CheckRange(true,
			r.FormValue("investor_situation"), uint64(is))
		if ok {
			u.InvestorSituation = investorSituation
		}
	}

	// Save however many changed
	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeUserUpdateError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u, nil)
}

func userPhotoIdHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	pifc := path.Join(dataDir, "user", fmt.Sprintf("%v", u.ID), "photo_id")
	parseFileDownload(w, r, u.PhotoIDType, pifc)
}

func userPhotoIdTokenHandler(w http.ResponseWriter, r *http.Request,
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

func userSellsHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	if u.UserState < UserStateActiveId {
		formatReturn(w, r, ps, ErrorCodeUserDealStateError, true, nil)
		return
	}

	var dshs []DealShareholder
	if dbConn.Model(u).Related(&dshs).Error != nil {
		formatReturn(w, r, ps, ErrorCodeUserDealStateError, true, nil)
		return
	}

	// Parse and return necessary fields
	sells := []map[string]interface{}{}
	for _, dsh := range dshs {
		var d Deal
		if dbConn.First(&d, dsh.DealID).RecordNotFound() {
			continue
		}
		var c Company
		var name string
		if dbConn.First(&c, d.CompanyID).RecordNotFound() {
			name = d.Name
		} else {
			name = c.Name
		}
		sells = append(sells, map[string]interface{}{
			"company_id":             d.CompanyID,
			"company_name":           name,
			"amount":                 dsh.SharesSellAmount,
			"deal_shareholder_state": dsh.DealShareholderState,
			"deal": map[string]interface{}{
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
			},
		})
	}

	saveLogin(w, r, ps, false, u, map[string]interface{}{"sells": sells})
}

func userBuysHandler(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User) {
	if u.UserState < UserStateActiveAccredId {
		formatReturn(w, r, ps, ErrorCodeUserDealStateError, true, nil)
		return
	}

	var dis []DealInvestor
	if dbConn.Model(u).Related(&dis).Error != nil {
		formatReturn(w, r, ps, ErrorCodeUserDealStateError, true, nil)
		return
	}

	// Parse and return necessary fields
	buys := []map[string]interface{}{}
	for _, di := range dis {
		var d Deal
		if dbConn.First(&d, di.DealID).RecordNotFound() {
			continue
		}
		var c Company
		var name string
		if dbConn.First(&c, d.CompanyID).RecordNotFound() {
			name = d.Name
		} else {
			name = c.Name
		}
		buys = append(buys, map[string]interface{}{
			"company_id":          d.CompanyID,
			"company_name":        name,
			"amount":              di.SharesBuyAmount,
			"deal_investor_state": di.DealInvestorState,
			"deal": map[string]interface{}{
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
			},
		})
	}

	saveLogin(w, r, ps, false, u, map[string]interface{}{"buys": buys})
}
