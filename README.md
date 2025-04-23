# Data Merge Tool

**Data Merge Tool** is a powerful desktop application that automates data extraction from PDF invoices and updates Excel spreadsheets with the extracted information. Designed with local businesses in mind, this tool simplifies and accelerates invoice processing workflows, significantly reducing the need for manual data entry.

---

## 🚀 Features

- **Intuitive User Interface**  
  Clean and user-friendly design for easy operation.

- **Batch Processing**  
  Process multiple PDF invoices simultaneously.

- **Automated Data Extraction**  
  Extracts barcode and quantity data from PDF invoices automatically.

- **Excel Integration**  
  Updates an Excel template with the extracted data and saves the result.

- **Real-time Progress Tracking**  
  Get visual feedback on processing status.

- **Organized Output**  
  All processed Excel files are saved in a `CM-Done` folder on your desktop.

---

## 📦 Installation

### For Users

1. Download the latest version from the provided location.
2. Run the executable file: `data-merge-tool.exe`.
3. Place `Template.xlsx` in the **same directory** as the application.
4. You're ready to go!

### For Developers

To build the application from source:

1. Install [Go](https://golang.org) (version 1.18 or later).
2. Install Wails CLI:  
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```
3. Clone this repository:
   ```bash
   git clone https://your-repo-url.git
   ```
4. Navigate to the project directory:
   ```bash
   cd data-merge-tool
   ```
5. Install dependencies:
   ```bash
   go mod tidy
   ```
6. Run in development mode:
   ```bash
   wails dev
   ```
7. Build for production:
   ```bash
   wails build
   ```

---

## 🛠️ Usage

1. Launch the application.
2. Click **"Select PDF Files"**.
3. Choose one or more PDF invoice files.
4. Monitor progress via the interface.
5. When complete, find the generated Excel files in the `CM-Done` folder on your desktop.  
   Each file will retain the name of its corresponding PDF.

---

## 📄 Template Requirements

The application requires a `Template.xlsx` file with the following specifications:

- Must contain headers:
  - `"Siframat"` — Barcode column
  - `"količina"` — Quantity column

To customize:

1. Open `Template.xlsx` in Excel.
2. Ensure the headers `"Siframat"` and `"količina"` are present.
3. Save the file, replacing the existing template if needed.

---

## ❗ Troubleshooting

| Issue | Solution |
|-------|----------|
| **Missing Template File** | Ensure `Template.xlsx` is in the same directory as the executable. |
| **Barcode Not Found** | Verify that the barcode exists in the `"Siframat"` column. |
| **Quantity Not Extracted** | The PDF format might be unusual or unsupported. |
| **Excel Format Error** | Ensure the template is in `.xlsx` format, not `.xls`. |

