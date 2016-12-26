// Config file for running MarketX server
// Reads in secret variables from the running environment.
// In the future it should read and parse default values with their
// respective types.
package main

import (
	"os"
	"strconv"
	"strings"
)

// getString is a string environment variable parsing and
// default-setting function
func getString(name string, def interface{}) string {
	val := os.Getenv(name)
	// Make sure certain values have must sets
	if val == "" && def == nil {
		panic(name)
	}

	if val != "" {
		return val
	}
	return def.(string)
}

// getBool is a bool environment variable parsing and
// default-setting function
func getBool(name string, def bool) bool {
	val := os.Getenv(name)
	ret, err := strconv.ParseBool(val)
	if err != nil {
		return def
	}
	return ret
}

// getInt is an int environment variable parsing and
// default-setting function
func getInt(name string, def int64) int64 {
	val := os.Getenv(name)
	ret, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return def
	}
	return ret
}

// getStringArray is a string array (separated by ",") environment variable
// parsing and default-setting function
func getStringArray(name string, def []string) []string {
	val := os.Getenv(name)
	if val == "" {
		return def
	}

	// Split into a slice
	return strings.Split(val, ",")
}

// Debugging configurations
var (
	dropTables = getBool("MX_DEBUG_DROP_TABLES", false)
	importData = getBool("MX_DEBUG_IMPORT_DATA", false)
)

// Server configurations
var (
	environment   = getString("MX_ENVIRONMENT", "development")
	logFileName   = getString("MX_LOG_FILE_NAME", "marketx.log")
	dbScheme      = getString("MX_DB_SCHEME", "postgres")
	dbUrl         = getString("DATABASE_URL", nil)
	dbLogFileName = getString("MX_DB_LOG_FILE_NAME", "marketx_db.log")
	serverProto   = getString("MX_SERVER_PROTO", "https://")
	serverDomain  = getString("MX_SERVER_DOMAIN", "https://127.0.0.1")
	serverPort    = getInt("PORT", 443)
	serverPPort   = getInt("MX_SERVER_P_PORT", 8080)
	clientDir     = getString("MX_CLIENT_DIR", "client")
	dataDir       = getString("MX_DATA_DIR", "data")
)

// Authentication configurations
var (
	jwtSecretKey         = getString("MX_JWT_SECRET_KEY", nil)
	aesTextKey           = getString("MX_AES_TEXT_KEY", nil)
	aesFileKey           = getString("MX_AES_FILE_KEY", nil)
	compSecretPrefix     = getString("MX_COMPANY_SECRET_PREFIX", "xtekram")
	supportEmailHostname = getString("MX_SUPPORT_EMAIL_HOSTNAME",
		"smtp.gmail.com")
	supportEmailPort     = getInt("MX_SUPPORT_EMAIL_PORT", 587)
	supportEmail         = getString("MX_SUPPORT_EMAIL", nil)
	supportEmailUsername = getString("MX_SUPPORT_EMAIL_USERNAME", nil)
	supportEmailPassword = getString("MX_SUPPORT_EMAIL_PASSWORD", nil)
	transactUrl          = getString("MX_TRANSACT_URL", nil)
	transactId           = getString("MX_TRANSACT_ID", nil)
	transactKey          = getString("MX_TRANSACT_KEY", nil)
	docusignUrl          = getString("MX_DOCUSIGN_URL",
		"https://demo.docusign.net/restapi/v2")
	docusignUsername             = getString("MX_DOCUSIGN_USERNAME", nil)
	docusignPassword             = getString("MX_DOCUSIGN_PASSWORD", nil)
	docusignAccountId            = getString("MX_DOCUSIGN_ACCOUNT_ID", nil)
	docusignIntegratorKey        = getString("MX_DOCUSIGN_INTEGRATOR_KEY", nil)
	docusignSellEngagementLetter = getString(
		"MX_DOCUSIGN_SELL_ENGAGEMENT_LETTER",
		"d59cb209-3602-4d77-ab60-e15a1a2af4e8")
	docusignBuyEngagementLetter = getString(
		"MX_DOCUSIGN_BUY_ENGAGEMENT_LETTER",
		"e63182bf-eaf9-40f6-9753-847f195578ff")
	docusignBuySummaryOfTerms = getString(
		"MX_DOCUSIGN_BUY_SUMMARY_OF_TERMS",
		"48f3a443-2312-43c9-8ebb-77218ef7989d")
	docusignBuyDePpm = getString(
		"MX_DOCUSIGN_BUY_DE_PPM",
		"5a84107e-aea9-476e-94d8-2e352ecf344c")
	docusignBuyDeOperatingAgreement = getString(
		"MX_DOCUSIGN_BUY_DE_OPEARTING_AGREEMENT",
		"ec794134-f795-41e0-b25f-c6d3249bd5a4")
	docusignBuyDeSubscriptionAgreement = getString(
		"MX_DOCUSIGN_BUY_DE_SUBSCRIPTION_AGREEMENT",
		"5d19273a-5495-4ddf-b942-2dfccc568730")
	wechatTokens     = getStringArray("MX_WECHAT_TOKENS", []string{})
	aliDayuAppKey    = getString("ALI_DAYU_APP_KEY", "23532365")
	aliDayuAppSecret = getString("ALI_DAYU_APP_SECRET", "c9883ac3d0cfbec594e995827ffbedc3")
	useSsl           = getBool("MX_USE_SSL", false)
)
