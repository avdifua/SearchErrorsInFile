package main

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var attrKeys struct {
	logFile        string
	error          string
	critical       int
	warning        int
	timePoint      int
	logFilePattern string
}

var semaphoreAttr struct {
	bufferSize     int
	maxGoroutines  int
	errorListMutex sync.Mutex
}

var sharedListofEroors struct {
	listOfErrors []string
}

var argsCli = &cobra.Command{
	Use:   "searchErrors",
	Short: "Search for specific errors in a log file",
	Long: `A command-line tool that scans a specified log file for certain error patterns or messages.
This tool allows setting thresholds for critical and warning alert levels based on the occurrence of these errors within a given time frame. It can handle both static and dynamically named log files.

For dynamically named log files, use the -l flag to specify a log file name pattern and the -p flag to specify the directory path. 
The pattern should include '2006-01-02' as a placeholder for the date, which the tool will replace with the current date in 'YYYY-MM-DD' format. 
For example, if the log files are named like 'ExampleLogFile-2023-01-02.log', use -l 'ExampleLogFile-2006-01-02.log'. The tool will automatically generate the correct file name for the current day.

Note: Ensure the time zone in the log file matches the local host's time zone for accurate time comparisons.`,
	Run: func(cmd *cobra.Command, args []string) {
		if attrKeys.warning > attrKeys.critical {
			fmt.Println("Warning number of errors must be less than critical number of errors")
			os.Exit(2)
		}
		if attrKeys.logFilePattern != "" {
			currentDate := time.Now().Format("2006-01-02")
			filename := strings.Replace(attrKeys.logFilePattern, "2006-01-02", currentDate, 1)
			attrKeys.logFile = filepath.Join(attrKeys.logFile, filename)
		}
	},
}

func init() {
	argsCli.PersistentFlags().StringVarP(&attrKeys.logFile, "path", "p", "", "Specify the full path to the log file to be analyzed")
	argsCli.PersistentFlags().StringVarP(&attrKeys.error, "error", "e", "", "Define the error string to search for in the log file")
	argsCli.PersistentFlags().IntVarP(&attrKeys.critical, "critical", "c", 0, "Set the threshold of errors to trigger a critical alert")
	argsCli.PersistentFlags().IntVarP(&attrKeys.warning, "warning", "w", 0, "Set the threshold of errors to trigger a warning alert")
	argsCli.PersistentFlags().IntVarP(&attrKeys.timePoint, "time-point", "t", 0, "Time frame in seconds to look back for errors from the current time")
	argsCli.PersistentFlags().StringVarP(&attrKeys.logFilePattern, "log-pattern", "l", "", "Pattern for the log file name with a date placeholder, e.g., 'SpamCop-2006-01-02.log'")

	argsCli.MarkFlagRequired("path")
	argsCli.MarkFlagRequired("error")
	argsCli.MarkFlagRequired("critical")
	argsCli.MarkFlagRequired("warning")
}

func exitCode() {
	fmt.Println(attrKeys.logFile)
	switch {
	case len(sharedListofEroors.listOfErrors) >= attrKeys.critical:
		fmt.Printf("CRITICAL: %d errors \"%s\" were found in the log file %s for the last %d sec!\n", len(sharedListofEroors.listOfErrors), attrKeys.error, attrKeys.logFile, attrKeys.timePoint)
		os.Exit(2)
	case len(sharedListofEroors.listOfErrors) >= attrKeys.warning:
		fmt.Printf("WARNING: %d errors \"%s\" were found in the log file %s for the last %d sec!\n", len(sharedListofEroors.listOfErrors), attrKeys.error, attrKeys.logFile, attrKeys.timePoint)
		os.Exit(1)
	default:
		fmt.Printf("OK: Errors not found in log file %s!\n", attrKeys.logFile)
		os.Exit(0)
	}
}

func processChunk(chunk []byte, sem chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() { <-sem }()
	lines := strings.Split(string(chunk), "\n")
	lowerErr := strings.ToLower(attrKeys.error)
	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, lowerErr) {
			result, err := getDateTimefromString(line)
			if err != nil {
				fmt.Println(err)
			}
			if int(result.Seconds()) <= attrKeys.timePoint {
				semaphoreAttr.errorListMutex.Lock()
				sharedListofEroors.listOfErrors = append(sharedListofEroors.listOfErrors, line)
				semaphoreAttr.errorListMutex.Unlock()
			}
		}
	}
}

func readFile() {
	semaphoreAttr.bufferSize = 64 * 1024
	semaphoreAttr.maxGoroutines = 5
	file, err := os.Open(attrKeys.logFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, semaphoreAttr.bufferSize)
	sem := make(chan struct{}, semaphoreAttr.maxGoroutines)
	var wg sync.WaitGroup
	for {
		chunk, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		wg.Add(1)
		sem <- struct{}{}
		go processChunk(chunk, sem, &wg)
	}
	wg.Wait()
}

func getDateTimefromString(logLine string) (time.Duration, error) {
	split := strings.Fields(logLine)
	if len(split) < 3 {
		return 0, fmt.Errorf("invalid log line format")
	}
	dateTimeStr := fmt.Sprintf("%s %s %s", split[0], split[1], split[2])
	currentYear := time.Now().Year()
	dateTimeStrWithYear := fmt.Sprintf("%d %s", currentYear, dateTimeStr)
	layoutWithYear := "2006 Jan 2 15:04:05"
	location := time.Local
	dateTime, err := time.ParseInLocation(layoutWithYear, dateTimeStrWithYear, location)
	if err != nil {
		return 0, fmt.Errorf("error parsing date: %w", err)
	}
	now := time.Now()
	diff := now.Sub(dateTime)

	return diff, nil
}

func main() {
	if err := argsCli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	readFile()
	exitCode()
}
