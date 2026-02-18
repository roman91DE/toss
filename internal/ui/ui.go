package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/roman91DE/toss/internal/db"
)

func Confirm(prompt string) (bool, error) {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}

func PickEntry(entries []db.Entry) (db.Entry, error) {
	fmt.Println("Multiple matches found:")
	for i, e := range entries {
		fmt.Printf("  %d) %s  (%s)\n", i+1, e.OriginalPath, e.TossedAt.Format("2006-01-02 15:04"))
	}
	fmt.Printf("Choose [1-%d]: ", len(entries))

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return db.Entry{}, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(entries) {
		return db.Entry{}, fmt.Errorf("invalid selection")
	}
	return entries[n-1], nil
}

func PrintTable(entries []db.Entry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NO.\tTOSSED AT\tSIZE\tPATH")
	for i, e := range entries {
		name := e.OriginalPath
		if e.IsDir {
			name += " [dir]"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			i+1,
			e.TossedAt.Format("2006-01-02 15:04"),
			FormatSize(e.SizeBytes),
			name,
		)
	}
	w.Flush()
}

func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
