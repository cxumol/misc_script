#!/usr/bin/env python3
# python dir2plaintext.py --get *.cs *.xaml *.csproj --skip bin obj --output dir2plaintext.txt
import argparse
from pathlib import Path

def create_text_bundle(get_patterns, skip_dirs, output_file):
    """
    Finds files, skips specified directories, and bundles their content into a single text file.
    """
    root = Path.cwd()
    skip_set = set(skip_dirs)
    all_files = []

    print(f"Searching for patterns: {get_patterns}")
    print(f"Skipping directories: {skip_dirs}")
    print(f"Working in root: {root}")

    for pattern in get_patterns:
        all_files.extend(root.rglob(pattern))

    unique_files = sorted(list(set(all_files))) # dedupe
    
    print(f"Found {len(unique_files)} initial files.")

    # Filter out files from skipped directories
    filtered_files = []
    if skip_set:
        for file_path in unique_files:
            # Check if any part of the file's path is in the skip set
            if not skip_set.intersection(file_path.parts):
                filtered_files.append(file_path)
    else:
        filtered_files = unique_files
    
    print(f"Processing {len(filtered_files)} files after filtering.")

    try:
        with open(output_file, 'w', encoding='utf-8') as outfile:
            for file_path in filtered_files:
                try:
                    rel_path = file_path.relative_to(root).as_posix() # Use forward slashes for consistency
                    content = file_path.read_text(encoding='utf-8')
                    ext = file_path.suffix.lstrip('.').lower()

                    lang_map = {
                        'h': 'cpp',
                        'conf': 'yaml',
                        'csproj': 'xml',
                        'gradle': 'groovy',
                    }
                    lang = lang_map.get(ext, ext)

                    # Construct the markdown block
                    block = (
                        f"{rel_path}\n"
                        f"```{lang}\n"
                        f"{content}\n"
                        f"```\n\n" # Add extra newline for spacing between blocks
                    )
                    
                    outfile.write(block)
                except Exception as e:
                    print(f"---Skipping file {file_path} due to error: {e}")

        print(f"✅ Success! Content written to {output_file}")

    except IOError as e:
        print(f"❌ Error writing to output file: {e}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="A script to concatenate specified project files into a single text file for LLM context.",
        formatter_class=argparse.RawTextHelpFormatter
    )
    parser.add_argument(
        '--get',
        nargs='+',  # Accepts one or more arguments
        required=True,
        help="List of file patterns to include (e.g., *.cs *.xaml)"
    )
    parser.add_argument(
        '--skip',
        nargs='*',  # Accepts zero or more arguments
        default=[],
        help="List of directory names to exclude (e.g., bin obj)"
    )
    parser.add_argument(
        '--output',
        type=str,
        required=True,
        help="The path for the output text file."
    )

    args = parser.parse_args()
    create_text_bundle(args.get, args.skip, args.output)
