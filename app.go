package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xuri/excelize/v2"
)

// App struct
type App struct {
    ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
    return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
    runtime.LogInfof(ctx, "App started")
}

// ProcessPDF is the function called from JS
func (a *App) ProcessPDF(pdfPath string) (string, error) {
    pdfName := filepath.Base(pdfPath)
    pdfNameWithoutExt := strings.TrimSuffix(pdfName, filepath.Ext(pdfName))
    outputPath := filepath.Join("output", pdfNameWithoutExt+".xlsx")

    barcodes, amounts, err := extractDataFromPDF(pdfPath)
    if err != nil {
        return "", err
    }

    err = updateExcelWithData("Template.xlsx", "output", pdfNameWithoutExt, barcodes, amounts)
    if err != nil {
        return "", err
    }

    return outputPath, nil
}
func (a *App) OpenPDFDialog() (string, error) {
    opts := runtime.OpenDialogOptions{
        Title: "Select PDF File",
        Filters: []runtime.FileFilter{
            {
                DisplayName: "PDF files (*.pdf)",
                Pattern:     "*.pdf",
            },
        },
    }
    return runtime.OpenFileDialog(a.ctx, opts)
}
func extractDataFromPDF(pdfPath string) ([]string, map[string]string, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var barcodes []string
	amounts := make(map[string]string)
	barcodeSet := make(map[string]bool)

	barcodeRegex := regexp.MustCompile(`\b\d{13}\b`)
	amountRegex := regexp.MustCompile(`\b(\d{1,3}(?:[.,]\d{2}))\b`)

	totalPages := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		page := r.Page(pageIndex)
		content, err := page.GetPlainText(nil)
		if err != nil {
			return nil, nil, err
		}

		foundBarcodes := barcodeRegex.FindAllStringIndex(content, -1)

		for _, match := range foundBarcodes {
			barcode := content[match[0]:match[1]]
			if barcodeSet[barcode] {
				continue
			}

			endIndex := match[1] + 100
			if endIndex > len(content) {
				endIndex = len(content)
			}
			contextAfter := content[match[1]:endIndex]

			amountMatch := amountRegex.FindStringSubmatch(contextAfter)
			if len(amountMatch) > 1 {
				rawAmount := strings.Replace(amountMatch[1], ",", ".", 1)
				barcodes = append(barcodes, barcode)
				amounts[barcode] = rawAmount
				barcodeSet[barcode] = true
			}
		}
	}
	return barcodes, amounts, nil
}

func updateExcelWithData(excelPath, outputDir, outputName string, barcodes []string, amounts map[string]string) error {
	excel, err := excelize.OpenFile(excelPath)
	if err != nil {
		return err
	}
	defer excel.Close()

	sheets := excel.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found in Excel file")
	}
	sheetName := sheets[0]

	rows, err := excel.GetRows(sheetName)
	if err != nil {
		return err
	}

	var barcodeCol, amountCol int = -1, -1

	if len(rows) > 0 {
		for i, cell := range rows[0] {
			if strings.Contains(strings.ToLower(cell), "bar code") {
				barcodeCol = i
			}
			if strings.Contains(strings.ToLower(cell), "kol") {
				amountCol = i
			}
		}
	}

	if barcodeCol == -1 || amountCol == -1 {
		return fmt.Errorf("could not detect 'Bar Code' or 'Kol.' columns in Excel")
	}

	for rowIndex, row := range rows {
		if barcodeCol < len(row) {
			cellValue := row[barcodeCol]
			for _, barcode := range barcodes {
				if strings.Contains(cellValue, barcode) {
					amount := amounts[barcode]
					colName, _ := excelize.ColumnNumberToName(amountCol + 1)
					cellRef := fmt.Sprintf("%s%d", colName, rowIndex+1)
					excel.SetCellValue(sheetName, cellRef, amount)
					fmt.Printf("✔ Updated barcode %s with amount %s at cell %s\n", barcode, amount, cellRef)
				}
			}
		}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, outputName+".xlsx")
	return excel.SaveAs(outputPath)
}