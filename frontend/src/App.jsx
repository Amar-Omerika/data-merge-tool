import { useState } from 'react';
import { ProcessPDF, OpenPDFDialog } from '../wailsjs/go/main/App';
import './App.css';

function App() {
  const [message, setMessage] = useState("Select a PDF file...");

  async function openFile() {
    try {
      // Call the Go method to open the file dialog
      const selectedFile = await OpenPDFDialog();

      if (selectedFile) {
        setMessage("Processing...");
        try {
          const result = await ProcessPDF(selectedFile);
          setMessage(`✅ Done! File saved as: ${result}`);
        } catch (err) {
          setMessage(`❌ Error: ${err.toString()}`);
        }
      }
    } catch (error) {
      setMessage(`❌ Error: ${error.toString()}`);
    }
  }

  return (
    <div className="container">
      <div className="app">
        <h1>PDF to Excel Converter</h1>
        <button className="btn" onClick={openFile}>Select PDF</button>
        <p className="message">{message}</p>
      </div>
    </div>
  );
}

export default App;