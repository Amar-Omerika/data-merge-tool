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

    // Regex to find 13-digit barcodes
    barcodeRegex := regexp.MustCompile(`\b\d{13}\b`)

	// Updated regex patterns to find Kol. Jmj values
	// Pattern 1: Look for "Kol. Jmj" column value
	kolJmjRegex := regexp.MustCompile(`Kol\.\s+Jmj\s+VPC\s+Rab%[\s\S]*?(\d+,\d{2})`)

	// Pattern 2: Look for the structure "number + t + amount"
	productRegex := regexp.MustCompile(`\d+\s+t\s+(\d+[.,]\d{2})`)

	// Pattern 3: Look for "Tabs" followed by a value (fallback)
	tabsRegex := regexp.MustCompile(`Tabs\s+(\d+[.,]\d{2})`)

	// Pattern 4: More precise table structure - specifically for Kol.Jmj column
	// This matches the table structure with columns Rb, Sifra, Naziv, Kol.Jmj, VPC, etc.
	kolJmjTableRegex := regexp.MustCompile(`(?:Rb|Rb Sifra)\s+Sifra\s+Naziv\s+Kol\.\s*Jmj\s+VPC\s+Rabat%[\s\S]+?\d+\s+\d+\s+[\S\s]+?\s+(\d+[.,]\d{2})\s+(\d+[.,]\d{2})`)

	totalPages := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		page := r.Page(pageIndex)
		content, err := page.GetPlainText(nil)
		if err != nil {
			return nil, nil, err
		}

		// Print entire content for debugging
		fmt.Printf("Page %d content: %s\n", pageIndex, content)

		foundBarcodes := barcodeRegex.FindAllStringIndex(content, -1)

		for _, match := range foundBarcodes {
			barcode := content[match[0]:match[1]]
			if barcodeSet[barcode] {
				continue
			}

			// Look in a wider context for the Kol. Jmj value
			// Search both before and after the barcode
			startContext := match[0] - 300
			if startContext < 0 {
				startContext = 0
			}
			endContext := match[1] + 300
			if endContext > len(content) {
				endContext = len(content)
			}

			surroundingContext := content[startContext:endContext]

			// Try first pattern - structured table with Kol. Jmj header
			kolMatch := kolJmjRegex.FindStringSubmatch(surroundingContext)
			if len(kolMatch) > 1 {
				rawAmount := strings.Replace(kolMatch[1], ",", ".", 1)
				barcodes = append(barcodes, barcode)
				amounts[barcode] = formatNumber(rawAmount)
				barcodeSet[barcode] = true
				fmt.Printf("Found barcode %s with Kol. Jmj quantity %s\n", barcode, amounts[barcode])
				continue
			}

			// Try the precise table structure pattern
			tableMatch := kolJmjTableRegex.FindStringSubmatch(surroundingContext)
			if len(tableMatch) > 1 {
				rawAmount := strings.Replace(tableMatch[1], ",", ".", 1)
				barcodes = append(barcodes, barcode)
				amounts[barcode] = formatNumber(rawAmount)
				barcodeSet[barcode] = true
				fmt.Printf("Found barcode %s with precise table Kol.Jmj quantity %s\n", barcode, amounts[barcode])
				continue
			}

			// Try second pattern - product description with quantity
			productMatch := productRegex.FindStringSubmatch(surroundingContext)
			if len(productMatch) > 1 {
				rawAmount := strings.Replace(productMatch[1], ",", ".", 1)
				barcodes = append(barcodes, barcode)
				amounts[barcode] = formatNumber(rawAmount)
				barcodeSet[barcode] = true
				fmt.Printf("Found barcode %s with product quantity %s\n", barcode, amounts[barcode])
				continue
			}

			// Try third pattern - Tabs followed by value
			tabsMatch := tabsRegex.FindStringSubmatch(surroundingContext)
			if len(tabsMatch) > 1 {
				rawAmount := strings.Replace(tabsMatch[1], ",", ".", 1)
				barcodes = append(barcodes, barcode)
				amounts[barcode] = formatNumber(rawAmount)
				barcodeSet[barcode] = true
				fmt.Printf("Found barcode %s with Tabs quantity %s\n", barcode, amounts[barcode])
				continue
			}

			// If none of the patterns matched, look for any number close to the barcode
			// that might represent the quantity
			simpleNumberRegex := regexp.MustCompile(`(\d+[.,]\d{2})`)
			numberMatches := simpleNumberRegex.FindAllStringSubmatch(surroundingContext, -1)
			if len(numberMatches) > 0 {
				// Use a value that's about 50 chars before or after the barcode
				// This is a last resort fallback
				for _, numMatch := range numberMatches {
					numPos := strings.Index(surroundingContext, numMatch[0])
					barcodePos := strings.Index(surroundingContext, barcode)
					distance := numPos - barcodePos
					if distance > -50 && distance < 100 {
						rawAmount := strings.Replace(numMatch[1], ",", ".", 1)
						barcodes = append(barcodes, barcode)
						amounts[barcode] = formatNumber(rawAmount)
						barcodeSet[barcode] = true
						fmt.Printf("Found barcode %s with nearby quantity %s\n", barcode, amounts[barcode])
						break
					}
				}
			}
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