import grpc
import pdf_transfer_pb2 # Imports the newly generated message classes
import pdf_transfer_pb2_grpc
import logging
import time
from concurrent import futures
import os # Import os module
import ntpath # For path manipulation (safer basename)

# --- Configuration ---
LOG_FORMAT = '%(asctime)s - %(levelname)s - %(message)s'
logging.basicConfig(level=logging.INFO, format=LOG_FORMAT)
PYTHON_SERVER_PORT = 50052
UPLOAD_DIR = "uploaded_pdfs" # Directory to save PDFs

# --- Service Implementation ---
class PdfProcessorServiceImpl(pdf_transfer_pb2_grpc.PdfProcessorServiceServicer):
    def ProcessPdf(self, request, context):
        """Receives PDF, saves it, simulates processing, returns structured response."""
        original_filename = request.filename
        pdf_content = request.pdf_content
        pdf_content_len = len(pdf_content)
        logging.info(f"[Python Server] Received ProcessPdf request for '{original_filename}'. Content length: {pdf_content_len}")

        saved_status = False
        saved_filename_on_server = None
        error_message = "" # Use empty string for no error

        # --- Save the PDF file ---
        try:
            # Ensure the upload directory exists
            os.makedirs(UPLOAD_DIR, exist_ok=True)

            # Sanitize filename: get only the base name to prevent path traversal
            safe_basename = ntpath.basename(original_filename)
            if not safe_basename: # Handle empty or malicious filenames
                safe_basename = "default_uploaded.pdf"
            saved_filename_on_server = safe_basename # Store the name used

            output_path = os.path.join(UPLOAD_DIR, safe_basename)

            logging.info(f"[Python Server] Attempting to save received PDF to: {output_path}")
            with open(output_path, 'wb') as f:
                f.write(pdf_content)
            logging.info(f"[Python Server] Successfully saved PDF to: {output_path}")
            saved_status = True

        except Exception as e:
            logging.error(f"[Python Server] Failed to save PDF '{original_filename}': {e}")
            error_message = f"Failed to save PDF file on server: {e}"
            # saved_status remains False

        # --- Simulate PDF Processing ---
        # In a real application, you would use request.pdf_content here
        # with a library like PyMuPDF to extract text.
        # For this example, we just return a fixed string.
        time.sleep(0.5) # Simulate some processing time
        # In a real scenario, you might extract actual text here
        simulated_extracted_text_summary = f"Text from '{saved_filename_on_server or original_filename}' processed."
        current_processing_status = "simulated_complete" if saved_status else "error_saving_file"

        logging.info(f"[Python Server] Finished processing '{original_filename}'. Returning structured response.")

        # --- Create and return the structured Protobuf message ---
        return pdf_transfer_pb2.StructuredProcessPdfResponse(
            original_filename=original_filename,
            save_attempted=True,
            saved_successfully=saved_status,
            saved_filename_server=saved_filename_on_server or "", # Ensure string, not None
            processing_status=current_processing_status,
            simulated_text_summary=simulated_extracted_text_summary,
            error_info=error_message # Will be empty if save succeeded
        )

# --- Start Server ---
def serve():
    """Starts the Python gRPC server."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    pdf_transfer_pb2_grpc.add_PdfProcessorServiceServicer_to_server(
        PdfProcessorServiceImpl(), server
    )
    server_address = f'[::]:{PYTHON_SERVER_PORT}'
    server.add_insecure_port(server_address)
    logging.info(f"[Python Server] Starting server listening on {server_address}")
    server.start()
    logging.info("[Python Server] Server started.")
    # Keep the server running
    try:
        server.wait_for_termination() # Blocks until server stops
    except KeyboardInterrupt:
        logging.info("[Python Server] Stopping server...")
        server.stop(0)
        logging.info("[Python Server] Server stopped.")

if __name__ == '__main__':
    serve() 