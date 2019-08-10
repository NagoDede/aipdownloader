package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
)

//
// The Airport Type contains the information for the definition of an airport in the AIP.
// The ICAO code is the main mean of identification of the airport.
// The individual charts are recorded in the PdfData tables.
// In order to manage the downloads, the structure contains information about the status of the downloads
// A waiting group is associated to the Airport structure is order to manage the downloads or any other tasks
// associated to the airport.
type Airport struct {
	title       string
	icao        string
	link        string
	airportType string
	downloadData
	adminData   AdminData
	navaids     []Navaids
	PdfData     []PdfData
	MergePdf    []MergedData
	com         []ComData
	airport     AirportInterface
	aipDocument *AipDocument
}

type AirportInterface interface {
	GetPDFFromHTML(cl *http.Client, aipURLDir string)
}

type downloadData struct {
	downloadCount int
	wg            sync.WaitGroup
	nbDownloaded  int
}

// AdminData contains the admnistrative information of the airport.
// Min information are related to the ARP coordinates, elevation, magnetic variations,...
type AdminData struct {
	arpCoord         string
	elevation        string
	mag_var          string
	mag_annualchange string
	geoid_undulation string
	traffic_types    string
}

// ComData describes the communication means available on the airport.
type ComData struct {
	service         string
	frequency       string
	callSign        string
	operationsHours string
	remarks         string
}

// Navaids descvribes the navigation means available on the airport/
type Navaids struct {
	id              string
	frequency       string
	navaidType      string
	magVar          string
	operationsHours string
	position        string
	elevation       string
	remarks         string
}

type PdfData struct {
	parentAirport   *Airport
	title           string
	dataContentType string
	link            string
	fileName        string
	downloadStatus  bool
}

type MergedData struct {
	parentAirport   *Airport
	title           string
	fileDirectory        string
	fileName	string
}

func (a *Airport) GetPDFFromHTML(cl *http.Client, aipURLDir string) {
	fmt.Println("Aiport not implemented")
}

func (a *Airport) DirDownload() string {
	return filepath.Join(a.aipDocument.DirMainDownload(), a.icao)
}

func (pdf *PdfData) FilePath() string {
	return filepath.Join(pdf.parentAirport.DirDownload(), pdf.fileName)
}

//setPdfDataListInChannel the content of the Airpot.PdfData in the indicated channel
//also, add each PdfData file as a task in the Waiting group of the Airport
func (apt *Airport) setPdfDataListInChannel(jobs *chan *PdfData) {
	for i := range apt.PdfData {
		apt.PdfData[i].parentAirport = apt
		*jobs <- &(apt.PdfData[i])
		apt.wg.Add(1) //add to the working group
	}
}

func (apt *Airport) DetermmineIsDownloaded() bool {
	var tempB bool
	tempB = true
	for _, pdf := range apt.PdfData {
		tempB = tempB && pdf.downloadStatus
	}
	if tempB {
		apt.downloadCount = apt.downloadCount + 1
	}
	return tempB
}
