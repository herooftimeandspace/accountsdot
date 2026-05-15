export const VISIBLE_SPACE_MARKER = "·";

/**
 * markEdgeWhitespaceForStudentData makes Aeries leading and trailing spaces visible on Student Data Cleanup.
 * The page preserves source-system text for comparison, but site secretaries need a stable visual marker for
 * whitespace-only defects that would otherwise collapse in table cells and drawer detail rows.
 */
export function markEdgeWhitespaceForStudentData(value) {
  const text = String(value ?? "");
  return text
    .replace(/^ +/, (spaces) => VISIBLE_SPACE_MARKER.repeat(spaces.length))
    .replace(/ +$/, (spaces) => VISIBLE_SPACE_MARKER.repeat(spaces.length));
}

/**
 * shouldShowSuggestedStudentNameValue returns whether the drawer should include a suggested first/last-name row.
 * Suggestions that render identically to the current Aeries value add noise for site secretaries, while whitespace
 * differences remain visible because the current value is marker-rendered before comparison.
 */
export function shouldShowSuggestedStudentNameValue(currentValue, suggestedValue) {
  return markEdgeWhitespaceForStudentData(currentValue) !== markEdgeWhitespaceForStudentData(suggestedValue);
}
