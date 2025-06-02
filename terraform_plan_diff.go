package comparison

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/pkg/errors"
)

// Static errors.
var (
	// ErrNoJSONOutput is returned when no JSON output is found in terraform show output.
	ErrNoJSONOutput = errors.New("no JSON output found in terraform show output")
)

// ComparePlansAndGenerateDiff compares two plan files and generates a diff.
func ComparePlansAndGenerateDiff(origPlanFileJSON, newPlanFileJSON string) (string, map[string]interface{}, bool, error) {
	// Parse the JSON
	var origPlan, newPlan map[string]interface{}
	err := json.Unmarshal([]byte(origPlanFileJSON), &origPlan)
	if err != nil {
		return "", nil, false, errors.Wrap(err, "error parsing original plan JSON")
	}

	err = json.Unmarshal([]byte(newPlanFileJSON), &newPlan)
	if err != nil {
		return "", nil, false, errors.Wrap(err, "error parsing new plan JSON")
	}

	log.Printf("Parsed both JSONs. Sorting maps now...")

	// Sort maps to ensure consistent ordering
	origPlan = sortMapKeys(origPlan)
	newPlan = sortMapKeys(newPlan)

	log.Printf("Sorted maps. Generating diff now...")

	// Generate the diff
	diff_string, diff_map, hasDiff := generatePlanDiff(origPlan, newPlan)

	// Print the diff
	if hasDiff {
		fmt.Fprintln(os.Stdout, "\nDiff Output")
		fmt.Fprintln(os.Stdout, "===========")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, diff_string)

		// Print the error message
		// u.PrintErrorMarkdown("", terrerrors.ErrPlanHasDiff, "")

		// Exit with code 2 to indicate that the plans are different
		// u.OsExit(2)

	} else {
		fmt.Fprintln(os.Stdout, "The planfiles are identical")
	}
	return diff_string, diff_map, hasDiff, nil
}

// extractJSONFromOutput extracts the JSON part from terraform show output.
func extractJSONFromOutput(output string) (string, error) {
	// Find the beginning of the JSON output (first '{' character)
	jsonStartIdx := strings.Index(output, "{")
	if jsonStartIdx == -1 {
		return "", ErrNoJSONOutput
	}

	// Extract just the JSON part
	jsonOutput := output[jsonStartIdx:]

	return jsonOutput, nil
}
