// MarketX specific use cases of North Capital's Transact API:
// https://api-docs.norcapsecurities.com
// Since this API is perhaps not going to be re-used in the future,
// an obligatory "package transact" was not created, but instead this
// customized version of MarketX Transact, MXT API is formed here.
// MXT API only handles the db User struct but is db agnostic to leave
// the API clean with the minimum requirements.
// All the API calls are in public domain in case of future package
// organizations and public exposures.
package main

/* Definitions */

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Transact struct {
	url             string
	clientID        string
	developerAPIKey string
	quiet           bool
}

var (
	TransactErrorUser = errors.New("Bad user data")
	TransactErrorArgs = errors.New("Bad arguments")
	TransactErrorRequest = errors.New("Request error")
	TransactErrorNetwork = errors.New("Network error")
	TransactErrorAPI = errors.New("API call error")
	TransactErrorResponse = errors.New("Response parsing error")
	TransactErrorFail = errors.New("Result error")
	TransactErrorCall = errors.New("Call error")
	TransactErrorKYCExpired = errors.New("KYC time expired")
	TransactErrorKYCFailed = errors.New("KYC failed multiple times")
)

var DummyUser = User{
	Email:        "johnsmith@egwog.com",
	FirstName:    "JOHN",
	LastName:     "SMITH",
	FullName:     "JOHN SMITH",
	Address1:     "222333 PEACHTREE PLACE",
	City:         "ATLANTA",
	State:        "GA",
	Zip:          "30318",
	Country:      "US",
	Dob:          "1975-02-28",
	SsnEncrypted: encField("112-22-3333"),
}

/* Initializers */

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

/* Private helpers */

var (
	kycAttempts uint64 = 3
	kycExpiring = 10 * time.Minute
	transactTimeout = 10 * time.Second
)

// log checks to see if we should log to server or just stdout
// for testing purposes
func (t *Transact) log(format string, a ...interface{}) {
	if t.quiet {
		return
	}
	if serverLog != nil {
		serverLog.Printf(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

// request takes a method, call and params to construct a call to TransactAPI
// and parses response and returns response body for parsing
func (t *Transact) request(method, call string,
params map[string]string) (map[string]interface{}, error) {
	form := url.Values{}
	// Set common fields
	form.Set("clientID", t.clientID)
	form.Set("developerAPIKey", t.developerAPIKey)
	// Create form with all params
	for k, v := range params {
		form.Set(k, v)
	}
	args := form.Encode()
	reqId := rand.Int63()
	url := t.url + "/" + call
	// Log before any error will happen
	t.log("[MXTAPI] (%v) %v: %v?%v\n", reqId, method, url, args)
	req, err := http.NewRequest(method, url, strings.NewReader(args))
	if err != nil {
		t.log("[MXTAPI] (%v) REQ: %v\n", reqId, err)
		return nil, TransactErrorRequest
	}
	// Always use url-encoded here
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create client to push the request
	client := &http.Client{Timeout: time.Duration(transactTimeout)}
	resp, err := client.Do(req)
	if err != nil {
		t.log("[MXTAPI] (%v) NET: %v\n", reqId, err)
		return nil, TransactErrorNetwork
	}
	// Caller must close
	defer resp.Body.Close()

	// Ignore error while parsing, read as many
	b, _ := ioutil.ReadAll(resp.Body)
	body := string(b)

	// If http response error, we should not parse the data structure at all
	if resp.StatusCode != http.StatusOK {
		t.log("[MXTAPI] (%v) API(%v): %v\n", reqId, resp.StatusCode, body)
		return nil, TransactErrorAPI
	}

	// Check if the response is json decodable
	var data map[string]interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		t.log("[MXTAPI] (%v) RES: %v\n", reqId, body)
		return nil, TransactErrorResponse
	}

	// Now check for API function failure
	sc, ok := data["statusCode"]
	if !ok || sc != "101" {
		t.log("[MXTAPI] (%v) CODE[%v]: %v\n", reqId, sc, body)
		return nil, TransactErrorFail
	}

	t.log("[MXTAPI] (%v) DONE: %v\n", reqId, data)
	return data, nil
}

// requestRetry is a wrapper around request with 3 network retries
func (t *Transact) requestRetry(method, call string,
params map[string]string) (map[string]interface{}, error) {
	for i := 0; i < 3; i++ {
		data, err := t.request(method, call, params)
		if err != TransactErrorNetwork {
			return data, err
		}
	}
	return nil, TransactErrorNetwork
}

// post is a wrapper for the POST request
func (t *Transact) post(call string,
params map[string]string) (map[string]interface{}, error) {
	return t.requestRetry("POST", call, params)
}

// put is a wrapper for the PUT request
func (t *Transact) put(call string,
params map[string]string) (map[string]interface{}, error) {
	return t.requestRetry("PUT", call, params)
}

/* KYC APIs */

/* Constants */
var (
	averageIncomeTexts = []string{
		"less than $200,000",
		"$200,000 to $300,000",
		"over $300,000",
	}
	householdNetworthTexts = []string{
		"less than $500,000",
		"$500,000 to $1,000,000",
		"$1,000,000 to $5,000,000",
		"over $5,000,000",
	}
)

// CreateInvestorRecord sets up an investor account with configurations
// from the db User structure
// On success, u *User is updated accordingly
func (t *Transact) CreateInvestorRecord(u *User) error {
	// Setup post-calculated values
	investingFor := "self"
	if u.InvestorType != InvestorTypeIndividual {
		investingFor = "agent"
	}
	resident := "yes"
	if u.CitizenType != CitizenTypePermanentResident {
		resident = "no"
	}
	citizen := "yes"
	if u.CitizenType != CitizenTypeCitizen {
		citizen = "no"
	}
	params := map[string]string{
		"firstName":           u.FirstName,
		"lastName":            u.LastName,
		"investorAccountName": u.FullName,
		"investingFor":        investingFor,
		"ageAbove18":          "yes",
		"residentUS":          resident,
		"citizenUS":           citizen,
		"residenceState":      formatUsState(u.State),
		"emailAddress":        u.Email,
		"averageIncome":       averageIncomeTexts[u.HouseholdIncome],
		"householdNetworth":   householdNetworthTexts[u.HouseholdNetworth],
		"createdIpAddress":    u.LastIpAddress,
	}
	data, err := t.put("createInvestorRecord", params)
	if err != nil {
		return err
	}

	// Now parse function specific return value
	details, ok := data["investorDetails"]
	if !ok {
		return TransactErrorCall
	}
	investorDetails, ok := details.([]interface{})
	if !ok {
		return TransactErrorCall
	}
	if len(investorDetails) < 1 {
		return TransactErrorCall
	}
	idetails, ok := investorDetails[0].(map[string]interface{})
	if !ok {
		return TransactErrorCall
	}
	iid, ok := idetails["investorId"]
	if !ok {
		return TransactErrorCall
	}
	iids, ok := iid.(string)
	if !ok {
		return TransactErrorCall
	}
	investorId, err := strconv.ParseUint(iids, 10, 64)
	if err != nil {
		return TransactErrorCall
	}

	// Can set investor id now
	u.TransactApiInvestorID = investorId
	return nil
}

// UpdateInvestor updates investor account information
func (t *Transact) UpdateInvestor(u *User) error {
	return nil
}

// DeleteInvestor deletes an investor account
// On success, u *User is updated accordingly
func (t *Transact) DeleteInvestor(u *User) error {
	if u.TransactApiInvestorID == 0 {
		return TransactErrorUser
	}

	params := map[string]string{
		"investorId": fmt.Sprintf("%v", u.TransactApiInvestorID),
	}
	_, err := t.post("deleteInvestor", params)
	if err != nil {
		return err
	}

	u.TransactApiInvestorID = 0
	return nil
}

// CreateInvestorAccount creates a financial account for an investor
// On success, u *User is updated accordingly
// Returns a list of questions to be answered
func (t *Transact) CreateInvestorAccount(u *User) ([]map[string]interface{},
error) {
	if u.TransactApiInvestorID == 0 {
		return nil, TransactErrorUser
	}

	// Setup post-calculated values
	params := map[string]string{
		"investorId":       fmt.Sprintf("%v", u.TransactApiInvestorID),
		"accountFullName":  u.FullName,
		"dob":              outputDate(u.Dob),
		"addressline1":     u.Address1,
		"addressline2":     u.Address2,
		"city":             u.City,
		"zip":              u.Zip,
		"country":          u.Country,
		"createdIpAddress": u.LastIpAddress,
	}
	// If non-US person ssn is not required
	if u.CitizenType != CitizenTypeOther {
		params["socialSecurityNumber"] = decField(u.SsnEncrypted)
	}
	data, err := t.put("createInvestorAccount", params)
	if err != nil {
		return nil, err
	}

	// Now parse function specific return value
	kyc, ok := data["kyc"]
	if !ok {
		return nil, TransactErrorCall
	}
	kycm, ok := kyc.(map[string]interface{})
	if !ok {
		return nil, TransactErrorCall
	}
	resp, ok := kycm["response"]
	if !ok {
		return nil, TransactErrorCall
	}
	respm, ok := resp.(map[string]interface{})
	if !ok {
		return nil, TransactErrorCall
	}
	kycid, ok := respm["id-number"]
	if !ok {
		return nil, TransactErrorCall
	}
	kycids, ok := kycid.(string)
	if !ok {
		return nil, TransactErrorCall
	}
	kycId, err := strconv.ParseUint(kycids, 10, 64)
	if err != nil {
		return nil, TransactErrorCall
	}
	ques, ok := respm["questions"]
	if !ok {
		return nil, TransactErrorCall
	}
	quesm, ok := ques.(map[string]interface{})
	if !ok {
		return nil, TransactErrorCall
	}
	questions, ok := quesm["question"]
	if !ok {
		return nil, TransactErrorCall
	}
	// Cannot parse any further than interface{} map
	questionss, ok := questions.([]interface{})
	if !ok {
		return nil, TransactErrorCall
	}
	var quess []map[string]interface{}
	for _, v := range questionss {
		vm, ok := v.(map[string]interface{})
		// Quit if anything is impossible to parse
		if !ok {
			return nil, TransactErrorCall
		}
		quess = append(quess, vm)
	}

	u.TransactApiKycID = kycId
	u.TransactApiKycExpire = time.Now().Add(kycExpiring)
	// Create new attempts if not in locked fail state
	if u.TransactApiKycAttempts == 0 && u.UserState != UserStateKycFailed {
		u.TransactApiKycAttempts = kycAttempts
	}
	return quess, nil
}

// DeleteInvestorFinancialAccount deletes the financial account
// associated with an investor
// On success, u *User is updated accordingly
func (t *Transact) DeleteInvestorFinancialAccount(u *User) error {
	if u.TransactApiInvestorID == 0 {
		return TransactErrorUser
	}

	params := map[string]string{
		"investorId": fmt.Sprintf("%v", u.TransactApiInvestorID),
	}
	_, err := t.post("deleteInvestorFinancialAccount", params)
	if err != nil {
		return err
	}

	u.TransactApiKycID = 0
	u.TransactApiKycExpire = time.Time{}
	u.TransactApiKycAttempts = 0
	return nil
}

// KycStatus checks for the answers for the KYC questions
// On success, u *User is updated accordingly
// On some failures, u *User may also be updated
func (t *Transact) KycStatus(u *User, ans map[string]string) error {
	// Development uses dummy user so these checks do not apply
	if environment == "production" {
		if u.TransactApiInvestorID == 0 || u.TransactApiKycID == 0 {
			return TransactErrorUser
		}
		if u.TransactApiKycExpire.Before(time.Now()) {
			return TransactErrorKYCExpired
		}
		if u.TransactApiKycAttempts == 0 ||
			u.UserState == UserStateKycFailed {
			return TransactErrorKYCFailed
		}
	}

	params := map[string]string{
		"investorId": fmt.Sprintf("%v", u.TransactApiInvestorID),
		"noOfqns":    fmt.Sprintf("%v", len(ans)),
		"idNumber":   fmt.Sprintf("%v", u.TransactApiKycID),
	}
	for k, v := range ans {
		var s string
		var m string
		if strings.HasPrefix(k, "type") {
			s = k[4:]
			m = "ans" + s
		} else if strings.HasPrefix(k, "ans") {
			s = k[3:]
			m = "type" + s
			// Convert to transact terms
			k = "qns" + s
		} else {
			return TransactErrorArgs
		}
		// Check if in range
		if s != "1" && s != "2" && s != "3" && s != "4" && s != "5" {
			return TransactErrorArgs
		}
		// Make sure type/ans match
		_, ok := ans[m]
		if !ok {
			return TransactErrorArgs
		}
		// Now save
		params[k] = v
	}
	// On development it is impossible to test automate kycstatus
	// so we do auto-kyc
	if environment == "production" {
		_, err := t.put("kycstatus", params)
		if err != nil {
			// Only subtract attempt on actual failures
			if err == TransactErrorAPI || err == TransactErrorResponse ||
				err == TransactErrorFail {
				u.TransactApiKycAttempts -= 1
				// Set failure status
				if u.TransactApiKycAttempts == 0 {
					u.UserState = UserStateKycFailed
				}
			}
			return err
		}
	}

	// Also clear the previous kyc settings
	u.TransactApiKycID = 0
	u.TransactApiKycExpire = time.Time{}
	u.TransactApiKycAttempts = 0
	return nil
}
