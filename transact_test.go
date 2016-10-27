// Testing for MXT API
package main

import (
	"testing"
	"time"
)

func TestKycGet(t *testing.T) {
	// Setup api
	mxt := Transact{transactUrl, transactId, transactKey, false}

	// Setup dummy user
	u := DummyUser

	err := mxt.CreateInvestorRecord(&u)
	if err != nil {
		t.Fatalf("Failed to create investor record: %v\n", err)
	}
	if u.TransactApiInvestorID == 0 {
		t.Fatalf("Failed to create investor id\n")
	}

	ques, err := mxt.CreateInvestorAccount(&u)
	if err != nil {
		t.Fatalf("Failed to create investor account: %v\n", err)
	}

	if len(ques) == 0 {
		t.Fatalf("Failed to get questions for investor: %v\n", err)
	}

	for _, v := range ques {
		// Check for specific types
		prompt, ok := v["prompt"]
		if !ok {
			t.Fatalf("Questions missing prompt: %v\n", v)
		}
		_, ok = prompt.(string)
		if !ok {
			t.Fatalf("Questions have wrong prompt type %T: %v\n", prompt, v)
		}
		qtype, ok := v["type"]
		if !ok {
			t.Fatalf("Questions missing type: %v\n", v)
		}
		_, ok = qtype.(string)
		if !ok {
			t.Fatalf("Questions have wrong type type %T: %v\n", qtype, v)
		}
		answer, ok := v["answer"]
		if !ok {
			t.Fatalf("Questions missing answer: %v\n", v)
		}
		answers, ok := answer.([]interface{})
		if !ok {
			t.Fatalf("Questions have wrong answer type %T: %v\n", answer, v)
		}
		for _, av := range answers {
			_, ok = av.(string)
			if !ok {
				t.Fatalf("Questions have a wrong answer type %T: %v\n", av, v)
			}
		}
	}

	if u.TransactApiKycID == 0 ||
		u.TransactApiKycExpire.Equal(time.Time{}) ||
		u.TransactApiKycAttempts == 0 {
		t.Fatalf("Failed to update kyc id\n")
	}

	err = mxt.KycStatus(&u,
		map[string]string{"type1": "test1", "ans1": "sth1",
			"type2": "test2", "ans2": "sth2",
			"type3": "test3", "ans3": "sth3",
			"type4": "test4", "ans4": "sth4",
			"type5": "test5", "ans5": "sth5"})
	if err != nil {
		t.Fatalf("Failed to check kyc status: %v\n", err)
	}
	if u.TransactApiKycID != 0 ||
		!u.TransactApiKycExpire.Equal(time.Time{}) ||
		u.TransactApiKycAttempts != 0 {
		t.Fatalf("Failed to clean up after kyc status\n")
	}

	err = mxt.DeleteInvestorFinancialAccount(&u)
	if err != nil {
		t.Fatalf("Failed to delete investor account: %v\n", err)
	}
	if u.TransactApiKycID != 0 ||
		!u.TransactApiKycExpire.Equal(time.Time{}) ||
		u.TransactApiKycAttempts != 0 {
		t.Fatalf("Failed to reset kyc id\n")
	}

	err = mxt.DeleteInvestor(&u)
	if err != nil {
		t.Fatalf("Failed to delete investor record: %v\n", err)
	}
	if u.TransactApiInvestorID != 0 {
		t.Fatalf("Failed to delete investor id\n")
	}
}
