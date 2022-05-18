package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/utils"
)

func statusIcon(ok bool) string {
	if ok {
		return "✓"
	}
	return "❌"
}

func logTestKeys(w io.Writer, testKeys string) error {
	colWidth := 24

	var out bytes.Buffer
	if err := json.Indent(&out, []byte(testKeys), "", " "); err != nil {
		return err
	}

	testKeysLines := strings.Split(string(out.Bytes()), "\n")
	if len(testKeysLines) > 1 {
		testKeysLines = testKeysLines[1 : len(testKeysLines)-1]
		testKeysLines[0] = "{" + testKeysLines[0][1:]
		testKeysLines[len(testKeysLines)-1] = testKeysLines[len(testKeysLines)-1] + "}"
	}
	for _, line := range testKeysLines {
		fmt.Fprintf(w, fmt.Sprintf("│ %s │\n",
			utils.RightPad(line, colWidth*2)))
	}
	return nil
}

func logMeasurementItem(w io.Writer, f log.Fields) error {
	colWidth := 24

	rID := f.Get("id").(int64)
	testName := f.Get("test_name").(string)

	// We currently don't use these fields in the view
	//testGroupName := f.Get("test_group_name").(string)
	//networkName := f.Get("network_name").(string)
	//asn := fmt.Sprintf("AS%d (%s)", f.Get("asn").(uint), f.Get("network_country_code").(string))
	testKeys := f.Get("test_keys").(string)

	isAnomaly := f.Get("is_anomaly").(bool)
	isFailed := f.Get("is_failed").(bool)
	isUploaded := f.Get("is_uploaded").(bool)
	url := f.Get("url").(string)
	urlCategoryCode := f.Get("url_category_code").(string)

	isFirst := f.Get("is_first").(bool)
	isLast := f.Get("is_last").(bool)
	if isFirst {
		fmt.Fprintf(w, "┏"+strings.Repeat("━", colWidth*2+2)+"┓\n")
	} else {
		fmt.Fprintf(w, "┢"+strings.Repeat("━", colWidth*2+2)+"┪\n")
	}

	anomalyStr := fmt.Sprintf("ok: %s", statusIcon(!isAnomaly))
	uploadStr := fmt.Sprintf("uploaded: %s", statusIcon(isUploaded))
	failureStr := fmt.Sprintf("success: %s", statusIcon(!isFailed))

	fmt.Fprintf(w, fmt.Sprintf("│ %s │\n",
		utils.RightPad(
			fmt.Sprintf("#%d", rID), colWidth*2)))

	if url != "" {
		fmt.Fprintf(w, fmt.Sprintf("│ %s │\n",
			utils.RightPad(
				fmt.Sprintf("%s (%s)", url, urlCategoryCode), colWidth*2)))
	}

	fmt.Fprintf(w, fmt.Sprintf("│ %s %s│\n",
		utils.RightPad(testName, colWidth),
		utils.RightPad(anomalyStr, colWidth)))

	fmt.Fprintf(w, fmt.Sprintf("│ %s %s│\n",
		utils.RightPad(failureStr, colWidth),
		utils.RightPad(uploadStr, colWidth)))

	if testKeys != "" {
		if err := logTestKeys(w, testKeys); err != nil {
			return err
		}
	}

	if isLast {
		fmt.Fprintf(w, "└┬────────────────────────────────────────────────┬┘\n")
	}
	return nil
}

func logMeasurementSummary(w io.Writer, f log.Fields) error {
	colWidth := 12

	totalCount := f.Get("total_count").(int64)
	anomalyCount := f.Get("anomaly_count").(int64)
	totalRuntime := f.Get("total_runtime").(float64)
	dataUp := f.Get("data_usage_up").(float64)
	dataDown := f.Get("data_usage_down").(float64)

	startTime := f.Get("start_time").(time.Time)

	asn := f.Get("asn").(uint)
	countryCode := f.Get("network_country_code").(string)
	networkName := f.Get("network_name").(string)

	fmt.Fprintf(w, " │ %s  │\n",
		utils.RightPad(startTime.Format(time.RFC822), (colWidth+3)*3),
	)
	fmt.Fprintf(w, " │ %s  │\n",
		utils.RightPad(fmt.Sprintf("AS%d, %s (%s)", asn, networkName, countryCode), (colWidth+3)*3),
	)
	fmt.Fprintf(w, " │ %s   %s   %s │\n",
		utils.RightPad(fmt.Sprintf("%.2fs", totalRuntime), colWidth),
		utils.RightPad(fmt.Sprintf("%d/%d anmls", anomalyCount, totalCount), colWidth),
		utils.RightPad(fmt.Sprintf("⬆ %s  ⬇ %s", formatSize(dataUp), formatSize(dataDown)), colWidth+4))
	fmt.Fprintf(w, " └────────────────────────────────────────────────┘\n")

	return nil
}

func logMeasurementJSON(w io.Writer, f log.Fields) error {
	m := f.Get("measurement_json").(map[string]interface{})

	json, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s", string(json))
	return nil
}
