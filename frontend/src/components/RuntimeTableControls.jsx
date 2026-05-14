import { useMemo, useState } from "react";

/**
 * nextSortState documents runtime data flow for frontend/src/components/RuntimeTableControls.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function nextSortState(current, key) {
  if (current.key !== key || current.direction === "none") {
    return { key, direction: "asc" };
  }
  if (current.direction === "asc") {
    return { key, direction: "desc" };
  }
  return { key: null, direction: "none" };
}

/**
 * stringifyValue documents runtime data flow for frontend/src/components/RuntimeTableControls.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function stringifyValue(value) {
  if (value === undefined || value === null) {
    return "";
  }
  if (Array.isArray(value)) {
    return value.map(stringifyValue).join(" ");
  }
  if (typeof value === "object") {
    return Object.values(value).map(stringifyValue).join(" ");
  }
  return String(value);
}

/**
 * runtimeTableColumnValue selects the plain value a runtime table should use for search and sort. Page-specific renderers can return JSX, so callers that build labels or compare rows should use this helper to avoid turning interactive cell markup into table data.
 */
export function runtimeTableColumnValue(row, column, purpose) {
  if (purpose === "sort" && typeof column.sortValue === "function") {
    return column.sortValue(row);
  }
  if (purpose === "search" && typeof column.searchValue === "function") {
    return column.searchValue(row);
  }
  if (typeof column.value === "function") {
    return column.value(row);
  }
  if (typeof column.render === "function") {
    return column.render(row);
  }
  return row?.[column.key];
}

/**
 * searchableRowText documents runtime data flow for frontend/src/components/RuntimeTableControls.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function searchableRowText(row, columns) {
  return columns
    .map((column) => stringifyValue(runtimeTableColumnValue(row, column, "search")))
    .join(" ")
    .toLowerCase();
}

/**
 * compareRows documents runtime data flow for frontend/src/components/RuntimeTableControls.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function compareRows(left, right, column, direction) {
  const leftValue = stringifyValue(runtimeTableColumnValue(left.row, column, "sort"));
  const rightValue = stringifyValue(runtimeTableColumnValue(right.row, column, "sort"));
  const comparison = leftValue.localeCompare(rightValue, undefined, {
    numeric: true,
    sensitivity: "base",
  });
  if (comparison !== 0) {
    return direction === "asc" ? comparison : -comparison;
  }
  return left.index - right.index;
}

/**
 * useRuntimeTableData derives reusable React state for frontend/src/components/RuntimeTableControls.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function useRuntimeTableData(rows, columns, { defaultSort = { key: null, direction: "none" } } = {}) {
  const [searchQuery, setSearchQuery] = useState("");
  const [sortState, setSortState] = useState(defaultSort);

  const visibleRows = useMemo(() => {
    const sourceRows = Array.isArray(rows) ? rows : [];
    const normalizedQuery = searchQuery.trim().toLowerCase();
    const filteredRows = normalizedQuery
      ? sourceRows.filter((row) => searchableRowText(row, columns).includes(normalizedQuery))
      : sourceRows;

    if (!sortState?.key || sortState.direction === "none") {
      return filteredRows;
    }

    const sortColumn = columns.find((column) => column.key === sortState.key);
    if (!sortColumn) {
      return filteredRows;
    }

    return [...filteredRows]
      .map((row, index) => ({ row, index }))
      .sort((left, right) => compareRows(left, right, sortColumn, sortState.direction))
      .map(({ row }) => row);
  }, [columns, rows, searchQuery, sortState]);

  return {
    visibleRows,
    searchQuery,
    setSearchQuery,
    sortState,
    toggleSort: (key) => setSortState((current) => nextSortState(current, key)),
  };
}

/**
 * RuntimeTableSearch renders the UI surface for frontend/src/components/RuntimeTableControls.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function RuntimeTableSearch({ value, onChange, label = "Search table", placeholder = "Search this table..." }) {
  return (
    <label className="runtime-table-search">
      <span>{label}</span>
      <input
        type="search"
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}

/**
 * RuntimeSortableHeader renders the UI surface for frontend/src/components/RuntimeTableControls.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function RuntimeSortableHeader({ column, sortState, onSort }) {
  const activeDirection = sortState?.key === column.key ? sortState.direction : "none";
  const indicator = activeDirection === "asc" ? "↑" : activeDirection === "desc" ? "↓" : "↕";
  const sortText =
    activeDirection === "asc"
      ? "ascending"
      : activeDirection === "desc"
        ? "descending"
        : "not sorted";

  return (
    <button
      type="button"
      className="runtime-table-sort-button"
      aria-label={`Sort ${column.label}; currently ${sortText}`}
      onClick={() => onSort(column.key)}
    >
      <span>{column.label}</span>
      <span aria-hidden="true">{indicator}</span>
    </button>
  );
}
