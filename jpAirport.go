package main

import (
	"fmt"
	"log"
	"net/http"

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
