package comparison

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/pkg/errors"
)

// Static errors.
var (
	// ErrNoJSONOutput is returned when no JSON output is found in terraform show output.
	ErrNoJSONOutput = errors.New("no JSON output found in terraform show output")
)

// TerraformPlanDiff represents the plan-diff command implementation.
// func TerraformPlanDiff(atmosConfig *schema.AtmosConfiguration, info *schema.ConfigAndStacksInfo) error {
func TerraformPlanDiff(tfplanpathOld string, tfplanpathNew string) (string, map[string]interface{}, bool, error) {
	// // Extract flags and setup paths
	// origPlanFile, newPlanFile, err := parsePlanDiffFlags(info.AdditionalArgsAndFlags)
	// if err != nil {
	// 	return err
	// }

	// Create a temporary directory for all temporary files
	tmpDir, err := os.MkdirTemp("", "terraform-plan-diff")
	if err != nil {
		return "", nil, false, errors.Wrap(err, "error creating temporary directory")
	}
	defer os.RemoveAll(tmpDir)

	//check if files exist
	if _, err := os.Stat(tfplanpathOld); os.IsNotExist(err) {
		return "", nil, false, errors.New("old plan file does not exist")
	}
	if _, err := os.Stat(tfplanpathNew); os.IsNotExist(err) {
		return "", nil, false, errors.New("new plan file does not exist")
	}
	origPlanFile := tfplanpathOld
	newPlanFile := tfplanpathNew

	// Compare the plans and generate diff

	return comparePlansAndGenerateDiff(origPlanFile, newPlanFile)
}

// comparePlansAndGenerateDiff compares two plan files and generates a diff.
func comparePlansAndGenerateDiff(origPlanFile, newPlanFile string) (string, map[string]interface{}, bool, error) {

	log.Printf("Getting JSON for original plan...")
	// Get the JSON representation of the original plan
	origPlanJSON, err := getTerraformPlanJSON(origPlanFile)
	if err != nil {
		return "", nil, false, errors.Wrap(err, "error getting JSON for original plan")
	}

	log.Printf("Getting JSON for new plan...")
	// Get the JSON representation of the new plan
	newPlanJSON, err := getTerraformPlanJSON(newPlanFile)
	if err != nil {
		return "", nil, false, errors.Wrap(err, "error getting JSON for new plan")
	}

	log.Printf("Got both JSONs. Parsing them now...")

	// Parse the JSON
	var origPlan, newPlan map[string]interface{}
	err = json.Unmarshal([]byte(origPlanJSON), &origPlan)
	if err != nil {
		return "", nil, false, errors.Wrap(err, "error parsing original plan JSON")
	}

	err = json.Unmarshal([]byte(newPlanJSON), &newPlan)
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

// getTerraformPlanJSON gets the JSON representation of a terraform plan.
func getTerraformPlanJSON(planFile string) (string, error) {

	// Run terraform show and capture output
	log.Printf("Running terraform show...")
	output, err := runTerraformShow(planFile)
	if err != nil {
		return "", err
	}

	log.Printf("Extracting JSON from output...")
	// Extract JSON from output
	return extractJSONFromOutput(output)
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


// runTerraformShow runs the terraform show command and captures its output.
func runTerraformShow(planFile string) (string, error) {
	cmd := exec.Command("terraform", "show", "-json", path.Base(planFile))
	cmd.Dir = path.Dir(planFile)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running terraform show: %w", err)
	}

	return string(output), nil
}
