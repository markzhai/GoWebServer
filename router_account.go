// Router branch for /account/ operations
package main

import (
	crand "crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
)

var cache, ignore = lru.NewARC(1000)

func mobileSendVerificationCodeHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params) {
	phoneNumber, of := CheckLengthForm("", r, "phone_number",
		PhoneMin, PhoneMax)

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	code := generateCode()
	cache.Add(phoneNumber, code)
	send(phoneNumber, code)

	formatReturn(w, r, ps, ErrorCodeNone, false,
		map[string]interface{}{
			"code":         code,
			"phone_number": phoneNumber,
		})
}

func mobileCheckVerificationCodeHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params) {
	phoneNumber, of := CheckLengthForm("", r, "phone_number", PhoneMin, PhoneMax)
	code, of := CheckFieldForm(of, r, "code")

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	val, found := cache.Get(phoneNumber)

	if !found || code != val {
		formatReturn(w, r, ps, ErrorCodePhoneCodeError, false, nil)
		return
	}

	formatReturn(w, r, ps, ErrorCodeNone, false, nil)
}

func wechatLoginHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	openid, of := CheckFieldForm("", r, "openid")
	unionid, of := CheckFieldForm(of, r, "unionid")

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	// Email must match
	var currentUser User

	if unionid != "" {
		if dbConn.First(&currentUser, "wx_open_id = ?", openid).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodePhoneUnknown, false, nil)
			return
		}
	} else {
		if dbConn.First(&currentUser, "wx_union_id = ?", unionid).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodeEmailUnknown, false, nil)
			return
		}
	}

	saveLogin(w, r, ps, true, &currentUser, nil)
}

func wechatBindHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	openid, of := CheckFieldForm("", r, "openid")
	unionid, of := CheckFieldForm(of, r, "unionid")
	password, of := CheckLengthForm(of, r, "password", PasswordMinMax, PasswordMinMax)

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	email, of := CheckEmailForm(of, r, "email")
	phoneNumber := ""
	if of != "" {
		phoneNumber, of = CheckLengthForm("", r, "email", PhoneMin, PhoneMax)
	}
	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	var currentUser User

	if phoneNumber != "" {
		if dbConn.First(&currentUser, "phone_number = ?", phoneNumber).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodePhoneUnknown, false, nil)
			return
		}
	} else {
		email = strings.ToLower(email)
		if dbConn.First(&currentUser, "email = ?", email).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodeEmailUnknown, false, nil)
			return
		}
	}

	// Password must match hash
	mismatch := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash),
		[]byte(password))
	if mismatch != nil {
		formatReturn(w, r, ps, ErrorCodeBadPassword, false, nil)
		return
	}

	if currentUser.WxOpenID != "" || currentUser.WxUnionID != "" {
		formatReturn(w, r, ps, ErrorCodeWechatBinded, false, nil)
		return
	}

	currentUser.WxUnionID = unionid
	currentUser.WxOpenID = openid

	saveLogin(w, r, ps, true, &currentUser, nil)
}

func accountRegisterHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	firstName, ignore := CheckFieldForm("", r, "first_name")
	lastName, ignore := CheckFieldForm("", r, "last_name")
	roleType, of := CheckRangeForm("", r, "role_type", RoleTypeInvestor)
	citizenType, of := CheckRangeForm(of, r, "citizen_type", CitizenTypeOther)

	var phoneNumber = ""
	var email = ""

	if citizenType == CitizenTypeOther {
		phoneNumber, of = CheckLengthForm(of, r, "phone_number", PhoneMin, PhoneMax)
		code, of := CheckFieldForm(of, r, "code")
		if of != "" {
			formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
			return
		}

		val, found := cache.Get(phoneNumber)

		if !found || code != val {
			formatReturn(w, r, ps, ErrorCodePhoneCodeError, false, nil)
			return
		}
		// phone verification code okay, just remove it.
		cache.Remove(phoneNumber)

	} else {
		email, of = CheckEmailForm(of, r, "email")
	}

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	password, of := CheckLengthForm("", r, "password", PasswordMinMax, PasswordMinMax)

	email = strings.ToLower(email)
	if citizenType == CitizenTypeOther {
		if !dbConn.First(&User{}, "phone_number = ?", phoneNumber).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodePhoneExists, false, nil)
			return
		}
	} else {
		if !dbConn.First(&User{}, "email = ?", email).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodeEmailExists, false, nil)
			return
		}
	}

	passwordHashStr := ""
	if password != "" {
		// Previous email check serves as an unsafe "barrier",
		// but if even two goroutines both pass through to here at the same time,
		// only one Create succeeds
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password),
			bcrypt.DefaultCost)
		// Return unknown register error here
		if err != nil {
			formatReturnInfo(w, r, ps, ErrorCodeRegisterError, err.Error(), false, nil)
			//formatReturn(w, r, ps, ErrorCodeRegisterError, false, nil)
			return
		}
		passwordHashStr = string(passwordHash)
	}

	// Generate email activation code
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	emailToken := fmt.Sprintf("%x", b)

	fullName := ""
	if ignore == "" {
		fullName = NameConventions[citizenType](firstName, lastName)
	}

	// Infer country from citizen type
	// In the future this should be an input as well
	country := "US"
	if citizenType == CitizenTypeOther {
		country = "CN"
	}
	newUser := User{
		FirstName:         firstName,
		LastName:          lastName,
		FullName:          fullName,
		Email:             email,
		EmailToken:        emailToken,
		EmailTokenExpire:  time.Now().Add(tokenExpiration),
		PhoneNumber:       phoneNumber,
		PasswordHash:      passwordHashStr,
		RoleType:          roleType,
		CitizenType:       citizenType,
		Country:           country,
		CreationIpAddress: getIp(r)}
	if dbConn.Create(&newUser).Error != nil {
		formatReturnInfo(w, r, ps, ErrorCodeRegisterError, dbConn.Create(&newUser).Error.Error(), false, nil)
		//formatReturn(w, r, ps, ErrorCodeRegisterError, false, nil)
		return
	}

	if citizenType != CitizenTypeOther {
		// Now send activation email non-blocking
		reqLang := ps[len(ps) - 2].Value
		author := EmailTexts[reqLang][EmailTextName]
		subject := EmailTexts[reqLang][EmailTextSubjectRegister]
		link := fmt.Sprintf(emailConfirmLink, serverDomain, emailToken)
		body := fmt.Sprintf(EmailTexts[reqLang][EmailTextBodyRegister],
			fullName, link, link)
		sendMail(author, email, fullName, subject, body)
	}

	saveLogin(w, r, ps, true, &newUser,
		map[string]interface{}{
			"email_token": emailToken,
		})
}

// TODO: it should be an account confirm function from phone code verification
func accountStateUpdate(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	if u.UserState < UserStateConfirmed {
		u.UserState = UserStateConfirmed
		if dbConn.Save(u).Error != nil {
			formatReturn(w, r, ps, ErrorCodeActivationError, true, nil)
			return
		}
	}

	saveLogin(w, r, ps, false, u, nil)
}

func accountLoginHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params) {

	confirm, cof := CheckLengthForm("", r, "confirm",
		TokenMinMax, TokenMinMax)

	// Confirming email path takes precedence
	if cof == "" {
		var currentUser User
		if dbConn.First(&currentUser,
			"email_token = ?", confirm).RecordNotFound() {
			// Could be refreshing the page right after confirmation
			// so we check the jwt
			u, ec := checkAuth(r)
			if ec != ErrorCodeNone {
				formatReturn(w, r, ps, ErrorCodeEmailTokenInvalid, false, nil)
				return
			}
			currentUser = *u
		} else if currentUser.EmailTokenExpire.Before(time.Now()) {
			// If token found, check for expiration
			formatReturn(w, r, ps, ErrorCodeEmailTokenExpired, false, nil)
			return
		}

		// If user state isn't inactive there is nothing to confirm
		if currentUser.UserState < UserStateInactive {
			formatReturn(w, r, ps, ErrorCodeActivationError, false, nil)
			return
		}

		// Clear token use and advance user state
		currentUser.EmailToken = ""
		currentUser.EmailTokenExpire = time.Time{}
		if currentUser.UserState < UserStateConfirmed {
			currentUser.UserState = UserStateNdaAgreed //UserStateConfirmed
		}
		if dbConn.Save(&currentUser).Error != nil {
			formatReturn(w, r, ps, ErrorCodeActivationError, false, nil)
			return
		}

		saveLogin(w, r, ps, true, &currentUser, nil)
		return
	}

	var phoneNumber = ""

	email, of := CheckEmailForm("", r, "email")
	// If error, try parse as phone number
	if of != "" {
		of = ""
		phoneNumber, of = CheckLengthForm("", r, "email", PhoneMin, PhoneMax)
	}

	password, of := CheckLengthForm(of, r, "password",
		PasswordMinMax, PasswordMinMax)

	// Now make sure we can proceed with valid login credentials
	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	// Email must match
	var currentUser User

	if phoneNumber != "" {
		if dbConn.First(&currentUser, "phone_number = ?", phoneNumber).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodePhoneUnknown, false, nil)
			return
		}
	} else {
		email = strings.ToLower(email)
		if dbConn.First(&currentUser, "email = ?", email).RecordNotFound() {
			formatReturn(w, r, ps, ErrorCodeEmailUnknown, false, nil)
			return
		}
	}

	// Password must match hash
	mismatch := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash),
		[]byte(password))
	if mismatch != nil {
		formatReturn(w, r, ps, ErrorCodeBadPassword, false, nil)
		return
	}

	saveLogin(w, r, ps, true, &currentUser, nil)
}

func accountConfirmHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	// If user state isn't inactive there is nothing to confirm
	if u.UserState != UserStateInactive {
		formatReturn(w, r, ps, ErrorCodeConfirmError, true, nil)
		return
	}

	// Generate a new email activation code
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	emailToken := fmt.Sprintf("%x", b)
	u.EmailToken = emailToken
	u.EmailTokenExpire = time.Now().Add(tokenExpiration)
	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodeConfirmError, true, nil)
		return
	}

	// Now send activation email non-blocking
	reqLang := ps[len(ps) - 2].Value
	author := EmailTexts[reqLang][EmailTextName]
	subject := EmailTexts[reqLang][EmailTextSubjectRegister]
	link := fmt.Sprintf(emailConfirmLink, serverDomain, emailToken)
	body := fmt.Sprintf(EmailTexts[reqLang][EmailTextBodyRegister],
		u.FullName, link, link)
	go sendMail(author, u.Email, u.FullName, subject, body)

	saveLogin(w, r, ps, false, u, nil)
}

func accountRecoverHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params) {
	confirm, of := CheckLengthForm("", r, "confirm",
		TokenMinMax, TokenMinMax)
	password, of := CheckLengthForm(of, r, "password",
		PasswordMinMax, PasswordMinMax)

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	var currentUser User
	if dbConn.First(&currentUser,
		"password_token = ?", confirm).RecordNotFound() {
		// Could be refreshing the page right after confirmation
		// so we check the jwt
		u, ec := checkAuth(r)
		if ec != ErrorCodeNone {
			formatReturn(w, r, ps, ErrorCodePasswordTokenInvalid, false, nil)
			return
		}
		currentUser = *u
	} else if currentUser.PasswordTokenExpire.Before(time.Now()) {
		// If token found, check for expiration
		formatReturn(w, r, ps, ErrorCodePasswordTokenExpired, false, nil)
		return
	}

	// Try to generate new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password),
		bcrypt.DefaultCost)
	// Return unknown register error here
	if err != nil {
		formatReturn(w, r, ps, ErrorCodePasswordResetError, false, nil)
		return
	}

	// Clear token use and change password
	currentUser.PasswordToken = ""
	currentUser.PasswordTokenExpire = time.Time{}
	currentUser.PasswordHash = string(passwordHash)
	if dbConn.Save(&currentUser).Error != nil {
		formatReturn(w, r, ps, ErrorCodePasswordResetError, false, nil)
		return
	}

	saveLogin(w, r, ps, true, &currentUser, nil)
}

func accountForgetHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params) {
	email, of := CheckEmailForm("", r, "email")

	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, false, nil)
		return
	}

	// Email must match
	var currentUser User
	email = strings.ToLower(email)
	if dbConn.First(&currentUser, "email = ?", email).RecordNotFound() {
		formatReturn(w, r, ps, ErrorCodeEmailUnknown, false, nil)
		return
	}

	// Generate a new password reset code
	b := make([]byte, TokenMinMax / 2)
	crand.Read(b)
	passwordToken := fmt.Sprintf("%x", b)
	currentUser.PasswordToken = passwordToken
	currentUser.PasswordTokenExpire = time.Now().Add(tokenExpiration)
	if dbConn.Save(&currentUser).Error != nil {
		formatReturn(w, r, ps, ErrorCodeConfirmError, false, nil)
		return
	}

	// Now send reset email non-blocking
	reqLang := ps[len(ps) - 2].Value
	author := EmailTexts[reqLang][EmailTextName]
	subject := EmailTexts[reqLang][EmailTextSubjectForgetPassword]
	link := fmt.Sprintf(emailForgetLink, serverDomain, passwordToken)
	body := fmt.Sprintf(EmailTexts[reqLang][EmailTextBodyForgetPassword],
		currentUser.FullName, link)
	go sendMail(author, email, currentUser.FullName, subject, body)

	// Not logged in
	formatReturn(w, r, ps, ErrorCodeNone, false, nil)
}

func accountChangeHandler(w http.ResponseWriter, r *http.Request,
ps httprouter.Params, u *User) {
	oldPassword, of := CheckLengthForm("", r, "old_password",
		PasswordMinMax, PasswordMinMax)
	newPassword, of := CheckLengthForm(of, r, "new_password",
		PasswordMinMax, PasswordMinMax)

	// Make sure arguments to update password are valid
	if of != "" {
		formatReturnInfo(w, r, ps, ErrorFmtCodeBadArgument, of, true, nil)
		return
	}

	// Password must match hash
	mismatch := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash),
		[]byte(oldPassword))
	if mismatch != nil {
		formatReturn(w, r, ps, ErrorCodeBadPassword, true, nil)
		return
	}

	// Update new password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword),
		bcrypt.DefaultCost)
	if err != nil {
		formatReturn(w, r, ps, ErrorCodePasswordUpdateError, true, nil)
		return
	}

	u.PasswordHash = string(passwordHash)
	if dbConn.Save(u).Error != nil {
		formatReturn(w, r, ps, ErrorCodePasswordUpdateError, true, nil)
		return
	}

	saveLogin(w, r, ps, false, u, nil)
}
