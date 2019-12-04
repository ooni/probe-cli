package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/utils"
)

func formatSpeed(speed float64) string {
	if speed < 1000 {
		return fmt.Sprintf("%.2f Kbit/s", speed)
	} else if speed < 1000*1000 {
		return fmt.Sprintf("%.2f Mbit/s", float32(speed)/1000)
	} else if speed < 1000*1000*1000 {
		return fmt.Sprintf("%.2f Gbit/s", float32(speed)/(1000*1000))
	}
	// WTF, you crazy?
	return fmt.Sprintf("%.2f Tbit/s", float32(speed)/(1000*1000*1000))
}

func formatSize(size float64) string {
	if size < 1024 {
		return fmt.Sprintf("%.1fK", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fM", size/1024.0)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1fG", size/(1024.0*1024.0))
	}
	// WTF, you crazy?
	return fmt.Sprintf("%.1fT", size/(1024*1024*1024))
}

var summarizers = map[string]func(uint64, uint64, string) []string{
	"websites": func(totalCount uint64, anomalyCount uint64, ss string) []string {
		return []string{
			fmt.Sprintf("%d tested", totalCount),
			fmt.Sprintf("%d blocked", anomalyCount),
			"",
		}
	},
	"performance": func(totalCount uint64, anomalyCount uint64, ss string) []string {
		var tk database.PerformanceTestKeys
		if err := json.Unmarshal([]byte(ss), &tk); err != nil {
			return nil
		}
		return []string{
			fmt.Sprintf("Download: %s", formatSpeed(tk.Download)),
			fmt.Sprintf("Upload: %s", formatSpeed(tk.Upload)),
			fmt.Sprintf("Ping: %.2fms", tk.Ping),
		}
	},
	"im": func(totalCount uint64, anomalyCount uint64, ss string) []string {
		return []string{
			fmt.Sprintf("%d tested", totalCount),
			fmt.Sprintf("%d blocked", anomalyCount),
			"",
		}
	},
	"middlebox": func(totalCount uint64, anomalyCount uint64, ss string) []string {
		return []string{
			fmt.Sprintf("Detected: %v", anomalyCount > 0),
			"",
			"",
		}
	},
}

func makeSummary(name string, totalCount uint64, anomalyCount uint64, ss string) []string {
	return summarizers[name](totalCount, anomalyCount, ss)
}

func logResultItem(w io.Writer, f log.Fields) error {
	colWidth := 24

	rID := f.Get("id").(int64)
	name := f.Get("name").(string)
	isDone := f.Get("is_done").(bool)
	startTime := f.Get("start_time").(time.Time)
	networkName := f.Get("network_name").(string)
	asn := fmt.Sprintf("AS%d (%s)", f.Get("asn").(uint), f.Get("network_country_code").(string))
	//runtime := f.Get("runtime").(float64)
	//dataUsageUp := f.Get("dataUsageUp").(int64)
	//dataUsageDown := f.Get("dataUsageDown").(int64)
	index := f.Get("index").(int)
	totalCount := f.Get("total_count").(int)
	if index == 0 {
		fmt.Fprintf(w, "┏"+strings.Repeat("━", colWidth*2+2)+"┓\n")
	} else {
		fmt.Fprintf(w, "┢"+strings.Repeat("━", colWidth*2+2)+"┪\n")
	}

	firstRow := utils.RightPad(fmt.Sprintf("#%d - %s", rID, startTime.Format(time.RFC822)), colWidth*2)
	fmt.Fprintf(w, "┃ "+firstRow+" ┃\n")
	fmt.Fprintf(w, "┡"+strings.Repeat("━", colWidth*2+2)+"┩\n")

	summary := makeSummary(name,
		f.Get("measurement_count").(uint64),
		f.Get("measurement_anomaly_count").(uint64),
		f.Get("test_keys").(string))

	fmt.Fprintf(w, fmt.Sprintf("│ %s %s│\n",
		utils.RightPad(name, colWidth),
		utils.RightPad(summary[0], colWidth)))
	fmt.Fprintf(w, fmt.Sprintf("│ %s %s│\n",
		utils.RightPad(networkName, colWidth),
		utils.RightPad(summary[1], colWidth)))
	fmt.Fprintf(w, fmt.Sprintf("│ %s %s│\n",
		utils.RightPad(asn, colWidth),
		utils.RightPad(summary[2], colWidth)))

	if index == totalCount-1 {
		if isDone == true {
			fmt.Fprintf(w, "└┬──────────────┬──────────────┬──────────────────┬┘\n")
		} else {
			// We want the incomplete section to not have a footer
			fmt.Fprintf(w, "└──────────────────────────────────────────────────┘\n")
		}
	}
	return nil
}

func logResultSummary(w io.Writer, f log.Fields) error {

	networks := f.Get("total_networks").(int64)
	tests := f.Get("total_tests").(int64)
	dataUp := f.Get("total_data_usage_up").(float64)
	dataDown := f.Get("total_data_usage_down").(float64)
	if tests == 0 {
		fmt.Fprintf(w, "No results\n")
		fmt.Fprintf(w, "Try running:\n")
		fmt.Fprintf(w, "  ooni run websites\n")
		return nil
	}
	//              └┬──────────────┬──────────────┬──────────────┬
	fmt.Fprintf(w, " │ %s │ %s │ %s │\n",
		utils.RightPad(fmt.Sprintf("%d tests", tests), 12),
		utils.RightPad(fmt.Sprintf("%d nets", networks), 12),
		utils.RightPad(fmt.Sprintf("⬆ %s  ⬇ %s", formatSize(dataUp), formatSize(dataDown)), 16))
	fmt.Fprintf(w, " └──────────────┴──────────────┴──────────────────┘\n")

	return nil
}
