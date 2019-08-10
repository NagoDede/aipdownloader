package main

import (
	"fmt"
	"net/http"
	"path/filepath"
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

func (aip *AipDocument) SentToFtp() {

	var ftpI FtpInfo
	ftpI.LoadJsonFile("./ftp.json")
	ftpClient, _ := NewFtpClient(&ftpI)

	for _, apt := range aip.airports {
		for _, chrt := range apt.MergePdf {
			remotePath := filepath.Join(ftpI.Directory, aip.countryCode, chrt.fileName)
			inPath := filepath.Join(chrt.fileDirectory, chrt.fileName)
			SendFile(ftpClient, inPath, remotePath)
		}
	}

	DisconnectFromServer(ftpClient)
}
