// In this example we'll look at how to implement
// a _worker pool_ using goroutines and channels.

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Here's the worker, of which we'll run several
// concurrent instances. These workers will receive
// work on the `jobs` channel and send the corresponding
// results on `results`. We'll sleep a second per job to
// simulate an expensive task.
func worker(id int, url string, client *http.Client, jobs chan *PdfData) {

	for j := range jobs {

		mainUrl := url + j.link
		status := downloadPDF(mainUrl, j.FilePath(), client)
		j.downloadStatus = status

		j.parentAirport.wg.Done() //set the task done in the airport working group
		j.parentAirport.nbDownloaded = j.parentAirport.nbDownloaded + 1
		fmt.Printf("%s downloaded %d / %d \n", j.parentAirport.icao, j.parentAirport.nbDownloaded, len(j.parentAirport.PdfData))

	}
}

func downloadPDF(url string, pathFile string, client *http.Client) bool {

	//create the directory
	os.MkdirAll(filepath.Dir(pathFile), os.ModePerm)

	newFile, err := os.Create(pathFile)
	if err != nil {
		log.Fatal(err)
	}
	defer newFile.Close()

	// HTTP GET request
	response, err := client.Get(url)
	defer response.Body.Close()

	// Write bytes from HTTP response to file.
	// response.Body satisfies the reader interface.
	// newFile satisfies the writer interface.
	// That allows us to use io.Copy which accepts
	// any type that implements reader and writer interface
	numBytesWritten, err := io.Copy(newFile, response.Body)
	if err != nil {

		log.Fatal(err)
	}
	log.Printf("Downloaded %d byte file %s.\n", numBytesWritten, pathFile)
	return true
}

// DownloadAndMergeAiportData will donwload the aiport pdf files (description and charts).
// In order to save time, will download only the files that are not up to date or not created before.
// DonwloadAndMergeAirportsData has the capability to retrieve and restart a donwload if:
// - the directory where the data are stored exists but the date is not in accordance with the effective date
// - by force
// - indicated files or directory do not exist or the dates are not in accordance with the effective date
// DownloadAndMergeAirportData does not download directly the files. Instead it puts the download files
// in the jobs channel. By this way it is possible to limit more easily the number of http client used to
// download the data.
// After download, the pdf data files are merged together in order to create _full pdf file and _chart pdf file.
// If for any reason the merge fails (mainly for file problem), a new download is performed for all the airport data.
// This new download is done only one time.
func DownloadAndMergeAiportData(apt *Airport, jobs *chan *PdfData, docWg *sync.WaitGroup, force bool) {

	//reset the number of pdf files downloaded
	apt.nbDownloaded = 0
	//Ensures that we try at worst two times the download
	if apt.downloadCount > 1 {
		//fmt.Println("*******" + apt.icao + " cannot perform the Merge process effciently - stop")
		log.Fatal("*******" + apt.icao + " cannot perform the Merge process effciently - stop")
	}

	DownloadAiportData(apt, jobs, force)
	//wait the waiting group of the airport
	apt.wg.Wait()

	//merge the pdf data if everything was done
	//thanks the Wait, the call of DetermineIsDownloaded is not mandatory.
	//But it provides a complementary means of verification
	if apt.DetermmineIsDownloaded() {
		fmt.Println("Airport: " + apt.icao + " all docs downloaded confirmed.")
		err := MergePdfDataOfAiport(apt)
		if err != nil {
			DownloadAndMergeAiportData(apt, jobs, docWg, true)
		} else {
			//All the airport downloads and merge have been done. The airport can be remove of the waiting group
			docWg.Done()
		}
	} else {
		//all the files have not been downloaded. Start again the download...
		fmt.Println("*******" + apt.icao + " is not completed. No PDF merge done. Start a New download")
		DownloadAndMergeAiportData(apt, jobs, docWg, true)
	}

}

func DownloadAiportData(apt *Airport, jobs *chan *PdfData, force bool) {

	di, err := os.Stat(apt.DirDownload())
	//determine if the target directory exists, or was created before the effective date.
	// Also, if force is done, all the files will be donwloaded again.
	if force || os.IsNotExist(err) || di.ModTime().Before(apt.aipDocument.effectiveDate) {
		//create the directory
		os.MkdirAll(apt.DirDownload(), os.ModePerm)
		//the directory does not exist or is not up to date
		apt.setPdfDataListInChannel(jobs)

		//set the directory time to the cuurent date
		if err := os.Chtimes(apt.DirDownload(), time.Now(), time.Now()); err != nil {
			log.Fatal(err)
			panic(err)
		}
	} else if err == nil {
		//if directory exists, then case by case in regard of the file description
		for i := range apt.PdfData {
			apt.PdfData[i].parentAirport = apt
			filePth := apt.PdfData[i].FilePath()
			fi, err := os.Stat(filePth)
			if os.IsNotExist(err) {
				//the files does not exist, download it

				*jobs <- &(apt.PdfData[i])
				apt.wg.Add(1) //add to the working group
			} else if err == nil {
				//the file exists, check if the file is before the effectiveDate.
				//As there is one directory by effective Date, there is no specific
				//ned to check if the file is after the next effective date.
				//This check is only to be sur that the directory is well up to date
				if fi.ModTime().Before(apt.aipDocument.effectiveDate) {
					*jobs <- &(apt.PdfData[i])
					apt.wg.Add(1) //add to the working group
				} else {
					apt.PdfData[i].downloadStatus = true
					apt.nbDownloaded = apt.nbDownloaded + 1
				}

			} else if err != nil {
				//there is an error... lets go for a panic
				log.Fatal(err)
				panic(err)
			}
		}
	} else {
		//there is an error with the directory. Lets go for a panic
		log.Fatal(err)
		panic(err)
	}
}