// Testing for MXDS API
package main

import (
	"fmt"
	"testing"
	"time"
)

// testDocusign is a private helper handling the generic interactive
// docusign signing and downloading process
func testDocusign(t *testing.T, template string,
args map[string]interface{}, dur time.Duration, pdf string) {
	// Setup api
	mxds := Docusign{docusignUrl, docusignUsername, docusignPassword,
		docusignAccountId, docusignIntegratorKey, false}

	cuid := "12345"
	email := "zhaiyifan56@gmail.com"
	name := "Eric Chen"
	eid, et, err := mxds.CreateEnvelopeWithTemplate(
		template, "client", cuid, "Sign", email, name, args)
	if err != nil || eid == "" || et.Equal(time.Time{}) {
		t.Fatalf("Failed to create envelope from template: %v\n", err)
	}

	url, err := mxds.CreateEmbeddedRecipientUrl("", eid, cuid, email, name)
	if err != nil || url == "" {
		t.Fatalf("Failed to create embedded recipient url: %v\n", err)
	}

	fmt.Println("Signing url", url)

	// Give some time to "sign" and check a couple of times
	for i := 0; i < 3; i++ {
		fmt.Println("Please sign asap...")
		time.Sleep(dur)

		completed, terminal, err := mxds.GetEnvelopeStatus(eid)
		if err != nil || !terminal {
			continue
		}
		if !completed {
			t.Fatalf("Failed to complete envelope: terminated\n")
		}

		err = mxds.DownloadEnvelopeDocument(eid, pdf)
		if err != nil {
			t.Fatalf("Failed to download envelope document: %v\n", err)
		}

		// Good, done
		return
	}

	t.Fatalf("Failed to complete envelope: %v\n", err)
}

func TestDocusignSellEngagementLetter(t *testing.T) {
	testDocusign(t, docusignSellEngagementLetter,
		map[string]interface{}{
			"client_name": "Eric Chen",
			"address":     "Earth, Universe",
			"shares_type": "Common Shares",
			"company":     "Palantir",
		}, time.Minute, "./test_enc_1.pdf")
}

func TestDocusignBuyEngagementLetter(t *testing.T) {
	testDocusign(t, docusignBuyEngagementLetter,
		map[string]interface{}{
			"client_name": "Eric Chen",
			"address":     "Earth, Universe",
			"shares_type": "Common Shares",
			"company":     "Palantir",
		}, time.Minute, "./test_enc_2.pdf")
}

func TestDocusignBuySummaryOfTerms(t *testing.T) {
	testDocusign(t, docusignBuySummaryOfTerms,
		map[string]interface{}{
			"client_name":  "Eric Chen",
			"amount":       "100,000",
			"amount_all":   "1,000,000",
			"shares_type":  "Common Shares",
			"company":      "Palantir",
			"closing_date": "3/15/2016",
		}, time.Minute, "./test_enc_3.pdf")
}

func TestDocusignBuyDePpm(t *testing.T) {
	testDocusign(t, docusignBuyDePpm,
		map[string]interface{}{
			"fund_name":           "MarketX Palantir Fund III, LLC",
			"actual_price":        "20.00",
			"actual_valuation":    "3.00B",
			"last_price":          "18.00",
			"last_valuation":      "3.00B",
			"shares_type":         "Common Shares",
			"last_valuation_date": "3/15/2016",
		}, 2 * time.Minute, "./test_enc_4.pdf")
}

func TestDocusignBuyDeOperatingAgreement(t *testing.T) {
	testDocusign(t, docusignBuyDeOperatingAgreement,
		map[string]interface{}{
			"client_name":   "Eric Chen",
			"fund_name":     "MarketX Palantir Fund III, LLC",
			"company":       "Palantir",
			"company_state": "California",
			"shares_type":   "Common Shares",
		}, 2 * time.Minute, "./test_enc_5.pdf")
}

func TestDocusignBuyDeSubscriptionAgreement(t *testing.T) {
	testDocusign(t, docusignBuyDeSubscriptionAgreement,
		map[string]interface{}{
			"client_name":      "Mark Zhai",
			"fund_name":        "MarketX Palantir Fund III, LLC",
			"amount_principal": "100,000",
			"amount_expense":   "5,000",
			"amount_total":     "105,000",
			"ssn":              "111-22-3333",
			"address":          "Some street",
			"phone_number":     "12333333333",
			"email":            "zhaiyifan56@gmail.com",
		}, 3 * time.Minute, "./test_enc_6.pdf")
}
