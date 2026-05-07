const COMMON_SHELL_TEXT = new Set([
  "The WIZARD",
  "Windsor Identity Zync,",
  "Access, & Retirement Dashboard",
  "Have you checked with The WIZARD?",
  "Dashboard",
  "Staff Onboarding",
  "Offboarding",
  "Room Moves",
  "Phone Directory",
  "Data Quality",
  "Frequent Fliers",
  "Student Data Cleanup",
  "Reports",
  "Admin",
]);

function collectTextNodes(node, textOverrides, target = []) {
  if (node?.type === "text") {
    const content = Object.prototype.hasOwnProperty.call(textOverrides, node.id)
      ? textOverrides[node.id]
      : node.content;
    const normalized = String(content ?? "").replace(/\s+/g, " ").trim();
    if (normalized) {
      target.push({
        content: normalized,
        fontSize: node.fontSize ?? 14,
        fontWeight: Number.parseInt(String(node.fontWeight ?? 400), 10),
      });
    }
  }

  for (const child of node?.children || []) {
    collectTextNodes(child, textOverrides, target);
  }

  return target;
}

function scoreTitleCandidate(entry) {
  return (entry.fontSize ?? 14) * 10 + ((entry.fontWeight ?? 400) >= 700 ? 20 : 0);
}

export function buildArtboardSemanticSummary(
  artboard,
  { fallbackTitle = "Page", textOverrides = {}, maxItems = 18 } = {}
) {
  const entries = collectTextNodes(artboard, textOverrides);
  const titleEntry = entries
    .filter((entry) => !COMMON_SHELL_TEXT.has(entry.content))
    .sort((left, right) => scoreTitleCandidate(right) - scoreTitleCandidate(left))[0];
  const title = titleEntry?.content || fallbackTitle;
  const seen = new Set([title]);
  const items = [];

  for (const entry of entries) {
    if (items.length >= maxItems) {
      break;
    }
    if (seen.has(entry.content) || COMMON_SHELL_TEXT.has(entry.content)) {
      continue;
    }
    seen.add(entry.content);
    items.push(entry.content);
  }

  return { title, items };
}
