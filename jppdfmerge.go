package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	pdf "github.com/unidoc/unipdf/v3/model"
)

func MergePdfDataOfAiport(apt *Airport) error {
	var outPath string
	pdfWriter := pdf.NewPdfWriter()
	pdf.SetPdfCreationDate(time.Now())
	pdf.SetPdfAuthor("Nagoy Dede")
	pdf.SetPdfTitle(apt.icao + "AIP charts ")
	pdf.SetPdfKeywords(apt.icao + " AIP Japan")
	pdf.SetPdfSubject(apt.icao + " merged charts")
	outPath = apt.aipDocument.DirMergeFiles()

	//create the directory
	os.MkdirAll(outPath, os.ModePerm)

	outFullMerge := MergedData{fileName: apt.icao + "_full.pdf", fileDirectory: outPath}
	apt.MergePdf = append(apt.MergePdf, outFullMerge)
	outChartMerge := MergedData{fileName: apt.icao + "_chart.pdf", fileDirectory: outPath}
	apt.MergePdf = append(apt.MergePdf, outChartMerge)

	outFullPath := filepath.Join(outPath, apt.icao+"_full.pdf")
	outChartPath := filepath.Join(outPath, apt.icao+"_chart.pdf")

	for _, pdfD := range apt.PdfData[1:] {
		err := mergeInPdfWriter(&pdfWriter, &pdfD)
		if err != nil {
			return err
		}
	}

	err2 := writePdfWriter(&pdfWriter, outChartPath)
	if err2 != nil {
		return err2
	}

	//create the full merge
	pdf.SetPdfTitle(apt.icao + " AIP document")
	pdf.SetPdfSubject(apt.icao + " merged AIP document")
	pdfFullWriter := pdf.NewPdfWriter()
	for _, pdfD := range apt.PdfData {
		err := mergeInPdfWriter(&pdfFullWriter, &pdfD)
		if err != nil {
			return err
		}
	}

	err2 = writePdfWriter(&pdfFullWriter, outFullPath)
	if err2 != nil {
		return err2
	}

	return nil

}

func mergeInPdfWriter(pdfWriter *pdf.PdfWriter, pdfD *PdfData) error {
	inPath := pdfD.FilePath()
	f, err := os.Open(inPath)
	if err != nil {
		fmt.Println("Error during  os.Open(inPath) " + inPath)
		return fmt.Errorf("Error while opening PDF file " + inPath)
	}

	defer f.Close()

	pdfReader, err2 := pdf.NewPdfReader(f)

	if err2 != nil {
		fmt.Println("Error during  pdf.NewPdfReader(f) " + inPath)
		//fmt.Println(err2)
		return fmt.Errorf("Error during PDFReader creation for file " + inPath)
	}
	numPages, err3 := pdfReader.GetNumPages()
	if err3 != nil {
		fmt.Println("Error during  pdf..GetNumPages()" + inPath)
		//fmt.Println(err3)
		return fmt.Errorf("Error when retrieving the number of pages of the file " + inPath)
	}

	for i := 0; i < numPages; i++ {
		pageNum := i + 1

		page, err4 := pdfReader.GetPage(pageNum)
		if err4 != nil {
			fmt.Println("Error while retrieving the page %d of file %s", pageNum, inPath)
			//fmt.Println(err4)
			return fmt.Errorf("Error while retrieving the page %d of file %s", pageNum, inPath)
		}

		err5 := pdfWriter.AddPage(page)
		if err5 != nil {
			fmt.Println("Error during  pdfWriter.AddPage(page)" + inPath)
			//fmt.Println(err5)
			return fmt.Errorf("Error while adding page " + inPath)
		}
	}
	return nil
}

func writePdfWriter(pdfWriter *pdf.PdfWriter, outPath string) error {

	fWrite, err6 := os.Create(outPath)
	if err6 != nil {
		fmt.Println("Error during  os.Create(outPath)" + outPath)
		//fmt.Println(err6)
		return fmt.Errorf("Error during  pdf creation of file " + outPath)
	}

	defer fWrite.Close()

	err7 := pdfWriter.Write(fWrite)
	if err7 != nil {
		fmt.Println("Error during  pdfWriter.Write(fWrite)" + outPath)
		//fmt.Println(err7)
		return fmt.Errorf("Error during pdf writing " + outPath)
	}

	return nil
}
