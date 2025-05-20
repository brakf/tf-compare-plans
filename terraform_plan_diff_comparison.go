package comparison

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// sortMapKeys recursively sorts map keys for consistent comparison.
func sortMapKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Get all keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	// Sort keys
	sort.Strings(keys)

	// Process each key in sorted order
	for _, k := range keys {
		v := m[k]
		result[k] = processValue(v)
	}

	return result
}

// processValue recursively processes a value, sorting map keys and handling slices.
func processValue(v interface{}) interface{} {
	// Handle maps
	if nestedMap, ok := v.(map[string]interface{}); ok {
		return sortMapKeys(nestedMap)
	}

	// Handle slices
	if nestedSlice, ok := v.([]interface{}); ok {
		processedSlice := make([]interface{}, len(nestedSlice))
		for i, item := range nestedSlice {
			processedSlice[i] = processValue(item)
		}
		return processedSlice
	}

	// Return unchanged for other types
	return v
}

// generatePlanDiff generates a diff between two terraform plans.
func generatePlanDiff(origPlan, newPlan map[string]interface{}) (string, map[string]interface{}, bool) {
	var diff strings.Builder
	hasDiff := false
	diffMap := make(map[string]interface{})

	// Compare variables
	if varsDiff, varsMap, varsHasDiff := compareVariables(origPlan, newPlan); varsHasDiff {
		hasDiff = true
		diff.WriteString(varsDiff)
		diffMap["variables"] = varsMap
	}

	// Compare resources
	if resourcesDiff, resourcesMap, resourcesHasDiff := compareResourceSections(origPlan, newPlan); resourcesHasDiff {
		hasDiff = true
		diff.WriteString(resourcesDiff)
		diffMap["resources"] = resourcesMap
	}

	// Compare outputs
	if outputsDiff, outputsMap, outputsHasDiff := compareOutputSections(origPlan, newPlan); outputsHasDiff {
		hasDiff = true
		diff.WriteString(outputsDiff)
		diffMap["outputs"] = outputsMap
	}

	return diff.String(), diffMap, hasDiff
}

// compareVariables compares variables between two plans and returns the diff.
func compareVariables(origPlan, newPlan map[string]interface{}) (string, map[string]interface{}, bool) {
	origVars, newVars := getVariables(origPlan), getVariables(newPlan)
	if reflect.DeepEqual(origVars, newVars) {
		return "", nil, false
	}

	diffMap := make(map[string]interface{})
	added := make([]map[string]interface{}, 0)
	removed := make([]map[string]interface{}, 0)
	changed := make([]map[string]interface{}, 0)

	var diff strings.Builder
	diff.WriteString("Variables:\n")
	diff.WriteString("----------\n")

	// Find added variables
	for k, v := range newVars {
		if _, exists := origVars[k]; !exists {
			diff.WriteString(fmt.Sprintf("+ %s: %v\n", k, formatValue(v)))
			added = append(added, map[string]interface{}{
				"name":  k,
				"value": v,
			})
		}
	}

	// Find removed variables
	for k, v := range origVars {
		if _, exists := newVars[k]; !exists {
			diff.WriteString(fmt.Sprintf("- %s: %v\n", k, formatValue(v)))
			removed = append(removed, map[string]interface{}{
				"name":  k,
				"value": v,
			})
		}
	}

	// Find changed variables
	for k, origV := range origVars {
		if newV, exists := newVars[k]; exists && !reflect.DeepEqual(origV, newV) {
			diff.WriteString(fmt.Sprintf("~ %s: %v => %v\n", k, formatValue(origV), formatValue(newV)))
			changed = append(changed, map[string]interface{}{
				"name": k,
				"old":  origV,
				"new":  newV,
			})
		}
	}

	diff.WriteString("\n")

	diffMap["added"] = added
	diffMap["removed"] = removed
	diffMap["changed"] = changed

	return diff.String(), diffMap, true
}

// compareResourceSections compares resource sections between two plans and returns the diff.
func compareResourceSections(origPlan, newPlan map[string]interface{}) (string, map[string]interface{}, bool) {
	origResources, newResources := getResources(origPlan), getResources(newPlan)
	if reflect.DeepEqual(origResources, newResources) {
		return "", nil, false
	}

	var diff strings.Builder
	diff.WriteString("Resources:\n")
	diff.WriteString("-----------\n")
	diff.WriteString("\n")

	resourceDiff, resourceDiffMap := compareResources(origResources, newResources)
	diff.WriteString(resourceDiff)
	diff.WriteString("\n")

	return diff.String(), resourceDiffMap, true
}

// compareOutputSections compares output sections between two plans and returns the diff.
func compareOutputSections(origPlan, newPlan map[string]interface{}) (string, map[string]interface{}, bool) {
	origOutputs, newOutputs := getOutputs(origPlan), getOutputs(newPlan)
	if reflect.DeepEqual(origOutputs, newOutputs) {
		return "", nil, false
	}

	var diff strings.Builder
	diff.WriteString("Outputs:\n")
	diff.WriteString("--------\n")

	outputDiff, outputDiffMap := compareOutputs(origOutputs, newOutputs)
	diff.WriteString(outputDiff)

	return diff.String(), outputDiffMap, true
}

// processPriorStateResources extracts resources from prior_state.
func processPriorStateResources(plan map[string]interface{}, result map[string]interface{}) {
	priorState, ok := plan["prior_state"].(map[string]interface{})
	if !ok {
		return
	}

	values, ok := priorState["values"].(map[string]interface{})
	if !ok {
		return
	}

	rootModule, ok := values["root_module"].(map[string]interface{})
	if !ok {
		return
	}

	processRootModuleResources(rootModule, result)
}

// processPlannedValuesResources extracts resources from planned_values.
func processPlannedValuesResources(plan map[string]interface{}, result map[string]interface{}) {
	plannedValues, ok := plan["planned_values"].(map[string]interface{})
	if !ok {
		return
	}

	rootModule, ok := plannedValues["root_module"].(map[string]interface{})
	if !ok {
		return
	}

	processRootModuleResources(rootModule, result)
}

// processRootModuleResources processes resources from a root_module.
func processRootModuleResources(rootModule map[string]interface{}, result map[string]interface{}) {
	resources, ok := rootModule["resources"].([]interface{})
	if !ok {
		return
	}

	for _, res := range resources {
		resMap, ok := res.(map[string]interface{})
		if !ok {
			continue
		}

		addressVal, ok := resMap["address"]
		if !ok {
			continue
		}

		address, ok := addressVal.(string)
		if !ok {
			continue
		}

		result[address] = resMap
	}
}

// processResourceChanges extracts resources from resource_changes.
func processResourceChanges(plan map[string]interface{}, result map[string]interface{}) {
	resourceChanges, ok := plan["resource_changes"].([]interface{})
	if !ok {
		return
	}

	for _, change := range resourceChanges {
		changeMap, ok := change.(map[string]interface{})
		if !ok {
			continue
		}

		addressVal, ok := changeMap["address"]
		if !ok {
			continue
		}

		address, ok := addressVal.(string)
		if !ok {
			continue
		}

		result[address] = changeMap
	}
}

// getVariables extracts variables from a terraform plan.
func getVariables(plan map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	if vars, ok := plan["variables"].(map[string]interface{}); ok {
		for k, v := range vars {
			if varMap, ok := v.(map[string]interface{}); ok {
				if value, exists := varMap["value"]; exists {
					result[k] = value
				}
			}
		}
	}

	return result
}

// getResources extracts resources from a terraform plan.
func getResources(plan map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Process resources from different sections
	processPriorStateResources(plan, result)
	processPlannedValuesResources(plan, result)
	processResourceChanges(plan, result)

	return result
}

// getOutputs extracts outputs from a terraform plan.
func getOutputs(plan map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Check planned_values for outputs
	if plannedValues, ok := plan["planned_values"].(map[string]interface{}); ok {
		if outputs, ok := plannedValues["outputs"].(map[string]interface{}); ok {
			for k, v := range outputs {
				result[k] = v
			}
		}
	}

	// Check output_changes
	if outputChanges, ok := plan["output_changes"].(map[string]interface{}); ok {
		for k, v := range outputChanges {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result
}

// compareOutputs compares outputs between two terraform plans.
func compareOutputs(origOutputs, newOutputs map[string]interface{}) (string, map[string]interface{}) {
	var diff strings.Builder
	diffMap := make(map[string]interface{})
	added := make([]map[string]interface{}, 0)
	removed := make([]map[string]interface{}, 0)
	changed := make([]map[string]interface{}, 0)

	// Find added outputs
	for k, v := range newOutputs {
		if _, exists := origOutputs[k]; !exists {
			diff.WriteString(fmt.Sprintf("+ %s: %v\n", k, formatValue(v)))
			added = append(added, map[string]interface{}{
				"name":  k,
				"value": v,
			})
		}
	}

	// Find removed outputs
	for k, v := range origOutputs {
		if _, exists := newOutputs[k]; !exists {
			diff.WriteString(fmt.Sprintf("- %s: %v\n", k, formatValue(v)))
			removed = append(removed, map[string]interface{}{
				"name":  k,
				"value": v,
			})
		}
	}

	// Find changed outputs
	for k, origV := range origOutputs {
		if newV, exists := newOutputs[k]; exists && !reflect.DeepEqual(origV, newV) {
			diff.WriteString(formatOutputChange(k, origV, newV))
			changed = append(changed, map[string]interface{}{
				"name": k,
				"old":  origV,
				"new":  newV,
			})
		}
	}

	diffMap["added"] = added
	diffMap["removed"] = removed
	diffMap["changed"] = changed

	return diff.String(), diffMap
}

// compareResources compares resources between two terraform plans.
func compareResources(origResources, newResources map[string]interface{}) (string, map[string]interface{}) {
	var diff strings.Builder
	diffMap := make(map[string]interface{})

	// Process resource additions and removals
	added, removed := processResourceAdditionsAndRemovals(&diff, origResources, newResources)
	diffMap["added"] = added
	diffMap["removed"] = removed

	// Process resource changes
	changed := processChangedResources(&diff, origResources, newResources)
	diffMap["changed"] = changed

	return diff.String(), diffMap
}

// processResourceAdditionsAndRemovals adds information about added and removed resources to the diff.
func processResourceAdditionsAndRemovals(diff *strings.Builder, origResources, newResources map[string]interface{}) ([]map[string]interface{}, []map[string]interface{}) {
	added := make([]map[string]interface{}, 0)
	removed := make([]map[string]interface{}, 0)

	// Find added resources
	for k, v := range newResources {
		if _, exists := origResources[k]; !exists {
			diff.WriteString(fmt.Sprintf("+ %s\n", k))
			added = append(added, map[string]interface{}{
				"address": k,
				"value":   v,
			})
		}
	}

	// Find removed resources
	for k, v := range origResources {
		if _, exists := newResources[k]; !exists {
			diff.WriteString(fmt.Sprintf("- %s\n", k))
			removed = append(removed, map[string]interface{}{
				"address": k,
				"value":   v,
			})
		}
	}

	return added, removed
}

// processChangedResources processes resources that exist in both but have changes.
func processChangedResources(diff *strings.Builder, origResources, newResources map[string]interface{}) []map[string]interface{} {
	changed := make([]map[string]interface{}, 0)

	for k, origV := range origResources {
		newV, exists := newResources[k]
		if !exists || reflect.DeepEqual(origV, newV) {
			continue
		}

		diff.WriteString(fmt.Sprintf("%s\n", k))

		// Compare resource attributes
		origAttrs := getResourceAttributes(origV)
		newAttrs := getResourceAttributes(newV)

		// Process attribute differences
		attrChanges := processAttributeDifferences(diff, origAttrs, newAttrs)

		changed = append(changed, map[string]interface{}{
			"address":    k,
			"attributes": attrChanges,
			// "old":        origV,
			// "new":        newV,
		})
	}

	return changed
}

// processAttributeDifferences handles comparing and generating diff for resource attributes.
func processAttributeDifferences(diff *strings.Builder, origAttrs, newAttrs map[string]interface{}) map[string]interface{} {
	// Important attributes to always show first if they exist
	priorityAttrs := []string{"id", "url", "content"}

	// Attributes to skip in the diff to keep it clean
	skipAttrs := map[string]bool{
		"response_body_base64": true,
		"content_base64sha256": true,
		"content_base64sha512": true,
		"content_md5":          true,
		"content_sha1":         true,
		"content_sha256":       true,
		"content_sha512":       true,
	}

	attrChanges := make(map[string]interface{})
	added := make([]map[string]interface{}, 0)
	removed := make([]map[string]interface{}, 0)
	changed := make([]map[string]interface{}, 0)

	// Process priority attributes first
	processPriorityAttributes(diff, origAttrs, newAttrs, priorityAttrs, &added, &removed, &changed)

	// Process other attribute changes (not priority, not skipped)
	processRegularAttributeChanges(diff, origAttrs, newAttrs, priorityAttrs, skipAttrs, &added, &removed, &changed)

	// Find added attributes (that weren't in the priority list)
	processAddedAttributes(diff, origAttrs, newAttrs, priorityAttrs, skipAttrs, &added)

	attrChanges["added"] = added
	attrChanges["removed"] = removed
	attrChanges["changed"] = changed

	return attrChanges
}

// processPriorityAttributes handles high-priority attributes that should be shown first.
func processPriorityAttributes(diff *strings.Builder, origAttrs, newAttrs map[string]interface{}, priorityAttrs []string, added, removed, changed *[]map[string]interface{}) {
	for _, attrK := range priorityAttrs {
		origAttrV, origExists := origAttrs[attrK]
		newAttrV, newExists := newAttrs[attrK]

		switch {
		case origExists && newExists && !reflect.DeepEqual(origAttrV, newAttrV):
			printAttributeDiff(diff, attrK, origAttrV, newAttrV)
			*changed = append(*changed, map[string]interface{}{
				"name": attrK,
				"old":  origAttrV,
				"new":  newAttrV,
			})
		case origExists && !newExists:
			diff.WriteString(fmt.Sprintf("  - %s: %v\n", attrK, formatValue(origAttrV)))
			*removed = append(*removed, map[string]interface{}{
				"name":  attrK,
				"value": origAttrV,
			})
		case !origExists && newExists:
			diff.WriteString(fmt.Sprintf("  + %s: %v\n", attrK, formatValue(newAttrV)))
			*added = append(*added, map[string]interface{}{
				"name":  attrK,
				"value": newAttrV,
			})
		}
	}
}

// processRegularAttributeChanges handles changed and removed attributes.
func processRegularAttributeChanges(diff *strings.Builder, origAttrs, newAttrs map[string]interface{}, priorityAttrs []string, skipAttrs map[string]bool, added, removed, changed *[]map[string]interface{}) {
	for attrK, origAttrV := range origAttrs {
		// Skip priority attributes (already processed) and attributes in the skip list
		if contains(priorityAttrs, attrK) || skipAttrs[attrK] {
			continue
		}

		if newAttrV, exists := newAttrs[attrK]; exists && !reflect.DeepEqual(origAttrV, newAttrV) {
			printAttributeDiff(diff, attrK, origAttrV, newAttrV)
			*changed = append(*changed, map[string]interface{}{
				"name": attrK,
				"old":  origAttrV,
				"new":  newAttrV,
			})
		} else if !exists {
			diff.WriteString(fmt.Sprintf("  - %s: %v\n", attrK, formatValue(origAttrV)))
			*removed = append(*removed, map[string]interface{}{
				"name":  attrK,
				"value": origAttrV,
			})
		}
	}
}

// processAddedAttributes handles new attributes that didn't exist before.
func processAddedAttributes(diff *strings.Builder, origAttrs, newAttrs map[string]interface{}, priorityAttrs []string, skipAttrs map[string]bool, added *[]map[string]interface{}) {
	for attrK, newAttrV := range newAttrs {
		if _, exists := origAttrs[attrK]; !exists && !contains(priorityAttrs, attrK) && !skipAttrs[attrK] {
			diff.WriteString(fmt.Sprintf("  + %s: %v\n", attrK, formatValue(newAttrV)))
			*added = append(*added, map[string]interface{}{
				"name":  attrK,
				"value": newAttrV,
			})
		}
	}
}

// getResourceAttributes extracts attributes from a resource.
func getResourceAttributes(resource interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	resMap, ok := resource.(map[string]interface{})
	if !ok {
		return result
	}

	// Extract values from the "values" field
	extractValuesField(resMap, result)

	// Extract values from the "change.after" field
	extractChangeAfterField(resMap, result)

	return result
}

// extractValuesField extracts attributes from the "values" field of a resource.
func extractValuesField(resMap map[string]interface{}, result map[string]interface{}) {
	values, ok := resMap["values"].(map[string]interface{})
	if !ok {
		return
	}

	for k, v := range values {
		result[k] = v
	}
}

// extractChangeAfterField extracts attributes from the "change.after" field of a resource.
func extractChangeAfterField(resMap map[string]interface{}, result map[string]interface{}) {
	change, ok := resMap["change"].(map[string]interface{})
	if !ok {
		return
	}

	after, ok := change["after"].(map[string]interface{})
	if !ok {
		return
	}

	for k, v := range after {
		result[k] = v
	}
}

// contains checks if a string is in a slice.
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
