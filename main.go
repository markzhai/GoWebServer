// Entry point for MarketX server
package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

var (
	serverLog      *log.Logger
	dbConn         *gorm.DB
	dbLog          *log.Logger
	serverTransact *Transact
	serverDocusign *Docusign
)

func main() {
	// Setup a watcher so the log always gets written to the right file
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("Error creating file watcher:", err)
	}
	defer watcher.Close()

	var sslf, sdlf *os.File
	ssl := func() {
		// Close previous instance if available
		if sslf != nil {
			sslf.Close()
			watcher.Remove(logFileName)
		}
		// Setup server logging file, exit if failed
		var err error
		sslf, err = os.OpenFile(logFileName,
			os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
		if err != nil {
			log.Fatalln("Error opening server log file:", err)
		}
		err = watcher.Add(logFileName)
		if err != nil {
			log.Fatalln("Error adding watcher for server log:", err)
		}
		// Create server logger
		serverLog = log.New(sslf, "", log.LstdFlags)
	}
	sdl := func() {
		// Close previous instance if available
		if sdlf != nil {
			sdlf.Close()
			watcher.Remove(dbLogFileName)
		}
		// Setup db logging file, exit if failed
		var err error
		sdlf, err = os.OpenFile(dbLogFileName,
			os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
		if err != nil {
			log.Fatalln("Error opening database log file:", err)
		}
		err = watcher.Add(dbLogFileName)
		if err != nil {
			log.Fatalln("Error adding watcher for database log:", err)
		}
		// Create db logger
		dbLog = log.New(sdlf, "", log.LstdFlags)
	}

	// Create endless loop to check for file states
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op & fsnotify.Create == fsnotify.Create ||
					event.Op & fsnotify.Remove == fsnotify.Remove ||
					event.Op & fsnotify.Rename == fsnotify.Rename ||
					event.Op & fsnotify.Chmod == fsnotify.Chmod {
					if event.Name == logFileName {
						ssl()
						defer sslf.Close()
					} else if event.Name == dbLogFileName {
						sdl()
						defer sdlf.Close()
					}
				}
			}
		}
	}()

	// Bootstrap log files
	ssl()
	defer sslf.Close()
	sdl()
	defer sdlf.Close()

	// Setup database connection, exit if failed
	dbConn, err = gorm.Open(dbScheme, dbUrl)
	if err != nil {
		log.Fatalln("Error opening database connection:", err)
	}
	defer dbConn.Close()

	// Setup database logging
	dbConn.LogMode(true)
	dbConn.SetLogger(dbLog)

	// Drop tables
	if dropTables {
		dbLog.Println("DROPPING ALL TABLES")
		for _, table := range allTables {
			dbConn.DropTable(table)
		}
	}

	// Setup tables
	dbLog.Println("AUTO MIGRATING ALL TABLES")
	dbConn.AutoMigrate(allTables...)

	// Importing data
	if importData {
		dbLog.Println("AUTO IMPORTING TABLE DATA")

		// Common helpers
		// Conversion of uint
		var parseNum = func(s string) uint64 {
			num, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return 0
			}
			return num
		}
		// Conversion of float
		var parseFloat = func(s string) float64 {
			fl, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return 0.0
			}
			return fl
		}

		// Import tags
		f, err := os.Open("data/tags.csv")
		// Skip 0 for better id
		tags := []Tag{Tag{}}
		if err != nil {
			dbLog.Fatalln("FAILED TO READ TAGS")
		} else {
			r := csv.NewReader(f)
			records, err := r.ReadAll()
			if err != nil {
				dbLog.Fatalln("FAILED TO PARSE TAGS")
			} else {
				for i, rec := range records {
					// Ignore first line of column names
					if i == 0 {
						continue
					}
					// Create tag with specific ids
					t := Tag{Name: rec[0], NameCn: rec[1]}
					t.ID = uint(i)
					if dbConn.Create(&t).Error != nil {
						dbLog.Fatalf("FAILED TO SAVE A TAG %v\n", i)
					}
					tags = append(tags, t)
				}
			}
		}
		f.Close()

		// Import fundings
		f, err = os.Open("data/fundings.csv")
		// Skip 0 for better id
		fundings := [][]Funding{[]Funding{}}
		if err != nil {
			dbLog.Fatalln("FAILED TO READ FUNDINGS")
		} else {
			r := csv.NewReader(f)
			records, err := r.ReadAll()
			if err != nil {
				dbLog.Fatalln("FAILED TO PARSE FUNDINGS")
			} else {
				var idx uint64 = 1
				funds := []Funding{}
				// Conversion from B/M to float in B
				var parseMoney = func(s string) float64 {
					if s == "0" {
						return 0.0
					}
					fl, err := strconv.ParseFloat(s[1:len(s) - 1], 64)
					if err != nil {
						return 0.0
					}
					if s[len(s) - 1] == 'M' {
						fl /= 1000.0
					}
					return fl
				}
				for i, rec := range records {
					// Ignore first line of column names
					if i == 0 {
						continue
					}
					fi, err := strconv.ParseUint(rec[0], 10, 64)
					if err != nil {
						dbLog.Fatalf("FAILED TO READ A FUNDING ID %v\n", i)
					}
					if fi > idx {
						// Going next
						fundings = append(fundings, funds)
						idx += 1
						funds = []Funding{}
					}

					// Add in
					funds = append(funds,
						Funding{CompanyID: fi, Type: rec[1], Date: rec[2],
							Amount:        parseMoney(rec[3]),
							RaisedToDate:  parseMoney(rec[4]),
							PreValuation:  parseMoney(rec[5]),
							PostValuation: parseMoney(rec[6]),
							Status:        rec[7], Stage: rec[8],
							NumShares:               parseNum(rec[9]),
							ParValue:                parseFloat(rec[10]),
							DividendRatePercent:     parseFloat(rec[11]) * 100,
							OriginalIssuePrice:      parseFloat(rec[12]),
							Liquidation:             parseFloat(rec[13]),
							LiquidationPrefMultiple: parseNum(rec[14][:1]),
							ConversionPrice:         parseFloat(rec[15]),
							PercentOwned:            parseFloat(rec[16]) * 100})
				}
				// Append the last one left
				if len(funds) > 0 {
					fundings = append(fundings, funds)
				}
			}
		}
		f.Close()

		// Import company executives
		f, err = os.Open("data/company_executives.csv")
		// Skip 0 for better id
		execs := [][]CompanyExecutive{[]CompanyExecutive{}}
		if err != nil {
			dbLog.Fatalln("FAILED TO READ COMPANY EXECUTIVES")
		} else {
			r := csv.NewReader(f)
			records, err := r.ReadAll()
			if err != nil {
				dbLog.Fatalln("FAILED TO PARSE COMPANY EXECUTIVES")
			} else {
				var idx uint64 = 1
				ces := []CompanyExecutive{}
				for i, rec := range records {
					// Ignore first line of column names
					if i == 0 {
						continue
					}
					fi, err := strconv.ParseUint(rec[0], 10, 64)
					if err != nil {
						dbLog.Fatalf("FAILED TO READ A COMPANY " +
							"EXECUTIVE ID %v\n", i)
					}
					if fi > idx {
						// Going next
						execs = append(execs, ces)
						idx += 1
						ces = []CompanyExecutive{}
					}

					// Add in
					ces = append(ces,
						CompanyExecutive{CompanyID: fi, Name: rec[1],
							Role: rec[2], Office: rec[3]})
				}
				// Append the last one left
				if len(ces) > 0 {
					execs = append(execs, ces)
				}
			}
		}
		f.Close()

		// Import company updates
		f, err = os.Open("data/company_updates.csv")
		// Skip 0 for better id
		updates := [][]CompanyUpdate{[]CompanyUpdate{}}
		if err != nil {
			dbLog.Fatalln("FAILED TO READ COMPANY UPDATES")
		} else {
			r := csv.NewReader(f)
			records, err := r.ReadAll()
			if err != nil {
				dbLog.Fatalln("FAILED TO PARSE COMPANY UPDATES")
			} else {
				var idx uint64 = 1
				// Start with Chinese language
				cus := []CompanyUpdate{}
				for i, rec := range records {
					// Ignore first line of column names
					if i == 0 {
						continue
					}
					fi, err := strconv.ParseUint(rec[0], 10, 64)
					if err != nil {
						dbLog.Fatalf("FAILED TO READ A COMPANY " +
							"UPDATE ID %v\n", i)
					}
					if fi > idx {
						updates = append(updates, cus)
						// Fill in empties in between
						for j := idx; j < fi - 1; j++ {
							updates = append(updates, []CompanyUpdate{})
						}
						idx = fi
						cus = []CompanyUpdate{}
					}

					// Add in with reverse ordering (newest last)
					cus = append([]CompanyUpdate{
						CompanyUpdate{CompanyID: fi, Title: rec[1],
							Url: rec[2], Date: rec[3], Language: rec[4]}},
						cus...)
				}
				// Append the last one left
				if len(cus) > 0 {
					updates = append(updates, cus)
					// Fill in more to prevent un-accounted companies
					for j := 0; j < 100; j++ {
						updates = append(updates, []CompanyUpdate{})
					}
				}
			}
		}
		f.Close()

		// Import companies
		f, err = os.Open("data/companies.csv")
		if err != nil {
			dbLog.Fatalln("FAILED TO READ COMPANIES")
		} else {
			r := csv.NewReader(f)
			records, err := r.ReadAll()
			if err != nil {
				dbLog.Fatalln("FAILED TO PARSE COMPANIES")
			} else {
				for i, rec := range records {
					// Ignore first line of column names
					if i == 0 {
						continue
					}
					// Create tag with specific ids
					tss := strings.Split(rec[12], ",")
					ts1 := parseNum(tss[0])
					ts2 := parseNum(tss[1])
					ts3 := parseNum(tss[2])
					c := Company{Name: rec[0], Description: rec[1],
						DescriptionCn: rec[2], YearFounded: parseNum(rec[3]),
						Hq: rec[4], HomePage: rec[5], KeyPerson: rec[6],
						NumEmployees:      parseNum(rec[7]),
						TotalValuation:    parseFloat(rec[8]),
						TotalFunding:      parseFloat(rec[9]),
						GrowthRatePercent: parseFloat(rec[10]),
						SizeMultiple:      parseFloat(rec[11]),
						Investors:         strings.Replace(rec[13], "\n", ",", -1),
						InvestorLogoPics:  strings.Replace(rec[14], "\n", ",", -1),
						NumSlides:         parseNum(rec[15]),
						CompanyExecutives: execs[i],
						Fundings:          fundings[i],
						CompanyUpdates:    updates[i],
						Tags:              []Tag{tags[ts1], tags[ts2], tags[ts3]}}
					c.ID = uint(i)
					// Push 6 live deals here
					if i == 1 || i == 2 || i == 3 || i == 5 || i == 6 ||
						i == 9 {
						c.Deals = []Deal{Deal{CompanyID: uint64(i),
							DealState:   DealStateOpen,
							DealSpecial: DealSpecialTwentyPercentOff}}
					}
					if dbConn.Create(&c).Error != nil {
						dbLog.Fatalf("FAILED TO SAVE A COMPANY %v\n", i)
					}
				}
			}
		}
		f.Close()
	}

	// Setup transact api
	serverLog.Println("[MX] Setting up Transact API")
	serverTransact = &Transact{transactUrl, transactId, transactKey, true}

	// Setup docusign api
	serverLog.Println("[MX] Setting up DocuSign API")
	serverDocusign = &Docusign{docusignUrl, docusignUsername, docusignPassword,
		docusignAccountId, docusignIntegratorKey, true}

	// Setup routes and start listening for requests
	router := newServerRouter()

	serverLog.Println("[MX] Starting server... @", serverPort)
	serverLog.Println("[MX] Stopping server...",
		http.ListenAndServe(fmt.Sprintf(":%v", serverPort), router))
}
