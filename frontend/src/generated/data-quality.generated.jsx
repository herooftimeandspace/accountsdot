import React from "react";
import artboard from "./data-quality.artboard.json";
import { PenArtboard } from "../lib/PenArtboard";

export const dataQualityDesign = {
  "key": "data-quality",
  "width": 1672,
  "height": 1080,
  "hotspots": {},
  "slots": {
    "shell": {
      "scopeTitle": "t12",
      "scopeSubtitle": "t13",
      "searchPlaceholder": "t25",
      "notificationCount": "t31",
      "userAvatar": "e36",
      "userInitials": "t37",
      "userName": "t38",
      "userRole": "t39",
      "platformStatus": "t98"
    },
    "page": {
      "title": "t99"
    },
    "summaryCards": [
      {
        "title": "t107",
        "count": "t108"
      },
      {
        "title": "t111",
        "count": "t112"
      },
      {
        "title": "t115",
        "count": "t116"
      },
      {
        "title": "t119",
        "count": "t120"
      }
    ],
    "queue": {
      "headers": {
        "issue": "t131",
        "source": "t132",
        "owner": "t133",
        "impact": "t134",
        "nextAction": "t135"
      },
      "rows": [
        {
          "issue": "t137",
          "source": "t138",
          "owner": "t139",
          "impact": [
            "t140"
          ],
          "nextAction": [
            "t141"
          ]
        },
        {
          "issue": "t143",
          "source": "t144",
          "owner": "t145",
          "impact": [
            "t146"
          ],
          "nextAction": [
            "t147"
          ]
        },
        {
          "issue": "t149",
          "source": "t150",
          "owner": "t151",
          "impact": [
            "t152"
          ],
          "nextAction": [
            "t153"
          ]
        },
        {
          "issue": "t155",
          "source": "t156",
          "owner": "t157",
          "impact": [
            "t158"
          ],
          "nextAction": [
            "t159"
          ]
        },
        {
          "issue": "t161",
          "source": "t162",
          "owner": "t163",
          "impact": [
            "t164"
          ],
          "nextAction": [
            "t166"
          ]
        }
      ]
    }
  }
};

export function DataQualityGeneratedView({ textOverrides = {}, hotspots = {}, hiddenNodeIds = [], imageNodeOverrides = {}, renderOverlay = null }) {
  return (
    <PenArtboard
      artboard={artboard}
      textOverrides={textOverrides}
      hotspots={hotspots}
      hiddenNodeIds={hiddenNodeIds}
      imageNodeOverrides={imageNodeOverrides}
      renderOverlay={renderOverlay}
    />
  );
}
