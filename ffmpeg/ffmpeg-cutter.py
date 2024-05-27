import subprocess

def generate_ffmpeg_commands(input_file, output_file):
    with open(input_file, 'r') as f:
        lines = f.readlines()

    m4a_file = lines[0].strip()
    timestamps = [line.strip() for line in lines[1:]]

    commands = []
    for i in range(len(timestamps) - 1):
        start_time = timestamps[i]
        end_time = timestamps[i + 1]

        command = f'ffmpeg -ss {start_time} -to {end_time} -i "{m4a_file}" -vn -acodec copy "{m4a_file[:-4]}_split{i}.m4a"'
        commands.append(command)

    with open(output_file, 'w') as f:
        for command in commands:
            f.write(command + '\n')

def run_ffmpeg_commands(command_file, log_file):
    with open(command_file, 'r') as f:
        commands = f.readlines()

    with open(log_file, 'w') as f:
        for command in commands:
            process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=True)
            stdout, _ = process.communicate()

            f.write(f'Command: {command}\n')
            f.write(f'Output: {stdout.decode()}\n')

generate_ffmpeg_commands('input.txt', 'cmd.txt')
run_ffmpeg_commands('cmd.txt', 'log.txt')
