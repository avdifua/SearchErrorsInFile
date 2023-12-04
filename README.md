# SearchErrorsInFile

## Overview
`SearchErrorsInFile` is a command-line utility designed to efficiently search for specific error patterns within log files. It allows users to define thresholds for critical and warning alert levels and analyze log entries within a specified time frame.

## Features
- Search for specific error messages or patterns within a log file.
- Set thresholds for critical and warning alert levels.
- Limit analysis to a specific time frame based on error occurrence.

## Prerequisites
- [Go](https://golang.org/dl/) (Version 1.x or higher)

## Installation
To install this tool, clone the repository and build using Go:

```bash
git clone https://github.com/avdifua/SearchErrorsInFile.git
cd SearchErrorsInFile
go build
```

## Usage
Execute the tool with the required flags:

`./searchErrors -p <path-to-log-file> -e "<error-pattern>" -c <critical-threshold> -w <warning-threshold> -t <time-frame-in-seconds>`

### Flags
- `-p`, `--path`      : Specify the full path to the log file.
- `-e`, `--error`     : Define the error string or pattern to search for.
- `-c`, `--critical`  : Set the number of errors to trigger a critical alert.
- `-w`, `--warning`   : Set the number of errors to trigger a warning alert.
- `-t`, `--time-point`: Time frame in seconds for searching errors.

### Example
`./searchErrors -p "/var/log/myapp.log" -e "Connection Error" -c 10 -w 5 -t 3600`

## Note
Ensure the time zone in the log file matches the local host's time zone for accurate time comparisons.

## Contributing
Contributions to this project are welcome. Please adhere to the code standards and guidelines.

