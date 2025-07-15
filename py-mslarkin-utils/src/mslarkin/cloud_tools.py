import os, os.path
from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from google_auth_oauthlib.flow import InstalledAppFlow
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

# If modifying these scopes, delete the file token.json.
# These scopes allow:
# 1. 'https://www.googleapis.com/auth/documents': Full access to Google Docs (create, read, update, delete).
# 2. 'https://www.googleapis.com/auth/drive.file': Allows the app to see, edit, create, and delete only the specific Google Drive files that it opens or creates.
SCOPES = ['https://www.googleapis.com/auth/documents', 'https://www.googleapis.com/auth/drive.file']

def get_google_oauth_token(
    scopes: list,
    client_secret_file: str = 'client_secret.json',
    token_file: str = 'token.json'
) -> Credentials:
    """
    Generates or retrieves a Google OAuth token.

    This function attempts to get credentials in the following order:
    1. From a previously saved token file (e.g., 'token.json').
    2. Using Default Application Credentials (ADC), which is automatic when
       running on Google Cloud environments (like GCE, Cloud Run).
    3. Interactively via a web browser flow using a client secrets file
       (e.g., 'client_secret.json') if ADC is not available.

    Args:
        scopes: A list of Google API scopes (e.g.,
                ['https://www.googleapis.com/auth/userinfo.email',
                 'https://www.googleapis.com/auth/drive.readonly']).
        client_secret_file: The path to your OAuth 2.0 client secret JSON file.
                            Download this from Google Cloud Console.
        token_file: The path to store/load the refresh token. This file
                    is created after the first successful authentication.

    Returns:
        A google.oauth2.credentials.Credentials object containing the
        access token, refresh token, and other credential details.
    """
    creds = None

    # 1. Try to load credentials from the token file if it exists.
    # This stores the refresh token, so subsequent runs don't require re-authentication.
    if os.path.exists(token_file):
        try:
            creds = Credentials.from_authorized_user_file(token_file, scopes)
        except Exception as e:
            print(f"Warning: Could not load credentials from {token_file}. Error: {e}")
            creds = None # Reset creds to try other methods

    # 2. If no valid credentials, try to refresh or obtain new ones.
    if not creds or not creds.valid:
        if creds and creds.expired and creds.refresh_token:
            # If credentials exist but are expired, try to refresh them.
            print("Refreshing existing token...")
            try:
                creds.refresh(Request())
            except Exception as e:
                print(f"Error refreshing token: {e}")
                creds = None # Reset creds to try full flow
        else:
            # If no creds or refresh failed, try Default Application Credentials (ADC)
            # This is standard for applications running on Google Cloud infrastructure.
            print("Attempting to get credentials via Default Application Credentials (ADC)...")
            try:
                # This will look for credentials in environment variables,
                # metadata server (for GCE), or gcloud config.
                import google.auth
                creds, project = google.auth.default(scopes=scopes)
                print(f"ADC successful. Project: {project}")
            except Exception as e:
                print(f"ADC failed or not configured: {e}")
                creds = None

        # 3. If ADC failed, fall back to the InstalledAppFlow (interactive browser login)
        if not creds or not creds.valid:
            print(f"Falling back to interactive browser flow using {client_secret_file}...")
            if not os.path.exists(client_secret_file):
                raise FileNotFoundError(
                    f"Client secret file not found at '{client_secret_file}'. "
                    "Please download it from Google Cloud Console."
                )
            try:
                # The `InstalledAppFlow` handles the authorization process.
                flow = InstalledAppFlow.from_client_secrets_file(client_secret_file, scopes)
                creds = flow.run_local_server(port=0) # Automatically opens browser
                print("Authentication successful via browser flow.")
            except Exception as e:
                raise Exception(f"Failed to complete interactive browser flow: {e}")

    # Save the credentials for the next run (including the refresh token)
    if creds and creds.valid:
        try:
            with open(token_file, 'w') as token:
                token.write(creds.to_json())
            print(f"Credentials saved to {token_file}")
        except Exception as e:
            print(f"Warning: Could not save token to file. Error: {e}")

    return creds

# if __name__ == '__main__':
#     # Define the scopes (permissions) your application needs.
#     # For email identity, 'userinfo.email' is commonly used.
#     # Add more scopes as needed for specific Google APIs (e.g., Drive, Calendar).
#     # You can find scopes in the documentation for each Google API.
#     REQUIRED_SCOPES = ['https://www.googleapis.com/auth/userinfo.email',
#                        'https://www.googleapis.com/auth/userinfo.profile']

#     # --- Example Usage ---
#     try:
#         credentials = get_google_oauth_token(REQUIRED_SCOPES)

#         if credentials:
#             print("\nOAuth Token obtained successfully!")
#             print(f"Access Token: {credentials.token}")
#             print(f"Token Expiry: {credentials.expiry}")
#             print(f"Token Scopes: {credentials.scopes}")

#             # You can now use 'credentials' to make authenticated API requests
#             # For example, to get user info:
#             # from googleapiclient.discovery import build
#             # service = build('oauth2', 'v2', credentials=credentials)
#             # user_info = service.userinfo().get().execute()
#             # print(f"Authenticated User Email: {user_info['email']}")

#     except Exception as e:
#         print(f"\nAn error occurred: {e}")


def write_to_google_doc(document_name: str, content_to_write: str):
    """
    Creates a new Google Doc with the given name and content, or
    appends content to an existing Google Doc of the same name.

    Args:
        document_name: The name of the Google Doc.
        content_to_write: The string content to write to the document.
    """
    REQUIRED_SCOPES = ['https://www.googleapis.com/auth/userinfo.email',
                       'https://www.googleapis.com/auth/userinfo.profile']
    
    creds = get_google_oauth_token(REQUIRED_SCOPES)

    try:
        # Build the Google Docs service client
        docs_service = build('docs', 'v1', credentials=creds)
        # Build the Google Drive service client (used to search for existing documents by name)
        drive_service = build('drive', 'v3', credentials=creds)

        document_id = None
        document_url = None

        # --- Step 1: Search for an existing document with the given name using the Drive API ---
        # The query filters for files that are Google Docs ('application/vnd.google-apps.document'),
        # have the exact 'document_name', and are not in the trash.
        query = f"name='{document_name}' and mimeType='application/vnd.google-apps.document' and trashed=false"
        results = drive_service.files().list(
            q=query,          # The search query
            spaces='drive',   # Limit search to Google Drive
            fields='files(id, name, webViewLink)' # Specify which fields to return
        ).execute()
        items = results.get('files', [])

        if items:
            # --- Step 2a: If document is found, append content to it ---
            doc_info = items[0] # Use the first found document
            document_id = doc_info['id']
            document_url = doc_info['webViewLink']
            print(f"Found existing document: '{document_name}' (ID: {document_id}). Appending content.")

            # Get the current document structure to find the end index for appending
            document = docs_service.documents().get(documentId=document_id).execute()
            
            # Determine the end index of the document for insertion.
            # If the document body has content, we append at the end of the last element's 'endIndex'.
            # Otherwise (e.g., an empty document), we insert at index 1 (the beginning of the body).
            end_index = 1 # Default to beginning
            if 'body' in document and 'content' in document['body'] and document['body']['content']:
                last_element = document['body']['content'][-1]
                if 'endIndex' in last_element:
                    end_index = last_element['endIndex']
                # If for some reason 'endIndex' is not present in a structural element,
                # we fall back to inserting at index 1.

            # Prepare the batch update request to insert text.
            # We add a newline character before the new content to ensure it's on a new line.
            requests = [
                {
                    'insertText': {
                        'location': {
                            'index': end_index,
                        },
                        'text': '\n' + content_to_write
                    }
                }
            ]
            
            # Execute the batch update request
            docs_service.documents().batchUpdate(documentId=document_id, body={'requests': requests}).execute()
            print("Content appended successfully.")

        else:
            # --- Step 2b: If document is not found, create a new one and write content ---
            print(f"Document '{document_name}' not found. Creating a new document.")
            new_document_body = {
                'title': document_name
            }
            new_document = docs_service.documents().create(body=new_document_body).execute()
            document_id = new_document.get('documentId')
            # Construct the URL for the newly created document
            document_url = f"https://docs.google.com/document/d/{document_id}/edit"
            print(f"Created new document with ID: {document_id}")

            # Prepare the batch update request to insert the initial content
            requests = [
                {
                    'insertText': {
                        'location': {
                            'index': 1, # Index 1 is typically the beginning of the document body
                        },
                        'text': content_to_write
                    }
                }
            ]
            # Execute the batch update request for the new document
            docs_service.documents().batchUpdate(documentId=document_id, body={'requests': requests}).execute()
            print("Initial content written successfully.")

        print(f"Document URL: {document_url}")

    except HttpError as err:
        # Handle API-specific errors (e.g., permission denied)
        print(f"An API error occurred: {err}")
    except Exception as e:
        # Handle any other unexpected errors
        print(f"An unexpected error occurred: {e}")

# --- Example Usage ---
if __name__ == '__main__':
    # First call: creates the document if it doesn't exist, and writes the content.
    doc_name = "My Automated Test Document"
    content = "This is the initial content written by the Python script."
    write_to_google_doc(doc_name, content)

    print("\n--- Calling the function again to append ---")
    # Second call with the same document name: appends new content to the existing document.
    content_to_append = "This is additional content appended in a subsequent call. It should appear on a new line."
    write_to_google_doc(doc_name, content_to_append)

    print("\n--- Calling the function with a new document name ---")
    # Third call with a different name: creates a new document.
    new_doc_name = "Another Script-Generated Document"
    new_doc_content = "This is a brand new document created with different content."
    write_to_google_doc(new_doc_name, new_doc_content)
