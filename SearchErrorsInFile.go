package main

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

var attrKeys struct {
	logFile   string
	error     string
	criticl   int
	warning   int
	timePoint int
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
        It allows setting thresholds for critical and warning alert levels based on the occurrence of these errors within a given time frame.
        Note: Ensure the time zone in the log file matches the local host's time zone for accurate time comparisons.`,
	Run: func(cmd *cobra.Command, args []string) {
		if attrKeys.warning > attrKeys.criticl {
			fmt.Println("Warning number of errors must be less than critical number of errors")
			os.Exit(2)
		}
	},
}

func init() {
	argsCli.PersistentFlags().StringVarP(&attrKeys.logFile, "path", "p", "", "Specify the full path to the log file to be analyzed")
	argsCli.PersistentFlags().StringVarP(&attrKeys.error, "error", "e", "", "Define the error string to search for in the log file")
	argsCli.PersistentFlags().IntVarP(&attrKeys.criticl, "critical", "c", 0, "Set the threshold of errors to trigger a critical alert")
	argsCli.PersistentFlags().IntVarP(&attrKeys.warning, "warning", "w", 0, "Set the threshold of errors to trigger a warning alert")
	argsCli.PersistentFlags().IntVarP(&attrKeys.timePoint, "time-point", "t", 0, "Time frame in seconds to look back for errors from the current time")

	argsCli.MarkFlagRequired("path")
	argsCli.MarkFlagRequired("error")
	argsCli.MarkFlagRequired("critical")
	argsCli.MarkFlagRequired("warning")

}

func exitCode() {
	switch {
	case len(sharedListofEroors.listOfErrors) >= attrKeys.criticl:
		fmt.Printf("CRITICAL: %d errors \"%s\" were found in the log file for the last %d sec!\n", len(sharedListofEroors.listOfErrors), attrKeys.error, attrKeys.timePoint)
		os.Exit(2)
	case len(sharedListofEroors.listOfErrors) >= attrKeys.warning:
		fmt.Printf("WARNING: %d errors \"%s\" were found in the log file for the last %d sec!\n", len(sharedListofEroors.listOfErrors), attrKeys.error, attrKeys.timePoint)
		os.Exit(1)
	default:
		fmt.Printf("OK: Errors not found or less than thresholds CRITICAL: %d or WARNING: %d\nNumber of detected errors: %d\n", attrKeys.criticl, attrKeys.warning, len(sharedListofEroors.listOfErrors))
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
