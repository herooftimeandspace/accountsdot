import assert from "node:assert/strict";
import test from "node:test";

import {
  DEFAULT_FREQUENT_FLIERS_FILTERS,
  FREQUENT_FLIER_ROWS,
  FREQUENT_FLIERS_REPRESENTATIVE_COMBINATIONS,
  FREQUENT_FLIERS_RANGE_OPTIONS,
  frequentFliersCombinationSignature,
  frequentFliersRowsForFilters,
  linkForDevice,
  linkForTicket,
  metricCountForRange,
  rangeLabelForValue,
  trendClass,
} from "./frequentFliersModel.mjs";

test("Frequent Fliers defaults match the documented threshold, metric, and lookback", () => {
  assert.deepEqual(DEFAULT_FREQUENT_FLIERS_FILTERS, { threshold: 2, metric: "devices", range: "90" });
  assert.deepEqual(FREQUENT_FLIERS_RANGE_OPTIONS.map((option) => option.label), [
    "30 days",
    "60 days",
    "90 days",
    "6 months",
    "1 year",
  ]);
  assert.equal(rangeLabelForValue("90"), "90 days");
  assert.equal(rangeLabelForValue("unknown"), "90 days");
});

test("Frequent Fliers filters use fixed greater-than-or-equal comparison for selected metric and range", () => {
  const deviceMatches = frequentFliersRowsForFilters(FREQUENT_FLIER_ROWS, {
    threshold: 2,
    metric: "devices",
    range: "90",
  }).map((row) => row.id);
  const ticketMatches = frequentFliersRowsForFilters(FREQUENT_FLIER_ROWS, {
    threshold: 2,
    metric: "tickets",
    range: "30",
  }).map((row) => row.id);

  assert.deepEqual(deviceMatches, [
    "jason-rodriguez",
    "maria-nguyen",
    "devon-price",
    "sophia-patel",
    "noah-kim",
    "omar-castillo",
  ]);
  assert.deepEqual(ticketMatches, ["devon-price", "aaliyah-brooks"]);
  assert.equal(metricCountForRange(FREQUENT_FLIER_ROWS[0], "devices", "60"), 3);
  assert.equal(metricCountForRange(FREQUENT_FLIER_ROWS[0], "tickets", "60"), 2);
});

test("Frequent Fliers representative dropdown combinations produce different mock row sets", () => {
  const signatures = FREQUENT_FLIERS_REPRESENTATIVE_COMBINATIONS.map((filters) => {
    return frequentFliersCombinationSignature(FREQUENT_FLIER_ROWS, filters);
  });

  assert.deepEqual(signatures, [
    "jason-rodriguez|maria-nguyen|devon-price|sophia-patel|noah-kim|omar-castillo",
    "noah-kim",
    "jason-rodriguez|maria-nguyen|noah-kim|omar-castillo",
    "devon-price|aaliyah-brooks",
    "jason-rodriguez|devon-price|aaliyah-brooks|omar-castillo",
  ]);
  assert.equal(new Set(signatures).size, signatures.length);
});

test("Frequent Fliers DEV links are deterministic IncidentIQ targets", () => {
  assert.equal(
    linkForDevice("CLA-24-27891"),
    "https://mock.wusd.local/incidentiq/assets/CLA-24-27891"
  );
  assert.equal(
    linkForTicket("INC-1782345"),
    "https://mock.wusd.local/incidentiq/tickets/INC-1782345"
  );
});

test("Frequent Fliers trend classes expose below-threshold, review, and critical marks", () => {
  assert.equal(
    trendClass(1, 2),
    "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--ready"
  );
  assert.equal(
    trendClass(2, 2),
    "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--review"
  );
  assert.equal(
    trendClass(4, 2),
    "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--critical"
  );
});
