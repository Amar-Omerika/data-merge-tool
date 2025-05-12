package main

import (
	"fmt"
)

// TestExtraction can be used for testing the PDF extraction functionality
func TestExtraction() {
	// Replace with a path to an actual PDF you want to test
	pdfPath := "path/to/your/pdf.pdf"
	
	fmt.Println("Testing PDF extraction from", pdfPath)
	barcodes, amounts, err := extractDataFromPDF(pdfPath)
	if err != nil {
		fmt.Printf("Error extracting data: %v\n", err)
		return
	}
	
	fmt.Println("Extracted data:")
	for i, barcode := range barcodes {
		fmt.Printf("%d. Barcode: %s, Amount: %s\n", i+1, barcode, amounts[barcode])
	}
}

/* Uncomment this to run the test directly:
func main() {
	TestExtraction()
}
*/
