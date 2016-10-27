// Provides an even more specific wrapper for a param validator
// Validator functions are exported as public since they are not
// MarketX specific checks.
package main

import (
	valid "github.com/asaskevich/govalidator"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	StringMax uint64 = 255
	StringPlusMax uint64 = 1000
	StringBlockMax uint64 = 10000
	JsonMax uint64 = 10 * 1024 * 1024
	PasswordMinMax uint64 = 128
	TokenMinMax uint64 = 64
	PhoneMin uint64 = 5
	PhoneMax uint64 = 20
	ZipMin uint64 = 5
	ZipMax uint64 = 20
	CountryMinMax uint64 = 2
	NoYesMax uint64 = 1
	NumberMax uint64 = ^uint64(0)
	PercentMax uint64 = 100
	OpenIdMinMax uint64 = 28
	TimeMax uint64 = 1 << 63 - 1
)

var (
	YearMax uint64 = uint64(time.Now().Year())
)

// CheckEmail makes sure s is at least an RFC compliant address
// Further email ownership verification is required afterwards
func CheckEmail(ok bool, s string) (string, bool) {
	return s, ok && valid.IsEmail(s)
}

// CheckEmailForm is the FormValue version of CheckEmail
func CheckEmailForm(of string, r *http.Request, f string) (string, string) {
	if of != "" {
		return "", of
	}
	rs, ok := CheckEmail(true, r.FormValue(f))
	if ok {
		return rs, ""
	}
	return rs, f
}

// CheckSSN makes sure s is a valid SSN string
func CheckSSN(ok bool, s string) (string, bool) {
	return s, ok && valid.IsSSN(s)
}

// CheckSSNForm is the FormValue version of CheckSSN
func CheckSSNForm(of string, r *http.Request, f string) (string, string) {
	if of != "" {
		return "", of
	}
	rs, ok := CheckSSN(true, r.FormValue(f))
	if ok {
		return rs, ""
	}
	return rs, f
}

// CheckDate makes sure s is a valid RFC3339 date of 2006-01-02 format
func CheckDate(ok bool, s string) (string, bool) {
	if !ok {
		return s, false
	}
	return s, inputDate(s) != ""
}

// CheckDateForm is the FormValue version of CheckDate
func CheckDateForm(of string, r *http.Request, f string) (string, string) {
	if of != "" {
		return "", of
	}
	rs, ok := CheckDate(true, r.FormValue(f))
	if ok {
		return rs, ""
	}
	return rs, f
}

// CheckField is a generic non-empty string checker
func CheckField(ok bool, s string) (string, bool) {
	return CheckLength(ok, s, 1, StringMax)
}

// CheckFieldForm is the FormValue version of CheckField
func CheckFieldForm(of string, r *http.Request, f string) (string, string) {
	if of != "" {
		return "", of
	}
	rs, ok := CheckField(true, r.FormValue(f))
	if ok {
		return rs, ""
	}
	return rs, f
}

// CheckFields checks a list of non-empty strings
func CheckFields(ok bool, ss ...string) bool {
	if !ok {
		return false
	}
	for _, s := range ss {
		_, ok = CheckField(true, s)
		if !ok {
			return false
		}
	}
	return true
}

// CheckFieldsForm is the FormValue version of CheckFields
func CheckFieldsForm(of string, r *http.Request, fs ...string) string {
	if of != "" {
		return of
	}
	// Make a FormValue slice
	var ffs []string
	for _, f := range fs {
		ffs = append(ffs, r.FormValue(f))
	}
	ok := CheckFields(true, ffs...)
	if ok {
		return ""
	}
	// Return list of fields
	return strings.Join(fs, ",")
}

// CheckLength checks a string length in between [min, max]
func CheckLength(ok bool, s string, min, max uint64) (string, bool) {
	l := uint64(len(s))
	return s, ok && l >= min && l <= max
}

// CheckLengthForm is the FormValue version of CheckLength
func CheckLengthForm(of string, r *http.Request, f string,
min, max uint64) (string, string) {
	if of != "" {
		return "", of
	}
	rs, ok := CheckLength(true, r.FormValue(f), min, max)
	if ok {
		return rs, ""
	}
	return rs, f
}

// CheckRange is a specialized check of an uint64 declaration wrapped
// inside a string, and returns a converted uint64 if possible
func CheckRange(ok bool, s string, upper uint64) (uint64, bool) {
	if !ok {
		return 0, false
	}
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return i, i <= upper
}

// CheckRangeForm is the FormValue version of CheckRange
func CheckRangeForm(of string, r *http.Request, f string,
upper uint64) (uint64, string) {
	if of != "" {
		return 0, of
	}
	rs, ok := CheckRange(true, r.FormValue(f), upper)
	if ok {
		return rs, ""
	}
	return rs, f
}

// CheckFloat is a specialized check of a float64 declaration wrapped
// inside a string, and returns a converted float64 if possible
func CheckFloat(ok bool, s string, pos bool) (float64, bool) {
	if !ok {
		return 0.0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, false
	}
	return f, !pos || f >= 0.0
}

// CheckFloatForm is the FormValue version of CheckFloat
func CheckFloatForm(of string, r *http.Request, f string,
pos bool) (float64, string) {
	if of != "" {
		return 0.0, of
	}
	rs, ok := CheckFloat(true, r.FormValue(f), pos)
	if ok {
		return rs, ""
	}
	return rs, f
}
