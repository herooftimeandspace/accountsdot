const ROUTE_SOURCE_NOTES = {
  "/dashboard/it-admin": "Product requirements dashboard and admin visibility scope; implementation plan implemented-page shell and IT Admin warnings.",
  "/dashboard/hr-lifecycle": "Product requirements HR lifecycle dashboard scope; implementation plan implemented-page shell and onboarding/offboarding routing.",
  "/dashboard/site-admin": "Product requirements site-admin visibility scope; implementation plan implemented-page shell and site-owned queue routing.",
  "/search": "Product requirements shared search and staff-only access scope; implementation plan implemented-page shell behavior.",
  "/onboarding": "Product requirements Onboarding pre-phase 0 behavior; implementation plan Onboarding runtime drawer and manual intake contracts.",
  "/offboarding": "Product requirements Offboarding pre-phase 0 behavior; implementation plan Offboarding runtime drawer and end-date ownership contracts.",
  "/departing-seniors": "Product requirements Departing Seniors requirements; implementation plan account retirement and device-return drawer contracts.",
  "/room-moves": "Product requirements Room Moves workflow; implementation plan room move execution, correction, warning, and revert contracts.",
  "/room-moves/bulk-draft": "Product requirements Room Moves batch/site rollover scope; implementation plan bulk draft review and cutover contracts.",
  "/phone-directory/by-person": "Product requirements Phone Directory person mode; implementation plan directory runtime view and detail-drawer contracts.",
  "/phone-directory/by-room": "Product requirements Phone Directory room mode; implementation plan directory runtime view and detail-drawer contracts.",
  "/phone-directory/by-department": "Product requirements Phone Directory department mode; implementation plan directory runtime view and detail-drawer contracts.",
  "/data-quality": "Product requirements Data Quality routing limits; implementation plan issue #57 decision and Data Quality queue behavior.",
  "/frequent-fliers": "Product requirements Frequent Fliers support-analysis scope; implementation plan configurable lookback, filters, and drawer contracts.",
  "/student-data-cleanup": "Product requirements Student Data Cleanup Aeries correction contract; implementation plan runtime table, filters, and drawer behavior.",
  "/reports": "Product requirements Reports inventory and refresh drawer requirements; implementation plan Reports runtime table/search/sort behavior.",
  "/reports/security-issues": "Product requirements Security Issues report migration; implementation plan read-only IT Admin report and drawer details.",
  "/reports/zoom-desk-phone-renames": "Product requirements Zoom Desk Phone rename report scope; implementation plan IncidentIQ asset-location correction and report behavior.",
  "/reports/sync-transparency": "Product requirements sync visibility goals; implementation plan sync transparency report and manual-action state guidance.",
  "/reports/ticketing-human-work": "Product requirements IncidentIQ human-work fallback; implementation plan ticketing report and manual owner contracts.",
  "/admin": "Product requirements IT Admin control scope; implementation plan Admin sync, exception, reversal, and emergency-control behavior.",
  "/admin/feature-flags": "Product requirements staged rollout and staff-only controls; implementation plan DEV feature flag control surface.",
  "/my-profile": "Product requirements account menu and staff profile scope; implementation plan shared shell profile affordance.",
};

const HELP_CONTENT_BY_ROUTE = {
  "/dashboard/it-admin": {
    title: "IT Admin Dashboard help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This dashboard is the IT Admin landing view for district-wide account, access, sync, data-quality, room-move, and provider-warning work. It is meant to help IT decide which operational queue needs attention first.",
          "Cards and tables summarize current exceptions rather than replacing the owning workflow pages. Counts are awareness signals; the detailed correction path usually lives on Onboarding, Offboarding, Room Moves, Data Quality, Reports, or Admin.",
        ],
      },
      {
        heading: "Controls and navigation",
        paragraphs: [
          "Use the sidebar to open the workflow that owns the issue you want to work. The header scope selector shows the current district or site context, and the shared search field can be used to find staff, students, rooms, phones, or workflow records when the route supports search.",
          "Use Refresh only when the page exposes it as a real page-level freshness control. It rechecks the current page data; it does not approve provider writes or change queue ownership.",
        ],
      },
      {
        heading: "How to interpret statuses",
        paragraphs: [
          "Ready and Healthy items can usually continue without manual intervention. Needs Review, Manual Action, Warning, Blocked, Invalid, Failed, and Security Risk items should be opened on the owning page so the row drawer can show the owner, external system, and resolution text.",
          "When a row names a source system, treat that system as the source of truth unless the page explicitly documents a local override. Do not correct provider data from the dashboard unless a visible control on the owning page supports that exact correction.",
        ],
      },
    ],
  },
  "/dashboard/hr-lifecycle": {
    title: "HR Lifecycle Dashboard help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This dashboard gives HR a lifecycle-focused view of upcoming onboarding, offboarding, missing intake data, and records that need HR-owned correction before IT automation can continue.",
          "The page is for deciding what HR should fix or review next. It does not replace Escape as the source of truth for Escape-backed employment dates or employee identity fields.",
        ],
      },
      {
        heading: "Controls and filters",
        paragraphs: [
          "Use the sidebar to move from the dashboard into Onboarding or Offboarding when a queue needs row-level detail. Use search for known people or workflow records when you need to jump directly to a person.",
          "If a visible row or card points to missing intake data, open the owning Onboarding row. Manual Non-Escape records can be continued there when the page shows an incomplete draft.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Escape-backed employee data should be corrected in Escape. Non-Escape or local override records may expose a dashboard-owned correction only where the row drawer explicitly provides that control.",
          "Rows marked Blocked, Incomplete Data, or Needs Review should be handled before routine Ready rows because downstream Google, Aeries, room, phone, and access work may be waiting on those fields.",
        ],
      },
    ],
  },
  "/dashboard/site-admin": {
    title: "Site Admin Dashboard help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This dashboard gives site staff a site-scoped view of people, rooms, phone-directory records, student cleanup, and local workflow issues they are allowed to see.",
          "It is an awareness and routing page. When a correction requires a source system such as Aeries, the dashboard should guide you to that source instead of editing the record locally.",
        ],
      },
      {
        heading: "Controls and filters",
        paragraphs: [
          "Use the header scope selector to confirm which site or district context you are viewing. Use sidebar routes such as Phone Directory, Student Data Cleanup, Frequent Fliers, or Room Moves when you need detailed rows and drawers.",
          "The shared search field is best for known names, student IDs, rooms, extensions, and emails. If search returns a result on a workflow page, select the row there for the current next action.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Student data corrections happen in Aeries. Room and phone corrections happen through the Room Moves or Phone Directory workflows only when those pages show the relevant control.",
          "If a row says IT owns the action, do not work around it on the site page. The owning IT Admin or report queue should show the detailed resolution path.",
        ],
      },
    ],
  },
  "/search": {
    title: "Search help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Search collects matching staff, student, room, phone-directory, and workflow records that the current persona is allowed to access.",
          "Results are visibility-filtered. A missing result may mean the record is outside your role, outside the current scope, or not available to this dashboard.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Search by a specific name, district email, student ID, room, extension, or known workflow term. Use a more specific query when common names return too many matches.",
          "Open the result that matches the work you need to perform. The destination page owns the actual controls, filters, table rows, drawers, and correction path for that record.",
        ],
      },
      {
        heading: "Status and corrections",
        paragraphs: [
          "Use status badges as routing hints. Search does not approve workflow steps, change source-system records, or bypass role authorization.",
          "If the result points to an upstream source such as Aeries or Escape, make the correction in that source unless the destination row drawer exposes a documented local override.",
        ],
      },
    ],
  },
  "/onboarding": {
    title: "Onboarding help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Onboarding tracks staff who need accounts, access, rooms, phones, or follow-up before they are fully ready. Rows include Escape-backed records and allowed manual Non-Escape drafts for contractors or other people who are not yet in Escape.",
          "The status badge tells you whether the record is Ready, In Progress, Waiting, Missing Information, Incomplete Data, Needs Review, or Blocked. Start with records that are blocked or incomplete because they can prevent downstream provisioning.",
        ],
      },
      {
        heading: "Controls, search, and table",
        paragraphs: [
          "Use table search and sortable column headers to find a person by date added, start date, name, site, current step, issue/action, or workflow status. Select any row to open the right drawer with the current workflow state.",
          "Use Add Non-Escape Record only for HR or IT manual intake that this page explicitly supports. The drawer asks for the required intake fields, optional replacing employee, room/classroom, and notes. It does not ask operators to guess district email, Google groups, trainings, keys, alarm codes, or ID card fields.",
        ],
      },
      {
        heading: "Drawers and warnings",
        paragraphs: [
          "The selected-row drawer lists workflow steps, status, owner, resolution instructions, and external-system links when a destination is known. If a step requires user interaction, follow the owner and resolution text in the drawer.",
          "Manual drafts autosave while edited and stay visible as Incomplete Data until required fields are complete. Missing-field red borders and the drawer status summary appear only after an operator explicitly selects Save.",
          "A Vegas Gold start-date warning means the start date is within three calendar days. Access to some systems may be delayed beyond that date.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Escape-backed employee data should be fixed in Escape. If a manual contractor entry collides with an active Escape employee, the drawer links the invalid manual record to the active employee and explains that Escape takes precedence.",
          "The system generates district email candidates. Operators do not enter or override district email on this page.",
        ],
      },
    ],
  },
  "/offboarding": {
    title: "Offboarding help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Offboarding tracks staff account retirement, license cleanup, device or asset follow-up, orphan-account review, and manual actions that must happen before access can be safely removed.",
          "Rows are source-aware. Escape-backed records use Escape as the owner for employment dates, while Non-Escape, orphan, or local override rows may expose limited local controls when the current role is allowed to use them.",
        ],
      },
      {
        heading: "Controls, filters, and table",
        paragraphs: [
          "Use search and sortable table columns to prioritize blocked, manual-action, security-risk, or soon-due rows. Select a row to open the right drawer with end-date source, workflow owner, external links, and manual action details.",
          "HR and IT Admin may see an End date picker only on records whose end date is not owned by Escape. Site Admin may view applicable rows but cannot update those dates.",
          "HR and IT Admin may use Emergency Offboarding only for immediate, non-scheduled access removal. Use Offboard Contractor when a manually created Non-Escape contractor needs a dated termination workflow.",
        ],
      },
      {
        heading: "Statuses and drawers",
        paragraphs: [
          "Ready or Scheduled rows can continue through the planned retirement path. Manual Action, Needs Review, Blocked, Security Risk, and Failed rows need the drawer's owner and resolution instructions before automation should proceed.",
          "Device-return warnings and security details belong in the row drawer. Use the drawer links to inspect the relevant external system when one is provided.",
          "Emergency Offboarding searches active employees and contractors by name, email, or employee ID, then schedules immediate deprovisioning after a person is selected. Offboard Contractor searches active contractors only, saves the selected termination date only when Schedule Offboarding is selected, and Cancel closes without saving.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Correct Escape-backed end dates in Escape. Use local controls only when the row drawer provides them for Non-Escape, orphan, or local override records.",
          "Account-security issues that are broader than ordinary offboarding review are surfaced in Reports > Security Issues for IT Admin review.",
        ],
      },
    ],
  },
  "/departing-seniors": {
    title: "Departing Seniors help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Departing Seniors tracks current and retained senior-year student account retirement alongside outstanding IncidentIQ device-return context.",
          "A student stays in the queue until the account is deprovisioned and assigned devices are cleared. The page is designed for retirement readiness and device follow-up, not for editing Aeries student identity data.",
        ],
      },
      {
        heading: "Controls and filters",
        paragraphs: [
          "Use the school-year dropdown to review the current senior class or retained previous senior years. Use table search to find a student by name, email, school year, student ID, asset serial, or asset ID.",
          "Select a row to open the right drawer with school-year context, account-retirement state, device-return details, and available IncidentIQ asset links.",
        ],
      },
      {
        heading: "Actions and statuses",
        paragraphs: [
          "IT and Device Wranglers can manage supported local end-date overrides and deprovision the account when the row is ready. If device-return context is still open, handle that follow-up before treating the row as complete.",
          "Read status and warning text literally. Outstanding device or account-retirement details belong in the drawer so the table stays scannable.",
        ],
      },
    ],
  },
  "/room-moves": {
    title: "Room Moves help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Room Moves reviews room and phone changes before scheduled cutover work runs. It covers one-person corrections, batch move drafts, site rollover work, warnings, cutover readiness, and completed-job reversal context for IT.",
          "The page is safety-oriented because room moves can affect phone assignments, shared line groups, and site ownership. Review warnings before scheduling or running a cutover.",
        ],
      },
      {
        heading: "Controls and draft types",
        paragraphs: [
          "Use Move Person for a one-person correction. Use Site Rollover for summer room updates across a site. Use Batch Move when you need to add selected people to a manually built room-move list.",
          "Batch Move and Site Rollover drafts update destination rooms for the selected people or site roster. If a person is moving sites, set the destination room to none until the destination site confirms the room.",
          "Rows that have not run yet may expose Cancel Move. Use it only for pending work that should not continue to cutover.",
        ],
      },
      {
        heading: "Drawers, warnings, and statuses",
        paragraphs: [
          "Select a move row to open the right drawer. The drawer shows current room context, destination room, warnings, owner, and resolution details. Rows marked for review need a person to resolve the warning before automation can safely continue.",
          "Primary-room conflicts do not automatically become manual tickets. The drawer should identify the active primary room owner, keep the existing primary phone assignment unchanged, and explain when automation will add the moving user to the destination shared line group.",
        ],
      },
      {
        heading: "Cutover and reversals",
        paragraphs: [
          "Five or fewer reviewed moves may run immediately after final review. More than five moves use a batch cutover. Non-IT cutovers run off-hours between 8:00 PM and 4:00 AM Pacific; IT may schedule broader multi-site windows when needed.",
          "IT can only fully revert a room move. To partially revert a room move, create a new Room Move draft for the affected employees.",
        ],
      },
    ],
  },
  "/room-moves/bulk-draft": {
    title: "Room Moves bulk draft help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "The bulk draft page is the review workspace for a Batch Move or Site Rollover room-move set. It lets operators confirm the people, origin rooms, destination rooms, warning state, and cutover readiness before work runs.",
          "Bulk drafts are not general-purpose room editing. They should be used only for documented room move sets that need shared review before cutover.",
        ],
      },
      {
        heading: "Controls and filters",
        paragraphs: [
          "Use the draft mode and site context to confirm you are editing the intended move set. Add or remove rows only when the draft type supports that action.",
          "Use destination-room controls to set the intended room. When a person is changing sites and the new room is not confirmed, set the destination room to none until the destination site confirms it.",
        ],
      },
      {
        heading: "Warnings and review",
        paragraphs: [
          "Review every warning before scheduling cutover. One-person warnings and primary-room conflict details belong in the right drawer so the table can stay compact.",
          "If automation cannot safely plan or verify a shared-line-group outcome, the drawer should show the manual owner, reason, resolution steps, and linked external systems.",
        ],
      },
      {
        heading: "Cutover rules",
        paragraphs: [
          "Small reviewed batches of five or fewer moves may run immediately after final review. Larger batches use a scheduled cutover, and non-IT cutovers run off-hours between 8:00 PM and 4:00 AM Pacific.",
          "IT can only fully revert a room move. To partially revert a room move, create a new Room Move draft for the affected employees.",
        ],
      },
    ],
  },
  "/phone-directory/by-person": {
    title: "Phone Directory by person help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "By Person shows directory entries around individual staff members. It helps operators find a person's extension, room, department, site, shared-line context, and phone coverage.",
          "The table is a projection of source systems and telephony data. It should preserve source values unless a documented directory workflow owns the correction.",
        ],
      },
      {
        heading: "Controls, filters, and table",
        paragraphs: [
          "Use the mode buttons to switch between By Person, By Room, and By Department. Switching modes clears the selected row and closes the detail drawer so the drawer never shows stale context.",
          "Use search, site or scope filters, and sortable headers to find a person by name, email, extension, room, department, or site. Select a row to open the right drawer with the current detail for that person.",
        ],
      },
      {
        heading: "Drawers and corrections",
        paragraphs: [
          "The detail drawer is read-oriented. It may show person context, room context, phone values, and source-system hints for follow-up.",
          "Correct room ownership through Room Moves when the change is a room move. Correct source-system identity or department values in the owning source system unless the directory page exposes a documented correction control.",
        ],
      },
    ],
  },
  "/phone-directory/by-room": {
    title: "Phone Directory by room help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "By Room groups directory information around rooms, extensions, primary room owners, shared lines, and the people associated with each room.",
          "Use this view when the question starts with a room, classroom, office, or extension rather than a person.",
        ],
      },
      {
        heading: "Controls, filters, and table",
        paragraphs: [
          "Use mode buttons to move between person, room, and department views. Use search and sortable headers to find a room by room label, extension, site, owner, or department.",
          "Select a row to open the right drawer. The drawer should show room context, associated people, phone assignment details, and any available source-system notes.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Room moves and primary-room conflicts should be handled through Room Moves. The Phone Directory view helps inspect the current projection; it is not the place to invent an undocumented phone reassignment.",
          "When the destination room already has an active primary room owner, Room Moves explains the shared-line-group outcome or the manual follow-up path.",
        ],
      },
    ],
  },
  "/phone-directory/by-department": {
    title: "Phone Directory by department help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "By Department groups directory information around departments, department phone coverage, related rooms, and people in those departments.",
          "Use this mode for department-level coverage questions, shared lines, call routing context, and gaps that are not tied to one person or one room.",
        ],
      },
      {
        heading: "Controls, filters, and table",
        paragraphs: [
          "Use the mode buttons, scope selector, search, and sortable columns to narrow the department list. Switching modes closes any open drawer so details match the current view.",
          "Select a department row to open the right drawer with the available coverage and source context for that department.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Department labels and source values should be corrected in their owning systems unless this page exposes a documented local directory control.",
          "When a department issue is really a room or person move, use Room Moves or By Person so the correction path matches the source of the change.",
        ],
      },
    ],
  },
  "/data-quality": {
    title: "Data Quality help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Data Quality is an IT Admin awareness page for district-wide data issues that can block or delay account, access, room, phone, or lifecycle work.",
          "Rows summarize owner, severity, source systems, next action, and the queue where the correction belongs. The page is not a generic mapping dashboard and does not expose unsupported shortcuts.",
        ],
      },
      {
        heading: "Controls, filters, and table",
        paragraphs: [
          "Use table search, issue filters, severity cues, and sortable headers to identify the highest-impact issues. Use Refresh when you need the latest queue data.",
          "Start with high-severity or blocked rows. The Next Action text tells you whether the correction belongs with HR lifecycle, Onboarding, Student Data Cleanup, site-owned work, Reports, or Admin.",
        ],
      },
      {
        heading: "Statuses and correction paths",
        paragraphs: [
          "HR lifecycle owns sensitive employee lifecycle and title issues. Onboarding owns missing intake data. Student Data Cleanup and site-owned workflows cover student and room corrections. Admin owns IT-only provider conflicts and security mismatches.",
          "If no supported destination is documented, this page should keep the guidance in help text rather than showing a button to an unsupported mapping workflow.",
        ],
      },
    ],
  },
  "/frequent-fliers": {
    title: "Frequent Fliers help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Frequent Fliers helps staff find students who may be repeatedly damaging district equipment or repeatedly needing device support during the selected lookback range.",
          "The page uses device assignment and IncidentIQ ticket history to surface patterns for planning support, repair, prevention, and follow-up.",
        ],
      },
      {
        heading: "Controls and filters",
        paragraphs: [
          "Choose Devices or Tickets, select the threshold, pick the lookback range, and select Apply. The table updates only after Apply so you can adjust filters before changing the committed view.",
          "Use table search and sortable headers to find a student by name, student ID, grade, site, device count, ticket count, or trend.",
        ],
      },
      {
        heading: "Drawers and links",
        paragraphs: [
          "Select a row to open the right drawer with student context, recent device assignments, recent tickets, trend context, and notes for follow-up.",
          "Device serial numbers and IncidentIQ ticket numbers open the matching asset or ticket destination when that source link is available. Use those links for follow-up, repair planning, and support coordination.",
        ],
      },
    ],
  },
  "/student-data-cleanup": {
    title: "Student Data Cleanup help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Student Data Cleanup lists active, unresolved student name issues detected during sync. These values can affect account creation, matching, or downstream identity work if they are not corrected in Aeries.",
          "The page preserves the current Aeries values, including visible markers for leading or trailing whitespace, so staff can see exactly what the source system contains.",
        ],
      },
      {
        heading: "Controls, filters, and table",
        paragraphs: [
          "Use table search, issue type filters, grade filters, and sortable columns to find a student. The Sync now control rechecks the queue after upstream corrections have had time to sync.",
          "Select a row to open the right drawer. The drawer compares current Aeries name values with suggested cleaned values only when the suggestion differs from the displayed current value.",
        ],
      },
      {
        heading: "Where corrections happen",
        paragraphs: [
          "This dashboard is informational only. Student records cannot be edited here. Open Aeries, search by the displayed Student ID, and make the correction in Aeries.",
          "The dashboard links to the configured Aeries website root. It must not imply that it can deep-link directly to a specific student record.",
        ],
      },
    ],
  },
  "/reports": {
    title: "Reports help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Reports collects operational report inventory and source refresh status for account, access, onboarding, offboarding, room, phone, data-quality, sync, and ticketing work.",
          "Report rows answer what the report covers and where to go next. Refresh rows answer whether the source-system projection has recently refreshed.",
        ],
      },
      {
        heading: "Controls, search, and sort",
        paragraphs: [
          "Use table search and sortable headers to find a report by name, scope, source system, last run, open-item count, or status. Select a report or refresh row to open the right drawer.",
          "A report drawer shows scope, source systems, included data, open-item count, last run, refresh cadence, status, and a plain-language explanation of what the row means.",
        ],
      },
      {
        heading: "Actions and statuses",
        paragraphs: [
          "Report rows may expose Open Report when the destination is an implemented page or report route. Refresh rows are informational and do not navigate.",
          "Up to date and Healthy rows indicate normal projection freshness. Needs Review, Warning, or stale refresh context should be followed to the owning implemented page or source-system investigation path.",
        ],
      },
    ],
  },
  "/reports/security-issues": {
    title: "Security Issues report help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Security Issues is an IT Admin report for account-security rows moved out of Offboarding when the problem needs focused security review rather than ordinary lifecycle processing.",
          "Rows can include orphaned accounts, recent Google activity after source-system inactivity, asset work, owner context, and review actions.",
        ],
      },
      {
        heading: "Controls, search, and sort",
        paragraphs: [
          "Use table search and three-way sortable headers to find a row by status, person/account, email, site, next action, asset work, or external reference.",
          "Summary cards are read-only counters. They help orient the report but do not replace row review.",
        ],
      },
      {
        heading: "Drawers and actions",
        paragraphs: [
          "Select a row to open the right drawer. The drawer shows status, email, site, end-date context, next action, asset work, reference, warning text, and review actions.",
          "Review actions name the owner, status, detail, resolution, and external-system links when a destination is known. This report is read-only; it does not edit Offboarding dates or provider records.",
        ],
      },
    ],
  },
  "/reports/zoom-desk-phone-renames": {
    title: "Zoom Desk Phone Renames report help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This IT Admin report shows Zoom desk phones whose current device name does not match the expected room or location name and where the row is actionable.",
          "Rows are limited to pending manual adjustment and error states. Healthy phones, completed renames, and phones merely waiting for a non-actionable sync are excluded from the table.",
        ],
      },
      {
        heading: "Controls, search, and table",
        paragraphs: [
          "Use table search and sortable headers to find a phone by serial number, MAC address, current name, new name, or IncidentIQ asset label.",
          "Select a row to open the right drawer with the status, MAC address, current and expected names, next action, IncidentIQ domain, and asset link.",
        ],
      },
      {
        heading: "Correction path",
        paragraphs: [
          "IncidentIQ asset location is the action lever for this report. Update the phone asset location in IncidentIQ so the next Zoom sync has a source change that forces the desk phone rename.",
          "The report is read-only in this dashboard. It points IT Admins to the IncidentIQ asset record and does not write to Zoom, IncidentIQ, or local provider state.",
        ],
      },
    ],
  },
  "/reports/sync-transparency": {
    title: "Sync Transparency report help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Sync Transparency explains how provider sync and workflow planning are progressing across source systems. Use it to understand stage, warning, retry, and manual-action context before assuming a workflow is stuck.",
          "The report is for visibility and triage. It should not become a hidden write surface for provider changes.",
        ],
      },
      {
        heading: "Controls and table",
        paragraphs: [
          "Use the report table to scan sync items by provider, phase, warning state, next action, last run, and open-item context. Sort and search where runtime controls are available to narrow the list to the provider or workflow you are investigating.",
          "Open row details when available to see the row's scope, source systems, included data, open-item count, last run or refresh time, cadence, status, and plain-language meaning.",
        ],
      },
      {
        heading: "Statuses and correction paths",
        paragraphs: [
          "Healthy or up-to-date sync rows usually mean the provider projection refreshed successfully. Warning, Manual Action, Needs Review, Failed, stale, or retry-related rows should be followed to the named owner or owning workflow page.",
          "If a provider write cannot complete safely, IncidentIQ tickets are the standard fallback. Ticket descriptions should be complete enough for the assignee to act without reopening the parent workflow.",
        ],
      },
    ],
  },
  "/reports/ticketing-human-work": {
    title: "Ticketing Human Work report help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Ticketing Human Work collects human-owned IncidentIQ tickets that unblock lifecycle, room, account, device, or provider workflows.",
          "The report helps operators see which work has left automation and is waiting for a person or team to act.",
        ],
      },
      {
        heading: "Controls and table",
        paragraphs: [
          "Use search and sorting to find tickets by category, owner, workflow, matching rule, status, or required action. Select rows when the report exposes details for ticket context.",
          "Use the owning workflow, owner, and required-action fields to decide whether the ticket belongs with IT, HR, site staff, device support, or another documented owner.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Work the ticket in IncidentIQ or the named external system when the row says the action is human-owned. Do not invent a dashboard-side correction unless a visible control on the owning workflow supports it.",
          "When automation can resume after the human action, the owning workflow page or report should show the updated status after its next refresh.",
        ],
      },
    ],
  },
  "/admin": {
    title: "Admin help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Admin is the IT Admin control surface for sync health, admin warnings, deprovisioning exceptions, Google-active/Aeries-inactive defaults, completed room-move reversal, feature flags, and emergency provisioning controls.",
          "This page can affect broad district behavior. Treat every control as operationally sensitive and review warning, scope, and expiration text before changing anything.",
        ],
      },
      {
        heading: "Controls and cards",
        paragraphs: [
          "Start with Sync Health and Admin Warnings. Slow provider convergence, repeated schedule overlap, stale sync times, or high-severity warnings should be investigated before changing write-capable workflow defaults.",
          "Use Deprovisioning Exceptions and Google-active / Aeries-inactive Defaults to confirm which accounts are intentionally held out of normal retirement. Review scope, expiration, and notification behavior before changing those settings.",
          "Use Room Move Reversal only for a completed room-move job that should be fully undone. Open Feature Flags for staged rollout controls.",
        ],
      },
      {
        heading: "Emergency and correction paths",
        paragraphs: [
          "Emergency controls such as Global Pause are for stopping unsafe provisioning or sync behavior while the underlying warning is investigated. They are not ordinary workflow shortcuts.",
          "IT can only fully revert a room move. To partially revert a room move, create a new Room Move draft for the affected employees.",
        ],
      },
    ],
  },
  "/admin/feature-flags": {
    title: "Feature Flags help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "Feature Flags is the IT Admin rollout-control surface for staged behavior. It shows which documented frontend or workflow affordances are enabled for the current environment.",
          "Flags are operational controls, not product documentation. A disabled flag may mean the related workflow is still under review, staged, or intentionally unavailable.",
        ],
      },
      {
        heading: "Controls and statuses",
        paragraphs: [
          "Review the flag name, current state, rollout scope, and any warning text before toggling a control. Use the page only for documented feature flags that already exist for this environment.",
          "Enabled means the flagged behavior is available in the configured environment. Disabled means the behavior should stay hidden or inactive even if related code exists.",
        ],
      },
      {
        heading: "Correction paths",
        paragraphs: [
          "Do not use Feature Flags to bypass missing product decisions, access rules, provider safety, or promotion gates. If a workflow is not defined in the PRD and implementation plan, it should not be exposed by adding a flag.",
          "Use Admin warnings, Reports, or the owning workflow page to investigate why a feature is not safe to enable.",
        ],
      },
    ],
  },
  "/my-profile": {
    title: "My Profile help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "My Profile shows the current staff account context used by the dashboard, including identity, role, and shell visibility information available to the current session.",
          "Use it to confirm which persona or account context is active before comparing route access, dashboard scope, or search results.",
        ],
      },
      {
        heading: "Controls and access",
        paragraphs: [
          "The profile page saves preferred display fields and advisory device preference only to DEV-local mock state in this slice. It does not edit Google, Aeries, Escape, Zoom, IncidentIQ, or local database records.",
          "If account details are wrong, correct the source system that owns the field or use the documented admin workflow for that data class when one exists.",
        ],
      },
      {
        heading: "Security",
        paragraphs: [
          "Do not copy secrets, tokens, service-account JSON, or private provider data into notes or tickets from this page. Diagnostics should use non-secret labels or one-way fingerprints when needed.",
          "Student users are not allowed to use the staff dashboard. If the active account does not match the expected staff role, sign out and use the correct staff account before continuing.",
        ],
      },
    ],
  },
};

const NAV_KEY_DEFAULT_ROUTE = {
  dashboard: "/dashboard/it-admin",
  onboarding: "/onboarding",
  offboarding: "/offboarding",
  departingSeniors: "/departing-seniors",
  roomMoves: "/room-moves",
  phoneDirectory: "/phone-directory/by-person",
  dataQuality: "/data-quality",
  frequentFliers: "/frequent-fliers",
  studentDataCleanup: "/student-data-cleanup",
  reports: "/reports",
  admin: "/admin",
};

const GENERIC_PAGE_HELP = {
  title: "Page help",
  sections: [
    {
      heading: "What this page shows",
      paragraphs: [
        "This staff-only page is part of The WIZARD dashboard. Use the visible page title, table labels, and route context to confirm the workflow before taking action.",
      ],
    },
    {
      heading: "How to use it",
      paragraphs: [
        "Use page controls exactly as labeled, select rows for details when available, and follow the owning source-system correction path shown by the page.",
      ],
    },
  ],
};

export function helpContentForRoute(routePath, navKey = null) {
  if (routePath && HELP_CONTENT_BY_ROUTE[routePath]) {
    return HELP_CONTENT_BY_ROUTE[routePath];
  }
  const fallbackRoute = navKey ? NAV_KEY_DEFAULT_ROUTE[navKey] : null;
  return (fallbackRoute && HELP_CONTENT_BY_ROUTE[fallbackRoute]) || GENERIC_PAGE_HELP;
}

export function helpSourceNoteForRoute(routePath) {
  return ROUTE_SOURCE_NOTES[routePath] ?? null;
}

export const routeHelpContent = HELP_CONTENT_BY_ROUTE;
export const routeHelpSourceNotes = ROUTE_SOURCE_NOTES;
