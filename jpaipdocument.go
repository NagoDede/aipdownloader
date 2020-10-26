package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func (aipdcs *AipDocument) GetNavaids(cl *http.Client) []Navaid {
	var indexUrl = aipdcs.fullURLDir + JapanAis.AipIndexPageName
	fmt.Println("   Retrieve RadioNavigation  in " + indexUrl)
	resp, err := cl.Get(indexUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
		log.Fatal(err)
	}

	var navaidpage string
	doc.Find(`div[id="ENR-4details"]`).Each(func(index int, divhtml *goquery.Selection) {
		divhtml.Find(`div[class="H3"]`).Each(func(index int, ahtml *goquery.Selection) {
			t, titleEx := ahtml.Find("a").Attr("title")
			if titleEx {
				if strings.Contains(t, "NAVIGATION AIDS") {
					href, hrefEx := ahtml.Find("a").Attr("href")
					if hrefEx {
						navaidpage = href
						fmt.Println("Page to the Radio Navigation" + href)
					}
				}
			}
		})
	})

	fmt.Println("Retrieve data from " + aipdcs.fullURLDir + navaidpage)
	navaidresp, err := cl.Get(aipdcs.fullURLDir + navaidpage)
	if err != nil {
		log.Fatal(err)
	}

	defer navaidresp.Body.Close()
	navaidsdoc, err := goquery.NewDocumentFromReader(navaidresp.Body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
		log.Fatal(err)
	}

	navaids, trCount := loadNavaidsFromHtmlDoc(navaidsdoc)
	//confirm we have the same number
	if trCount == len(navaids) {
		return nil
	} else {
		log.Println("Number of rows in the table and identified Navaids differs")
		return nil
	}
}

func loadNavaidsFromHtmlDoc(navaidsdoc *goquery.Document) (map[string]Navaid, int) {
	//navs := //[]Navaid{}
	var navs = make(map[string]Navaid)
	trCount := 0
	navaidsdoc.Find(`table`).Each(func(index int, divhtml *goquery.Selection) {
		tbody := divhtml.Find(`tbody`).First()
		trCount = 0
		tbody.Find("tr").Each(func(index int, tr *goquery.Selection) {
			id, titleEx := tr.Attr("id")
			if titleEx {
				fmt.Println(id)
				if strings.HasPrefix(id, "NAV-") {
					nav := Navaid{}
					nav.SetFromHtmlSelection(tr)
					if nav.key != "" {
						if val, ok := navs[nav.key]; ok {
							log.Printf("%s appears several time", val.key)
						} else {
							navs[nav.key] = nav
							trCount++
						}
					} else {
						log.Printf("%s is disregarded - not NAV data", id)
					}
				} else {
					log.Printf("%s is disregarded - not NAV data \n", id)
				}
			}
		})
	})

	return navs, trCount
}

func (aipdcs *AipDocument) GetAirports(cl *http.Client) []Airport {
	var indexUrl = aipdcs.fullURLDir + JapanAis.AipIndexPageName
	apts := []Airport{}

	fmt.Println("   Retrieve Airports list from: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("Problem while reading %s \n", indexUrl)
		log.Fatal(err)
	} else {

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			fmt.Println("No url found for airports extraction")
			log.Fatal(err)
		} else {

			var countWkr int
			var wg sync.WaitGroup
			doc.Find(`div[id="AD-2details"]`).Each(func(index int, divhtml *goquery.Selection) {
				divhtml.Find(`div[class="H3"]`).Each(func(index int, h3html *goquery.Selection) {
					countWkr = countWkr + 1
					fmt.Println("Main: Starting worker", countWkr)
					wg.Add(1)
					go aipdcs.retrieveAirport(&wg, h3html, cl)
				})
			})

			fmt.Println("Main: Waiting for workers to finish")
			wg.Wait()
			fmt.Println("Main: Completed")
		}
	}
	return apts
}

func (aipDoc *AipDocument) retrieveAirport(wg *sync.WaitGroup, h3html *goquery.Selection, cl *http.Client) {
	defer wg.Done()
	h3html.Find("a").Each(func(index int, ahtml *goquery.Selection) {
		idAd, exist := ahtml.Attr("title")
		if exist {
			if strings.Contains(idAd, "AERO") || strings.Contains(idAd, "aero") {
				idId, idEx := ahtml.Attr("id")
				if idEx {
					ad := NewJpAirport()
					ad.aipDocument = aipDoc
					//ad.aipDocument = aipDoc
					ad.icao = idId[5:9]
					ad.title = ahtml.Text()[7:]
					href, hrefEx := ahtml.Attr("href")
					if hrefEx {
						ad.link = href
					}
					ad.PdfData = []PdfData{}
					fmt.Println(ad.icao)
					fmt.Println(ad.title)
					ad.airport.DownloadPage(cl)
					ad.GetPDFFromHTML(cl, aipDoc.fullURLDir)
					maps, i := ad.GetNavaids()
					fmt.Println(maps)
					fmt.Println(i)
					aipDoc.airports = append(aipDoc.airports, *ad.Airport)
				}
			}
		}
	})
}

func (aipDoc *AipDocument) DownloadAllAiportsHtmlPage(cl *http.Client) {
	var docsWg sync.WaitGroup
	for i, _ := range aipDoc.airports {
		docsWg.Add(1)
		apt := &aipDoc.airports[i]
		apt.aipDocument = aipDoc
		apt.DownloadAirportPageSync(cl, &docsWg)
	}
	docsWg.Wait()
}

func (aipDoc *AipDocument) DownloadAllAiportsData(client *http.Client) {
	jobs := make(chan *PdfData, 10)

	var w int
	var docsWg sync.WaitGroup
	for i, _ := range aipDoc.airports {
		docsWg.Add(1)

		apt := &aipDoc.airports[i]
		apt.aipDocument = aipDoc //refresh the pointer (case we miss something)
		//create the workers. the number is limited by 5 at the time being
		if w < 5 {
			w = w + 1
			go worker(w, aipDoc.fullURLDir, client, jobs)
		}

		DownloadAndMergeAiportData(apt, &jobs, &docsWg, false)
	}
	docsWg.Wait()

	fmt.Println("Download and merge - done")

}
