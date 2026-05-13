export const generatedArtboardLoaders = {
  "data-quality": () => import("./data-quality.artboard.json").then((module) => module.default),
  "dashboard-it-admin": () => import("./dashboard-it-admin.artboard.json").then((module) => module.default),
  "dashboard-hr-lifecycle": () => import("./dashboard-hr-lifecycle.artboard.json").then((module) => module.default),
  "dashboard-site-admin": () => import("./dashboard-site-admin.artboard.json").then((module) => module.default),
  "onboarding": () => import("./onboarding.artboard.json").then((module) => module.default),
  "offboarding": () => import("./offboarding.artboard.json").then((module) => module.default),
  "room-moves": () => import("./room-moves.artboard.json").then((module) => module.default),
  "room-moves-bulk-draft": () => import("./room-moves-bulk-draft.artboard.json").then((module) => module.default),
  "phone-directory-by-person": () => import("./phone-directory-by-person.artboard.json").then((module) => module.default),
  "phone-directory-by-room": () => import("./phone-directory-by-room.artboard.json").then((module) => module.default),
  "phone-directory-by-department": () => import("./phone-directory-by-department.artboard.json").then((module) => module.default),
  "frequent-fliers": () => import("./frequent-fliers.artboard.json").then((module) => module.default),
  "student-data-cleanup": () => import("./student-data-cleanup.artboard.json").then((module) => module.default),
  "reports": () => import("./reports.artboard.json").then((module) => module.default),
  "reports-sync-transparency": () => import("./reports-sync-transparency.artboard.json").then((module) => module.default),
  "reports-ticketing-human-work": () => import("./reports-ticketing-human-work.artboard.json").then((module) => module.default),
  "admin": () => import("./admin.artboard.json").then((module) => module.default),
  "admin-feature-flags": () => import("./admin-feature-flags.artboard.json").then((module) => module.default),
  "my-profile": () => import("./my-profile.artboard.json").then((module) => module.default),
  "login": () => import("./login.artboard.json").then((module) => module.default),
  "error-logged-in": () => import("./error-logged-in.artboard.json").then((module) => module.default),
  "error-logged-out": () => import("./error-logged-out.artboard.json").then((module) => module.default),
};

const generatedArtboardCache = new Map();

export function loadGeneratedArtboard(key) {
  const loader = generatedArtboardLoaders[key];
  if (!loader) {
    return Promise.reject(new Error(`Unknown generated artboard: ${key}`));
  }
  if (!generatedArtboardCache.has(key)) {
    generatedArtboardCache.set(key, loader());
  }
  return generatedArtboardCache.get(key);
}

export const generatedArtboardMeta = {
  "data-quality": { key: "data-quality", sourcePen: "docs/mocks/wireframes/wireframe-data-quality-dashboard.pen", activeNav: "dataQuality" },
  "dashboard-it-admin": { key: "dashboard-it-admin", sourcePen: "docs/mocks/wireframes/wireframe-it-admin-overview.pen", activeNav: "dashboard" },
  "dashboard-hr-lifecycle": { key: "dashboard-hr-lifecycle", sourcePen: "docs/mocks/wireframes/wireframe-hr-lifecycle-overview.pen", activeNav: "dashboard" },
  "dashboard-site-admin": { key: "dashboard-site-admin", sourcePen: "docs/mocks/wireframes/wireframe-site-admin-dashboard.pen", activeNav: "dashboard" },
  "onboarding": { key: "onboarding", sourcePen: "docs/mocks/wireframes/wireframe-onboarding-dashboard.pen", activeNav: "onboarding" },
  "offboarding": { key: "offboarding", sourcePen: "docs/mocks/wireframes/wireframe-offboarding-dashboard.pen", activeNav: "offboarding" },
  "room-moves": { key: "room-moves", sourcePen: "docs/mocks/wireframes/wireframe-room-moves.pen", activeNav: "roomMoves" },
  "room-moves-bulk-draft": { key: "room-moves-bulk-draft", sourcePen: "docs/mocks/wireframes/wireframe-room-moves-bulk-draft.pen", activeNav: null },
  "phone-directory-by-person": { key: "phone-directory-by-person", sourcePen: "docs/mocks/wireframes/wireframe-phone-directory-by-person.pen", activeNav: "phoneDirectory" },
  "phone-directory-by-room": { key: "phone-directory-by-room", sourcePen: "docs/mocks/wireframes/wireframe-phone-directory-by-room.pen", activeNav: "phoneDirectory" },
  "phone-directory-by-department": { key: "phone-directory-by-department", sourcePen: "docs/mocks/wireframes/wireframe-phone-directory-by-department.pen", activeNav: "phoneDirectory" },
  "frequent-fliers": { key: "frequent-fliers", sourcePen: "docs/mocks/wireframes/wireframe-device-wrangler-frequent-fliers.pen", activeNav: "frequentFliers" },
  "student-data-cleanup": { key: "student-data-cleanup", sourcePen: "docs/mocks/wireframes/wireframe-site-secretary-student-data-cleanup.pen", activeNav: "studentDataCleanup" },
  "reports": { key: "reports", sourcePen: "docs/mocks/wireframes/wireframe-it-admin-reports.pen", activeNav: "reports" },
  "reports-sync-transparency": { key: "reports-sync-transparency", sourcePen: "docs/mocks/wireframes/wireframe-sync-transparency-dashboard.pen", activeNav: "reports" },
  "reports-ticketing-human-work": { key: "reports-ticketing-human-work", sourcePen: "docs/mocks/wireframes/wireframe-ticketing-human-work.pen", activeNav: "reports" },
  "admin": { key: "admin", sourcePen: "docs/mocks/wireframes/wireframe-it-admin-admin-controls.pen", activeNav: "admin" },
  "admin-feature-flags": { key: "admin-feature-flags", sourcePen: "docs/mocks/wireframes/wireframe-admin-feature-flags.pen", activeNav: "admin" },
  "my-profile": { key: "my-profile", sourcePen: "docs/mocks/wireframes/wireframe-faculty-staff-my-profile.pen", activeNav: null },
  "login": { key: "login", sourcePen: "docs/mocks/wireframes/wireframe-login.pen", activeNav: null },
  "error-logged-in": { key: "error-logged-in", sourcePen: "docs/mocks/wireframes/wireframe-http-error.pen", activeNav: null },
  "error-logged-out": { key: "error-logged-out", sourcePen: "docs/mocks/wireframes/wireframe-http-error.pen", activeNav: null },
};

export const sharedShellSpec = {
  "sharedShellIds": {
    "scopeField": "f11",
    "scopeTitle": "t12",
    "scopeSubtitle": "t13",
    "searchField": "f22",
    "searchIcon": "p23",
    "searchPlaceholder": "t25",
    "notificationBubble": "f30",
    "notificationCount": "t31",
    "helpIcon": "p34",
    "accountBox": "f35",
    "avatar": "e36",
    "initials": "t37",
    "userName": "t38",
    "userRole": "t39",
    "navHighlight": "f64",
    "supportIcon": "p92",
    "supportLabel": "t95",
    "platformStatusLabel": "t96",
    "platformStatusDot": "e97",
    "platformStatusValue": "t98"
  },
  "navGroups": {
    "dashboard": [
      "p41",
      "p42",
      "p43",
      "p44",
      "t45"
    ],
    "onboarding": [
      "p46",
      "p47",
      "p48",
      "p49",
      "t50",
      "p51"
    ],
    "offboarding": [
      "p52",
      "p53",
      "p54",
      "t55",
      "p56"
    ],
    "roomMoves": [
      "p57",
      "p58",
      "p59",
      "p60",
      "t61"
    ],
    "phoneDirectory": [
      "p62",
      "t63"
    ],
    "dataQuality": [
      "p65",
      "p66",
      "p67",
      "t68"
    ],
    "frequentFliers": [
      "p69",
      "p70",
      "p71",
      "p72",
      "t73"
    ],
    "studentDataCleanup": [
      "p74",
      "p75",
      "p76",
      "p77",
      "p78",
      "t79"
    ],
    "reports": [
      "p80",
      "p81",
      "p82",
      "p83",
      "t84",
      "p85"
    ],
    "admin": [
      "p86",
      "p87",
      "p88",
      "p89",
      "t90",
      "p91"
    ]
  },
  "navLabelIds": {
    "dashboard": "t45",
    "onboarding": "t50",
    "offboarding": "t55",
    "roomMoves": "t61",
    "phoneDirectory": "t63",
    "dataQuality": "t68",
    "frequentFliers": "t73",
    "studentDataCleanup": "t79",
    "reports": "t84",
    "admin": "t90"
  }
};

export const implementedPageDesignManifest = {
  "schemaVersion": 1,
  "sourceOfTruth": [
    "README.md",
    "IMPLEMENTATION_PLAN.md",
    "PRODUCT_REQUIREMENTS.md",
    "TEST_MATRIX.md",
    "AGENTS.md",
    "docs/mocks/wireframes/implemented-page-design-contract.md"
  ],
  "generatedBy": "scripts/sync_implemented_pages.mjs",
  "generatedFileGlobs": [
    "frontend/src/generated/*.artboard.json",
    "frontend/src/generated/artboards.generated.js",
    "frontend/src/generated/data-quality.generated.jsx",
    "frontend/src/generated/implemented-page-design-manifest.generated.json"
  ],
  "artboards": [
    {
      "key": "data-quality",
      "sourcePen": "docs/mocks/wireframes/wireframe-data-quality-dashboard.pen",
      "mode": "merge-shell",
      "activeNav": "dataQuality",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "dashboard-it-admin",
      "sourcePen": "docs/mocks/wireframes/wireframe-it-admin-overview.pen",
      "mode": "merge-shell",
      "activeNav": "dashboard",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "dashboard-hr-lifecycle",
      "sourcePen": "docs/mocks/wireframes/wireframe-hr-lifecycle-overview.pen",
      "mode": "merge-shell",
      "activeNav": "dashboard",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "dashboard-site-admin",
      "sourcePen": "docs/mocks/wireframes/wireframe-site-admin-dashboard.pen",
      "mode": "merge-shell",
      "activeNav": "dashboard",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "onboarding",
      "sourcePen": "docs/mocks/wireframes/wireframe-onboarding-dashboard.pen",
      "mode": "merge-shell",
      "activeNav": "onboarding",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "offboarding",
      "sourcePen": "docs/mocks/wireframes/wireframe-offboarding-dashboard.pen",
      "mode": "merge-shell",
      "activeNav": "offboarding",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "room-moves",
      "sourcePen": "docs/mocks/wireframes/wireframe-room-moves.pen",
      "mode": "merge-shell",
      "activeNav": "roomMoves",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "room-moves-bulk-draft",
      "sourcePen": "docs/mocks/wireframes/wireframe-room-moves-bulk-draft.pen",
      "mode": "merge-shell",
      "activeNav": null,
      "loggedInShell": true,
      "standardPrimitives": []
    },
    {
      "key": "phone-directory-by-person",
      "sourcePen": "docs/mocks/wireframes/wireframe-phone-directory-by-person.pen",
      "mode": "merge-shell",
      "activeNav": "phoneDirectory",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "phone-directory-by-room",
      "sourcePen": "docs/mocks/wireframes/wireframe-phone-directory-by-room.pen",
      "mode": "merge-shell",
      "activeNav": "phoneDirectory",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "phone-directory-by-department",
      "sourcePen": "docs/mocks/wireframes/wireframe-phone-directory-by-department.pen",
      "mode": "merge-shell",
      "activeNav": "phoneDirectory",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "frequent-fliers",
      "sourcePen": "docs/mocks/wireframes/wireframe-device-wrangler-frequent-fliers.pen",
      "mode": "merge-shell",
      "activeNav": "frequentFliers",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "student-data-cleanup",
      "sourcePen": "docs/mocks/wireframes/wireframe-site-secretary-student-data-cleanup.pen",
      "mode": "merge-shell",
      "activeNav": "studentDataCleanup",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "reports",
      "sourcePen": "docs/mocks/wireframes/wireframe-it-admin-reports.pen",
      "mode": "merge-shell",
      "activeNav": "reports",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "reports-sync-transparency",
      "sourcePen": "docs/mocks/wireframes/wireframe-sync-transparency-dashboard.pen",
      "mode": "merge-shell",
      "activeNav": "reports",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "reports-ticketing-human-work",
      "sourcePen": "docs/mocks/wireframes/wireframe-ticketing-human-work.pen",
      "mode": "merge-shell",
      "activeNav": "reports",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "admin",
      "sourcePen": "docs/mocks/wireframes/wireframe-it-admin-admin-controls.pen",
      "mode": "merge-shell",
      "activeNav": "admin",
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "admin-feature-flags",
      "sourcePen": "docs/mocks/wireframes/wireframe-admin-feature-flags.pen",
      "mode": "merge-shell",
      "activeNav": "admin",
      "loggedInShell": true,
      "standardPrimitives": []
    },
    {
      "key": "my-profile",
      "sourcePen": "docs/mocks/wireframes/wireframe-faculty-staff-my-profile.pen",
      "mode": "merge-shell",
      "activeNav": null,
      "loggedInShell": true,
      "standardPrimitives": [
        "refresh"
      ]
    },
    {
      "key": "login",
      "sourcePen": "docs/mocks/wireframes/wireframe-login.pen",
      "mode": "passthrough",
      "activeNav": null,
      "loggedInShell": false,
      "standardPrimitives": []
    },
    {
      "key": "error-logged-in",
      "sourcePen": "docs/mocks/wireframes/wireframe-http-error.pen",
      "mode": "error-logged-in",
      "activeNav": null,
      "loggedInShell": true,
      "standardPrimitives": []
    },
    {
      "key": "error-logged-out",
      "sourcePen": "docs/mocks/wireframes/wireframe-http-error.pen",
      "mode": "error-logged-out",
      "activeNav": null,
      "loggedInShell": false,
      "standardPrimitives": []
    }
  ],
  "sharedShell": {
    "sourcePen": "docs/mocks/wireframes/wireframe-shared-shell.pen",
    "loggedInPane": {
      "x": 264,
      "y": 76
    },
    "sharedShellIds": {
      "scopeField": "f11",
      "scopeTitle": "t12",
      "scopeSubtitle": "t13",
      "searchField": "f22",
      "searchIcon": "p23",
      "searchPlaceholder": "t25",
      "notificationBubble": "f30",
      "notificationCount": "t31",
      "helpIcon": "p34",
      "accountBox": "f35",
      "avatar": "e36",
      "initials": "t37",
      "userName": "t38",
      "userRole": "t39",
      "navHighlight": "f64",
      "supportIcon": "p92",
      "supportLabel": "t95",
      "platformStatusLabel": "t96",
      "platformStatusDot": "e97",
      "platformStatusValue": "t98"
    },
    "navGroups": {
      "dashboard": [
        "p41",
        "p42",
        "p43",
        "p44",
        "t45"
      ],
      "onboarding": [
        "p46",
        "p47",
        "p48",
        "p49",
        "t50",
        "p51"
      ],
      "offboarding": [
        "p52",
        "p53",
        "p54",
        "t55",
        "p56"
      ],
      "roomMoves": [
        "p57",
        "p58",
        "p59",
        "p60",
        "t61"
      ],
      "phoneDirectory": [
        "p62",
        "t63"
      ],
      "dataQuality": [
        "p65",
        "p66",
        "p67",
        "t68"
      ],
      "frequentFliers": [
        "p69",
        "p70",
        "p71",
        "p72",
        "t73"
      ],
      "studentDataCleanup": [
        "p74",
        "p75",
        "p76",
        "p77",
        "p78",
        "t79"
      ],
      "reports": [
        "p80",
        "p81",
        "p82",
        "p83",
        "t84",
        "p85"
      ],
      "admin": [
        "p86",
        "p87",
        "p88",
        "p89",
        "t90",
        "p91"
      ]
    },
    "navLabelIds": {
      "dashboard": "t45",
      "onboarding": "t50",
      "offboarding": "t55",
      "roomMoves": "t61",
      "phoneDirectory": "t63",
      "dataQuality": "t68",
      "frequentFliers": "t73",
      "studentDataCleanup": "t79",
      "reports": "t84",
      "admin": "t90"
    }
  },
  "standardPrimitives": {
    "refresh": {
      "label": "Refresh",
      "role": "standard-header-action",
      "frame": {
        "x": 1540,
        "y": 90,
        "width": 112,
        "height": 38,
        "fill": "#CEB770",
        "stroke": "#CEB770",
        "cornerRadius": 8
      },
      "text": {
        "y": 101,
        "fontSize": 13,
        "fontWeight": "700",
        "fill": "#01161E",
        "textAlign": "center"
      }
    }
  },
  "lintPolicy": {
    "initialPosture": "warn broadly, fail high-confidence regressions",
    "warningPromotion": "Promote stable warning checks to failures after false positives are resolved.",
    "minimumVisualGapPx": 5,
    "recoveryLayers": [
      "pipeline",
      ".pen layout",
      "docs/new behavior",
      "runtime behavior",
      "review artifact"
    ]
  }
};
