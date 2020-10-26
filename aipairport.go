package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	navaids     map[string]Navaid
	PdfData     []PdfData
	MergePdf    []MergedData
	com         []ComData
	airport     AirportInterface
	aipDocument *AipDocument
	htmlPage    string
}

type AirportInterface interface {
	GetPDFFromHTML(cl *http.Client, aipURLDir string)
	DownloadPage(cl *http.Client)
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

type PdfData struct {
	parentAirport   *Airport
	title           string
	dataContentType string
	link            string
	fileName        string
	downloadStatus  bool
}

type MergedData struct {
	parentAirport *Airport
	title         string
	fileDirectory string
	fileName      string
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

func (apt *Airport) DownloadPage(cl *http.Client) { //, aipURLDir string) {

	var indexUrl = apt.aipDocument.fullURLDir + apt.link // aipURLDir + apt.link
	fmt.Println("     Download the airport page: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	// HTTP GET request

	filePth := filepath.Join(apt.DirDownload(), apt.icao+".html")

	if apt.shouldIDownloadHtmlPage(filePth, resp.ContentLength) {
		//create the directory
		os.MkdirAll(apt.DirDownload(), os.ModePerm)
		newFile, err := os.Create(filePth)
		// Write bytes from HTTP response to file.
		// response.Body satisfies the reader interface.
		// newFile satisfies the writer interface.
		// That allows us to use io.Copy which accepts
		// any type that implements reader and writer interface

		numBytesWritten, err := io.Copy(newFile, resp.Body)
		if err != nil {
			log.Printf("Unable to write the webpage %s in directory %s \n", indexUrl, filePth)
			log.Fatal(err)
		}
		log.Printf("Airport %s - downloaded %d byte file %s.\n", apt.icao, numBytesWritten, filePth)
	} else {
		log.Printf("Airport %s - page %s not saved, local copy is good %s.\n", apt.icao, indexUrl, filePth)
	}
	apt.htmlPage = filePth
}

func (apt *Airport) DownloadAirportPageSync(cl *http.Client, docWg *sync.WaitGroup) {
	go apt.DownloadPage(cl)
	docWg.Done()
}

func (apt *Airport) shouldIDownloadHtmlPage(realPath string, bodySize int64) bool {
	if st, err := os.Stat(realPath); err == nil {
		//The file exists, check the date of the file
		if st.ModTime().After(apt.aipDocument.effectiveDate) && st.ModTime().Before(apt.aipDocument.nextEffectiveDate) {
			if bodySize == st.Size() {
				return false
			}
		}
		return true

	} else if os.IsNotExist(err) {
		return true
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
		log.Printf("File %s is not writeable or readable \n", realPath)
		return true
	}
	return true
}
