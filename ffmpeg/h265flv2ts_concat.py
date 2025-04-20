import os
import glob
import sys
import subprocess # Needed for ffprobe to capture output
import collections # For defaultdict

# --- Configuration --- (Keep your existing settings)
TEMP_DIR = "r:/"
OUTPUT_FILENAME_PREFIX = "final_output_concat_"
OUTPUT_FILENAME_SUFFIX = ".ts"
FLV_PATTERN = "*.flv"
# --- End Configuration ---

def run_command(cmd):
    """Runs a command using os.system and checks for errors."""
    print(f"--- Running Command ---")
    print(cmd)
    print(f"-----------------------")
    # Flush output buffer to ensure messages appear before ffmpeg/ffprobe output
    sys.stdout.flush()
    return_code = os.system(cmd)
    success = (return_code == 0)
    if not success:
        print(f"\n!!! Error: Command failed with return code {return_code} !!!")
        print(f"Command: {cmd}")
    print(f"--- Command {'Finished' if success else 'Failed'} ---")
    return success # Return True on success, False on failure

def get_video_resolution(filepath):
    """
    Uses ffprobe to get the video resolution (WxH) of a file.
    Handles potential multi-line output from ffprobe.
    Returns resolution string 'WidthxHeight' or None if error/not found.
    """
    # Ensure ffprobe exists (basic check)
    try:
        # Use specific ffprobe path if needed, otherwise rely on PATH
        ffprobe_cmd = 'ffprobe'
        subprocess.run([ffprobe_cmd, '-version'], stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True, timeout=10) # Added timeout
    except FileNotFoundError:
        print("!!! Error: ffprobe command not found. Make sure FFmpeg (including ffprobe) is installed and in your system's PATH. !!!")
        return None
    except subprocess.TimeoutExpired:
        print("!!! Error: ffprobe version check timed out. ffprobe might be hanging. !!!")
        return None
    except subprocess.CalledProcessError:
         print("!!! Warning: ffprobe -version check failed. Proceeding, but ffprobe might not work correctly. !!!")
         # Decide if you want to return None here or try anyway
         # return None

    command = [
        ffprobe_cmd,
        '-v', 'error',
        '-select_streams', 'v:0',
        '-show_entries', 'stream=width,height',
        '-of', 'csv=s=x:p=0',
        filepath
    ]
    print(f"Probing resolution for: {os.path.basename(filepath)}")
    sys.stdout.flush()

    try:
        # Increased timeout for potentially larger files or slower disks
        result = subprocess.run(command, capture_output=True, text=True, check=True, encoding='utf-8', timeout=30)
        raw_output = result.stdout.strip()

        # --- Robust Parsing Logic ---
        lines = raw_output.splitlines() # Split into lines based on \n, \r, \r\n
        resolution = None
        for line in lines:
            potential_resolution = line.strip()
            if potential_resolution: # Find the first non-empty line
                resolution = potential_resolution
                break # Use the first non-empty line found
        # --- End Robust Parsing Logic ---

        if resolution and 'x' in resolution and all(part.isdigit() for part in resolution.split('x')):
             print(f"  -> Detected resolution: {resolution}")
             return resolution
        else:
             # Log the *raw* output for better debugging if it fails again
             print(f"  -> Warning: Could not parse valid WxH resolution from ffprobe output. Raw output was: '{raw_output}'. Skipping file for grouping.")
             return None

    except subprocess.CalledProcessError as e:
        error_message = e.stderr.strip() if e.stderr else "No stderr output"
        # Check if the error indicates no video stream
        if "Stream specifier 'v:0' in filtergraph description" in error_message and "matches no streams" in error_message:
             print(f"  -> Info: No video stream found in {os.path.basename(filepath)}. Skipping file for grouping.")
        else:
             print(f"  -> Error probing resolution for {os.path.basename(filepath)}: {error_message}")
        return None
    except subprocess.TimeoutExpired:
        print(f"  -> Error: ffprobe timed out while probing {os.path.basename(filepath)}. Skipping file for grouping.")
        return None
    except Exception as e:
        print(f"  -> Unexpected error running ffprobe for {os.path.basename(filepath)}: {e}")
        return None

# --- main() function remains the same as the previous version ---
# (Copy the main function from the previous response here, it doesn't need changes for this specific fix)

def main():
    # Ensure temp directory exists
    try:
        os.makedirs(TEMP_DIR, exist_ok=True)
        print(f"Ensured temporary directory exists: {TEMP_DIR}")
    except OSError as e:
        print(f"Error: Could not create temporary directory {TEMP_DIR}: {e}")
        print("Please ensure the drive/path is valid and writable.")
        return

    # Find FLV files
    flv_files = glob.glob(FLV_PATTERN)
    if not flv_files:
        print(f"No FLV files found matching '{FLV_PATTERN}' in the current directory.")
        return

    print(f"\nFound {len(flv_files)} FLV files.")

    # Get creation times and sort
    file_info = []
    print("Getting file creation times...")
    if sys.platform != 'win32':
         print("Note: On non-Windows systems, os.path.getctime() might reflect last metadata change time, not strict creation time.")

    for flv_file in flv_files:
        try:
            abs_flv_path = os.path.abspath(flv_file)
            creation_time = os.path.getctime(abs_flv_path)
            file_info.append((creation_time, abs_flv_path))
        except OSError as e:
            print(f"Warning: Could not get stats for {flv_file}: {e}. Skipping.")

    file_info.sort(key=lambda item: item[0])

    print("\nFiles sorted by creation date (oldest first):")
    for _, filepath in file_info:
        print(f"- {os.path.basename(filepath)}")

    # Dictionary to hold intermediate TS files grouped by resolution
    resolution_groups = collections.defaultdict(list)
    all_intermediate_ts_files = []
    conversion_errors = False

    try:
        print("\n--- Starting FLV to TS Conversion & Resolution Probing ---")
        for i, (_, flv_path) in enumerate(file_info):
            base_name = os.path.splitext(os.path.basename(flv_path))[0]
            ts_filename = f"temp_{i:04d}_{base_name}.ts"
            ts_path = os.path.join(TEMP_DIR, ts_filename)
            ts_path_ffmpeg_list = ts_path.replace('\\', '/')

            print(f"\n[{i+1}/{len(file_info)}] Converting '{os.path.basename(flv_path)}' to '{ts_path}'...")

            # Conversion command (NO -bsf:v h264_mp4toannexb)
            convert_cmd = f'ffmpeg -i "{flv_path}" -c copy -f mpegts "{ts_path}"'

            if run_command(convert_cmd):
                # Conversion succeeded, now probe resolution
                resolution = get_video_resolution(ts_path) # Calls the updated function
                if resolution:
                    resolution_groups[resolution].append(ts_path_ffmpeg_list)
                    all_intermediate_ts_files.append(ts_path)
                    print(f"  -> Added to group: {resolution}")
                else:
                    print(f"  -> Warning: Could not determine resolution for {ts_filename}. It will not be concatenated.")
                    all_intermediate_ts_files.append(ts_path)
            else:
                print(f"!!! Conversion failed for {flv_path}. Skipping this file. !!!")
                conversion_errors = True


        # --- Concatenation Phase (Group by Group) ---
        print("\n--- Starting TS Concatenation (Grouped by Resolution) ---")

        if not resolution_groups:
            print("No files were successfully converted and grouped by resolution. Nothing to concatenate.")
            # If there were conversion errors, mention it
            if conversion_errors:
                print("Note: There were also errors during the initial FLV to TS conversion phase.")
        else:
            concatenation_attempts = 0
            concatenation_successes = 0

            for resolution, ts_files_in_group in resolution_groups.items():
                if len(ts_files_in_group) > 1:
                    concatenation_attempts += 1
                    print(f"\nConcatenating {len(ts_files_in_group)} files for resolution: {resolution}")

                    concat_input_string = "concat:" + "|".join(ts_files_in_group)
                    output_file = f"{OUTPUT_FILENAME_PREFIX}{resolution}{OUTPUT_FILENAME_SUFFIX}"
                    output_file = os.path.join(os.getcwd(), output_file)

                    concat_cmd = f'ffmpeg -i "{concat_input_string}" -c copy "{output_file}"'

                    if run_command(concat_cmd):
                        print(f"Successfully concatenated group {resolution} into '{os.path.basename(output_file)}'")
                        concatenation_successes += 1
                    else:
                        print(f"!!! Concatenation failed for group {resolution}. !!!")
                elif len(ts_files_in_group) == 1:
                    print(f"\nSkipping concatenation for resolution {resolution}: Only one file in this group.")

            print(f"\nConcatenation Summary: Attempted={concatenation_attempts}, Succeeded={concatenation_successes}")

        if conversion_errors and not resolution_groups: # Case where ONLY conversion errors happened
             print("\nNote: There were errors during the initial FLV to TS conversion phase for one or more files.")


    finally:
        # --- Cleanup ---
        print("\n--- Cleaning up temporary files ---")
        cleaned_count = 0
        if not all_intermediate_ts_files:
            print("No temporary files were created or marked for cleanup.")
        else:
            for ts_file_to_remove in all_intermediate_ts_files:
                if os.path.exists(ts_file_to_remove):
                    try:
                        os.remove(ts_file_to_remove)
                        cleaned_count += 1
                    except OSError as e:
                        print(f"Warning: Could not remove temporary file {ts_file_to_remove}: {e}")
                # else: pass # Don't report non-existent files here
            if cleaned_count > 0:
                print(f"Cleaned up {cleaned_count} temporary TS files from {TEMP_DIR}.")
            elif all_intermediate_ts_files: # If list wasn't empty but nothing removed
                print("Attempted cleanup, but no temporary files were found to remove (or removal failed).")


if __name__ == "__main__":
    main()
    print("\nScript finished.")
