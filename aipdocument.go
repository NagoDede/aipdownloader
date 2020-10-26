package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AipDocument struct {
	isActive          bool
	effectiveDate     time.Time
	publicationDate   time.Time
	nextEffectiveDate time.Time
	isValidDate       bool
	partialURL        string
	isPartialURLValid bool
	fullURLDir        string
	fullURLPage       string
	airports          []Airport
	countryCode       string
}

type AipDocumentInterface interface {
	GetAirports(cl *http.Client) []Airport
	DownloadAllAiportsData(client *http.Client)
}

func (aip *AipDocument) DirMainDownload() string {
	dir := filepath.Join(ConfData.MainLocalDir, aip.countryCode)
	t := aip.effectiveDate
	dateDir := fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())
	return filepath.Join(dir, dateDir)
}

func (aip *AipDocument) DirMergeFiles() string {
	return filepath.Join(aip.DirMainDownload(), ConfData.MergeDir)
}

func (aip *AipDocument) TestFtp() {
	var ftpI FtpInfo
	ftpI.LoadJsonFile("./ftp.json")
	ftpClient, _ := NewFtpClient(&ftpI)

	//Delete the directory to start from scracth and be sure to have the latest files
	remoteDir := filepath.Join(ftpI.Directory, aip.countryCode, "RORA_full.pdf")
	fmt.Printf("Delete FTP Directory: %s \n", remoteDir)
	entries, _ := ftpClient.List(filepath.ToSlash(remoteDir))
	fmt.Println(entries)

	DisconnectFromFtpServer(ftpClient)
}

func (aip *AipDocument) SentToFtp() {

	var ftpI FtpInfo
	ftpI.LoadJsonFile("./ftp.json")
	ftpClient, _ := NewFtpClient(&ftpI)

	//Delete the directory to start from scracth and be sure to have the latest files
	remoteDir := filepath.Join(ftpI.Directory, aip.countryCode)
	fmt.Printf("Do we need to delete FTP Directory %s ?\n", remoteDir)
	entries, err := ftpClient.List(filepath.ToSlash(remoteDir))
	if err != nil {
		fmt.Println("Try to deleted, due to error")
		DeleteFtpDirectory(ftpClient, remoteDir)
		fmt.Printf("Create FTP Directory: %s \n", remoteDir)
		CreateFtpDirectory(ftpClient, remoteDir)
	} else {
		if len(entries) > 0 {
			//keep the directory is the creation date is in accordance with the effective date
			e := entries[len(entries)-1]
			if e.Time.After(aip.effectiveDate) && e.Time.Before(aip.nextEffectiveDate) {
				fmt.Println("Keep current directory")
			} else {
				fmt.Println("Delete outdated directory %s \n", remoteDir)
				DeleteFtpDirectory(ftpClient, remoteDir)
				fmt.Printf("Create FTP Directory: %s \n", remoteDir)
				CreateFtpDirectory(ftpClient, remoteDir)
			}
		}
	}

	fmt.Println("** Upload the files t othe FTP server**")

	nameList, err := ftpClient.NameList(filepath.ToSlash(remoteDir))
	if err != nil {
		log.Println(err)
	}
	fmt.Println(nameList)
	nameList = nameList[2:] //remove the ".." and '.'

	if len(nameList) == 0 {
		for _, apt := range aip.airports {
			for _, chrt := range apt.MergePdf {
				remotePath := filepath.Join(remoteDir, chrt.fileName)
				inPath := filepath.Join(chrt.fileDirectory, chrt.fileName)
				SendFtpFile(ftpClient, inPath, remotePath)
			}
		}
	} else {
		for _, apt := range aip.airports {
			for _, chrt := range apt.MergePdf {
				remotePath := filepath.Join(remoteDir, chrt.fileName)
				inPath := filepath.Join(chrt.fileDirectory, chrt.fileName)
				var wasfound bool = false
				for _, rf := range nameList {
					if strings.Compare(rf, chrt.fileName) == 0 {
						wasfound = true
						rSize, err := ftpClient.FileSize(filepath.ToSlash(remotePath))
						if err != nil {
							//in case of error, upload the file (error could also means there is no file)
							log.Printf("Upload, unable to retrieve file info %s \n", remotePath)
							SendFtpFile(ftpClient, inPath, remotePath)
						} else {
							fi, err := os.Stat(inPath) //get the local info

							if err != nil {
								log.Printf("Unable to retrieve info for %s \n", remotePath)
								log.Panic("Quit, unable to retrieve file info, suspect pb on local dir")
							}

							if rSize != fi.Size() {
								log.Printf("File size discrepency  %s in: %d remote: %d \n", remotePath, fi.Size(), rSize)
								SendFtpFile(ftpClient, inPath, remotePath)
							}
						}
					}
				}
				if !wasfound {
					log.Printf("File not set on remote host %s  \n", remotePath)
					SendFtpFile(ftpClient, inPath, remotePath)
				}
			}
		}
	}
	DisconnectFromFtpServer(ftpClient)
}

func (aip *AipDocument) SentToMongoDb() {

}
