package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const SentinelMarker = "__TAIL_SENTINEL__"

var syncConfigCells = map[string]string{
	"Zoom_SLG":        "B2",
	"Zoom_Users":      "B3",
	"Zoom_CallQueues": "B4",
	"Zoom_CommonArea": "B5",
	"Zoom_AR":         "B6",
}

// ChecksumRows documents the data flow for internal/provider/sheets.go. Provider contract tests and future provider implementations reach this function; debug it by checking normalized payloads, error classes, and external-write safety notes. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func ChecksumRows(rows [][]string) string {
	hasher := sha256.New()
	for _, row := range rows {
		hasher.Write([]byte(strings.Join(row, "\x1f")))
		hasher.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// BuildSentinelRow documents the data flow for internal/provider/sheets.go. Provider contract tests and future provider implementations reach this function; debug it by checking normalized payloads, error classes, and external-write safety notes. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func BuildSentinelRow(rowCount int, checksum string, version int64) []string {
	return []string{
		SentinelMarker,
		strconv.Itoa(rowCount),
		checksum,
		strconv.FormatInt(version, 10),
	}
}

// SyncConfigCell documents the data flow for internal/provider/sheets.go. Provider contract tests and future provider implementations reach this function; debug it by checking normalized payloads, error classes, and external-write safety notes. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func SyncConfigCell(tab string) (string, error) {
	cell, ok := syncConfigCells[tab]
	if !ok {
		return "", fmt.Errorf("unknown sync tab %q", tab)
	}
	return cell, nil
}

// VisibleTabFormula documents the data flow for internal/provider/sheets.go. Provider contract tests and future provider implementations reach this function; debug it by checking normalized payloads, error classes, and external-write safety notes. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func VisibleTabFormula(tab string) (string, error) {
	cell, err := SyncConfigCell(tab)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`=QUERY(INDIRECT(Sync_Config!%s & "!A:Z"), "select *", 0)`, cell), nil
}
