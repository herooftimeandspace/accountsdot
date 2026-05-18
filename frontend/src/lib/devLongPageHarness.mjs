export const LONG_PAGE_ROUTE_HARNESS = [
  { path: "/dashboard/it-admin", selectors: [], fallback: "dashboard rows" },
  { path: "/dashboard/hr-lifecycle", selectors: [], fallback: "HR lifecycle rows" },
  { path: "/dashboard/site-admin", selectors: [], fallback: "site dashboard rows" },
  { path: "/search", selectors: [".global-search-runtime__result", ".global-search-runtime__group"], fallback: "search results" },
  { path: "/onboarding", selectors: [".onboarding-runtime__row"], fallback: "onboarding rows" },
  { path: "/offboarding", selectors: [".offboarding-runtime__row"], fallback: "offboarding rows" },
  { path: "/departing-seniors", selectors: [".departing-seniors-runtime__row"], fallback: "departing senior rows" },
  { path: "/room-moves", selectors: [".room-moves-runtime__row"], fallback: "room move rows" },
  { path: "/room-moves/bulk-draft", selectors: [".room-moves-runtime__bulk-row", ".room-moves-runtime__row"], fallback: "bulk draft rows" },
  { path: "/phone-directory/by-person", selectors: [".phone-directory-runtime__row"], fallback: "person directory rows" },
  { path: "/phone-directory/by-department", selectors: [".phone-directory-runtime__row"], fallback: "department directory rows" },
  { path: "/phone-directory/by-room", selectors: [".phone-directory-runtime__row"], fallback: "room directory rows" },
  { path: "/data-quality", selectors: [".data-quality-runtime tbody tr", ".data-quality-runtime tr"], fallback: "data quality rows" },
  { path: "/frequent-fliers", selectors: [".frequent-fliers-runtime__row"], fallback: "frequent flier rows" },
  { path: "/student-data-cleanup", selectors: [".student-data-runtime__row"], fallback: "student cleanup rows" },
  { path: "/reports", selectors: [".reports-runtime__row", ".reports-runtime__card"], fallback: "report rows" },
  { path: "/reports/security-issues", selectors: [".security-issues-runtime__row", ".reports-runtime__row"], fallback: "security issue rows" },
  { path: "/reports/sync-transparency", selectors: [], fallback: "sync transparency rows" },
  { path: "/admin", selectors: [], fallback: "admin control rows" },
  { path: "/admin/feature-flags", selectors: [".feature-flags-runtime__flag", ".feature-flags-runtime__target-row"], fallback: "feature flag rows" },
  { path: "/my-profile", selectors: [], fallback: "profile history rows" },
];

const DEFAULT_ROW_COUNT = 72;
const HARNESS_TIMEOUT_MS = 1500;
const HARNESS_RETRY_MS = 100;

function longPageHarnessEnabled(search) {
  return new URLSearchParams(search).get("longPageHarness") === "1";
}

function longPageHarnessRowCount(search) {
  const value = Number.parseInt(new URLSearchParams(search).get("longPageHarnessRows") || "", 10);
  return Number.isFinite(value) && value > 0 ? value : DEFAULT_ROW_COUNT;
}

function visibleElements(elements) {
  return elements.filter((element) => element instanceof HTMLElement && element.offsetParent !== null);
}

function cloneExistingRows(selectors, count) {
  for (const selector of selectors) {
    const rows = visibleElements([...document.querySelectorAll(selector)]).filter(
      (row) => !row.dataset.longPageHarnessClone
    );
    if (!rows.length) {
      continue;
    }
    const parent = rows[rows.length - 1].parentElement;
    if (!parent) {
      continue;
    }
    for (let index = 0; index < count; index += 1) {
      const clone = rows[index % rows.length].cloneNode(true);
      clone.dataset.longPageHarnessClone = "true";
      clone.querySelectorAll("[id]").forEach((node) => node.removeAttribute("id"));
      clone.querySelectorAll("button, a, input, select, textarea").forEach((node) => {
        node.setAttribute("tabindex", "-1");
        node.setAttribute("aria-hidden", "true");
      });
      parent.appendChild(clone);
    }
    return { strategy: "clone", selector, count };
  }
  return null;
}

function injectFallbackRows(root, label, count) {
  const block = document.createElement("section");
  block.dataset.longPageHarnessClone = "true";
  block.setAttribute("aria-label", "Long page harness rows");
  Object.assign(block.style, {
    position: "absolute",
    left: "306px",
    top: "940px",
    width: "1180px",
    display: "grid",
    gap: "8px",
    fontFamily: "Atkinson Hyperlegible, Arial, sans-serif",
    color: "#01161E",
  });

  for (let index = 0; index < count; index += 1) {
    const row = document.createElement("div");
    row.textContent = `${label} overflow row ${String(index + 1).padStart(2, "0")}`;
    Object.assign(row.style, {
      minHeight: "42px",
      border: "1px solid #DDE2E3",
      borderRadius: "6px",
      background: "#FFFFFF",
      padding: "10px 14px",
      boxSizing: "border-box",
    });
    block.appendChild(row);
  }

  root.appendChild(block);
  return { strategy: "fallback", count };
}

function finishLongPageHarness(routePath, root, result) {
  document.documentElement.dataset.longPageHarness = "active";
  root.dataset.longPageHarness = "active";
  window.WizardLongPageHarnessResult = { route: routePath, ...result };
}

/**
 * scheduleLongPageHarness is a DEV-only route stress helper. It waits briefly
 * for route-owned table/card rows so runtime pages exercise their real row
 * containers, then falls back to synthetic rows for static implemented pages.
 */
export function scheduleLongPageHarness({ root, routePath, search }) {
  if (!root || !longPageHarnessEnabled(search) || root.querySelector('[data-long-page-harness-clone="true"]')) {
    return () => {};
  }

  const recipe = LONG_PAGE_ROUTE_HARNESS.find((entry) => entry.path === routePath);
  if (!recipe) {
    return () => {};
  }

  const count = longPageHarnessRowCount(search);
  const startedAt = window.performance.now();
  let disposed = false;
  let timer = 0;

  const attempt = () => {
    if (disposed || root.querySelector('[data-long-page-harness-clone="true"]')) {
      return;
    }

    const cloneResult = cloneExistingRows(recipe.selectors || [], count);
    if (cloneResult) {
      finishLongPageHarness(routePath, root, cloneResult);
      return;
    }

    if (window.performance.now() - startedAt >= HARNESS_TIMEOUT_MS) {
      finishLongPageHarness(routePath, root, injectFallbackRows(root, recipe.fallback, count));
      return;
    }

    timer = window.setTimeout(attempt, HARNESS_RETRY_MS);
  };

  timer = window.setTimeout(attempt, 0);

  return () => {
    disposed = true;
    window.clearTimeout(timer);
  };
}
