// Constants for MarketX server
// These are not "typed" for simplicity as name prefixes imply underlying
// types and database saving schema.
// If manipulations need to be done on the types, they should be aliased.
package main

const (
	ResultFail = iota
	ResultSuccess
)

const (
	LoginLoggedOut = iota
	LoginLoggedIn
	LoginNone
)

const (
	RoleTypeShareholder = iota
	RoleTypeInvestor
)

const (
	UserStateInactive = iota
	UserStateConfirmed
	UserStateAccred
	UserStateNdaAgreed
	UserStateKycWaitingQuestions
	UserStateKycWaitingId
	UserStateKycWaitingApproval
	UserStateKycFailed
	UserStateActive
	UserStateActiveId
	UserStateActiveAccred
	UserStateActiveAccredId
	UserStateBanned = 999
)

const (
	UserLevelNormal = iota
	UserLevelAdmin
)

const (
	InvestorTypeIndividual = iota
	InvestorTypeEntity
	InvestorTypeAdvisor
)

const (
	InvestorSituationUSIndividualJoint1M = iota
	InvestorSituationUSIndividualIncome200KJoint300K
	InvestorSituationUSBusiness5M
	InvestorSituationUSBusinessIndividualAbove
)

const (
	InvestorSituationCNIndividual300W = iota
	InvestorSituationCNIndividualIncome50W
	InvestorSituationCNBusiness1000W
	InvestorSituationCNBusinessIndividualAbove
)

const (
	CitizenTypeCitizen = iota
	CitizenTypePermanentResident
	CitizenTypeOther
)

const (
	EmploymentTypeEmployed = iota
	EmploymentTypeSelfEmployed
	EmploymentTypeRetired
	EmploymentTypeStudent
	EmploymentTypeHomemaker
	EmploymentTypeUnemployed
)

const (
	PublicCompanyPolicyMakerNo = iota
	PublicCompanyPolicyMakerYes
)

const (
	EmployedByBrokerDealerNo = iota
	EmployedByBrokerDealerYes
)

const (
	RiskToleranceConservative = iota
	RiskToleranceModerate
	RiskToleranceAggressive
)

const (
	MaritalStatusSingle = iota
	MaritalStatusMarried
)

const (
	HouseholdIncomeLess200K = iota
	HouseholdIncome200KTo300K
	HouseholdIncomeMore300K
)

const (
	HouseholdNetworthLess500K = iota
	HouseholdNetworth500KTo1M
	HouseholdNetworth1MTo5M
	HouseholdNetworthMore5M
)

const (
	InvestPrivateCompanyNo = iota
	InvestPrivateCompanyYes
)

const (
	InvestPortfolioHorizonLess5 = iota
	InvestPortfolioHorizon5To10
	InvestPortfolioHorizonMore10
)

const (
	EducationLevelHighSchool = iota
	EducationLevelUndergraduate
	EducationLevelGraduate
)

const (
	InvestKnowledgeNovice = iota
	InvestKnowledgeAverage
	InvestKnowledgeHigh
	InvestKnowledgeExpert
)

const (
	WorkWithFinancialAdvisorsNo = iota
	WorkWithFinancialAdvisorsYes
)

const (
	OwnTypeShares = iota
	OwnTypeRsu
	OwnTypeOptions
	OwnTypeSharesRsu
	OwnTypeSharesOptions
	OwnTypeRsuOptions
	OwnTypeSharesRsuOptions
)

const (
	StockTypePreferred = iota
	StockTypeCommon
	StockTypeBoth
	StockTypeOther
)

const (
	AccountTypeChecking = iota
	AccountTypeSaving
)

const (
	DealStateOpen = iota
	DealStatePreview
	DealStateClosed
	DealStateUserSubmitted
)

const (
	DealSpecialNone = iota
	DealSpecialTwentyPercentOff
)

const (
	DealUserStateNone = iota
	DealUserStateShareholder
	DealUserStateInvestor
)

const (
	DealShareholderStateEngagementStarted = iota
	DealShareholderStateEngagementLetterSigned
	DealShareholderStateOfferCreated
	DealShareholderStateOfferCompleted
	DealShareholderStateAdminApprovedOffer
	DealShareholderStateBankInfoSubmitted
	DealShareholderStateBankVerified
	DealShareholderStateWaitingCompanyRofr
	DealShareholderStateCompanyApprovedRofr
	DealShareholderStatePurchaseAgreementSigned
	DealShareholderStateAdminApprovedDeal
	DealShareholderStateDealClosed
)

const (
	DealInvestorStateEngagementStarted = iota
	DealInvestorStateEngagementLetterSigned
	DealInvestorStateInterestSubmitted
	DealInvestorStateSummaryTermsSigned
	DealInvestorStateIndicationSigned
	DealInvestorStateAdminApprovedInterest
	DealInvestorStateBankInfoSubmitted
	DealInvestorStateBankVerified
	DealInvestorStateLpAgreementSigned
	DealInvestorStateGpAgreementSigned
	DealInvestorStateSpcApplicationSigned
	DealInvestorStateSpcPpmSigned
	DealInvestorStateSpcSupplementSigned
	DealInvestorStateDePpmSigned
	DealInvestorStateDeOperatingAgreementSigned
	DealInvestorStateDeSubscriptionAgreementSigned
	DealInvestorStateDePosSigned
	DealInvestorStateTaxFormUploaded
	DealInvestorStateEscrowStarted
	DealInvestorStateEscrowDocUploaded
	DealInvestorStateWaitingFundTransfer
	DealInvestorStateWaitingCompanyRofr
	DealInvestorStateCompanyApprovedRofr
	DealInvestorStateEscrowBroke
	DealInvestorStateAdminApprovedDeal
	DealInvestorStateDealClosed
)

const (
	DealSharesTypePreferred = iota
	DealSharesTypeCommon
)

const (
	WechatStateCreated = iota
	WechatStateApproved
)

const (
	InvestmentAmountLess20K = iota
	InvestmentAmount20KTo100K
	InvestmentAmountMore100K
)

const (
	Base64No = iota
	Base64Yes
)

// 0 denotes default
var PageSizes = map[uint64]uint64{
	0:   10,
	10:  10,
	20:  20,
	50:  50,
	100: 100,
	200: 200,
	500: 500,
}

const (
	DocusignCheckingNo = iota
	DocusignCheckingYes
)
