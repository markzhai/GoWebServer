// Database schema for MarketX
// Uses gorm to manage postgresql connections and structures.
package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strings"
	"time"
)

var allTables = []interface{}{&User{}, &Company{}, &CompanyExecutive{},
	&Funding{}, &CompanyUpdate{}, &Tag{}, &Bank{}, &Offer{}, &Deal{},
	&DealShareholder{}, &DealInvestor{}, &Wechat{}}

// encField makes writing field easier by ignoring errors
func encField(f string) string {
	ef, err := EncryptString(aesTextSecret, f)
	if err != nil {
		return ""
	}
	return ef
}

// decField makes reading field easier by ignoring errors
func decField(f string) string {
	df, err := DecryptString(aesTextSecret, f)
	if err != nil {
		return ""
	}
	return df
}

// unixTime is a convenience helper to check out of bounds time conversion
func unixTime(t time.Time) uint64 {
	tv := t.Unix()
	if tv < 0 {
		return 0
	}
	if uint64(tv) > TimeMax {
		return TimeMax
	}
	return uint64(tv)
}

var (
	m0 = []string{"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX"}
	m1 = []string{"", "X", "XX", "XXX", "XL", "L", "LX", "LXX", "LXXX", "XC"}
	m2 = []string{"", "C", "CC", "CCC", "CD", "D", "DC", "DCC", "DCCC", "CM"}
	m3 = []string{"", "M", "MM", "MMM", "I̅V̅", "V̅", "V̅I̅", "V̅I̅I̅", "V̅I̅I̅I̅", "I̅X̅"}
	m4 = []string{"", "X̅", "X̅X̅", "X̅X̅X̅", "X̅L̅", "L̅", "L̅X̅", "L̅X̅X̅", "L̅X̅X̅X̅", "X̅C̅"}
	m5 = []string{"", "C̅", "C̅C̅", "C̅C̅C̅", "C̅D̅", "D̅", "D̅C̅", "D̅C̅C̅", "D̅C̅C̅C̅", "C̅M̅"}
	m6 = []string{"", "M̅", "M̅M̅", "M̅M̅M̅"}
)

// formatRoman takes an int and returns a Roman numeral representation if
// possible, ignore errors version
func formatRoman(n uint64) string {
	if n < 1 || n >= 4e6 {
		return ""
	}
	// This is efficient in Go.
	// The seven operands are evaluated, then a single allocation is
	// made of the exact size needed for the result.
	return m6[n/1e6] + m5[n%1e6/1e5] + m4[n%1e5/1e4] + m3[n%1e4/1e3] +
		m2[n%1e3/1e2] + m1[n%100/10] + m0[n%10]
}

// formatMoney takes a float of B and returns M representation when fit
func formatMoney(m float64) string {
	if m < 1 {
		return fmt.Sprintf("%vM", 1000.0*m)
	}
	return fmt.Sprintf("%vB", m)
}

// inputDate checks whether date is in 2006-01-02 format
// and returns the 2006-01-02 format on success and "" on error
func inputDate(date string) string {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return date
}

// outputDate checks whether date is in 2006-01-02 format
// and returns the 01-02-2006 format on success and "" on error
func outputDate(date string) string {
	tp, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return tp.Format("01-02-2006")
}

// https://www.irs.gov/pub/irs-utl/zip%20code%20and%20state%20abbreviations.pdf
var usStates = map[string]string{
	// States
	"Alabama":              "AL",
	"Alaska":               "AK",
	"Arizona":              "AZ",
	"Arkansas":             "AR",
	"California":           "CA",
	"Colorado":             "CO",
	"Connecticut":          "CT",
	"Delaware":             "DE",
	"District of Columbia": "DC",
	"Florida":              "FL",
	"Georgia":              "GA",
	"Hawaii":               "HI",
	"Idaho":                "ID",
	"Illinois":             "IL",
	"Indiana":              "IN",
	"Iowa":                 "IA",
	"Kansas":               "KS",
	"Kentucky":             "KY",
	"Louisiana":            "LA",
	"Maine":                "ME",
	"Maryland":             "MD",
	"Massachusetts":        "MA",
	"Michigan":             "MI",
	"Minnesota":            "MN",
	"Mississippi":          "MS",
	"Missouri":             "MO",
	"Montana":              "MT",
	"Nebraska":             "NE",
	"Nevada":               "NV",
	"New Hampshire":        "NH",
	"New Jersey":           "NJ",
	"New Mexico":           "NM",
	"New York":             "NY",
	"North Carolina":       "NC",
	"North Dakota":         "ND",
	"Ohio":                 "OH",
	"Oklahoma":             "OK",
	"Oregon":               "OR",
	"Pennsylvania":         "PA",
	"Rhode Island":         "RI",
	"South Carolina":       "SC",
	"South Dakota":         "SD",
	"Tennessee":            "TN",
	"Texas":                "TX",
	"Utah":                 "UT",
	"Vermont":              "VT",
	"Virginia":             "VA",
	"Washington":           "WA",
	"West Virginia":        "WV",
	"Wisconsin":            "WI",
	"Wyoming":              "WY",
	// Territories
	"American Somoa":                 "AS",
	"Federated States of Micronesia": "FM",
	"Guam":                                         "GU",
	"Marshall Islands":                             "MH",
	"Commonwealth of the Northern Mariana Islands": "MP",
	"Palau":               "PW",
	"Puerto Rico":         "PR",
	"U.S. Virgin Islands": "VI",
	// Military
	"U.S. Armed Forces - Americas": "AA",
	"U.S. Armed Forces - Europe":   "AE",
	"U.S. Armed Forces - Pacific":  "AP",
}

// formatUsState does a case-insensitive search of US state abbreviations
// returns "" on error and 2-char abbreviation on success
func formatUsState(state string) string {
	state = strings.ToUpper(state)
	for s, a := range usStates {
		if strings.ToUpper(s) == state || a == state {
			return a
		}
	}
	return ""
}

type User struct {
	gorm.Model
	Companies                          []Company `gorm:"many2many:user_companies;"`
	Banks                              []Bank
	Offers                             []Offer
	DealShareholders                   []DealShareholder
	DealInvestors                      []DealInvestor
	Wechat                             Wechat
	WxOpenID                           string `sql:"index"`
	WxUnionID                          string `sql:"index"`
	WxAccessToken                      string
	UserState                          uint64
	UserLevel                          uint64
	FirstName                          string
	MiddleInitial                      string
	IDCardNumber                       string
	LastName                           string
	FullName                           string
	Email                              string `sql:"index"`
	EmailToken                         string `sql:"index"`
	EmailTokenExpire                   time.Time
	PasswordHash                       string
	PasswordToken                      string `sql:"index"`
	PasswordTokenExpire                time.Time
	RoleType                           uint64
	CreationIpAddress                  string
	LastIpAddress                      string
	LastLanguage                       string
	Affiliation                        string
	InvestorType                       uint64
	InvestorSituation                  uint64
	PhoneNumber                        string `sql:"index"`
	PhotoIDPic                         string
	PhotoIDName                        string
	PhotoIDType                        string
	PhotoIDToken                       string `sql:"index"`
	Dob                                string
	Address1                           string
	Address2                           string
	City                               string
	State                              string
	Zip                                string
	Country                            string
	CitizenType                        uint64
	SsnEncrypted                       string
	EmploymentType                     uint64
	Employer                           string
	Occupation                         string
	PublicCompanyPolicyMaker           uint64
	EmployedByBrokerDealer             uint64
	RiskTolerance                      uint64
	MaritalStatus                      uint64
	HouseholdIncome                    uint64
	HouseholdNetworth                  uint64
	InvestPortfolioTotal               uint64
	InvestExpTotal                     uint64
	InvestRealEstatePortion            uint64
	InvestConvertCashNinetyDaysPortion uint64
	InvestAlternativePortion           uint64
	InvestPrivateCompany               uint64
	InvestPortfolioHorizon             uint64
	EducationLevel                     uint64
	InvestKnowledge                    uint64
	WorkWithFinancialAdvisors          uint64
	TransactApiInvestorID              uint64
	TransactApiIssuerID                uint64
	TransactApiKycID                   uint64
	TransactApiKycExpire               time.Time
	TransactApiKycAttempts             uint64
}

type Company struct {
	gorm.Model
	Users             []User `gorm:"many2many:user_companies;"`
	Offers            []Offer
	Deals             []Deal
	Tags              []Tag `gorm:"many2many:company_tags;"`
	CompanyExecutives []CompanyExecutive
	Fundings          []Funding
	CompanyUpdates    []CompanyUpdate
	Name              string
	Description       string `sql:"size:10000"`
	DescriptionCn     string `sql:"size:10000"`
	YearFounded       uint64
	StateFounded      string
	Hq                string
	HomePage          string
	KeyPerson         string
	NumEmployees      uint64
	TotalValuation    float64
	TotalFunding      float64
	GrowthRatePercent float64
	SizeMultiple      float64
	Investors         string `sql:"size:10000"`
	InvestorLogoPics  string `sql:"size:1000"`
	NumSlides         uint64
	VideoUrl          string
}

type CompanyExecutiveSort []CompanyExecutive

func (ces CompanyExecutiveSort) Len() int {
	return len(ces)
}
func (ces CompanyExecutiveSort) Swap(i, j int) {
	ces[i], ces[j] = ces[j], ces[i]
}
func (ces CompanyExecutiveSort) Less(i, j int) bool {
	return ces[i].ID < ces[j].ID
}

type CompanyExecutive struct {
	gorm.Model
	CompanyID uint64 `sql:"index"`
	Name      string
	Role      string
	Office    string
}

// TODO: Should sort by date
type FundingSort []Funding

func (fs FundingSort) Len() int {
	return len(fs)
}
func (fs FundingSort) Swap(i, j int) {
	fs[i], fs[j] = fs[j], fs[i]
}
func (fs FundingSort) Less(i, j int) bool {
	return fs[i].ID < fs[j].ID
}

type Funding struct {
	gorm.Model
	CompanyID               uint64 `sql:"index"`
	Type                    string
	Date                    string
	Amount                  float64
	RaisedToDate            float64 `json:"raised_to_date"`
	PreValuation            float64 `json:"pre_valuation"`
	PostValuation           float64 `json:"post_valuation"`
	Status                  string
	Stage                   string
	NumShares               uint64  `json:"num_shares"`
	ParValue                float64 `json:"par_value"`
	DividendRatePercent     float64 `json:"dividend_rate_percent"`
	OriginalIssuePrice      float64 `json:"original_issue_price"`
	Liquidation             float64
	LiquidationPrefMultiple uint64  `json:"liquidation_pref_multiple"`
	ConversionPrice         float64 `json:"conversion_price"`
	PercentOwned            float64 `json:"percent_owned"`
}

// TODO: Should sort by date
type CompanyUpdateSort []CompanyUpdate

func (cus CompanyUpdateSort) Len() int {
	return len(cus)
}
func (cus CompanyUpdateSort) Swap(i, j int) {
	cus[i], cus[j] = cus[j], cus[i]
}
func (cus CompanyUpdateSort) Less(i, j int) bool {
	return cus[i].ID < cus[j].ID
}

type CompanyUpdate struct {
	gorm.Model
	CompanyID uint64 `sql:"index"`
	Title     string
	Url       string
	Date      string
	Language  string
}

type Tag struct {
	gorm.Model
	Companies []Company `gorm:"many2many:company_tags;"`
	Name      string
	NameCn    string
}

type Bank struct {
	gorm.Model
	DealShareholders       []DealShareholder
	DealInvestors          []DealInvestor
	UserID                 uint64 `sql:"index"`
	FullName               string
	NickName               string
	RoutingNumberEncrypted string
	AccountNumberEncrypted string
	AccountType            uint64
}

type Offer struct {
	gorm.Model
	UserID                    uint64 `sql:"index"`
	CompanyID                 uint64 `sql:"index"`
	DealID                    uint64 `sql:"index"`
	OwnType                   uint64
	Vested                    uint64
	Restrictions              uint64
	SharesTotalOwn            uint64
	StockType                 uint64
	ExerciseDate              string
	ExercisePrice             float64
	SharesToSell              uint64
	DesirePrice               float64
	ShareCertificateDoc       string
	ShareCertificateName      string
	ShareCertificateType      string
	ShareCertificateToken     string `sql:"index"`
	CompanyByLawsDoc          string
	CompanyByLawsName         string
	CompanyByLawsType         string
	CompanyByLawsToken        string `sql:"index"`
	ShareholderAgreementDoc   string
	ShareholderAgreementName  string
	ShareholderAgreementType  string
	ShareholderAgreementToken string `sql:"index"`
	StockOptionPlanDoc        string
	StockOptionPlanName       string
	StockOptionPlanType       string
	StockOptionPlanToken      string `sql:"index"`
}

type Deal struct {
	gorm.Model
	Offers           []Offer
	DealShareholders []DealShareholder
	DealInvestors    []DealInvestor
	CompanyID        uint64 `sql:"index"`
	Name             string
	DealState        uint64
	DealSpecial      uint64
	FundNum          uint64
	Note             string
	SharesAmount     uint64
	SharesLeft       uint64
	SharesType       uint64
	ActualPrice      float64
	ActualValuation  float64
	StartDate        time.Time
	EndDate          time.Time
	EscrowAccount    string
	EscrowAccountCn  string
}

type DealShareholder struct {
	gorm.Model
	BankID                      uint64 `sql:"index"`
	UserID                      uint64 `sql:"index"`
	DealID                      uint64 `sql:"index"`
	DealShareholderState        uint64
	EngagementLetterSign        string
	EngagementLetterSignId      string
	EngagementLetterSignUrl     string
	EngagementLetterSignExpire  time.Time
	EngagementLetterSignCheck   time.Time
	EngagementLetterToken       string `sql:"index"`
	SharesSellAmount            uint64
	RofrWaiverSign              string
	RofrWaiverSignId            string
	RofrWaiverSignUrl           string
	RofrWaiverSignExpire        time.Time
	PurchaseAgreementSign       string
	PurchaseAgreementSignId     string
	PurchaseAgreementSignUrl    string
	PurchaseAgreementSignExpire time.Time
}

type DealInvestor struct {
	gorm.Model
	BankID                            uint64 `sql:"index"`
	UserID                            uint64 `sql:"index"`
	DealID                            uint64 `sql:"index"`
	DealInvestorState                 uint64
	EngagementLetterSign              string
	EngagementLetterSignId            string
	EngagementLetterSignUrl           string
	EngagementLetterSignExpire        time.Time
	EngagementLetterSignCheck         time.Time
	EngagementLetterToken             string `sql:"index"`
	SharesBuyAmount                   uint64
	SummaryOfTermsSign                string
	SummaryOfTermsSignId              string
	SummaryOfTermsSignUrl             string
	SummaryOfTermsSignExpire          time.Time
	SummaryOfTermsSignCheck           time.Time
	SummaryOfTermsToken               string `sql:"index"`
	IndicationSign                    string
	IndicationSignId                  string
	IndicationSignUrl                 string
	IndicationSignExpire              time.Time
	RofrWaiverSign                    string
	RofrWaiverSignId                  string
	RofrWaiverSignUrl                 string
	RofrWaiverSignExpire              time.Time
	LpAgreementSign                   string
	LpAgreementSignId                 string
	LpAgreementSignUrl                string
	LpAgreementSignExpire             time.Time
	GpAgreementSign                   string
	GpAgreementSignId                 string
	GpAgreementSignUrl                string
	GpAgreementSignExpire             time.Time
	SpcApplicationSign                string
	SpcApplicationSignId              string
	SpcApplicationSignUrl             string
	SpcApplicationSignExpire          time.Time
	SpcPpmSign                        string
	SpcPpmSignId                      string
	SpcPpmSignUrl                     string
	SpcPpmSignExpire                  time.Time
	SpcSupplementSign                 string
	SpcSupplementSignId               string
	SpcSupplementSignUrl              string
	SpcSupplementSignExpire           time.Time
	DePpmSign                         string
	DePpmSignId                       string
	DePpmSignUrl                      string
	DePpmSignExpire                   time.Time
	DePpmSignCheck                    time.Time
	DePpmToken                        string `sql:"index"`
	DeOperatingAgreementSign          string
	DeOperatingAgreementSignId        string
	DeOperatingAgreementSignUrl       string
	DeOperatingAgreementSignExpire    time.Time
	DeOperatingAgreementSignCheck     time.Time
	DeOperatingAgreementToken         string `sql:"index"`
	DeSubscriptionAgreementSign       string
	DeSubscriptionAgreementSignId     string
	DeSubscriptionAgreementSignUrl    string
	DeSubscriptionAgreementSignExpire time.Time
	DeSubscriptionAgreementSignCheck  time.Time
	DeSubscriptionAgreementToken      string `sql:"index"`
}

type Wechat struct {
	gorm.Model
	UserID      uint64 `sql:"index"`
	WechatState uint64
	OpenID      string `sql:"index"`
	UnionID     string `sql:"index"`

	PhotoIDPic       string
	PhotoIDName      string
	PhotoIDType      string
	CitizenType      uint64
	OverseasBank     uint64
	InvestmentAmount uint64

	Nickname      string
	Sex           int
	City          string
	Country       string
	Province      string
	Language      string
	HeadImgUrl    string
	SubscribeTime string
	GroupID       string `sql:"index"`
	Remark        string
}
