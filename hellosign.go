// MarketX specific use cases of HelloSign API
// Currently only supports the embedded signature request with template.
// In public domain due to the fact that these APIs should be.
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
	"strings"
	"time"
)

type Hellosign struct {
	url      string
	clientID string
	apiKey   string
	quiet    bool
}

var (
	HellosignErrorRequest = errors.New("Request error")
	HellosignErrorNetwork = errors.New("Network error")
	HellosignErrorAPI = errors.New("API call error")
	HellosignErrorResponse = errors.New("Response parsing error")
	HellosignErrorCall = errors.New("Call error")
)

/* Initializers */

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

/* Private helpers */

var (
	hellosignTimeout = 30 * time.Second
)

// log checks to see if we should log to server or just stdout
// for testing purposes
func (h *Hellosign) log(format string, a ...interface{}) {
	if h.quiet {
		return
	}
	if serverLog != nil {
		serverLog.Printf(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

// request takes a call and params to construct a call to HelloSign API
// and parses response and returns response body for parsing
func (h *Hellosign) request(call string,
params map[string]string) (map[string]interface{}, error) {
	form := url.Values{}
	// Create form with all params
	for k, v := range params {
		form.Set(k, v)
	}
	args := form.Encode()
	reqId := rand.Int63()
	url := h.url + "/" + call
	// Log before any error will happen
	h.log("[MXHSAPI] (%v) %v: %v?%v\n", reqId, "POST", url, args)
	req, err := http.NewRequest("POST", url, strings.NewReader(args))
	if err != nil {
		h.log("[MXHSAPI] (%v) REQ: %v\n", reqId, err)
		return nil, HellosignErrorRequest
	}
	// Always use url-encoded here
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Set authentication
	req.SetBasicAuth(h.apiKey, "")

	// Create client to push the request
	client := &http.Client{Timeout: time.Duration(hellosignTimeout)}
	resp, err := client.Do(req)
	if err != nil {
		h.log("[MXHSAPI] (%v) NET: %v\n", reqId, err)
		return nil, HellosignErrorNetwork
	}
	// Caller must close
	defer resp.Body.Close()

	// Ignore error while parsing, read as many
	b, _ := ioutil.ReadAll(resp.Body)
	body := string(b)

	// If http response error, we should not parse the data structure at all
	if resp.StatusCode != http.StatusOK {
		h.log("[MXHSAPI] (%v) API(%v): %v\n", reqId, resp.StatusCode, body)
		return nil, HellosignErrorAPI
	}

	// Check if the response is json decodable
	var data map[string]interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		h.log("[MXHSAPI] (%v) RES: %v\n", reqId, body)
		return nil, HellosignErrorResponse
	}

	h.log("[MXHSAPI] (%v) DONE: %v\n", reqId, data)
	return data, nil
}

/* APIs */

// CreateEmbeddedWithTemplate sets up a hellosign signing request
// with a pre-set template id, and captures the request id on success
func (h *Hellosign) CreateEmbeddedWithTemplate(ti, sn, email, name string,
ps map[string]interface{}) (string, error) {
	params := map[string]string{"template_id": ti,
		"client_id": h.clientID,
		fmt.Sprintf("signers[%v][email_address]", sn): email,
		fmt.Sprintf("signers[%v][name]", sn):          name}
	if environment != "production" {
		params["test_mode"] = "1"
	}
	// Fill in custom data
	for k, v := range ps {
		params[fmt.Sprintf("custom_fields[%v]", k)] = fmt.Sprintf("%v", v)
	}
	data, err := h.request("signature_request/create_embedded_with_template",
		params)
	if err != nil {
		return "", err
	}

	// Now parse function specific return value
	sigreq, ok := data["signature_request"]
	if !ok {
		return "", HellosignErrorCall
	}
	sigr, ok := sigreq.(map[string]interface{})
	if !ok {
		return "", HellosignErrorCall
	}
	sigs, ok := sigr["signatures"]
	if !ok {
		return "", HellosignErrorCall
	}
	signatures, ok := sigs.([]interface{})
	if !ok {
		return "", HellosignErrorCall
	}
	if len(signatures) < 1 {
		return "", HellosignErrorCall
	}
	sig, ok := signatures[0].(map[string]interface{})
	if !ok {
		return "", HellosignErrorCall
	}
	sigid, ok := sig["signature_id"]
	if !ok {
		return "", HellosignErrorCall
	}
	sid, ok := sigid.(string)
	if !ok {
		return "", HellosignErrorCall
	}

	return sid, nil
}

// CreateEmbeddedSignUrl sets up a hellosign signing url
// with a pre-captured signature id, and captures the embedded url
// plus the expiration time on success
func (h *Hellosign) CreateEmbeddedSignUrl(si string) (string,
time.Time, error) {
	data, err := h.request(fmt.Sprintf("embedded/sign_url/%v", si), nil)
	if err != nil {
		return "", time.Time{}, err
	}

	// Now parse function specific return value
	embedded, ok := data["embedded"]
	if !ok {
		return "", time.Time{}, HellosignErrorCall
	}
	emb, ok := embedded.(map[string]interface{})
	if !ok {
		return "", time.Time{}, HellosignErrorCall
	}
	signUrl, ok := emb["sign_url"]
	if !ok {
		return "", time.Time{}, HellosignErrorCall
	}
	su, ok := signUrl.(string)
	if !ok {
		return "", time.Time{}, HellosignErrorCall
	}
	expiresAt, ok := emb["expires_at"]
	if !ok {
		return "", time.Time{}, HellosignErrorCall
	}
	ea, ok := expiresAt.(float64)
	if !ok {
		return "", time.Time{}, HellosignErrorCall
	}

	return su, time.Unix(int64(ea), 0), nil
}
