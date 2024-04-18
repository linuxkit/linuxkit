import tarfile
import sys

def list_pax_headers(archive_path):
    # Open the tar archive
    try:
        with tarfile.open(archive_path, 'r') as tar:
            # Iterate over each member in the tar archive
            for member in tar.getmembers():
                # ignore the root directory, which just exists
                if member.name == '.':
                    continue
                # Check if there are any PAX headers
                if member.pax_headers:
                    # Check for the specific PAX header
                    if 'LINUXKIT.source' not in member.pax_headers:
                        print(f"File: {member.name} is missing LINUXKIT.source PAX Header.")
                    if 'LINUXKIT.location' not in member.pax_headers:
                        print(f"File: {member.name} is missing LINUXKIT.source PAX Header.")
                else:
                    print(f"File: {member.name} has No PAX Headers.")
    except Exception as e:
        print("Failed to read tar archive:", e)
        sys.exit(1)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python list_pax_headers.py <archive.tar>")
        sys.exit(1)
    archive_filename = sys.argv[1]
    list_pax_headers(archive_filename)
