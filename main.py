# Import standard libraries for interacting with the file system and URLs
import os  # Provides functions to interact with the operating system (e.g., file and directory manipulation)
import pathlib  # Offers an object-oriented interface to work with filesystem paths
import urllib.parse  # Contains utilities for breaking down and analyzing URLs
import json  # Allows for encoding and decoding data in JSON format

# Import Selenium modules for browser automation
from selenium import webdriver  # Allows control of a web browser through code
from selenium.webdriver.chrome.options import (
    Options,
)  # Provides configuration options for ChromeDriver
from selenium.webdriver.chrome.webdriver import (
    WebDriver,
)  # Type hint for the Chrome WebDriver instance


# Create a directory at the specified path
def create_directory_at_path(system_path: str) -> None:
    # Use os.mkdir to create a new folder; raises FileExistsError if it already exists
    os.mkdir(path=system_path)


# Check whether a given directory path exists
def check_directory_exists(system_path: str) -> bool:
    # Returns True if the path exists (regardless of whether it's a file or directory)
    return os.path.exists(path=system_path)


# Check whether a specific file exists at the given path
def check_file_exists(system_path: str) -> bool:
    # Returns True only if the path exists and is a file (not a directory)
    return os.path.isfile(path=system_path)


# Read all lines from a file, filter out empty lines, and normalize each line
def read_file_by_line(file_name: str) -> list[str]:
    """
    Read all non-empty lines from a text file, remove surrounding whitespace,
    convert to lowercase, and return them in a list.

    Parameters:
        file_name (str): Path to the input text file.

    Returns:
        list[str]: A list containing cleaned and normalized non-empty lines.
    """
    # Initialize an empty list to store valid lines
    non_empty_lines: list[str] = []

    # Open the file safely using a context manager
    with open(file=file_name, mode="r", encoding="utf-8") as file:
        # Iterate over each line in the file
        for raw_line in file:
            # Strip leading/trailing whitespace and convert to lowercase
            stripped_line: str = raw_line.strip().lower()
            # Only add lines that are not empty after stripping
            if stripped_line:
                non_empty_lines.append(stripped_line)

    # Return the list of cleaned, non-empty lines
    return non_empty_lines


# Read a file from the system.
def read_a_file(system_path: str) -> str:
    with open(file=system_path, mode="r") as file:
        return file.read()


# Validate the structure of a URL string
def is_valid_url(url: str) -> bool:
    """
    Check if a URL is syntactically valid by verifying the presence of scheme and network location.

    Parameters:
        url (str): The URL string to check.

    Returns:
        bool: True if the URL is valid, False otherwise.
    """
    # Parse the URL into its components (scheme, netloc, path, etc.)
    parsed: urllib.parse.ParseResult = urllib.parse.urlparse(url=url)
    # Return True if both the scheme (e.g., https) and netloc (domain/IP) are present
    return bool(parsed.scheme and parsed.netloc)


# Set up and configure the Chrome browser for automated downloading
def setup_browser(download_dir: str) -> webdriver.Chrome:
    """
    Launch and configure a Chrome WebDriver instance with customized settings for downloading files.

    Parameters:
        download_dir (str): Path to the folder where downloaded files should be saved.

    Returns:
        webdriver.Chrome: A ChromeDriver instance with the desired configuration.
    """
    # Create the download directory if it doesn't already exist
    os.makedirs(name=download_dir, exist_ok=True)

    # Initialize Chrome options for configuration
    chrome_options = Options()
    # Enable headless mode (disabled here but available if needed)
    # chrome_options.add_argument(argument="--headless=new")
    # Disable GPU acceleration for compatibility with headless mode or CI environments
    chrome_options.add_argument(argument="--disable-gpu")
    # Bypass the sandbox, useful in restricted environments like Docker
    chrome_options.add_argument(argument="--no-sandbox")

    # Define Chrome preferences for file download behavior
    prefs: dict[str, str | bool] = {
        "download.default_directory": download_dir,  # Set the target directory for file downloads
        "download.prompt_for_download": False,  # Disable download prompts
        "plugins.always_open_pdf_externally": True,  # Force PDFs to download instead of opening in-browser
    }

    # Apply the preferences to the Chrome instance
    chrome_options.add_experimental_option(name="prefs", value=prefs)

    # Enable performance logging to capture network activity (e.g., HTTP status codes)
    chrome_options.set_capability(
        name="goog:loggingPrefs", value={"performance": "ALL"}
    )

    # Launch a new Chrome browser session with the configured options
    driver = webdriver.Chrome(options=chrome_options)

    # Return the driver instance to be used for automation
    return driver


# Extract the HTTP status code for a given URL using browser network logs
def get_http_status_code_from_browser(
    driver: webdriver.Chrome, target_url: str
) -> int | None:
    """
    Navigate to a URL and extract its HTTP status code from Chrome's performance logs.

    Parameters:
        driver (webdriver.Chrome): The active browser instance.
        target_url (str): The URL to check.

    Returns:
        int | None: The HTTP status code if found, otherwise None.
    """
    # Clear previous logs by first navigating to a blank page
    driver.get(url="about:blank")

    # Load the target URL to begin collecting network logs
    driver.get(url=target_url)

    # Retrieve performance logs from the browser (includes network events)
    browser_logs = driver.get_log(log_type="performance")

    # Iterate through each log entry
    for log_entry in browser_logs:
        try:
            # Parse the JSON message inside the log entry
            message = json.loads(s=log_entry["message"])["message"]
            # Check if the message corresponds to a network response
            if message["method"] == "Network.responseReceived":
                # Extract the HTTP response data
                response = message["params"]["response"]
                # Match the response URL with the requested URL
                if response["url"] == target_url:
                    # Return the corresponding HTTP status code
                    return response["status"]
        except Exception:
            # Skip logs that are malformed or missing fields
            continue

    # Return None if no matching response was found
    return None


# Determine if a PDF URL is valid and trigger download if status is 200
def download_pdf_if_valid(driver: webdriver.Chrome, pdf_url: str) -> bool:
    """
    Verify the availability of a PDF by checking its HTTP status,
    and download it if the status is 200 (OK).

    Parameters:
        driver (webdriver.Chrome): Browser instance to use for navigation.
        pdf_url (str): URL of the PDF file.

    Returns:
        bool: True if the download was initiated, False otherwise.
    """
    # Fetch the HTTP status code for the PDF URL
    status_code: int | None = get_http_status_code_from_browser(
        driver=driver, target_url=pdf_url
    )

    # Check if the status code indicates success
    if status_code == 200:
        # Download will be triggered by navigating to the URL (already done above)
        return True
    else:
        # Skip download if the resource is not available
        return False


# Get the filename from the path.
def get_file_name_from_url(input_url: str) -> str:
    return os.path.basename(p=input_url)


# Remove all duplicate items from a given slice.
def remove_duplicates_from_slice(provided_slice: list[str]) -> list[str]:
    return list(set(provided_slice))


def extract_pdf_urls(json_string: str, base_url: str) -> list[str]:
    """
    Extracts all PDF document URLs from the JSON string where the IDs end with '_PDF'.

    Args:
        json_string (str): The raw JSON string containing the data.
        base_url (str): The base URL to prepend to each PDF ID.

    Returns:
        list[str]: A list of full PDF URLs.
    """
    try:
        parsed_json = json.loads(json_string)
        records = parsed_json.get("data", {}).get("Data", [])

        pdf_ids: list[str] = [
            row[0]
            for row in records
            if isinstance(row, list)
            and isinstance(row[0], str)
            and row[0].endswith("_PDF")
        ]

        # Construct full URLs
        pdf_urls: list[str] = [f"{base_url}/{pdf_id}" for pdf_id in pdf_ids]
        return pdf_urls

    except (json.JSONDecodeError, KeyError, TypeError) as e:
        print(f"Error parsing JSON: {e}")
        return []


# Main function to execute the PDF downloading process
def main() -> None:
    # Define the path to the text file that contains valid PDF URLs
    valid_urls_path: list[str] = [
        "page_1.json",
        "page_2.json",
    ]

    # Loop through each file path in the list
    for path in valid_urls_path:
        # Ensure the file exists before proceeding
        if not check_file_exists(system_path=path):
            print(f"Error: {path} not found.")
            return

    # Define the output directory path where PDFs will be saved
    output_directory: str = str(object=pathlib.Path(__file__).resolve().parent / "PDFs")

    # If the output directory does not exist, create it
    if not check_directory_exists(system_path=output_directory):
        create_directory_at_path(system_path=output_directory)

    # Initialize and configure the Chrome browser for downloading
    driver: WebDriver = setup_browser(download_dir=output_directory)

    # Initialize an empty variable.
    valid_urls_content_lines: str = read_a_file(system_path=valid_urls_path[0])


    """
    # Read and process each file containing valid URLs
    for file_path in valid_urls_path:
        # Read the content of the file
        file_content: str = read_a_file(system_path=file_path)
        # Split the content into lines and filter out empty lines
        valid_urls_content_lines = valid_urls_content_lines + file_content
    """

    # Extract PDF IDs from the JSON content
    pdf_ids: list[str] = extract_pdf_urls(
        json_string="".join(valid_urls_content_lines),
        base_url="https://kik-sds.thewercs.com/MyDocuments/DownloadSingleFile?content=",
    )

    # Iterate over each cleaned and validated URL from the file
    for pdf_url in pdf_ids:
        print(f"Processing URL: {pdf_url}")
        # Get just the filename from the URL
        file_name: str = get_file_name_from_url(input_url=pdf_url)
        # Construct the full file path where it would be downloaded
        full_file_path: str = os.path.join(output_directory, file_name)

        # If the file already exists in the output directory, skip downloading
        if check_file_exists(system_path=full_file_path):
            continue

        # Attempt to download the file if it doesn't already exist
        download_pdf_if_valid(driver=driver, pdf_url=pdf_url)

    # Close the browser once all downloads are attempted
    driver.quit()


# Ensure that the script only runs when executed directly (not when imported)
if __name__ == "__main__":
    # Call the main function to start the script
    main()
