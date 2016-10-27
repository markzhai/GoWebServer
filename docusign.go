// MarketX specific use cases of DocuSign API
// Currently only supports envelope creation from template, embedded signing
// and document download.
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
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Docusign struct {
	url           string
	username      string
	password      string
	accountId     string
	integratorKey string
	quiet         bool
}

var (
	DocusignErrorParams   = errors.New("Params error")
	DocusignErrorRequest  = errors.New("Request error")
	DocusignErrorNetwork  = errors.New("Network error")
	DocusignErrorAPI      = errors.New("API call error")
	DocusignErrorResponse = errors.New("Response parsing error")
	DocusignErrorCall     = errors.New("Call error")
)

/* Initializers */

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

/* Private helpers */

var (
	docusignTimeout     = 30 * time.Second
	docusignExpire      = 120 * 24 * time.Hour
	docusignStatusDelay = 15 * time.Minute
	docusignReturnUrl   = serverDomain + "/u/#/pages/sign/"
)

// log checks to see if we should log to server or just stdout
// for testing purposes
func (d *Docusign) log(format string, a ...interface{}) {
	fmt.Printf(format, a...)
	if d.quiet {
		return
	}
	if serverLog != nil {
		serverLog.Printf(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

// request takes a call and params to construct a call to DocuSign API
// and parses response and returns response body for parsing
func (d *Docusign) request(method, call string,
	params map[string]interface{}, raw bool) (map[string]interface{}, error) {
	fmt.Printf("on docusign requested")
	reqId := rand.Int63()
	// Encode POST and GET differently
	var args string
	if method == "POST" {
		jp, err := json.Marshal(params)
		if err != nil {
			d.log("[MXDSAPI] (%v) REQ: %v\n", reqId, err)
			return nil, DocusignErrorParams
		}
		args = string(jp)
	} else {
		form := url.Values{}
		for k, v := range params {
			vs, ok := v.(string)
			if !ok {
				continue
			}
			form.Set(k, vs)
		}
		args = form.Encode()
	}
	url := d.url + "/accounts/" + d.accountId + "/" + call
	// Log before any error will happen
	d.log("[MXDSAPI] (%v) %v: %v?%v\n", reqId, method, url, args)
	req, err := http.NewRequest(method, url, strings.NewReader(args))
	if err != nil {
		d.log("[MXDSAPI] (%v) REQ: %v\n", reqId, err)
		return nil, DocusignErrorRequest
	}
	// Always use json format as required
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	// Set authentication
	req.Header.Set("X-DocuSign-Authentication",
		"<DocuSignCredentials><Username>"+d.username+
			"</Username><Password>"+d.password+
			"</Password><IntegratorKey>"+d.integratorKey+
			"</IntegratorKey></DocuSignCredentials>")

	// Create client to push the request
	client := &http.Client{Timeout: time.Duration(docusignTimeout)}
	resp, err := client.Do(req)
	if err != nil {
		d.log("[MXDSAPI] (%v) NET: %v\n", reqId, err)
		return nil, DocusignErrorNetwork
	}
	// Caller must close
	defer resp.Body.Close()

	// Ignore error while parsing, read as many
	b, _ := ioutil.ReadAll(resp.Body)
	body := string(b)

	// If http response error, we should not parse the data structure at all
	if (method == "POST" && resp.StatusCode != http.StatusCreated) ||
		(method != "POST" && resp.StatusCode != http.StatusOK) {
		d.log("[MXDSAPI] (%v) API(%v): %v\n", reqId, resp.StatusCode, body)
		return nil, DocusignErrorAPI
	}

	// Return bytes if reading binary
	if raw {
		d.log("[MXDSAPI] (%v) DONE: <binary>\n", reqId)
		return map[string]interface{}{"raw": b}, nil
	}

	// Check if the response is json decodable
	var data map[string]interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		d.log("[MXDSAPI] (%v) RES: %v\n", reqId, body)
		return nil, DocusignErrorResponse
	}

	d.log("[MXDSAPI] (%v) DONE: %v\n", reqId, data)
	return data, nil
}

/* APIs */

// CreateEnvelopeWithTemplate sets up a new docusign envelope
// with a pre-set template id, and captures the envelope id and expiration
// time on success
func (d *Docusign) CreateEnvelopeWithTemplate(ti, rn, cuid, es,
	email, name string, ps map[string]interface{}) (string,
	time.Time, error) {
	// The outer arguments to use
	params := map[string]interface{}{
		"status":       "sent",
		"emailSubject": es,
		"templateId":   ti,
	}
	// Normally just a single signer
	tr := map[string]interface{}{
		"email":        email,
		"name":         name,
		"roleName":     rn,
		"clientUserId": cuid,
	}
	// Fill in custom data
	var ttabs []map[string]string
	for k, v := range ps {
		ttabs = append(ttabs, map[string]string{
			"tabLabel": "\\*" + k,
			"value":    fmt.Sprintf("%v", v),
		})
	}
	// Set tabs in place
	tr["tabs"] = map[string][]map[string]string{"textTabs": ttabs}
	// Set params in place
	params["templateRoles"] = []map[string]interface{}{tr}
	data, err := d.request("POST", "envelopes", params, false)
	if err != nil {
		return "", time.Time{}, err
	}

	// Now parse function specific return value
	envId, ok := data["envelopeId"]
	if !ok {
		return "", time.Time{}, DocusignErrorCall
	}
	eid, ok := envId.(string)
	if !ok {
		return "", time.Time{}, DocusignErrorCall
	}
	startTime, ok := data["statusDateTime"]
	if !ok {
		return "", time.Time{}, DocusignErrorCall
	}
	stime, ok := startTime.(string)
	if !ok {
		return "", time.Time{}, DocusignErrorCall
	}
	et, err := time.Parse(time.RFC3339Nano, stime)
	if err != nil {
		return "", time.Time{}, DocusignErrorCall
	}

	// Add default expiration time
	et = et.Add(docusignExpire)
	return eid, et, nil
}

// GetEnvelopeRecipient returns the initial information of the first
// recipient when the envelope is created; this is useful because certain
// information can be modified by other apis or docusign interface and
// we have no valid way to verify the correct recipient information locally
// without destroying the current envelope - a call could be costly but it
// saves the hassle of voiding an envelope
// The return consists of a map with recipient name, email, clientUserId and
// roleName with possibly more information as fit
func (d *Docusign) GetEnvelopeRecipient(eid string) (map[string]string,
	error) {
	data, err := d.request("GET", fmt.Sprintf("/envelopes/%v/recipients",
		eid), nil, false)
	if err != nil {
		return nil, err
	}

	// Now parse function specific return value
	signersList, ok := data["signers"]
	if !ok {
		return nil, DocusignErrorCall
	}
	signers, ok := signersList.([]interface{})
	if !ok || len(signers) < 1 {
		return nil, DocusignErrorCall
	}
	recipientData, ok := signers[0].(map[string]interface{})
	if !ok {
		return nil, DocusignErrorCall
	}
	// Create new map with only necessary information
	recipient := map[string]string{}
	nameData, ok := recipientData["name"]
	if !ok {
		return nil, DocusignErrorCall
	}
	recipient["name"], ok = nameData.(string)
	if !ok {
		return nil, DocusignErrorCall
	}
	emailData, ok := recipientData["email"]
	if !ok {
		return nil, DocusignErrorCall
	}
	recipient["email"], ok = emailData.(string)
	if !ok {
		return nil, DocusignErrorCall
	}
	cuidData, ok := recipientData["clientUserId"]
	if !ok {
		return nil, DocusignErrorCall
	}
	recipient["clientUserId"], ok = cuidData.(string)
	if !ok {
		return nil, DocusignErrorCall
	}
	roleNameData, ok := recipientData["roleName"]
	if !ok {
		return nil, DocusignErrorCall
	}
	recipient["roleName"], ok = roleNameData.(string)
	if !ok {
		return nil, DocusignErrorCall
	}

	return recipient, nil
}

// CreateEmbeddedRecipientUrl sets up a docusign signing url
// with a pre-captured envelope id, and captures the embedded url
func (d *Docusign) CreateEmbeddedRecipientUrl(host, eid, cuid,
	email, name string) (string, error) {
	// The outer arguments to use
	rurl := docusignReturnUrl
	if host != "" {
		rurl = strings.Replace(rurl, serverDomain, serverProto+host, -1)
	}
	params := map[string]interface{}{
		"userName":             name,
		"email":                email,
		"clientUserId":         cuid,
		"authenticationMethod": "email",
		"returnUrl":            rurl + eid,
	}
	data, err := d.request("POST",
		fmt.Sprintf("/envelopes/%v/views/recipient", eid), params, false)
	// We retry get recipient information once in case of name changes
	if err == DocusignErrorAPI {
		recp, rerr := d.GetEnvelopeRecipient(eid)
		if rerr != nil {
			return "", err
		}
		params["userName"] = recp["name"]
		params["email"] = recp["email"]
		params["clientUserId"] = recp["clientUserId"]
		// Check again
		data, err = d.request("POST",
			fmt.Sprintf("/envelopes/%v/views/recipient", eid), params, false)
	}
	if err != nil {
		return "", err
	}

	// Now parse function specific return value
	urlData, ok := data["url"]
	if !ok {
		return "", DocusignErrorCall
	}
	url, ok := urlData.(string)
	if !ok {
		return "", DocusignErrorCall
	}

	return url, nil
}

// GetEnvelopeStatus checks whether an envelope has finished
// signing (terminal state) and can be downloaded (completed state)
// Returns (completed, terminal, error) tuple
func (d *Docusign) GetEnvelopeStatus(eid string) (bool, bool, error) {
	data, err := d.request("GET", fmt.Sprintf("/envelopes/%v", eid), nil,
		false)
	if err != nil {
		return false, false, err
	}

	// Now parse function specific return value
	statusData, ok := data["status"]
	if !ok {
		return false, false, DocusignErrorCall
	}
	status, ok := statusData.(string)
	if !ok {
		return false, false, DocusignErrorCall
	}

	// Completed, Declined and Voided are the 3 valid terminal states
	status = strings.ToLower(status)
	return status == "completed", status == "completed" ||
		status == "declined" || status == "voided", nil
}

// DownloadEnvelopeDocument downloads a previously signed document
// It assumes that the document has been completed
// On success, the document is saved to path
func (d *Docusign) DownloadEnvelopeDocument(eid, path string) error {
	// Always the first document in the envelope, for now
	data, err := d.request("GET",
		fmt.Sprintf("/envelopes/%v/documents/1", eid), nil, true)
	if err != nil {
		return err
	}

	// Now parse function specific return value
	raw, ok := data["raw"]
	if !ok {
		return DocusignErrorCall
	}
	file, ok := raw.([]byte)
	if !ok {
		return DocusignErrorCall
	}

	// Create directory
	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return DocusignErrorCall
	}

	// Create file
	out, err := os.Create(path)
	if err != nil {
		return DocusignErrorCall
	}
	defer out.Close()

	// Apply encryption
	err = EncryptStreamBytes(aesFileSecret, file, out)
	if err != nil {
		return DocusignErrorCall
	}

	return nil
}
