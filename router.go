// Router (based on httprouter) for MarketX functions
package main

import (
	"bytes"
	"compress/gzip"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
)

var (
	jwtSigningSecret, _ = hex.DecodeString(jwtSecretKey)
	jwtSigningMethod    = jwt.SigningMethodHS512
	aesTextSecret, _    = hex.DecodeString(aesTextKey)
	aesFileSecret, _    = hex.DecodeString(aesFileKey)
	companySecret       = compSecretPrefix
	gzipFiles           = []string{".html", ".css", ".js"}
)

const (
	jwtSigningDuration = time.Hour * 24 * 7
	maxRequestBody     = 10 * 1024 * 1024
)

// formatReturnJson returns a json-formatted http response to client
func formatReturnJson(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, ec ErrorCode, eci string, login int64,
	ie map[string]interface{}) {
	// Create new map not to taint the argument
	args := map[string]interface{}{}
	for k, v := range ie {
		args[k] = v
	}
	if ec == ErrorCodeNone {
		args["result"] = ResultSuccess
	} else {
		args["result"] = ResultFail
		reqLang := ps[len(ps)-2].Value
		ecs := ErrorCodes[reqLang][ec]
		if eci != "" {
			args["error"] = fmt.Sprintf(ecs, eci)
		} else {
			args["error"] = ecs
		}
	}
	if login != LoginNone {
		args["login"] = login
	}
	ret, err := json.Marshal(args)
	var rs string
	if err != nil {
		// Should not happen but return an indicator
		rs = `{"result":0,"error":"?"}`
	} else {
		rs = string(ret)
	}
	fmt.Fprintf(w, rs)

	// Log the response (the only exit point)
	reqId := ps[len(ps)-1].Value
	serverLog.Printf("[MXAPI] (%v) |%v| RETURN: %v\n", reqId,
		getIp(r), rs)
}

// formatReturnInfo is a wrapper for formatReturnJson that is only available
// for website-related apis (with jwt) and attaches more information
// about the return message
func formatReturnInfo(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, ec ErrorCode, eci string, login bool,
	ie map[string]interface{}) {
	loginState := LoginLoggedOut
	if login {
		loginState = LoginLoggedIn
	}
	formatReturnJson(w, r, ps, ec, eci, int64(loginState), ie)
}

// formatReturn is a wrapper for formatReturnInfo that represents a generic return
func formatReturn(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, ec ErrorCode, login bool,
	ie map[string]interface{}) {
	formatReturnInfo(w, r, ps, ec, "", login, ie)
}

// parseFileUpload reads a file in a multipart form post upload
// and saves to a destined location with encryption
func parseFileUpload(r *http.Request, name string, fileDir string,
	fileDirId interface{}, fileName string) (string, string, string, error) {
	// Read the file contents
	f, header, err := r.FormFile(name)
	if err != nil {
		return "", "", "", err
	}
	defer f.Close()

	// Directory to store uploaded file
	pidc := path.Join(dataDir, fileDir, fmt.Sprintf("%v", fileDirId))
	err = os.MkdirAll(pidc, os.ModePerm)
	if err != nil {
		return "", "", "", err
	}

	// Now store file
	pifc := path.Join(pidc, fileName)
	out, err := os.Create(pifc)
	if err != nil {
		return "", "", "", err
	}
	defer out.Close()

	// Apply encryption
	err = EncryptStream(aesFileSecret, f, out)
	if err != nil {
		return "", "", "", err
	}

	// Save link in case we change storage types in the future
	return pifc, header.Filename, header.Header.Get("Content-Type"), nil
}

// saveFileUpload reads a file in a multipart form post upload
// and saves to a destined location directly
func saveFileUpload(r *http.Request, name string,
	saveDir, fileName string) error {
	// Read the file contents
	f, _, err := r.FormFile(name)
	if err != nil {
		return err
	}
	defer f.Close()

	// Directory to store uploaded file
	pidc := path.Join(saveDir)
	err = os.MkdirAll(pidc, os.ModePerm)
	if err != nil {
		return err
	}

	// Now store file
	pifc := path.Join(pidc, fileName)
	out, err := os.Create(pifc)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write file
	_, err = io.Copy(out, f)
	if err != nil {
		return err
	}

	return nil
}

// createFileTokenName takes a filebase and filetype then creates a token for
// download/preview along with the filename extended with suffix
func createFileTokenName(fbase, ftype string) (string, string) {
	b := make([]byte, TokenMinMax/2)
	crand.Read(b)
	token := fmt.Sprintf("%x", b)
	if ftype == "application/octet-stream" {
		return token, fbase
	}
	cts := strings.Split(ftype, "/")
	return token, fbase + "." + cts[len(cts)-1]
}

// parseFileDownload is the counterpart to parseFileUpload - which decrypts
// files from storages and returns to clients
func parseFileDownload(w http.ResponseWriter, r *http.Request,
	ct string, pifc string) error {
	f, err := os.Open(pifc)
	// TODO: Return an error file here
	if err != nil {
		return err
	}
	defer f.Close()

	// Perform decryption of file
	file, err := DecryptStreamBytes(aesFileSecret, f)
	// TODO: Return an error file here
	if err != nil {
		return err
	}

	// Backwards compatible - default to file instead of base64
	b64, _ := CheckRange(true, r.FormValue("base64"), NoYesMax)
	if b64 == Base64Yes {
		fmt.Fprintf(w, "%s", base64.StdEncoding.EncodeToString(file))
		return nil
	}

	// Set header for accuracy
	w.Header().Set("Content-Type", ct)
	// If type is readable, infer name
	if ct != "application/octet-stream" {
		cts := strings.Split(ct, "/")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("inline; filename=\"%v.%v\"",
				filepath.Base(pifc), cts[len(cts)-1]))
	}
	// Other fields will be auto-filled
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(file))
	return nil
}

// mxHandle is the handler function for MarketX specific functionalities
type mxHandle func(http.ResponseWriter, *http.Request, httprouter.Params,
	*User)

// checkAuth performs the basic operations for jwt-authentication
// and returns error code if failed
func checkAuth(r *http.Request) (*User, ErrorCode) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, ErrorCodeJwtError
	}

	// Check jwt validity
	token, err := jwt.Parse(auth[7:], func(token *jwt.Token) (interface{},
		error) {
		if reflect.ValueOf(token.Method).Pointer() !=
			reflect.ValueOf(jwtSigningMethod).Pointer() {
			return nil, fmt.Errorf("Unexpected signing method: %v",
				token.Header["alg"])
		}
		return jwtSigningSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrorCodeJwtError
	}

	// Check for jwt expiration
	claims := token.Claims.(jwt.MapClaims)
	expire, found := claims["expire"]
	if !found {
		return nil, ErrorCodeJwtStoreError
	}
	exp, ok := expire.(float64)
	if !ok {
		return nil, ErrorCodeJwtStoreError
	}
	if time.Now().After(time.Unix(int64(exp), 0)) {
		return nil, ErrorCodeJwtExpired
	}

	// Check user id value availability
	userId, found := claims["user_id"]
	if !found {
		return nil, ErrorCodeJwtStoreError
	}
	uid, ok := userId.(float64)
	if !ok {
		return nil, ErrorCodeJwtStoreError
	}

	// Check user availability
	var currentUser User
	if dbConn.First(&currentUser, int64(uid)).RecordNotFound() {
		return nil, ErrorCodeUserUnknown
	}

	// Clean up if user is banned
	if currentUser.UserState == UserStateBanned {
		return nil, ErrorCodeUserBanned
	}

	return &currentUser, ErrorCodeNone
}

// authProtect checks user credientials and returns
// generic failure json if permission is denied
func authProtect(h mxHandle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request,
		ps httprouter.Params) {
		u, ec := checkAuth(r)
		if ec != ErrorCodeNone {
			formatReturn(w, r, ps, ec, false, nil)
			return
		}

		// Passed all checks, continue
		h(w, r, ps, u)
	}
}

// adminProtect checks user credentials and only allows
// admin-level users to proceed, otherwise return failure
func adminProtect(h mxHandle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request,
		ps httprouter.Params) {
		u, ec := checkAuth(r)
		if ec != ErrorCodeNone {
			formatReturn(w, r, ps, ec, false, nil)
			return
		}

		if u.UserLevel != UserLevelAdmin {
			formatReturn(w, r, ps, ErrorCodeNoPermission, false, nil)
			return
		}

		// Passed all checks, continue
		h(w, r, ps, u)
	}
}

// tokenProtect checks whether the access token of the api call
// is valid and returns failure result + error message
func tokenProtect(h httprouter.Handle, tokens []string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request,
		ps httprouter.Params) {
		// token is a required field
		token, ok := CheckLength(true, r.FormValue("token"),
			TokenMinMax, TokenMinMax)
		if !ok {
			formatReturnJson(w, r, ps, ErrorCodeTokenError, "",
				LoginNone, nil)
			return
		}
		for _, t := range tokens {
			if token == t {
				h(w, r, ps)
				return
			}
		}
		formatReturnJson(w, r, ps, ErrorCodeTokenError, "", LoginNone, nil)
	}
}

// saveLogin is called after login is successful and a new jwt is created
// if on initial (login) call api (register/login/recover)
// Now serves as a common "success" return point after login
func saveLogin(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, initial bool, u *User,
	ie map[string]interface{}) {
	// Clear out user session
	if u.UserState == UserStateBanned {
		formatReturn(w, r, ps, ErrorCodeUserBanned, false, nil)
		return
	}

	if ie == nil {
		ie = map[string]interface{}{}
	}
	if initial {
		token := jwt.New(jwtSigningMethod)
		// Set claims for user id and expiration
		claims := token.Claims.(jwt.MapClaims)
		claims["user_id"] = u.ID
		claims["expire"] = time.Now().Add(jwtSigningDuration).Unix()
		tokenString, err := token.SignedString(jwtSigningSecret)
		if err != nil {
			formatReturn(w, r, ps, ErrorCodeJwtError, false, nil)
			return
		}
		// Save for return
		ie["token"] = tokenString
	}

	// TODO: Probably need to merge this somewhere to save one db save
	u.LastIpAddress = getIp(r)
	u.LastLanguage = ps[len(ps)-2].Value
	// Do not fail on ip address save
	dbConn.Save(u)

	// We continue with good state even if failed to save:
	// we can continue and let the next call fail, otherwise we could end
	// up at a state where user logged in but state says no (orphaned)
	ie["user_state"] = u.UserState
	ie["user_level"] = u.UserLevel
	ie["role_type"] = u.RoleType
	formatReturn(w, r, ps, ErrorCodeNone, true, ie)
}

// saveAdmin is the normal return point after an admin operation
// Future common ground for priviledged returns
func saveAdmin(w http.ResponseWriter, r *http.Request,
	ps httprouter.Params, u *User, ie map[string]interface{}) {
	// TODO: Probably need to merge this somewhere to save one db save
	u.LastIpAddress = getIp(r)
	// Do not fail on ip address save
	dbConn.Save(u)

	formatReturn(w, r, ps, ErrorCodeNone, true, ie)
}

// getIpAddress reads in a concatenated ip:port from http.Request
// and returns the ip address if available, "" otherwise
func getIp(r *http.Request) string {
	i := strings.LastIndex(r.RemoteAddr, ":")
	if i < 0 {
		return ""
	}
	return strings.Trim(r.RemoteAddr[:i], "[]")
}

// logProtect wraps REST-like requests with properly searchable
// logging so we don't log everything such as static files
func logProtect(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request,
		ps httprouter.Params) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		r.ParseMultipartForm(0)
		reqId := fmt.Sprintf("%v", rand.Int63())
		serverLog.Printf("[MXAPI] (%v) |%v| %v: %v?%v\n", reqId,
			getIp(r), r.Method, r.URL, r.Form.Encode())
		// Store current language (default to English)
		reqLang := r.Header.Get("Language")
		if reqLang != "en-US" && reqLang != "zh-CN" {
			reqLang = "en-US"
		}
		// Store request language as a hack in httprouter params
		ps = append(ps, httprouter.Param{Key: "", Value: reqLang})
		// Store request id as a hack in httprouter params
		ps = append(ps, httprouter.Param{Key: "", Value: reqId})
		h(w, r, ps)
	}
}

// Gzip-related responses
type GzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// newServerRouter returns a valid router to handle
func newServerRouter() *httprouter.Router {
	// Init rand seed for request ids
	rand.Seed(time.Now().UTC().UnixNano())

	router := httprouter.New()

	// --- Account ---
	router.POST("/account/send_mobile_code", logProtect(mobileSendVerificationCodeHandler))
	router.POST("/account/verify_mobile_code", logProtect(mobileCheckVerificationCodeHandler))
	router.POST("/account/register", logProtect(accountRegisterHandler))
	router.POST("/account/login", logProtect(accountLoginHandler))
	router.POST("/account/confirm", logProtect(authProtect(accountConfirmHandler)))
	router.POST("/account/recover", logProtect(accountRecoverHandler))
	router.POST("/account/forget", logProtect(accountForgetHandler))
	router.POST("/account/change", logProtect(authProtect(accountChangeHandler)))

	router.PUT("/account/stateupdate", logProtect(authProtect(accountStateUpdate)))

	router.PUT("/account/wxlogin", logProtect(wechatLoginHandler))
	router.PUT("/account/wxbind", logProtect(wechatBindHandler))

	// --- User ---
	router.PUT("/user/nda", logProtect(authProtect(userNdaHandler)))
	router.PUT("/user/kyc", logProtect(authProtect(userKycHandler)))
	router.GET("/user/kyc_questions",
		logProtect(authProtect(userKycQuestionsHandler)))
	router.PUT("/user/kyc_check",
		logProtect(authProtect(userKycCheckHandler)))
	router.PUT("/user/self_accred",
		logProtect(authProtect(userSelfAccredHandler)))
	router.POST("/user/upload_id",
		logProtect(authProtect(userUploadIdHandler)))
	router.PUT("/user/self_accred_switch",
		logProtect(authProtect(userSelfAccredSwitchHandler)))
	router.GET("/user", logProtect(authProtect(userHandler)))
	router.PUT("/user", logProtect(authProtect(userUpdateHandler)))
	router.GET("/user/photo_id", logProtect(authProtect(userPhotoIdHandler)))
	router.GET("/user/photo_id/:token", logProtect(userPhotoIdTokenHandler))
	router.GET("/user/sells", logProtect(authProtect(userSellsHandler)))
	router.GET("/user/buys", logProtect(authProtect(userBuysHandler)))

	// --- Company ---
	router.GET("/companies", logProtect(companiesHandler))
	router.GET("/company/:id", logProtect(authProtect(companyIdHandler)))

	// --- Deal ---
	router.GET("/deals", logProtect(authProtect(dealsHandler)))

	// --- Deal / Sell ---
	router.POST("/deal/:id/sell/engagement_letter_sign",
		logProtect(authProtect(dealSellEngagementLetterSignHandler)))
	router.GET("/deal/:id/sell/engagement_letter_check",
		logProtect(authProtect(dealSellEngagementLetterCheckHandler)))
	router.POST("/deal/:id/sell/new_offer",
		logProtect(authProtect(dealSellNewOfferHandler)))
	router.POST("/deal_new/sell/new_offer",
		logProtect(authProtect(dealNewSellNewOfferHandler)))
	router.GET("/deal/:id/sell",
		logProtect(authProtect(dealSellHandler)))
	router.GET("/deal/:id/sell/share_certificate",
		logProtect(authProtect(dealSellShareCertificateHandler)))
	router.GET("/deal/:id/sell/share_certificate/:token",
		logProtect(dealSellShareCertificateTokenHandler))
	router.GET("/deal/:id/sell/company_by_laws",
		logProtect(authProtect(dealSellCompanyByLawsHandler)))
	router.GET("/deal/:id/sell/company_by_laws/:token",
		logProtect(dealSellCompanyByLawsTokenHandler))
	router.GET("/deal/:id/sell/shareholder_agreement",
		logProtect(authProtect(dealSellShareholderAgreementHandler)))
	router.GET("/deal/:id/sell/shareholder_agreement/:token",
		logProtect(dealSellShareholderAgreementTokenHandler))
	router.GET("/deal/:id/sell/stock_option_plan",
		logProtect(authProtect(dealSellStockOptionPlanHandler)))
	router.GET("/deal/:id/sell/stock_option_plan/:token",
		logProtect(dealSellStockOptionPlanTokenHandler))
	router.GET("/deal/:id/sell/engagement_letter",
		logProtect(authProtect(dealSellEngagementLetterHandler)))
	router.GET("/deal/:id/sell/engagement_letter/:token",
		logProtect(dealSellEngagementLetterTokenHandler))
	router.POST("/deal/:id/sell/bank_info",
		logProtect(authProtect(dealSellBankInfoHandler)))

	// --- Deal / Buy ---
	router.POST("/deal/:id/buy/engagement_letter_sign",
		logProtect(authProtect(dealBuyEngagementLetterSignHandler)))
	router.GET("/deal/:id/buy/engagement_letter_check",
		logProtect(authProtect(dealBuyEngagementLetterCheckHandler)))
	router.POST("/deal/:id/buy/new_interest",
		logProtect(authProtect(dealBuyNewInterestHandler)))
	router.GET("/deal/:id/buy",
		logProtect(authProtect(dealBuyHandler)))
	router.GET("/deal/:id/buy/engagement_letter",
		logProtect(authProtect(dealBuyEngagementLetterHandler)))
	router.GET("/deal/:id/buy/engagement_letter/:token",
		logProtect(dealBuyEngagementLetterTokenHandler))
	router.GET("/deal/:id/buy/summary_terms",
		logProtect(authProtect(dealBuySummaryTermsHandler)))
	router.GET("/deal/:id/buy/summary_terms/:token",
		logProtect(dealBuySummaryTermsTokenHandler))
	router.GET("/deal/:id/buy/de_ppm",
		logProtect(authProtect(dealBuyDePpmHandler)))
	router.GET("/deal/:id/buy/de_ppm/:token",
		logProtect(dealBuyDePpmTokenHandler))
	router.GET("/deal/:id/buy/de_operating",
		logProtect(authProtect(dealBuyDeOperatingHandler)))
	router.GET("/deal/:id/buy/de_operating/:token",
		logProtect(dealBuyDeOperatingTokenHandler))
	router.GET("/deal/:id/buy/de_subscription",
		logProtect(authProtect(dealBuyDeSubscriptionHandler)))
	router.GET("/deal/:id/buy/de_subscription/:token",
		logProtect(dealBuyDeSubscriptionTokenHandler))
	router.POST("/deal/:id/buy/summary_terms_sign",
		logProtect(authProtect(dealBuySummaryTermsSignHandler)))
	router.GET("/deal/:id/buy/summary_terms_check",
		logProtect(authProtect(dealBuySummaryTermsCheckHandler)))
	router.POST("/deal/:id/buy/bank_info",
		logProtect(authProtect(dealBuyBankInfoHandler)))
	router.POST("/deal/:id/buy/de_ppm_sign",
		logProtect(authProtect(dealBuyDePpmSignHandler)))
	router.GET("/deal/:id/buy/de_ppm_check",
		logProtect(authProtect(dealBuyDePpmCheckHandler)))
	router.POST("/deal/:id/buy/de_operating_sign",
		logProtect(authProtect(dealBuyDeOperatingSignHandler)))
	router.GET("/deal/:id/buy/de_operating_check",
		logProtect(authProtect(dealBuyDeOperatingCheckHandler)))
	router.POST("/deal/:id/buy/de_subscription_sign",
		logProtect(authProtect(dealBuyDeSubscriptionSignHandler)))
	router.GET("/deal/:id/buy/de_subscription_check",
		logProtect(authProtect(dealBuyDeSubscriptionCheckHandler)))

	// --- Admin / User ---
	router.GET("/admin/users", logProtect(adminProtect(adminUsersHandler)))
	router.GET("/admin/user/:id",
		logProtect(adminProtect(adminUserIdHandler)))
	router.PUT("/admin/user/:id",
		logProtect(adminProtect(adminUserUpdateHandler)))
	router.GET("/admin/user/:id/photo_id",
		logProtect(adminProtect(adminUserPhotoIdHandler)))
	router.GET("/admin/user/:id/photo_id/:token",
		logProtect(adminUserPhotoIdTokenHandler))
	router.POST("/admin/user/:id/photo_id",
		logProtect(adminProtect(adminUserPhotoIdUploadHandler)))
	router.DELETE("/admin/user/:id",
		logProtect(adminProtect(adminUserDeleteHandler)))

	// --- Admin / Company ---
	router.GET("/admin/companies",
		logProtect(adminProtect(adminCompaniesHandler)))
	router.GET("/admin/company_tags",
		logProtect(adminProtect(adminCompanyTagsHandler)))
	router.GET("/admin/company/:id",
		logProtect(adminProtect(adminCompanyIdHandler)))
	router.GET("/admin/company_tag/:id",
		logProtect(adminProtect(adminCompanyTagIdHandler)))
	router.PUT("/admin/company/:id",
		logProtect(adminProtect(adminCompanyUpdateHandler)))
	router.POST("/admin/company",
		logProtect(adminProtect(adminCompanyAddHandler)))
	router.POST("/admin/company/:id/logo",
		logProtect(adminProtect(adminCompanyLogoUpdateHandler)))
	router.POST("/admin/company/:id/bg",
		logProtect(adminProtect(adminCompanyBgUpdateHandler)))
	router.POST("/admin/company/:id/slide_en/:slide_id",
		logProtect(adminProtect(adminCompanySlideEnUpdateHandler)))
	router.POST("/admin/company/:id/slide_zh/:slide_id",
		logProtect(adminProtect(adminCompanySlideZhUpdateHandler)))
	router.PUT("/admin/company_tag/:id",
		logProtect(adminProtect(adminCompanyTagUpdateHandler)))
	router.POST("/admin/company_tag",
		logProtect(adminProtect(adminCompanyTagAddHandler)))
	router.DELETE("/admin/company/:id",
		logProtect(adminProtect(adminCompanyDeleteHandler)))

	// --- Admin / Deal ---
	router.GET("/admin/deals", logProtect(adminProtect(adminDealsHandler)))
	router.GET("/admin/deal/:id",
		logProtect(adminProtect(adminDealIdHandler)))
	router.PUT("/admin/deal/:id",
		logProtect(adminProtect(adminDealUpdateHandler)))
	router.POST("/admin/deal", logProtect(adminProtect(adminDealAddHandler)))
	router.DELETE("/admin/deal/:id",
		logProtect(adminProtect(adminDealDeleteHandler)))

	// --- Admin / Deal / Sell ---
	router.GET("/admin/deal/:id/sells",
		logProtect(adminProtect(adminDealSellsHandler)))
	router.GET("/admin/deal/:id/sell/:sell_id",
		logProtect(adminProtect(adminDealSellHandler)))
	router.GET("/admin/deal/:id/sell/:sell_id/share_certificate",
		logProtect(adminProtect(adminDealSellShareCertificateHandler)))
	router.GET("/admin/deal/:id/sell/:sell_id/share_certificate/:token",
		logProtect(adminDealSellShareCertificateTokenHandler))
	router.GET("/admin/deal/:id/sell/:sell_id/company_by_laws",
		logProtect(adminProtect(adminDealSellCompanyByLawsHandler)))
	router.GET("/admin/deal/:id/sell/:sell_id/company_by_laws/:token",
		logProtect(adminDealSellCompanyByLawsTokenHandler))
	router.GET("/admin/deal/:id/sell/:sell_id/shareholder_agreement",
		logProtect(adminProtect(adminDealSellShareholderAgreementHandler)))
	router.GET("/admin/deal/:id/sell/:sell_id/shareholder_agreement/:token",
		logProtect(adminDealSellShareholderAgreementTokenHandler))
	router.GET("/admin/deal/:id/sell/:sell_id/stock_option_plan",
		logProtect(adminProtect(adminDealSellStockOptionPlanHandler)))
	router.GET("/admin/deal/:id/sell/:sell_id/stock_option_plan/:token",
		logProtect(adminDealSellStockOptionPlanTokenHandler))
	router.GET("/admin/deal/:id/sell/:sell_id/engagement_letter",
		logProtect(adminProtect(adminDealSellEngagementLetterHandler)))
	router.GET("/admin/deal/:id/sell/:sell_id/engagement_letter/:token",
		logProtect(adminDealSellEngagementLetterTokenHandler))
	router.POST("/admin/deal/:id/sell/:sell_id",
		logProtect(adminProtect(adminDealSellUpdateHandler)))
	router.DELETE("/admin/deal/:id/sell/:sell_id",
		logProtect(adminProtect(adminDealSellDeleteHandler)))

	// --- Admin / Deal / Buy ---
	router.GET("/admin/deal/:id/buys",
		logProtect(adminProtect(adminDealBuysHandler)))
	router.GET("/admin/deal/:id/buy/:buy_id",
		logProtect(adminProtect(adminDealBuyHandler)))
	router.GET("/admin/deal/:id/buy/:buy_id/engagement_letter",
		logProtect(adminProtect(adminDealBuyEngagementLetterHandler)))
	router.GET("/admin/deal/:id/buy/:buy_id/engagement_letter/:token",
		logProtect(adminDealBuyEngagementLetterTokenHandler))
	router.GET("/admin/deal/:id/buy/:buy_id/summary_terms",
		logProtect(adminProtect(adminDealBuySummaryTermsHandler)))
	router.GET("/admin/deal/:id/buy/:buy_id/summary_terms/:token",
		logProtect(adminDealBuySummaryTermsTokenHandler))
	router.GET("/admin/deal/:id/buy/:buy_id/de_ppm",
		logProtect(adminProtect(adminDealBuyDePpmHandler)))
	router.GET("/admin/deal/:id/buy/:buy_id/de_ppm/:token",
		logProtect(adminDealBuyDePpmTokenHandler))
	router.GET("/admin/deal/:id/buy/:buy_id/de_operating",
		logProtect(adminProtect(adminDealBuyDeOperatingHandler)))
	router.GET("/admin/deal/:id/buy/:buy_id/de_operating/:token",
		logProtect(adminDealBuyDeOperatingTokenHandler))
	router.GET("/admin/deal/:id/buy/:buy_id/de_subscription",
		logProtect(adminProtect(adminDealBuyDeSubscriptionHandler)))
	router.GET("/admin/deal/:id/buy/:buy_id/de_subscription/:token",
		logProtect(adminDealBuyDeSubscriptionTokenHandler))
	router.POST("/admin/deal/:id/buy/:buy_id",
		logProtect(adminProtect(adminDealBuyUpdateHandler)))
	router.DELETE("/admin/deal/:id/buy/:buy_id",
		logProtect(adminProtect(adminDealBuyDeleteHandler)))

	// --- Wechat ---
	router.POST("/wechat",
		logProtect(tokenProtect(wechatAddHandler, wechatTokens)))
	router.GET("/wechat/:id",
		logProtect(tokenProtect(wechatIdHandler, wechatTokens)))

	// Redirect not found to home page
	router.HandleMethodNotAllowed = false
	router.NotFound = http.RedirectHandler("/", http.StatusMovedPermanently)

	// Loop the (static) client dir and add relevant handlers
	files, err := ioutil.ReadDir(clientDir)
	if err != nil {
		serverLog.Println("[MX] Cannot read client directory")
	} else {
		// Mimic httprouter behavior of serving static files
		// without all the panics
		fserver := http.FileServer(http.Dir(clientDir))
		for _, f := range files {
			name := "/"
			fname := f.Name()
			// Save company secret if found
			if strings.HasPrefix(fname, compSecretPrefix) {
				companySecret = fname
			}
			if f.IsDir() {
				name += fname + "/*filepath"
			} else if fname != "index.html" {
				name += fname
			} // special case "/"
			rh := func(w http.ResponseWriter, r *http.Request,
				ps httprouter.Params) {
				// Disable directory listing if no index.html is available
				fp := strings.Replace(ps.ByName("filepath"),
					"/", string(filepath.Separator), -1)
				rp := path.Join(clientDir, fname, fp)
				fi, err := os.Stat(rp)
				if err != nil {
					router.NotFound.ServeHTTP(w, r)
					return
				}
				if fi.IsDir() {
					// Folder is a special case of index.html
					rp = path.Join(rp, "index.html")
					ifi, err := os.Stat(rp)
					// If index.html is not available then nothing to show
					if err != nil || ifi.IsDir() {
						router.NotFound.ServeHTTP(w, r)
						return
					}
				}

				// Compress text files when client supports for our
				// allowed files
				if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
					for _, tfs := range gzipFiles {
						if !strings.HasSuffix(rp, tfs) {
							continue
						}
						w.Header().Set("Vary", "Accept-Encoding")
						w.Header().Set("Content-Encoding", "gzip")
						gz := gzip.NewWriter(w)
						defer gz.Close()
						// Replace current with gzipped writer
						w = GzipResponseWriter{Writer: gz, ResponseWriter: w}
						break
					}
				}
				fserver.ServeHTTP(w, r)
			}
			router.GET(name, rh)
		}
		serverLog.Println("[MX] Mounted client file handlers")
	}

	return router
}
