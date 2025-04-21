import { useState } from 'react';
import { ProcessPDF, OpenMultiplePDFDialog } from "../wailsjs/go/main/App";
import "./App.css";

function App() {
	const [messages, setMessages] = useState([
		{ text: "Select PDF files to convert..." },
	]);
	const [isProcessing, setIsProcessing] = useState(false);

	async function openFiles() {
		if (isProcessing) return;

		try {
			// Call the Go method to open multiple files
			const selectedFiles = await OpenMultiplePDFDialog();

			if (selectedFiles && selectedFiles.length > 0) {
				setIsProcessing(true);
				setMessages([{ text: `Processing ${selectedFiles.length} files...` }]);

				const results = [];

				// Process each file individually
				for (let i = 0; i < selectedFiles.length; i++) {
					const file = selectedFiles[i];
					const fileName = file.split(/[\\\/]/).pop(); // Get just the filename

					try {
						// Add processing message
						setMessages((prev) => [
							...prev,
							{
								text: `Processing: ${fileName} (${i + 1}/${
									selectedFiles.length
								})`,
							},
						]);

						// Process the file
						const result = await ProcessPDF(file);
						const outputFileName = result.split(/[\\\/]/).pop();

						// Update message with success
						setMessages((prev) => [
							...prev,
							{
								text: `✅ Created: ${outputFileName}`,
								success: true,
							},
						]);
					} catch (err) {
						// Update message with error
						setMessages((prev) => [
							...prev,
							{
								text: `❌ Error: ${fileName} - ${err.toString()}`,
								error: true,
							},
						]);
					}
				}

				// Add completion message
				setMessages((prev) => [
					...prev,
					{
						text: `Completed processing ${selectedFiles.length} files.`,
						summary: true,
					},
				]);
				setIsProcessing(false);
			}
		} catch (error) {
			setMessages([{ text: `❌ Error: ${error.toString()}`, error: true }]);
			setIsProcessing(false);
		}
	}

	return (
		<div className="container">
			<div className="app">
				<h1>Automation invoices</h1>
				<button
					className={`btn ${isProcessing ? "disabled" : ""}`}
					onClick={openFiles}
					disabled={isProcessing}
				>
					{isProcessing ? "Processing..." : "Select PDF Files"}
				</button>

				<div className="messages-container">
					{messages.map((msg, index) => (
						<p
							key={index}
							className={`message ${msg.success ? "success" : ""} ${
								msg.error ? "error" : ""
							} ${msg.summary ? "summary" : ""}`}
						>
							{msg.text}
						</p>
					))}
				</div>
			</div>
		</div>
	);
}

export default App;