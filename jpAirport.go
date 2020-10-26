package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"path/filepath"

	"github.com/PuerkitoBio/goquery"
)

type JpAirport struct {
	*Airport
}

func NewJpAirport() *JpAirport {
	ft := &JpAirport{&Airport{}}
	ft.airport = ft
	return ft
}

// GetPDFFromHTML will retrieve the PDF information (which will be downloaded later) in a HTML
// indicated by combination of the fullURLDir and the content of the Airport.link.
// The function will populate the PdfData table of the Airport.
// This approach allows a simple way to conbsider the fact that the main directory evolves for each new AIP vesion.
// As there is the web pages do not contain a direct link to the main PDF file, a dedicated entry
// is done during the process.
// There is no need to sort the identified PDF files. The natural sorting, done by the data recovery ensures
// the correct order. The name of the files is not sufficient to set them in the corect order
func (apt *JpAirport) GetPDFFromHTML(cl *http.Client, aipURLDir string) {

	apt.downloadCount = 0 //reinit the download counter
	var indexUrl = aipURLDir + apt.link
	divWord := `div[id="` + apt.icao + "-AD-2.24" + `"]`

	fmt.Println("     Retrieve PDF pathes from: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("No url found for airports extraction")
		log.Fatal(err)
	}

	//create and retrieve the main PDF page
	//the order shall be respected, else the page sequence could not be respected during the merge process
	//So first, it is the text/description pdf
	apt.PdfData = append(apt.PdfData, apt.GetTxtPDFFile())

	doc.Find(divWord).Each(func(index int, divhtml *goquery.Selection) {
		divhtml.Find("a").Each(func(index int, ahtml *goquery.Selection) {
			pdfLink, ext := ahtml.Attr("href")
			if ext {
				apt.PdfData = append(apt.PdfData, apt.GetChartPDFFile(pdfLink))
			}

		})
	})
}

// mainPDFFile creates the path to the main PDF as there is no associated link in the webpage
// and provides it in a PdfData structure (dataContentType is associated to Text)
func (apt *JpAirport) GetTxtPDFFile() PdfData {
	pdfTxt := PdfData{}
	pdfTxt.parentAirport = apt.Airport
	pdfTxt.dataContentType = "Text"
	pdfTxt.link = fmt.Sprintf("pdf/JP-AD-2-%s-en-JP.pdf", apt.icao)
	pdfTxt.fileName = fmt.Sprintf("JP-AD-2-%s-en-JP.pdf", apt.icao)

	return pdfTxt
}

func (apt *JpAirport) GetChartPDFFile(partialLink string) PdfData {
	pdfChart := PdfData{}
	pdfChart.parentAirport = apt.Airport
	pdfChart.dataContentType = "Chart"
	pdfChart.link = partialLink
	pdfChart.fileName = filepath.Base(partialLink)
	return pdfChart
}

func (apt *JpAirport) GetNavaids() (map[string]Navaid, int) {

	if apt.htmlPage == "" {
		log.Println("Html File is not downloaded")
		return nil, 0
	}
	//div[id="ENR-4details"]`
	divId := fmt.Sprintf(`div[id="%s-AD-2.19"]`, apt.icao)
	//divId := `div=["` + apt.icao + "-AD-2.19" + `"]`

	f, err := os.Open(apt.htmlPage)
	if err != nil {
		log.Println("Unable to open " + apt.htmlPage)
	}
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		fmt.Println("Unable to parse")
		log.Fatal(err)
	}

	sel := doc.Find(divId).First()
	navaids, trcount := apt.loadNavaidsFromHtmlDoc(sel)

	fmt.Println(navaids)
	fmt.Println(trcount)
	return navaids, trcount
}

func (apt *JpAirport) loadNavaidsFromHtmlDoc(div *goquery.Selection) (map[string]Navaid, int) {
	//navs := //[]Navaid{}
	apt.navaids = make(map[string]Navaid)
	trCount := 0
	div.Find("table").Each(func(index int, divhtml *goquery.Selection) {
		tbody := divhtml.Find(`tbody`).First()
		trCount = 0
		tbody.Find("tr").Each(func(index int, tr *goquery.Selection) {
			aids, isok := apt.loadNavaidsFromTr(tr)
			if isok {
				apt.navaids[aids.key] = aids
			}
			fmt.Println(aids)
		})
	})

	return apt.navaids, trCount
}

func (apt *JpAirport) loadNavaidsFromTr(tr *goquery.Selection) (Navaid, bool) {
	var n Navaid
	tr.Find("td").Each(func(index int, td *goquery.Selection) {
		switch index {
		case 0:

			n.navaidType = strings.TrimSpace(td.Text())
			if strings.Contains(n.navaidType, "(") {
				n.navaidType = strings.TrimSpace(n.navaidType[0:strings.Index(n.navaidType, "(")])
			}
			n.magVar = getMagVariationFromTextOfjpAirportData(td.Text())
		case 1:
			n.id = strings.TrimSpace(td.Text())
		case 2:
			n.frequency = strings.TrimSpace(td.Text())
		case 3:
			n.operationsHours = strings.TrimSpace(td.Text())
		case 4:
			n.position.Latitude = getLatitudeFromTextOfjpAirportData(td.Text())
			n.position.Longitude = getLongitudeFromTextOfjpAirportData(td.Text())
		case 5:
			n.elevation = strings.TrimSpace(td.Text())
		case 6:
			n.remarks = strings.TrimSpace(td.Text())
		}

		if (n.id != "") && (n.id != "-") {
			n.key = n.id + " " + n.navaidType
		} else {
			n.key = apt.icao + " " + n.navaidType
		}

	})

	//Determine if the identifed raw is a real Navaids or the titles of the table
	//If it is title, we return false
	//To do this, we test the column where only text should be used.
	//If a number is defined, it is a title row

	_, errCol1 := strconv.Atoi(n.navaidType)
	_, errCol2 := strconv.Atoi(n.id)

	if (strings.Compare(n.name, "ID") == 0) ||
		strings.Contains(n.frequency, "requency") ||
		(errCol1 == nil) || (errCol2 == nil) ||
		(strings.Contains(n.navaidType, "Nil")) ||
		(strings.Contains(n.id, "Nil")) ||
		(strings.TrimSpace(n.navaidType) == "") {
		return n, false
	} else {
		return n, true
	}
}

func getLatitudeFromTextOfjpAirportData(t string) float32 {
	latre := regexp.MustCompile(`[0-9]*\.?[0-9]+[N|S]`)
	latitude := string(latre.Find([]byte(t)))
	lat, err := convertDDMMSSSSLatitudeToFloat(latitude)
	if err != nil {
		log.Printf("%s Latitude Conversion problem %f \n", t, lat)
		log.Println(err)
		return 0
	} else {
		return lat
	}
}

func getLongitudeFromTextOfjpAirportData(t string) float32 {
	longre := regexp.MustCompile(`[0-9]*\.?[0-9]+[E|W]`)
	longitude := string(longre.Find([]byte(t)))
	long, err := convertDDDMMSSSSLongitudeToFloat(longitude)
	if err != nil {
		log.Printf("%s Longitude Conversion problem %f \n", t, long)
		log.Println(err)
		return 0
	} else {
		return long
	}
}

//getMagVariationFromTextOfjpAirportData extracts from a text
//the magnetic variation which is usually associated to a VOR, TACAN.
//Exemple VOR(4°W0.25W/y) --> (4°W0.25W/y)
func getMagVariationFromTextOfjpAirportData(t string) string {
	magre := regexp.MustCompile(`\((.*?)\)`)
	mag := string(magre.Find([]byte(t)))
	return strings.TrimSpace(mag)
}
