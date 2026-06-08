package verify

// ToolCallF1 computes the harmonic mean of precision and recall for
// tool-call behavior across a slice of CaseResults.
//
// Precision = TP / (TP + FP)  — of calls made, how many were correct
// Recall    = TP / (TP + FN)  — of calls expected, how many were made
// F1        = 2 * P * R / (P + R)
//
// For each CaseResult:
//
//	TP: expected a tool call AND got the right tool  (ExpectedTool && CorrectTool)
//	FP: got a tool call but expected none            (!ExpectedTool && CalledAnyTool)
//	   or got the wrong tool                         (ExpectedTool && CalledAnyTool && !CorrectTool)
//	FN: expected a tool call but got none            (ExpectedTool && !CalledAnyTool)
//	TN: expected no tool call AND got none           (ignored in F1)
//
// Returns (1, 1, 1) when there are no tool-call-relevant cases (vacuous).
func ToolCallF1(results []CaseResult, _ []Case) (precision, recall, f1 float64) {
	var tp, fp, fn float64

	for _, r := range results {
		switch {
		case r.ExpectedTool && r.CorrectTool:
			tp++
		case !r.ExpectedTool && r.CalledAnyTool:
			fp++
		case r.ExpectedTool && r.CalledAnyTool && !r.CorrectTool:
			fp++
			fn++
		case r.ExpectedTool && !r.CalledAnyTool:
			fn++
			// TN: !ExpectedTool && !CalledAnyTool — not counted.
		}
	}

	if tp+fp == 0 {
		precision = 1.0
	} else {
		precision = tp / (tp + fp)
	}

	if tp+fn == 0 {
		recall = 1.0
	} else {
		recall = tp / (tp + fn)
	}

	if precision+recall == 0 {
		f1 = 0
	} else {
		f1 = 2 * precision * recall / (precision + recall)
	}

	return precision, recall, f1
}
