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

/*
 The Airport Type contains the information for the definition of an airport in the AIP.
 The ICAO code is the main mean of identification of the airport.
 The individual charts are recorded in the PdfData tables.
 In order to manage the downloads, the structure contains information about the status of the downloads.
 A waiting group is associated to the Airport structure in order to manage the downloads
 or any other tasks associated to the airport.
*/
type Airport struct {
	Title       string
	Icao        string
	link        string `json:"-"`
	airportType string `json:"-"`
	downloadData
	AdminData   AdminData
	navaids     map[string]Navaid
	PdfData     []PdfData    `json:"-"`
	MergePdf    []MergedData `json:"-"`
	com         []ComData
	airport     AirportInterface `json:"-"`
	aipDocument *AipDocument     `json:"-"`
	htmlPage    string           `json:"-"`
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

/*
AdminData contains the admnistrative information of the airport.
Basic information are related to the ARP coordinates, elevation, magnetic variations,...
*/
type AdminData struct {
	ArpCoord         string
	Elevation        string
	Mag_var          string
	Mag_annualchange string
	Geoid_undulation string
	Traffic_types    string
}

/*
 ComData describes the communication means available on the airport.
*/
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

/*
	Get PDF from HTML
*/
func (a *Airport) GetPDFFromHTML(cl *http.Client, aipURLDir string) {
	fmt.Println("Aiport not implemented")
}

/*
	Get the download directory.
*/
func (a *Airport) DirDownload() string {
	return filepath.Join(a.aipDocument.DirMainDownload(), a.Icao)
}

func (pdf *PdfData) FilePath() string {
	return filepath.Join(pdf.parentAirport.DirDownload(), pdf.fileName)
}

/*
Set in channel the content of the Airport.PdfData in the indicated channel.
Also, add each PdfData file as a task in the Waiting group of the Airport
*/
func (a *Airport) setPdfDataListInChannel(jobs *chan *PdfData) {
	for i := range a.PdfData {
		a.PdfData[i].parentAirport = a
		*jobs <- &(a.PdfData[i])
		a.wg.Add(1) //add to the working group
	}
}

/*
	Determine if all airport's data have been downloaded.
*/
func (a *Airport) DetermmineIsDownloaded() bool {
	var tempB bool
	tempB = true
	for _, pdf := range a.PdfData {
		tempB = tempB && pdf.downloadStatus
	}
	if tempB {
		a.downloadCount = a.downloadCount + 1
	}
	return tempB
}

/*
	Download the AIP webpage of the airport.
	This webpage will be used to retrieve all the relevant information.
	The path to the downloaded file will be indicated in the airport.htmlPage field.
*/
func (a *Airport) DownloadPage(cl *http.Client) { //, aipURLDir string) {

	var indexUrl = a.aipDocument.FullURLDir + a.link // aipURLDir + apt.link
	fmt.Println("     Download the airport page: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	// HTTP GET request

	filePth := filepath.Join(a.DirDownload(), a.Icao+".html")

	if a.shouldIDownloadHtmlPage(filePth, resp.ContentLength) {
		//create the directory
		os.MkdirAll(a.DirDownload(), os.ModePerm)
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
		log.Printf("Airport %s - downloaded %d byte file %s.\n", a.Icao, numBytesWritten, filePth)
	} else {
		log.Printf("Airport %s - page %s not saved, local copy is good %s.\n", a.Icao, indexUrl, filePth)
	}
	a.htmlPage = filePth
}

/*
	Download the airport page in a synchronous way.
*/
func (a *Airport) DownloadAirportPageSync(cl *http.Client, docWg *sync.WaitGroup) {
	go a.DownloadPage(cl)
	docWg.Done()
}

/*
	Determine if the airport web page shall be downloaded.
	Return true if the page shall be downloaded.
	By default, we download the page.
*/
func (apt *Airport) shouldIDownloadHtmlPage(realPath string, bodySize int64) bool {
	if st, err := os.Stat(realPath); err == nil {
		//The file exists, check the date of the file
		if st.ModTime().After(apt.aipDocument.EffectiveDate) && st.ModTime().Before(apt.aipDocument.NextEffectiveDate) {
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
