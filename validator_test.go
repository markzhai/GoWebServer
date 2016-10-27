// Testing for param validator
package main

import (
	"fmt"
	"net/http"
	"testing"
)

func TestCheckEmail(t *testing.T) {
	s, ok := CheckEmail(true, "taiyang.chen@gmail.com")
	if !ok || s != "taiyang.chen@gmail.com" {
		t.Fatal("[Validator] CheckEmail fails on good email\n")
	}
	s, ok = CheckEmail(false, "taiyang.chen@gmail.com")
	if ok || s != "taiyang.chen@gmail.com" {
		t.Fatal("[Validator] CheckEmail does not fail on previous check " +
			"with good email\n")
	}
	s, ok = CheckEmail(true, "DEADBEEF@@@!!!")
	if ok || s != "DEADBEEF@@@!!!" {
		t.Fatal("[Validator] CheckEmail does not fail on bad email\n")
	}
	s, ok = CheckEmail(false, "DEADBEEF@@@!!!")
	if ok || s != "DEADBEEF@@@!!!" {
		t.Fatal("[Validator] CheckEmail does not fail on previous check " +
			"with bad email\n")
	}
}

func TestCheckSSN(t *testing.T) {
	s, ok := CheckSSN(true, "112-22-3333")
	if !ok || s != "112-22-3333" {
		t.Fatal("[Validator] CheckSSN fails on good ssn\n")
	}
	s, ok = CheckSSN(false, "112-22-3333")
	if ok || s != "112-22-3333" {
		t.Fatal("[Validator] CheckSSN does not fail on previous check " +
			"with good ssn\n")
	}
	s, ok = CheckSSN(true, "DEADBEEF@@@!!!")
	if ok || s != "DEADBEEF@@@!!!" {
		t.Fatal("[Validator] CheckSSN does not fail on bad ssn\n")
	}
	s, ok = CheckSSN(false, "DEADBEEF@@@!!!")
	if ok || s != "DEADBEEF@@@!!!" {
		t.Fatal("[Validator] CheckSSN does not fail on previous check " +
			"with bad ssn\n")
	}
}

func TestCheckDate(t *testing.T) {
	s, ok := CheckDate(true, "2016-06-06")
	if !ok || s != "2016-06-06" {
		t.Fatal("[Validator] CheckDate fails on good date\n")
	}
	s, ok = CheckDate(false, "2016-06-06")
	if ok || s != "2016-06-06" {
		t.Fatal("[Validator] CheckDate does not fail on previous check " +
			"with good date\n")
	}
	s, ok = CheckDate(true, "2016haha")
	if ok || s != "2016haha" {
		t.Fatal("[Validator] CheckDate does not fail on bad date\n")
	}
	s, ok = CheckDate(false, "2016haha")
	if ok || s != "2016haha" {
		t.Fatal("[Validator] CheckDate does not fail on previous check " +
			"with bad date\n")
	}
	s, ok = CheckDate(true, "2016-13-29")
	if ok || s != "2016-13-29" {
		t.Fatal("[Validator] CheckDate does not fail on out-of-range date\n")
	}
	s, ok = CheckDate(false, "2016-13-29")
	if ok || s != "2016-13-29" {
		t.Fatal("[Validator] CheckDate does not fail on previous check " +
			"with out-of-range date\n")
	}
}

func TestCheckField(t *testing.T) {
	s, ok := CheckField(true, "Name")
	if !ok || s != "Name" {
		t.Fatal("[Validator] CheckField fails on good field\n")
	}
	s, ok = CheckField(false, "Name")
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckField does not fail on previous check " +
			"with good field\n")
	}
	s, ok = CheckField(true, "")
	if ok || s != "" {
		t.Fatal("[Validator] CheckField does not fail on bad field\n")
	}
	s, ok = CheckField(false, "")
	if ok || s != "" {
		t.Fatal("[Validator] CheckField does not fail on previous check " +
			"with bad field\n")
	}
}

func TestCheckFields(t *testing.T) {
	if !CheckFields(true, "Name", "First", "Last") {
		t.Fatal("[Validator] CheckFields fails on good fields\n")
	}
	if CheckFields(false, "Name", "First", "Last") {
		t.Fatal("[Validator] CheckFields does not fail on previous check " +
			"with good fields\n")
	}
	if CheckFields(true, "Name", "First", "", "Last") {
		t.Fatal("[Validator] CheckFields does not fail on bad fields\n")
	}
	if CheckFields(false, "Name", "First", "", "Last") {
		t.Fatal("[Validator] CheckFields does not fail on previous check " +
			"with bad fields\n")
	}
}

func TestCheckLength(t *testing.T) {
	s, ok := CheckLength(true, "Name", 2, 8)
	if !ok || s != "Name" {
		t.Fatal("[Validator] CheckLength fails on good length\n")
	}
	s, ok = CheckLength(false, "Name", 2, 8)
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckLength does not fail on previous check " +
			"with good length\n")
	}
	s, ok = CheckLength(true, "Name", 5, 8)
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckLength does not fail on bad min\n")
	}
	s, ok = CheckLength(false, "Name", 5, 8)
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckLength does not fail on previous check " +
			"with bad min\n")
	}
	s, ok = CheckLength(true, "Name", 2, 3)
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckLength does not on bad max\n")
	}
	s, ok = CheckLength(false, "Name", 2, 3)
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckLength does not fail on previous check " +
			"with bad max\n")
	}
	s, ok = CheckLength(true, "Name", 5, 3)
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckLength does not fail on bad min and max\n")
	}
	s, ok = CheckLength(false, "Name", 5, 3)
	if ok || s != "Name" {
		t.Fatal("[Validator] CheckLength does not fail on previous check " +
			"with bad min and max\n")
	}
}

func TestCheckRange(t *testing.T) {
	n, ok := CheckRange(true, "123", 130)
	if !ok {
		t.Fatal("[Validator] CheckRange fails on good number and range\n")
	}
	if fmt.Sprintf("%v", n) != "123" {
		t.Fatal("[Validator] CheckRange returns invalid number with " +
			"good number and range\n")
	}
	n, ok = CheckRange(true, "123", 100)
	if ok {
		t.Fatal("[Validator] CheckRange does not fail on good number " +
			"and bad range\n")
	}
	if fmt.Sprintf("%v", n) != "123" {
		t.Fatal("[Validator] CheckRange returns invalid number with " +
			"good number and bad range\n")
	}
	_, ok = CheckRange(true, "ASDF123", 100)
	if ok {
		t.Fatal("[Validator] CheckRange does not fail on bad number " +
			"and range\n")
	}
	_, ok = CheckRange(false, "123", 130)
	if ok {
		t.Fatal("[Validator] CheckRange does not fail on previous check " +
			"with good number and range\n")
	}
	_, ok = CheckRange(false, "123", 100)
	if ok {
		t.Fatal("[Validator] CheckRange does not fail on previous check " +
			"with good number and bad range\n")
	}
	_, ok = CheckRange(false, "ASDF123", 100)
	if ok {
		t.Fatal("[Validator] CheckRange does not fail on previous check " +
			"with bad number and range\n")
	}
}

func TestCheckFloat(t *testing.T) {
	n, ok := CheckFloat(true, "123.4", false)
	if !ok {
		t.Fatal("[Validator] CheckFloat fails on good number and sign\n")
	}
	if fmt.Sprintf("%v", n) != "123.4" {
		t.Fatal("[Validator] CheckFloat returns invalid number with " +
			"good number and sign\n")
	}
	n, ok = CheckFloat(true, "123.4", true)
	if !ok {
		t.Fatal("[Validator] CheckFloat fails on good number and strict sign\n")
	}
	if fmt.Sprintf("%v", n) != "123.4" {
		t.Fatal("[Validator] CheckFloat returns invalid number with " +
			"good number and strict sign\n")
	}
	_, ok = CheckFloat(false, "123.4", false)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on previous check " +
			"with good number and sign\n")
	}
	_, ok = CheckFloat(false, "123.4", true)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on previous check " +
			"with good number and strict sign\n")
	}
	n, ok = CheckFloat(true, "-123.4", false)
	if !ok {
		t.Fatal("[Validator] CheckFloat fails on good neg number and sign\n")
	}
	if fmt.Sprintf("%v", n) != "-123.4" {
		t.Fatal("[Validator] CheckFloat returns invalid number with " +
			"good neg number and sign\n")
	}
	_, ok = CheckFloat(true, "-123.4", true)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on good neg number " +
			"and strict sign\n")
	}
	_, ok = CheckFloat(false, "-123.4", false)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on previous check " +
			"with good neg number and sign\n")
	}
	_, ok = CheckFloat(false, "-123.4", true)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on previous check " +
			"with good neg number and strict sign\n")
	}
	_, ok = CheckFloat(true, "ASDF@@", false)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on bad number and sign\n")
	}
	_, ok = CheckFloat(true, "ASDF@@", true)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on bad number " +
			"and strict sign\n")
	}
	_, ok = CheckFloat(false, "ASDF@@", false)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on previous check " +
			"with bad number and sign\n")
	}
	_, ok = CheckFloat(false, "ASDF@@", true)
	if ok {
		t.Fatal("[Validator] CheckFloat does not fail on previous check " +
			"with bad number and strict sign\n")
	}
}

func TestCheckEmailForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"email_good": []string{"taiyang.chen@gmail.com"},
		"email_bad":  []string{"DEADBEEF@@@!!!"},
	}}
	s, of := CheckEmailForm("", r, "email_good")
	if of != "" || s != "taiyang.chen@gmail.com" {
		t.Fatal("[Validator] CheckEmailForm fails on good email\n")
	}
	s, of = CheckEmailForm("xxx", r, "email_good")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckEmailForm does not fail on previous check " +
			"with good email\n")
	}
	s, of = CheckEmailForm("", r, "email_bad")
	if of != "email_bad" || s != "DEADBEEF@@@!!!" {
		t.Fatal("[Validator] CheckEmailForm does not fail on bad email\n")
	}
	s, of = CheckEmailForm("xxx", r, "email_bad")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckEmailForm does not fail on previous check " +
			"with bad email\n")
	}
}

func TestCheckSSNForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"ssn_good": []string{"112-22-3333"},
		"ssn_bad":  []string{"DEADBEEF@@@!!!"},
	}}
	s, of := CheckSSNForm("", r, "ssn_good")
	if of != "" || s != "112-22-3333" {
		t.Fatal("[Validator] CheckSSNForm fails on good ssn\n")
	}
	s, of = CheckSSNForm("xxx", r, "ssn_good")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckSSNForm does not fail on previous check " +
			"with good ssn\n")
	}
	s, of = CheckSSNForm("", r, "ssn_bad")
	if of != "ssn_bad" || s != "DEADBEEF@@@!!!" {
		t.Fatal("[Validator] CheckSSNForm does not fail on bad ssn\n")
	}
	s, of = CheckSSNForm("xxx", r, "ssn_bad")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckSSNForm does not fail on previous check " +
			"with bad ssn\n")
	}
}

func TestCheckDateForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"date_good":  []string{"2016-06-06"},
		"date_bad":   []string{"2016haha"},
		"date_range": []string{"2016-13-29"},
	}}
	s, of := CheckDateForm("", r, "date_good")
	if of != "" || s != "2016-06-06" {
		t.Fatal("[Validator] CheckDateForm fails on good date\n")
	}
	s, of = CheckDateForm("xxx", r, "date_good")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckDateForm does not fail on previous check " +
			"with good date\n")
	}
	s, of = CheckDateForm("", r, "date_bad")
	if of != "date_bad" || s != "2016haha" {
		t.Fatal("[Validator] CheckDateForm does not fail on bad date\n")
	}
	s, of = CheckDateForm("xxx", r, "date_bad")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckDateForm does not fail on previous check " +
			"with bad date\n")
	}
	s, of = CheckDateForm("", r, "date_range")
	if of != "date_range" || s != "2016-13-29" {
		t.Fatal("[Validator] CheckDateForm does not fail on " +
			"out-of-range date\n")
	}
	s, of = CheckDateForm("xxx", r, "2016-13-29")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckDateForm does not fail on previous check " +
			"with out-of-range date\n")
	}
}

func TestCheckFieldForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"field_good": []string{"Name"},
	}}
	s, of := CheckFieldForm("", r, "field_good")
	if of != "" || s != "Name" {
		t.Fatal("[Validator] CheckFieldForm fails on good field\n")
	}
	s, of = CheckFieldForm("xxx", r, "field_good")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckFieldForm does not fail on previous check " +
			"with good field\n")
	}
	s, of = CheckFieldForm("", r, "field")
	if of != "field" || s != "" {
		t.Fatal("[Validator] CheckFieldForm does not fail on bad field\n")
	}
	s, of = CheckFieldForm("xxx", r, "field")
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckFieldForm does not fail on previous check " +
			"with bad field\n")
	}
}

func TestCheckFieldsForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"field_1": []string{"Name"},
		"field_2": []string{"First"},
		"field_3": []string{"Last"},
	}}
	if CheckFieldsForm("", r, "field_1", "field_2", "field_3") != "" {
		t.Fatal("[Validator] CheckFieldsForm fails on good fields\n")
	}
	if CheckFieldsForm("xxx", r, "field_1", "field_2", "field_3") != "xxx" {
		t.Fatal("[Validator] CheckFieldsForm does not fail on " +
			"previous check with good fields\n")
	}
	if CheckFieldsForm("", r, "field_1", "field", "field_3") !=
		"field_1,field,field_3" {
		t.Fatal("[Validator] CheckFieldsForm does not fail on bad fields\n")
	}
	if CheckFieldsForm("xxx", r, "field_1", "field", "field_3") != "xxx" {
		t.Fatal("[Validator] CheckFieldsForm does not fail on " +
			"previous check with bad fields\n")
	}
}

func TestCheckLengthForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"length_good": []string{"Name"},
	}}
	s, of := CheckLengthForm("", r, "length_good", 2, 8)
	if of != "" || s != "Name" {
		t.Fatal("[Validator] CheckLengthForm fails on good length\n")
	}
	s, of = CheckLengthForm("xxx", r, "length_good", 2, 8)
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckLengthForm does not fail on " +
			"previous check with good length\n")
	}
	s, of = CheckLengthForm("", r, "length_good", 5, 8)
	if of != "length_good" || s != "Name" {
		t.Fatal("[Validator] CheckLengthForm does not fail on bad min\n")
	}
	s, of = CheckLengthForm("xxx", r, "length_good", 5, 8)
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckLengthForm does not fail on " +
			"previous check with bad min\n")
	}
	s, of = CheckLengthForm("", r, "length_good", 2, 3)
	if of != "length_good" || s != "Name" {
		t.Fatal("[Validator] CheckLengthForm does not on bad max\n")
	}
	s, of = CheckLengthForm("xxx", r, "length_good", 2, 3)
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckLengthForm does not fail on " +
			"previous check with bad max\n")
	}
	s, of = CheckLengthForm("", r, "length_good", 5, 3)
	if of != "length_good" || s != "Name" {
		t.Fatal("[Validator] CheckLengthForm does not fail on " +
			"bad min and max\n")
	}
	s, of = CheckLengthForm("xxx", r, "length_good", 5, 3)
	if of != "xxx" || s != "" {
		t.Fatal("[Validator] CheckLengthForm does not fail on " +
			"previous check with bad min and max\n")
	}
}

func TestCheckRangeForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"range_good": []string{"123"},
		"range_bad":  []string{"ASDF123"},
	}}
	n, of := CheckRangeForm("", r, "range_good", 130)
	if of != "" {
		t.Fatal("[Validator] CheckRangeForm fails on good number and range\n")
	}
	if fmt.Sprintf("%v", n) != "123" {
		t.Fatal("[Validator] CheckRangeForm returns invalid number with " +
			"good number and range\n")
	}
	n, of = CheckRangeForm("", r, "range_good", 100)
	if of != "range_good" {
		t.Fatal("[Validator] CheckRangeForm does not fail on good number " +
			"and bad range\n")
	}
	if fmt.Sprintf("%v", n) != "123" {
		t.Fatal("[Validator] CheckRangeForm returns invalid number with " +
			"good number and bad range\n")
	}
	_, of = CheckRangeForm("", r, "range_bad", 100)
	if of != "range_bad" {
		t.Fatal("[Validator] CheckRangeForm does not fail on bad number " +
			"and range\n")
	}
	_, of = CheckRangeForm("xxx", r, "range_good", 130)
	if of != "xxx" {
		t.Fatal("[Validator] CheckRangeForm does not fail on previous check " +
			"with good number and range\n")
	}
	_, of = CheckRangeForm("xxx", r, "range_good", 100)
	if of != "xxx" {
		t.Fatal("[Validator] CheckRangeForm does not fail on previous check " +
			"with good number and bad range\n")
	}
	_, of = CheckRangeForm("xxx", r, "range_bad", 100)
	if of != "xxx" {
		t.Fatal("[Validator] CheckRangeForm does not fail on previous check " +
			"with bad number and range\n")
	}
}

func TestCheckFloatForm(t *testing.T) {
	r := &http.Request{Form: map[string][]string{
		"float_good_pos": []string{"123.4"},
		"float_good_neg": []string{"-123.4"},
		"float_bad":      []string{"ASDF@@"},
	}}
	n, of := CheckFloatForm("", r, "float_good_pos", false)
	if of != "" {
		t.Fatal("[Validator] CheckFloatForm fails on good number and sign\n")
	}
	if fmt.Sprintf("%v", n) != "123.4" {
		t.Fatal("[Validator] CheckFloatForm returns invalid number with " +
			"good number and sign\n")
	}
	n, of = CheckFloatForm("", r, "float_good_pos", true)
	if of != "" {
		t.Fatal("[Validator] CheckFloatForm fails on " +
			"good number and strict sign\n")
	}
	if fmt.Sprintf("%v", n) != "123.4" {
		t.Fatal("[Validator] CheckFloatForm returns invalid number with " +
			"good number and strict sign\n")
	}
	_, of = CheckFloatForm("xxx", r, "float_good_pos", false)
	if of != "xxx" {
		t.Fatal("[Validator] CheckFloatForm does not fail on previous check " +
			"with good number and sign\n")
	}
	_, of = CheckFloatForm("xxx", r, "float_good_pos", true)
	if of != "xxx" {
		t.Fatal("[Validator] CheckFloatForm does not fail on previous check " +
			"with good number and strict sign\n")
	}
	n, of = CheckFloatForm("", r, "float_good_neg", false)
	if of != "" {
		t.Fatal("[Validator] CheckFloatForm fails on " +
			"good neg number and sign\n")
	}
	if fmt.Sprintf("%v", n) != "-123.4" {
		t.Fatal("[Validator] CheckFloatForm returns invalid number with " +
			"good neg number and sign\n")
	}
	_, of = CheckFloatForm("", r, "float_good_neg", true)
	if of != "float_good_neg" {
		t.Fatal("[Validator] CheckFloatForm does not fail on " +
			"good neg number and strict sign\n")
	}
	_, of = CheckFloatForm("xxx", r, "-123.4", false)
	if of != "xxx" {
		t.Fatal("[Validator] CheckFloatForm does not fail on previous check " +
			"with good neg number and sign\n")
	}
	_, of = CheckFloatForm("xxx", r, "-123.4", true)
	if of != "xxx" {
		t.Fatal("[Validator] CheckFloatForm does not fail on previous check " +
			"with good neg number and strict sign\n")
	}
	_, of = CheckFloatForm("", r, "float_bad", false)
	if of != "float_bad" {
		t.Fatal("[Validator] CheckFloatForm does not fail on " +
			"bad number and sign\n")
	}
	_, of = CheckFloatForm("", r, "float_bad", true)
	if of != "float_bad" {
		t.Fatal("[Validator] CheckFloatForm does not fail on bad number " +
			"and strict sign\n")
	}
	_, of = CheckFloatForm("xxx", r, "float_bad", false)
	if of != "xxx" {
		t.Fatal("[Validator] CheckFloatForm does not fail on previous check " +
			"with bad number and sign\n")
	}
	_, of = CheckFloatForm("xxx", r, "float_bad", true)
	if of != "xxx" {
		t.Fatal("[Validator] CheckFloatForm does not fail on previous check " +
			"with bad number and strict sign\n")
	}
}
