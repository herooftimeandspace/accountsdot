package web

import (
	"net/http"
	"slices"
	"time"
)

const devZoomDeskPhoneRenamesRoute = "/reports/zoom-desk-phone-renames"

type zoomDeskPhoneRenameReportPagePayload struct {
	PageID      string                               `json:"page_id"`
	Persona     devPersona                           `json:"persona"`
	Shell       devShellPayload                      `json:"shell"`
	GeneratedAt string                               `json:"generated_at"`
	Page        zoomDeskPhoneRenameReportPageContent `json:"page"`
}

type zoomDeskPhoneRenameReportPageContent struct {
	Title         string                          `json:"title"`
	Description   string                          `json:"description"`
	HelpText      string                          `json:"help_text"`
	LastRefreshed string                          `json:"last_refreshed"`
	SummaryCards  []summaryCardPayload            `json:"summary_cards"`
	Rows          []zoomDeskPhoneRenameRowPayload `json:"rows"`
}

type zoomDeskPhoneRenameRowPayload struct {
	ID                    string `json:"id"`
	SerialNumber          string `json:"serial_number"`
	MACAddress            string `json:"mac_address"`
	CurrentName           string `json:"current_name"`
	NewName               string `json:"new_name"`
	Status                string `json:"status"`
	NextAction            string `json:"next_action"`
	IncidentIQAssetLabel  string `json:"incidentiq_asset_label"`
	IncidentIQAssetURL    string `json:"incidentiq_asset_url"`
	IncidentIQAssetDomain string `json:"incidentiq_asset_domain"`
}

type zoomDeskPhoneRenameSeedRecord struct {
	ID                    string
	SerialNumber          string
	MACAddress            string
	CurrentName           string
	NewName               string
	Status                string
	NextAction            string
	IncidentIQAssetLabel  string
	IncidentIQAssetURL    string
	IncidentIQAssetDomain string
}

var actionableZoomDeskPhoneRenameStatuses = []string{
	"Pending manual adjustment",
	"Error",
}

// handleDevZoomDeskPhoneRenamesReportPage serves the IT Admin-only DEV report
// for Zoom desk phones whose expected Zoom display name needs an IncidentIQ
// asset-location correction. The handler returns read-only mock rows and does
// not write IncidentIQ, Zoom, or local database state.
func handleDevZoomDeskPhoneRenamesReportPage(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can view this page.",
		})
		return
	}
	if !routeAllowed(r.Context(), config, devZoomDeskPhoneRenamesRoute) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Zoom desk phone rename reports are available only to IT Admin.",
			"persona": config.Persona,
		})
		return
	}

	now := time.Now().UTC()
	rows := zoomDeskPhoneRenameRows()
	writeJSON(w, http.StatusOK, zoomDeskPhoneRenameReportPagePayload{
		PageID:      "reports-zoom-desk-phone-renames",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: now.Format(time.RFC3339),
		Page: zoomDeskPhoneRenameReportPageContent{
			Title:         "Zoom Desk Phone Renames",
			Description:   "Desk phones whose Zoom device name needs manual IncidentIQ asset-location follow-up.",
			HelpText:      "Update the phone asset location in IncidentIQ to force the next Zoom sync to rename the desk phone.",
			LastRefreshed: "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			SummaryCards: []summaryCardPayload{
				{Title: "Actionable Phones", Count: "2"},
				{Title: "Manual Adjustment", Count: "1"},
				{Title: "Error Follow-up", Count: "1"},
			},
			Rows: rows,
		},
	})
}

func zoomDeskPhoneRenameRows() []zoomDeskPhoneRenameRowPayload {
	rows := []zoomDeskPhoneRenameRowPayload{}
	for _, seed := range zoomDeskPhoneRenameSeedRows() {
		if !slices.Contains(actionableZoomDeskPhoneRenameStatuses, seed.Status) {
			continue
		}
		rows = append(rows, zoomDeskPhoneRenameRowPayload(seed))
	}
	return rows
}

func zoomDeskPhoneRenameSeedRows() []zoomDeskPhoneRenameSeedRecord {
	return []zoomDeskPhoneRenameSeedRecord{
		{
			ID:                    "phone-a108-pending-location",
			SerialNumber:          "SN-ZP-A108-0427",
			MACAddress:            "00:15:5D:3A:10:27",
			CurrentName:           "Clover HS Room A106",
			NewName:               "Clover HS Room A108",
			Status:                "Pending manual adjustment",
			NextAction:            "Update IncidentIQ asset location to Clover HS A108.",
			IncidentIQAssetLabel:  "IIQ Asset ZP-A108",
			IncidentIQAssetURL:    "https://wusd.incidentiq.com/agent/assets/asset-zp-a108",
			IncidentIQAssetDomain: "wusd.incidentiq.com",
		},
		{
			ID:                    "phone-b212-error-location",
			SerialNumber:          "SN-ZP-B212-1184",
			MACAddress:            "00:15:5D:3A:21:84",
			CurrentName:           "District Office Spare",
			NewName:               "Clover HS Room B212",
			Status:                "Error",
			NextAction:            "Correct the IncidentIQ asset location and recheck the next Zoom reconciliation.",
			IncidentIQAssetLabel:  "IIQ Asset ZP-B212",
			IncidentIQAssetURL:    "https://wusd.incidentiq.com/agent/assets/asset-zp-b212",
			IncidentIQAssetDomain: "wusd.incidentiq.com",
		},
		{
			ID:                    "phone-c301-complete",
			SerialNumber:          "SN-ZP-C301-2261",
			MACAddress:            "00:15:5D:3A:30:61",
			CurrentName:           "Clover HS Room C301",
			NewName:               "Clover HS Room C301",
			Status:                "Completed",
			NextAction:            "No manual action needed.",
			IncidentIQAssetLabel:  "IIQ Asset ZP-C301",
			IncidentIQAssetURL:    "https://wusd.incidentiq.com/agent/assets/asset-zp-c301",
			IncidentIQAssetDomain: "wusd.incidentiq.com",
		},
		{
			ID:                    "phone-d118-healthy",
			SerialNumber:          "SN-ZP-D118-7719",
			MACAddress:            "00:15:5D:3A:71:19",
			CurrentName:           "Clover HS Room D118",
			NewName:               "Clover HS Room D118",
			Status:                "Healthy",
			NextAction:            "No manual action needed.",
			IncidentIQAssetLabel:  "IIQ Asset ZP-D118",
			IncidentIQAssetURL:    "https://wusd.incidentiq.com/agent/assets/asset-zp-d118",
			IncidentIQAssetDomain: "wusd.incidentiq.com",
		},
		{
			ID:                    "phone-e104-waiting",
			SerialNumber:          "SN-ZP-E104-5512",
			MACAddress:            "00:15:5D:3A:55:12",
			CurrentName:           "Clover HS Room E104",
			NewName:               "Clover HS Room E104",
			Status:                "Waiting for sync",
			NextAction:            "No manual action yet.",
			IncidentIQAssetLabel:  "IIQ Asset ZP-E104",
			IncidentIQAssetURL:    "https://wusd.incidentiq.com/agent/assets/asset-zp-e104",
			IncidentIQAssetDomain: "wusd.incidentiq.com",
		},
	}
}
