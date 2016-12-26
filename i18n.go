// Internationalization and localization maps
// Golang text package is not ready so custom construction is done here.
package main

type ErrorCode uint64

const (
	ErrorCodeEmailExists ErrorCode = iota
	ErrorCodePhoneExists
	ErrorCodeRegisterError
	ErrorFmtCodeBadArgument
	ErrorCodeEmailUnknown
	ErrorCodePhoneUnknown
	ErrorCodeIdCardInvalid
	ErrorCodeBadPassword
	ErrorCodePhoneCodeError
	ErrorCodeEmailTokenInvalid
	ErrorCodeEmailTokenExpired
	ErrorCodeActivationError
	ErrorCodePasswordTokenInvalid
	ErrorCodePasswordTokenExpired
	ErrorCodePasswordResetError
	ErrorCodePasswordUpdateError
	ErrorCodeConfirmError
	ErrorCodeNdaError
	ErrorCodeKycError
	ErrorCodeKycRecordError
	ErrorCodeKycAccountError
	ErrorCodeKycHardFail
	ErrorCodeKycGetQuestions
	ErrorCodeKycCheckError
	ErrorCodeKycCheckQuestions
	ErrorCodeUploadIdError
	ErrorCodePhotoIdError
	ErrorCodeUserNotOnboard
	ErrorCodeUserUnknown
	ErrorCodeUserUpdateError
	ErrorCodeUserDealStateError
	ErrorCodeCompanyUnknown
	ErrorCodeCompanyTagUnknown
	ErrorCodeCompanySlideUnknown
	ErrorCodeCompanyInfoError
	ErrorCodeSelfAccredError
	ErrorCodeSelfAccredSwitchError
	ErrorCodeDealUnknown
	ErrorCodeDealOfferUnknown
	ErrorCodeDealShareholderUnknown
	ErrorCodeDealInvestorUnknown
	ErrorCodeDealUserUnknown
	ErrorCodeDealNotLive
	ErrorCodeDealNotLiveOrSubmitted
	ErrorCodeDealError
	ErrorCodeDealAlreadySelling
	ErrorCodeDealAlreadyBuying
	ErrorCodeDealAlreadyOffering
	ErrorCodeDealUserStateError
	ErrorCodeDealOperationError
	ErrorCodeDealSellUploadOffer
	ErrorCodeDealSellNoSuchOffer
	ErrorCodeDealNoBankInfo
	ErrorCodeDealHellosignError
	ErrorCodeDealDocusignError
	ErrorCodeDealDocusignEnvelopeError
	ErrorCodeDealDocusignRecipientError
	ErrorCodeDealDocusignSignError
	ErrorCodeDealDocusignStatusError
	ErrorCodeDealDocusignDownloadError
	ErrorCodeDealNotEnoughShares
	ErrorCodeNoPermission
	ErrorCodeServerInternal
	ErrorCodeUserBanned
	ErrorCodeJwtError
	ErrorCodeJwtStoreError
	ErrorCodeJwtExpired
	ErrorCodeTokenError
	ErrorCodeWechatExists
	ErrorCodeWechatError
	ErrorCodeWechatBinded
	ErrorCodeAdminError
	ErrorCodeAdminUserStateError
	ErrorCodeAdminPageError
	ErrorCodeAdminCompanyTagsError
	ErrorCodeAdminCompanyExecutivesError
	ErrorCodeAdminCompanyFundingsError
	ErrorCodeAdminCompanyUpdatesError
	ErrorCodeKeywordError
	ErrorCodeIdError
	ErrorCodeUnknown
	ErrorCodeNone = 99999
)

var ErrorCodes = map[string][]string{
	"en-US": []string{
		"Email has already been registered",
		"Phone number has already been taken",
		"Register error",
		"Argument field %v is invalid",
		"Email has not been registered",
		"Phone number has not been registered",
		"Id card number is invalid",
		"Wrong password",
		"Phone verification code is error",
		"Invalid email activation token",
		"Email activation token expired",
		"Account activation error",
		"Invalid password reset token",
		"Password reset token expired",
		"Password reset error",
		"Password update error",
		"Confirm error",
		"NDA error",
		"KYC error",
		"Failed to verify KYC: please go back and check your name and address.",
		"Failed to verify KYC: please go back and check your name, date of birth, address, and social security number.",
		"Failed to verify KYC: please contact support for more information.",
		"Failed to get KYC questions",
		"Failed to check KYC",
		"Failed to answer KYC questions",
		"Unable to upload id",
		"Cannot get photo id",
		"User has not finished onboarding process",
		"User does not exist",
		"User information update error",
		"User deal state is invalid",
		"Company does not exist",
		"Company tag does not exist",
		"Company slide page does not exist",
		"Cannot read company information",
		"Failed to submit self accreditation",
		"Failed to submit self accreditation when switching to investor",
		"Deal does not exist",
		"Deal offer does not exist",
		"Deal shareholder does not exist",
		"Deal investor does not exist",
		"Deal user does not exist",
		"Deal is not live yet or has closed",
		"Deal is not live or has not been submitted by user",
		"Cannot start trading deal yet",
		"You are already selling shares for this deal",
		"You are already buying shares for this deal",
		"You have already created an offer for this new deal",
		"Deal user state is invalid",
		"Deal operation error",
		"Failed to upload offer documents",
		"Offer does not exist",
		"Bank information does not exist",
		"Failed to display signing document",
		"Failed to display signing document",
		"Failed to create a new signing document",
		"Failed to create a new embedded url for signing",
		"Failed to request signing document",
		"Failed to request signing document status",
		"Failed to download signing document",
		"Not enough shares left",
		"Permission denied",
		"Internal server error",
		"You are banned from site",
		"Login JWT is invalid",
		"Login JWT contents are invalid",
		"Login JWT expired",
		"Access token is invalid",
		"Wechat account is already registered",
		"Wechat account is invalid",
		"Wechat account is binded",
		"Admin operation error",
		"User state modification is invalid for citizen type",
		"Paging arguments are invalid",
		"Company tags are invalid",
		"Company executives are invalid",
		"Company fundings are invalid",
		"Company updates are invalid",
		"Keyword is invalid",
		"ID is invalid",
		"Unknown error",
	},
	"zh-CN": []string{
		"邮箱已经被注册",
		"手机号已被注册",
		"注册错误",
		"参数 %v 格式错误",
		"邮箱未被注册",
		"手机号未被注册",
		"身份证号码不合法",
		"密码错误",
		"手机验证码错误",
		"邮箱确认码无效",
		"邮箱确认码已过期",
		"帐户激活错误",
		"密码重置码无效",
		"密码重置码已过期",
		"密码重置错误",
		"密码更新错误",
		"确认错误",
		"保密协议错误",
		"客户认证错误",
		"客户认证失败：请返回并检查您的姓名和地址。",
		"客户认证失败：请返回并检查您的姓名、生日、地址、SSN。",
		"客户认证失败：请联系客服。",
		"获取客户认证问题失败",
		"检查客户认证问题失败",
		"回答客户认证问题失败",
		"无法上传证件",
		"无法读取证件文件",
		"用户还未完成初始验证过程",
		"用户不存在",
		"用户信息更新失败",
		"用户项目状态无效",
		"公司不存在",
		"公司标签不存在",
		"公司介绍页面不存在",
		"无法读取公司信息",
		"提交自我认证失败",
		"切换到投资人时提交自我认证失败",
		"项目不存在",
		"项目股权信息不存在",
		"项目持股者不存在",
		"项目投资人不存在",
		"项目用户不存在",
		"项目还未开放或已结束",
		"项目还未开放或未被用户提交",
		"还未能开始交易股权",
		"您已在出售该股权的交易中，请点击菜单中的“我的股权”栏目确认该交易的进度",
		"您已在投资该股权的交易中，请点击菜单中的“我的股权”栏目确认该交易的进度",
		"您已经提交了该股权信息，请点击菜单中的“我的股权”栏目查看审核进度",
		"项目用户状态无效",
		"项目操作错误",
		"上传股权证明失败",
		"项目意向不存在",
		"银行信息不存在",
		"显示签字文档失败",
		"显示签字文档失败",
		"创建新签字文档失败",
		"创建新签字嵌入链接失败",
		"读取签字文档失败",
		"读取签字文档状态失败",
		"下载签字文档失败",
		"股权剩余不足",
		"权限错误",
		"服务器错误",
		"您已被屏蔽",
		"登录密钥无效",
		"登录加密内容无效",
		"登录密钥已经过期",
		"访问密钥无效",
		"微信帐号已经被注册",
		"微信帐号不合法",
		"微信账号已经被绑定",
		"管理员操作错误",
		"此用户状态修改不符合用户国籍",
		"分页参数不合法",
		"公司标签不合法",
		"公司成员不合法",
		"公司融资不合法",
		"公司新闻不合法",
		"关键词不合法",
		"ID 不合法",
		"未知错误",
	},
}

type FullName func(first, last string) string

var NameConventions = map[uint64]FullName{
	CitizenTypeCitizen: func(first, last string) string {
		return first + " " + last
	},
	CitizenTypePermanentResident: func(first, last string) string {
		return first + " " + last
	},
	CitizenTypeOther: func(first, last string) string {
		return last + first
	},
}

const (
	EmailTextName = iota
	EmailTextSubjectRegister
	EmailTextBodyRegister
	EmailTextSubjectForgetPassword
	EmailTextBodyForgetPassword
	EmailTextSubjectWireTransferAccount
	EmailTextBodyWireTransferAccount
	EmailTextSubjectInvestorApproved
	EmailTextBodyInvestorApproved
	EmailTextSubjectShareholderApproved
	EmailTextBodyShareholderApproved
)

var EmailTexts = map[string][]string{
	"en-US": []string{
		"MarketX",
		"Confirmation Email from MarketX",
		`<p>Hello %v,</p>

<p>Welcome to MarketX, an online investment platform connecting overseas investors and shareholders of U.S. pre-IPO companies. Please click link below to confirm your account.</p>

<p><a href="%v">Confirm Registration</a></p>

<p>If the link doesn't work, you could also use this link to confirm your registration:<br>
%v</p>
<p>Best regards,</p>

<p>Team MarketX</p>
`,
		"Reset Your MarketX Password",
		`<p>Hello %v,</p>

<p>Someone has requested a link to change your password, and you can do this through the link below.</p>

<p><a href="%v">Change Your Password</a></p>

<p>If you didn't request this, please ignore this email.</p>

<p>Your password won't change until you access the link above and create a new one.</p>

<p>Best regards,</p>

<p>Team MarketX</p>
`,
		"Account Information",
		`<p>Hello %v,</p>

<p>Please send your payment to our account and include your name in memo or reference for wires. Our account information is as follows:</p>

<p>Bank Name: Wells Fargo Bank, N.A.<br>
Bank Address: 420 Montgomery, San Francisco, CA 94104<br>
Beneficiary Account Number: 3595722426<br>
Beneficiary Account Name: MarketX Ventures LLC %v Fund %v<br>
Beneficiary ABA/Routing Number: 121000248<br>
Account Type: Checking: 3595722426<br>
International SWIFT BIC: WFBIUS6S</p>

<p>Thanks,</p>

<p>Team MarketX</p>
`,
		"Your account has been approved!",
		`<p>Hello %v,</p>

<p>Congratulations! We are pleased to inform you that you account has been approved. You can start browsing the investment opportunities through our investment portal. Happy investing!</p>

<p>Please click %v to continue.</p>

<p>Best regards,</p>

<p>Team MarketX</p>
`,
		"Your account has been approved!",
		`<p>Hello %v,</p>

<p>Congratulations! We are pleased to inform you that you account has been approved. Please list the opportunity to create an offer.</p>

<p>Please click %v to continue.</p>

<p>Best regards,</p>

<p>Team MarketX</p>
`,
	},
	"zh-CN": []string{
		"MarketX (源投金融)",
		"源投金融确认信",
		`<p>%v您好，</p>

<p>欢迎来到源投金融，我们是专注于帮助中国投资者购买美国未上市公司股权的跨境投资平台，项目包括Uber、Airbnb、SpaceX等。请点击下方链接确认您的账号。</p>

<p><a href="%v">确认注册</a></p>

<p>如果以上链接失效，您也可以使用以下链接确认注册：<br>
%v</p>

<p>致礼！</p>

<p>源投金融团队</p>
`,
		"重置您的源投金融密码",
		`<p>%v您好，</p>

<p>有人申请了更改您密码的链接，您可以点击以下链接进行操作。</p>

<p><a href="%v">更改您的密码</a></p>

<p>如果您并没有做此申请，请您忽略这条邮件。</p>

<p>您的密码在您点击以上链接并重新设置之前不会改变。</p>

<p>致礼！</p>

<p>源投金融团队</p>
`,
		"账户信息",
		`<p>%v您好，</p>

<p>请将您的投资额汇入我们的帐户并在备注中填写您的姓名。</p>

<p>我们的帐号信息是：</p>

<p>银行名称：Wells Fargo Bank, N.A.<br>
银行地址：420 Montgomery, San Francisco, CA 94104<br>
受益人帐号：3595722426<br>
受益人帐号名：MarketX Ventures LLC %v Fund %v<br>
受益人ABA/Routing Number：121000248<br>
帐号类型：活期：3595722426<br>
国际SWIFT BIC：WFBIUS6S</p>

<p>致礼！</p>

<p>源投金融团队</p>
	`,
		"您的帐号已通过审批！",
		`<p>%v您好！</p>

<p>恭喜您！我们很荣幸地通知您，您的帐号已通过审核。您现在就可以登陆我们的投资平台浏览投资机遇。我们衷心地希望您拥有愉悦的投资体验！</p>

<p>请您点击 %v 开启您的创投之旅！</p>

<p>致礼！</p>

<p>源投金融团队</p>
		`,
		"您的帐号已通过审批！",
		`<p>%v您好！</p>

<p>恭喜您！我们很荣幸地通知您，您的帐号已通过审核。请您登陆我们的投资平台列出您想要出售的股票。</p>

<p>请您点击 %v 继续出售流程。</p>

<p>致礼！</p>

<p>源投金融团队</p>
		`,
	},
}

const (
	SignSellEngagementLetter = iota
	SignBuyEngagementLetter
	SignBuySummaryOfTerms
	SignBuyDePpm
	SignBuyDeOperatingAgreement
	SignBuyDeSubscriptionAgreement
)

var SignTexts = map[string][]string{
	"en-US": []string{
		"Please sign Engagement Letter - Seller.",
		"Please sign Engagement Letter - Buyer.",
		"Please sign Summary of Terms.",
		"Please sign Delaware Private Placement Memorandum.",
		"Please sign Delaware Operating Agreement.",
		"Please sign Delaware Subscription Agreement.",
	},
	"zh-CN": []string{
		"请签署卖家参与协议书。",
		"请签署买家参与协议书。",
		"请签署条款概要。",
		"请签署 Delaware 私募备忘录。",
		"请签署 Delaware 经营协议。",
		"请签署 Delaware 认购协议。",
	},
}

var SharesTypeTexts = map[string][]string{
	"en-US": []string{
		"Preferred Shares",
		"Common Shares",
	},
	"zh-CN": []string{
		"优先股",
		"普通股",
	},
}

var WireTransferInfos = map[string]string{
	"en-US": `Bank Name: Wells Fargo Bank, N.A.
Bank Address: 420 Montgomery, San Francisco, CA 94104
Beneficiary Account Number:
Beneficiary Account Name: MarketX Ventures LLC %v Fund %v
Beneficiary ABA/Routing Number: 121000248
Account Type: Checking:
International SWIFT BIC: WFBIUS6S`,
	"zh-CN": `银行名称：Wells Fargo Bank, N.A.
银行地址：420 Montgomery, San Francisco, CA 94104
受益人帐号：
受益人帐号名：MarketX Ventures LLC %v Fund %v
受益人ABA/Routing Number：121000248
帐号类型：活期：
国际SWIFT BIC：WFBIUS6S`,
}
