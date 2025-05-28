package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// OpenMultiplePDFDialog allows selecting multiple PDF files
func (a *App) OpenMultiplePDFDialog() ([]string, error) {
	opts := runtime.OpenDialogOptions{
		Title: "Select PDF Files",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PDF files (*.pdf)",
				Pattern:     "*.pdf",
			},
		},
	}
	return runtime.OpenMultipleFilesDialog(a.ctx, opts)
}

// ProcessMultiplePDFs processes multiple PDF files
func (a *App) ProcessMultiplePDFs(pdfPaths []string) ([]string, []error) {
	results := make([]string, 0, len(pdfPaths))
	errors := make([]error, 0)

	for _, pdfPath := range pdfPaths {
		result, err := a.ProcessPDF(pdfPath)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", filepath.Base(pdfPath), err))
		} else {
			results = append(results, result)
		}
	}

	return results, errors
}

// ProcessPDF is the function called from JS
func (a *App) ProcessPDF(pdfPath string) (string, error) {
	pdfName := filepath.Base(pdfPath)
	pdfNameWithoutExt := strings.TrimSuffix(pdfName, filepath.Ext(pdfName))

	// Get desktop path
	desktopDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	desktopDir = filepath.Join(desktopDir, "Desktop")

	// Create CM-Done folder on desktop
	outputDir := filepath.Join(desktopDir, "CM-Done")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	outputPath := filepath.Join(outputDir, pdfNameWithoutExt+".xlsx")

	barcodes, amounts, err := extractDataFromPDF(pdfPath)
	if err != nil {
		return "", err
	}

	err = updateExcelWithData("Template.xlsx", outputDir, pdfNameWithoutExt, barcodes, amounts)
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

	totalPages := r.NumPage()
	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		page := r.Page(pageIndex)
		content, err := page.GetPlainText(nil)
		if err != nil {
			return nil, nil, err
		}

		lines := strings.Split(content, "\n")
		headerIdx := -1
		kolJmjCol := -1
		barcodeCol := -1

		// Find header row and column indices
		for idx, line := range lines {
			cols := strings.Fields(line)
			for i, col := range cols {
				if strings.HasPrefix(strings.ToLower(col), "kol.") || strings.Contains(strings.ToLower(col), "jmj") {
					kolJmjCol = i
				}
				if strings.Contains(strings.ToLower(col), "sifra") || strings.Contains(strings.ToLower(col), "barcode") {
					barcodeCol = i
				}
			}
			if kolJmjCol != -1 && barcodeCol != -1 {
				headerIdx = idx
				break
			}
		}

		if headerIdx == -1 || kolJmjCol == -1 || barcodeCol == -1 {
			continue // Could not find header or required columns
		}

		// Process data rows
		for i := headerIdx + 1; i < len(lines); i++ {
			cols := strings.Fields(lines[i])
			if len(cols) <= kolJmjCol || len(cols) <= barcodeCol {
				continue
			}
			barcodeCandidate := cols[barcodeCol]
			if !barcodeRegex.MatchString(barcodeCandidate) {
				continue
			}
			if barcodeSet[barcodeCandidate] {
				continue
			}
			qtyRaw := cols[kolJmjCol]
			qtyRaw = strings.Replace(qtyRaw, ",", ".", 1)
			barcodes = append(barcodes, barcodeCandidate)
			amounts[barcodeCandidate] = formatNumber(qtyRaw)
			barcodeSet[barcodeCandidate] = true
			fmt.Printf(" Found barcode %s with Kol.Jmj value: %s\n", barcodeCandidate, amounts[barcodeCandidate])
		}
	}
	return barcodes, amounts, nil
}

// formatNumber formats a number string, removing .00 decimal places
func formatNumber(number string) string {
	f, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return number // Return original if parsing fails
	}

	// Check if the number has no decimal part (or .00)
	if f == float64(int(f)) {
		return fmt.Sprintf("%d", int(f))
	}

	// Return with decimal places
	return number
}

// Helper function to get absolute value of an int
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
			cellLower := strings.ToLower(cell)
			// Fix: "Siframat" to "Siframat" (remove the extra 'i')
			if strings.Contains(cellLower, "siframat") {
				barcodeCol = i
			}
			if strings.Contains(cellLower, "količina") {
				amountCol = i
			}
		}
	}

	if barcodeCol == -1 || amountCol == -1 {
		return fmt.Errorf("could not detect 'Siframat' or 'količina.' columns in Excel")
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
