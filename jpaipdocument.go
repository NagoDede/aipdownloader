package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func (aipdcs *AipDocument) GetAirports(cl *http.Client) []Airport {
	var indexUrl = aipdcs.fullURLDir + JapanAis.AipIndexPageName
	apts := []Airport{}

	fmt.Println("   Retrieve Airports list from: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("No url found for airports extractiob")
		log.Fatal(err)
	}

	var countWkr int
	var wg sync.WaitGroup
	doc.Find(`div[id="AD-2details"]`).Each(func(index int, divhtml *goquery.Selection) {
		divhtml.Find(`div[class="H3"]`).Each(func(index int, h3html *goquery.Selection) {
			countWkr = countWkr + 1
			fmt.Println("Main: Starting worker", countWkr)
			wg.Add(1)
			//go worker(&wg, i)

			go aipdcs.retrieveAirport(&wg, h3html, cl)
		})
	})

	fmt.Println("Main: Waiting for workers to finish")
	wg.Wait()
	fmt.Println("Main: Completed")
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

					ad.GetPDFFromHTML(cl, aipDoc.fullURLDir)
					aipDoc.airports = append(aipDoc.airports, *ad.Airport)
				}
			}
		}
	})
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
