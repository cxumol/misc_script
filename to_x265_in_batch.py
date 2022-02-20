import os
from glob import glob

input_path = './'
all_ext = ["flv", "mkv", "webm", "mp4", "rmvb", "MP4", "avi","wmv","ts"]

vid_other_ext = []
for ext in all_ext:
    vid_other_ext += glob(
        f"{input_path}/*.{ext}")
print(len(vid_other_ext))

# Remove coverted files from list
vids_to_conv = []
converted = glob(f"{input_path}/*.x265.mp4") + glob(f"{input_path}/*.hevc_nvenc.mp4") + glob(f"{input_path}/*.hevc_amf.mp4")
print(len(converted))
for vid_ext in vid_other_ext:
    is_converted = False
    for x265 in converted:
        if (x265[:-len(".x265.mp4")] == vid_ext[:-len(".mp4")]) or (x265[:-len(".hevc_nvenc.mp4")] == vid_ext[:-len(".mp4")]) or (x265[:-len(".hevc_amf.mp4")] == vid_ext[:-len(".mp4")]) or x265 == vid_ext:
            is_converted = True
            # print(x265[:-len(".x265.mp4")], vid_ext)
    if not is_converted:
        vids_to_conv.append(vid_ext)

# Show files to convert
print("To convert:\n")
i=0
for each in vids_to_conv:
    print(i, each)
    i+=1
print()


# -----------

import time

def to_mp4(filepath):
    newfilepath = filepath[:filepath.rfind('.')]+".x265"+".mp4"
    cmd = f'''
ffmpeg  -i "{filepath}" -c:a aac -c:v libx265 "{newfilepath}"
'''
    os.system(cmd)
    
for v in vids_to_conv[:]:
#     if '1591' in v: continue
    to_mp4(v)
    time.sleep(1)
    print('\n\n=============\n\n')
