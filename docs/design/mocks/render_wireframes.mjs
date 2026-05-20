import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.join(__dirname, "wireframes");
const lucideVendorDir = path.join(__dirname, "..", "vendor", "lucide-static", "icons");
const brandLogoPath = path.join(
  __dirname,
  "..",
  "reference-inputs",
  "branding",
  "Wordmarks",
  "Gold W black outline.png"
);
const brandLogoDataUri = `data:image/png;base64,${fs.readFileSync(brandLogoPath).toString("base64")}`;

const W = 1672;
const H = 1080;
const NAV_W = 264;
const TOP_H = 76;
const MAIN_X = 288;
const MAIN_W = W - MAIN_X - 24;

const colors = {
  bg: "#FFFFFF",
  appBg: "#FBFBFD",
  panel: "#FFFFFF",
  text: "#111827",
  subtext: "#4B5563",
  faint: "#6B7280",
  line: "#E5E7EB",
  lineSoft: "#EEF2F7",
  gold: "#E6B64C",
  goldSoft: "#FFF3D6",
  goldDeep: "#A16207",
  blue: "#2563EB",
  blueSoft: "#EAF1FF",
  green: "#16A34A",
  greenSoft: "#E8F7EB",
  red: "#DC2626",
  redSoft: "#FDECEC",
  orange: "#EA580C",
  orangeSoft: "#FFF1E7",
  purple: "#7C3AED",
  purpleSoft: "#F2EBFF",
  cyan: "#0891B2",
  cyanSoft: "#E8FAFF",
  shadow: "#F3F4F6",
};

const fontFamily = "Arial, Helvetica, sans-serif";
const penFont = "Arial";
const brandTitle = "The WIZARD";
const brandSubtitle = [
  "Windsor Identity Zync,",
  "Access, & Retirement Dashboard",
];
const brandTagline = "Have you checked with The WIZARD?";
const implementedPageFiles = new Set(["wireframe-data-quality-dashboard"]);

const navSets = {
  it: [
    "Dashboard",
    "Staff Onboarding",
    "Offboarding",
    "Room Moves",
    "Phone Directory",
    "Data Quality",
    "Frequent Fliers",
    "Cleanup",
    "Reports",
    "Admin",
  ],
  hr: [
    "Dashboard",
    "Staff Onboarding",
    "Offboarding",
    "Manual Intake",
    "Preferred Name Requests",
    "Provisioning Profiles",
    "Phone Directory",
    "Data Quality",
    "Reports",
  ],
  siteAdmin: [
    "Dashboard",
    "Staff Onboarding",
    "Offboarding",
    "Room Moves",
    "Phone Directory",
    "Frequent Fliers",
    "Reports",
  ],
  secretary: [
    "Dashboard",
    "Invalid Student Names",
    "Room Moves",
    "Phone Directory",
    "Reports",
  ],
  wrangler: ["Dashboard", "Frequent Fliers", "Phone Directory", "Reports"],
  faculty: ["My Profile", "Preferred Name", "Phone Directory"],
};

const lucideIcons = {
  Dashboard: "layout-grid",
  "Staff Onboarding": "user-plus",
  Offboarding: "user-minus",
  "Room Moves": "arrow-left-right",
  "Phone Directory": "phone",
  "Data Quality": "triangle-alert",
  "Frequent Fliers": "sparkles",
  Cleanup: "trash-2",
  Reports: "chart-column",
  Admin: "settings-2",
  "Manual Intake": "square-pen",
  "Preferred Name Requests": "star",
  "Provisioning Profiles": "files",
  "Invalid Student Names": "triangle-alert",
  "My Profile": "contact-round",
  "Preferred Name": "star",
  scope: "building-2",
  search: "search",
  alert: "bell",
  help: "circle-help",
  close: "x",
  expand: "chevron-down",
  info: "info",
  refresh: "refresh-cw",
  warning: "triangle-alert"
};

const lucideCache = new Map();

const esc = (value) =>
  String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");

function textWidth(text, size) {
  return String(text).length * size * 0.54;
}

function truncateToWidth(text, maxWidth, size) {
  const source = String(text);
  if (textWidth(source, size) <= maxWidth) return source;
  let value = source;
  while (value.length > 1 && textWidth(`${value}...`, size) > maxWidth) {
    value = value.slice(0, -1);
  }
  return `${value}...`;
}

function wrapToken(token, maxWidth, size) {
  const value = String(token);
  if (!value) return [""];
  if (textWidth(value, size) <= maxWidth) return [value];
  const parts = [];
  let current = "";
  for (const char of value) {
    const candidate = `${current}${char}`;
    if (current && textWidth(candidate, size) > maxWidth) {
      parts.push(current);
      current = char;
    } else {
      current = candidate;
    }
  }
  if (current) parts.push(current);
  return parts;
}

function wrapText(text, maxWidth, size) {
  const raw = String(text).trim();
  if (!raw) return [""];
  const words = raw.split(/\s+/);
  const lines = [];
  let current = "";
  for (const word of words) {
    if (textWidth(word, size) > maxWidth) {
      if (current) {
        lines.push(current);
        current = "";
      }
      lines.push(...wrapToken(word, maxWidth, size));
      continue;
    }
    const candidate = current ? `${current} ${word}` : word;
    if (current && textWidth(candidate, size) > maxWidth) {
      lines.push(current);
      current = word;
    } else {
      current = candidate;
    }
  }
  if (current) lines.push(current);
  return lines.length ? lines : [raw];
}

function wrapLines(lines, maxWidth, size, maxLines = Infinity) {
  const prepared = [];
  for (const line of linesFor(lines)) {
    const subLines = String(line).split("\n");
    for (const part of subLines) {
      if (!maxWidth) {
        prepared.push(part);
      } else {
        prepared.push(...wrapText(part, maxWidth, size));
      }
    }
  }
  if (prepared.length <= maxLines) return prepared;
  const clipped = prepared.slice(0, maxLines);
  clipped[maxLines - 1] = truncateToWidth(clipped[maxLines - 1], maxWidth, size);
  return clipped;
}

function textBlockHeight(lines, size, lineHeight, maxWidth, maxLines = Infinity) {
  return wrapLines(lines, maxWidth, size, maxLines).length * lineHeight;
}

function linesFor(cell) {
  if (cell == null) return [];
  if (Array.isArray(cell)) return cell.map(String);
  if (typeof cell === "object") {
    if (cell.lines) return cell.lines.map(String);
    if (cell.text) return [String(cell.text)];
  }
  return [String(cell)];
}

function slugTitle(slug) {
  return slug
    .replace(/^wireframe-/, "")
    .split("-")
    .map((part) => part[0].toUpperCase() + part.slice(1))
    .join(" ");
}

function parseSvgAttributes(fragment) {
  const attrs = {};
  const re = /([:@A-Za-z_][\w:.-]*)="([^"]*)"/g;
  let match;
  while ((match = re.exec(fragment))) {
    attrs[match[1]] = match[2];
  }
  return attrs;
}

function rectGeometry(x, y, w, h) {
  return `M ${x} ${y} H ${x + w} V ${y + h} H ${x} Z`;
}

function circleGeometry(cx, cy, r) {
  return `M ${cx + r} ${cy} A ${r} ${r} 0 1 1 ${cx - r} ${cy} A ${r} ${r} 0 1 1 ${cx + r} ${cy}`;
}

function pointsGeometry(points, close = false) {
  const parts = String(points)
    .trim()
    .split(/\s+/)
    .map((point) => point.split(",").map(Number))
    .filter((pair) => pair.length === 2 && pair.every((value) => !Number.isNaN(value)));
  if (!parts.length) return null;
  const [first, ...rest] = parts;
  return `M ${first[0]} ${first[1]} ${rest.map(([x, y]) => `L ${x} ${y}`).join(" ")}${close ? " Z" : ""}`.trim();
}

function geometryForLucideElement(tag, attrs) {
  if (tag === "path") return attrs.d ?? null;
  if (tag === "line") return `M ${attrs.x1} ${attrs.y1} L ${attrs.x2} ${attrs.y2}`;
  if (tag === "circle") return circleGeometry(Number(attrs.cx), Number(attrs.cy), Number(attrs.r));
  if (tag === "rect") return rectGeometry(Number(attrs.x ?? 0), Number(attrs.y ?? 0), Number(attrs.width), Number(attrs.height));
  if (tag === "polyline") return pointsGeometry(attrs.points, false);
  if (tag === "polygon") return pointsGeometry(attrs.points, true);
  return null;
}

function loadLucideIcon(name) {
  if (lucideCache.has(name)) return lucideCache.get(name);
  const iconPath = path.join(lucideVendorDir, `${name}.svg`);
  if (!fs.existsSync(iconPath)) {
    throw new Error(`Vendored Lucide icon not found: ${iconPath}`);
  }
  const source = fs.readFileSync(iconPath, "utf8");
  const rootMatch = source.match(/<svg\b([^>]*)>/m);
  if (!rootMatch) {
    throw new Error(`Invalid Lucide SVG: ${name}`);
  }
  const rootAttrs = parseSvgAttributes(rootMatch[1]);
  const viewBox = String(rootAttrs.viewBox ?? "0 0 24 24").split(/\s+/).map(Number);
  const defaults = {
    strokeWidth: Number(rootAttrs["stroke-width"] ?? 2),
    linecap: rootAttrs["stroke-linecap"] ?? "round",
    linejoin: rootAttrs["stroke-linejoin"] ?? "round",
    stroke: rootAttrs.stroke ?? "currentColor",
    fill: rootAttrs.fill ?? "none",
  };
  const shapes = [];
  const elementRe = /<(path|line|circle|rect|polyline|polygon)\b([^>]*)\/>/g;
  let match;
  while ((match = elementRe.exec(source))) {
    const tag = match[1];
    const attrs = parseSvgAttributes(match[2]);
    const geometry = geometryForLucideElement(tag, attrs);
    if (!geometry) continue;
    shapes.push({
      geometry,
      strokeWidth: Number(attrs["stroke-width"] ?? defaults.strokeWidth),
      linecap: attrs["stroke-linecap"] ?? defaults.linecap,
      linejoin: attrs["stroke-linejoin"] ?? defaults.linejoin,
      stroke: attrs.stroke ?? defaults.stroke,
      fill: attrs.fill ?? defaults.fill,
    });
  }
  const icon = { name, viewBox, shapes };
  lucideCache.set(name, icon);
  return icon;
}

class Scene {
  constructor(name) {
    this.name = name;
    this.nodes = [];
    this.penId = 1;
  }

  nextId(prefix) {
    this.penId += 1;
    return `${prefix}${this.penId}`;
  }

  rect(x, y, w, h, opts = {}) {
    this.nodes.push({
      type: "rect",
      x,
      y,
      w,
      h,
      fill: opts.fill ?? colors.panel,
      stroke: opts.stroke ?? null,
      strokeWidth: opts.strokeWidth ?? 1,
      radius: opts.radius ?? 8,
    });
  }

  line(x1, y1, x2, y2, opts = {}) {
    this.nodes.push({
      type: "line",
      x1,
      y1,
      x2,
      y2,
      stroke: opts.stroke ?? colors.lineSoft,
      strokeWidth: opts.strokeWidth ?? 1,
    });
  }

  path(geometry, x, y, w, h, opts = {}) {
    this.nodes.push({
      type: "path",
      geometry: String(geometry),
      x,
      y,
      w,
      h,
      viewBox: opts.viewBox ?? [0, 0, 24, 24],
      fill: opts.fill ?? null,
      stroke: opts.stroke ?? colors.subtext,
      strokeWidth: opts.strokeWidth ?? 2,
      linecap: opts.linecap ?? "round",
      linejoin: opts.linejoin ?? "round",
    });
  }

  text(text, x, y, opts = {}) {
    this.nodes.push({
      type: "text",
      text: String(text),
      x,
      y,
      size: opts.size ?? 14,
      weight: opts.weight ?? 400,
      fill: opts.fill ?? colors.text,
      anchor: opts.anchor ?? "start",
      width: opts.width ?? null,
      family: opts.family ?? fontFamily,
      penFamily: opts.penFamily ?? penFont,
    });
  }

  icon(name, x, y, size, opts = {}) {
    const icon = loadLucideIcon(name);
    const width = opts.width ?? size;
    const height = opts.height ?? size;
    for (const shape of icon.shapes) {
      this.path(shape.geometry, x, y, width, height, {
        viewBox: icon.viewBox,
        fill: shape.fill !== "none" ? shape.fill : null,
        stroke: opts.stroke ?? colors.subtext,
        strokeWidth: opts.strokeWidth ?? shape.strokeWidth,
        linecap: shape.linecap,
        linejoin: shape.linejoin,
      });
    }
  }

  multiline(lines, x, y, opts = {}) {
    const lineHeight = opts.lineHeight ?? Math.round((opts.size ?? 14) * 1.4);
    const prepared = opts.width
      ? wrapLines(lines, opts.width, opts.size ?? 14, opts.maxLines ?? Infinity)
      : linesFor(lines);
    prepared.forEach((line, index) => {
      this.text(line, x, y + index * lineHeight, opts);
    });
    return prepared.length * lineHeight;
  }

  pill(text, x, y, w, opts = {}) {
    const h = opts.height ?? 26;
    this.rect(x, y, w, h, {
      fill: opts.fill ?? colors.goldSoft,
      stroke: opts.stroke ?? colors.gold,
      radius: opts.radius ?? 7,
    });
    this.text(text, x + w / 2, y + 17, {
      size: opts.size ?? 12,
      weight: opts.weight ?? 700,
      fill: opts.color ?? colors.goldDeep,
      anchor: "middle",
    });
  }

  circle(x, y, d, opts = {}) {
    this.nodes.push({
      type: "circle",
      x,
      y,
      d,
      fill: opts.fill ?? colors.goldSoft,
      stroke: opts.stroke ?? null,
      strokeWidth: opts.strokeWidth ?? 1,
    });
  }

  image(x, y, w, h, href, opts = {}) {
    this.nodes.push({
      type: "image",
      x,
      y,
      w,
      h,
      href,
      penHref: opts.penHref ?? href,
      radius: opts.radius ?? 0,
      mode: opts.mode ?? "fill",
      stroke: opts.stroke ?? null,
      strokeWidth: opts.strokeWidth ?? 1,
      background: opts.background ?? null,
    });
  }

  svg() {
    const parts = [
      '<?xml version="1.0" encoding="UTF-8"?>',
      `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">`,
      `<rect width="${W}" height="${H}" fill="${colors.appBg}"/>`,
    ];
    for (const node of this.nodes) {
      if (node.type === "rect") {
        parts.push(
          `<rect x="${node.x}" y="${node.y}" width="${node.w}" height="${node.h}" rx="${node.radius}" fill="${node.fill}"${node.stroke ? ` stroke="${node.stroke}" stroke-width="${node.strokeWidth}"` : ""}/>`
        );
      } else if (node.type === "line") {
        parts.push(
          `<line x1="${node.x1}" y1="${node.y1}" x2="${node.x2}" y2="${node.y2}" stroke="${node.stroke}" stroke-width="${node.strokeWidth}"/>`
        );
      } else if (node.type === "text") {
        parts.push(
          `<text x="${node.x}" y="${node.y}" font-family="${esc(node.family)}" font-size="${node.size}" font-weight="${node.weight}" text-anchor="${node.anchor}" fill="${node.fill}">${esc(node.text)}</text>`
        );
      } else if (node.type === "circle") {
        parts.push(
          `<circle cx="${node.x + node.d / 2}" cy="${node.y + node.d / 2}" r="${node.d / 2}" fill="${node.fill}"${node.stroke ? ` stroke="${node.stroke}" stroke-width="${node.strokeWidth}"` : ""}/>`
        );
      } else if (node.type === "image") {
        if (node.radius > 0) {
          const clipId = `clip-${parts.length}`;
          parts.push(`<defs><clipPath id="${clipId}"><rect x="${node.x}" y="${node.y}" width="${node.w}" height="${node.h}" rx="${node.radius}"/></clipPath></defs>`);
          if (node.background) {
            parts.push(
              `<rect x="${node.x}" y="${node.y}" width="${node.w}" height="${node.h}" rx="${node.radius}" fill="${node.background}"${node.stroke ? ` stroke="${node.stroke}" stroke-width="${node.strokeWidth}"` : ""}/>`
            );
          }
          parts.push(
            `<image x="${node.x}" y="${node.y}" width="${node.w}" height="${node.h}" href="${esc(node.href)}" clip-path="url(#${clipId})" preserveAspectRatio="xMidYMid meet"/>`
          );
        } else {
          parts.push(
            `<image x="${node.x}" y="${node.y}" width="${node.w}" height="${node.h}" href="${esc(node.href)}" preserveAspectRatio="xMidYMid meet"/>`
          );
        }
      } else if (node.type === "path") {
        const [vx, vy, vw, vh] = node.viewBox;
        parts.push(
          `<svg x="${node.x}" y="${node.y}" width="${node.w}" height="${node.h}" viewBox="${vx} ${vy} ${vw} ${vh}"><path d="${esc(node.geometry)}" fill="${node.fill ?? "none"}"${node.stroke ? ` stroke="${node.stroke}" stroke-width="${node.strokeWidth}" stroke-linecap="${node.linecap}" stroke-linejoin="${node.linejoin}"` : ""}/></svg>`
        );
      }
    }
    parts.push("</svg>");
    return `${parts.join("\n")}\n`;
  }

  pen() {
    const root = {
      type: "frame",
      id: "root",
      name: this.name,
      x: 0,
      y: 0,
      width: W,
      height: H,
      fill: colors.appBg,
      cornerRadius: 0,
      layout: "none",
      children: [],
    };
    for (const node of this.nodes) {
      if (node.type === "rect") {
        root.children.push({
          type: "frame",
          id: this.nextId("f"),
          name: "Rect",
          x: Math.round(node.x),
          y: Math.round(node.y),
          width: Math.round(node.w),
          height: Math.round(node.h),
          fill: node.fill,
          cornerRadius: node.radius,
          layout: "none",
          ...(node.stroke
            ? {
                stroke: {
                  thickness: node.strokeWidth,
                  fill: node.stroke,
                  align: "inside",
                },
              }
            : {}),
        });
      } else if (node.type === "line") {
        const x = Math.min(node.x1, node.x2);
        const y = Math.min(node.y1, node.y2);
        const w = Math.max(1, Math.abs(node.x2 - node.x1) || node.strokeWidth);
        const h = Math.max(1, Math.abs(node.y2 - node.y1) || node.strokeWidth);
        root.children.push({
          type: "frame",
          id: this.nextId("l"),
          name: "Line",
          x: Math.round(x),
          y: Math.round(y),
          width: Math.round(w),
          height: Math.round(h),
          fill: node.stroke,
          cornerRadius: 0,
          layout: "none",
        });
      } else if (node.type === "text") {
        const width = Math.max(20, Math.ceil(node.width ?? textWidth(node.text, node.size)));
        let x = node.x;
        if (node.anchor === "middle") x -= width / 2;
        if (node.anchor === "end") x -= width;
        root.children.push({
          type: "text",
          id: this.nextId("t"),
          content: node.text,
          x: Math.round(x),
          y: Math.round(node.y - node.size),
          fontFamily: node.penFamily ?? penFont,
          fontSize: node.size,
          fontWeight: String(node.weight),
          fill: node.fill,
          width,
          textGrowth: "auto",
        });
      } else if (node.type === "circle") {
        root.children.push({
          type: "ellipse",
          id: this.nextId("e"),
          x: Math.round(node.x),
          y: Math.round(node.y),
          width: Math.round(node.d),
          height: Math.round(node.d),
          fill: node.fill,
          ...(node.stroke
            ? {
                stroke: {
                  thickness: node.strokeWidth,
                  fill: node.stroke,
                  align: "inside",
                },
              }
            : {}),
        });
      } else if (node.type === "image") {
        root.children.push({
          type: "frame",
          id: this.nextId("img"),
          name: "Image",
          x: Math.round(node.x),
          y: Math.round(node.y),
          width: Math.round(node.w),
          height: Math.round(node.h),
          cornerRadius: node.radius,
          layout: "none",
          ...(node.background ? { fill: node.background } : {}),
          fill: {
            type: "image",
            enabled: true,
            url: node.penHref,
            mode: node.mode,
          },
          ...(node.stroke
            ? {
                stroke: {
                  thickness: node.strokeWidth,
                  fill: node.stroke,
                  align: "inside",
                },
              }
            : {}),
        });
      } else if (node.type === "path") {
        const pathNode = {
          type: "path",
          id: this.nextId("p"),
          x: Math.round(node.x),
          y: Math.round(node.y),
          width: Math.round(node.w),
          height: Math.round(node.h),
          geometry: node.geometry,
          viewBox: node.viewBox,
        };
        if (node.fill) {
          pathNode.fill = node.fill;
        }
        if (node.stroke) {
          pathNode.stroke = {
            thickness: node.strokeWidth,
            fill: node.stroke,
            align: "center",
            cap: node.linecap,
            join: node.linejoin,
          };
        }
        root.children.push(pathNode);
      }
    }
    return JSON.stringify({ version: "2.11", children: [root] }, null, 2);
  }
}

function navKindFor(role) {
  if (role === "Human Resources") return "hr";
  if (role === "Site Admin") return "siteAdmin";
  if (role === "Site Secretary") return "secretary";
  if (role === "Device Wrangler") return "wrangler";
  if (role === "Faculty / Staff") return "faculty";
  return "it";
}

function iconCenter(scene, name, cx, cy, size, opts = {}) {
  scene.icon(name, cx - size / 2, cy - size / 2, size, opts);
}

function navGlyph(scene, item, x, y, color = colors.subtext) {
  const iconName = lucideIcons[item];
  if (!iconName) return;
  iconCenter(scene, iconName, x, y, 18, { stroke: color, strokeWidth: 2 });
}

function chevronDown(scene, cx, cy, opts = {}) {
  iconCenter(scene, lucideIcons.expand, cx, cy, opts.size ?? 12, {
    stroke: opts.stroke ?? colors.subtext,
    strokeWidth: opts.strokeWidth ?? 2,
  });
}

function dropdownButton(scene, x, y, w, h, label, opts = {}) {
  const leftPad = opts.leftPad ?? 16;
  const rightPad = opts.rightPad ?? 34;
  const fill = opts.fill ?? colors.bg;
  const stroke = opts.stroke ?? colors.line;
  const radius = opts.radius ?? 8;
  const size = opts.size ?? 12;
  const weight = opts.weight ?? 700;
  const fillColor = opts.textColor ?? colors.text;
  const iconColor = opts.iconColor ?? colors.subtext;
  const align = opts.align ?? "center";
  const textWidthLimit = Math.max(32, w - leftPad - rightPad);
  const text = truncateToWidth(label, textWidthLimit, size);
  scene.rect(x, y, w, h, { fill, stroke, radius });
  if (align === "center") {
    const centerX = x + leftPad + textWidthLimit / 2;
    scene.text(text, centerX, y + Math.round(h / 2) + 5, {
      size,
      weight,
      fill: fillColor,
      anchor: "middle",
    });
  } else {
    scene.text(text, x + leftPad, y + Math.round(h / 2) + 5, {
      size,
      weight,
      fill: fillColor,
      width: textWidthLimit,
    });
  }
  chevronDown(scene, x + w - 18, y + Math.round(h / 2) - 1, {
    stroke: iconColor,
    size: opts.iconSize ?? 8,
    strokeWidth: opts.iconStrokeWidth ?? 2,
  });
}

function dropdownValueField(scene, x, y, w, h, label, value, opts = {}) {
  const leftPad = opts.leftPad ?? 16;
  const rightPad = opts.rightPad ?? 34;
  const labelGap = opts.labelGap ?? 14;
  const fill = opts.fill ?? colors.bg;
  const stroke = opts.stroke ?? colors.line;
  const radius = opts.radius ?? 8;
  const labelSize = opts.labelSize ?? 12;
  const valueSize = opts.valueSize ?? 12;
  const iconColor = opts.iconColor ?? colors.subtext;
  const labelWidth = Math.min(w * 0.45, textWidth(label, labelSize) + 8);
  const valueMaxWidth = Math.max(24, w - leftPad - rightPad - labelWidth - labelGap);
  scene.rect(x, y, w, h, { fill, stroke, radius });
  scene.text(truncateToWidth(label, labelWidth, labelSize), x + leftPad, y + Math.round(h / 2) + 5, {
    size: labelSize,
    weight: 700,
    fill: colors.text,
    width: labelWidth,
  });
  scene.text(truncateToWidth(value, valueMaxWidth, valueSize), x + w - rightPad, y + Math.round(h / 2) + 5, {
    size: valueSize,
    fill: colors.subtext,
    anchor: "end",
  });
  chevronDown(scene, x + w - 18, y + Math.round(h / 2) - 1, {
    stroke: iconColor,
    size: opts.iconSize ?? 8,
    strokeWidth: opts.iconStrokeWidth ?? 2,
  });
}

function searchField(scene, x, y, w, h, placeholder, opts = {}) {
  const leftPad = opts.leftPad ?? 20;
  const rightPad = opts.rightPad ?? 16;
  const iconSize = opts.iconSize ?? 16;
  const iconGap = opts.iconGap ?? 12;
  const textMaxWidth = Math.max(32, w - leftPad - rightPad - iconSize - iconGap);
  scene.rect(x, y, w, h, {
    fill: opts.fill ?? colors.bg,
    stroke: opts.stroke ?? colors.line,
    radius: opts.radius ?? 8,
  });
  iconCenter(scene, lucideIcons.search, x + leftPad + iconSize / 2, y + Math.round(h / 2) + 1, iconSize, {
    stroke: opts.iconColor ?? colors.faint,
    strokeWidth: 2,
  });
  scene.text(truncateToWidth(placeholder, textMaxWidth, opts.size ?? 12), x + leftPad + iconSize + iconGap, y + Math.round(h / 2) + 5, {
    size: opts.size ?? 12,
    fill: opts.textColor ?? colors.faint,
    width: textMaxWidth,
  });
}

function shell(scene, spec) {
  scene.rect(0, 0, W, H, { fill: colors.bg, stroke: colors.bg, radius: 0 });
  scene.rect(NAV_W, 0, W - NAV_W, TOP_H, { fill: colors.bg, stroke: colors.line, radius: 0 });
  scene.rect(0, 0, NAV_W, H, { fill: colors.bg, stroke: colors.line, radius: 0 });

  scene.image(18, 14, 34, 22, brandLogoDataUri, { penHref: brandLogoPath });
  scene.text(brandTitle, 58, 34, { size: 24, weight: 800, width: 182 });
  scene.multiline(brandSubtitle, 58, 54, {
    size: 10,
    weight: 600,
    fill: colors.subtext,
    lineHeight: 13,
    width: 184,
    maxLines: 2,
  });
  scene.text(brandTagline, 58, 86, {
    size: 10,
    fill: colors.goldDeep,
    width: 184,
  });
  scene.line(16, 108, NAV_W - 16, 108, { stroke: colors.lineSoft });

  const scopeLabel = spec.scopeLabel ?? "District-wide";
  const scopeSub = spec.scopeSub ?? "All Sites";
  scene.rect(740, 10, 220, 50, { fill: colors.bg, stroke: colors.line });
  const scopeTextWidth = 220 - 78;
  scene.text(truncateToWidth(scopeLabel, scopeTextWidth, 14), 792, 33, { size: 14, weight: 700 });
  scene.text(truncateToWidth(scopeSub, scopeTextWidth, 12), 792, 52, { size: 12, fill: colors.faint });
  chevronDown(scene, 934, 34, { stroke: colors.subtext, size: 8 });
  iconCenter(scene, lucideIcons.scope, 770, 40, 18, { stroke: colors.subtext, strokeWidth: 2 });

  if (spec.search !== false) {
    searchField(scene, 980, 10, 404, 50, spec.searchPlaceholder ?? "Search by name, email, phone, extension, or ID...", {
      iconSize: 18,
      leftPad: 18,
      rightPad: 64,
    });
    scene.rect(1322, 23, 40, 24, { fill: colors.bg, stroke: colors.line, radius: 6 });
    scene.text("K", 1342, 40, { size: 11, weight: 700, fill: colors.faint, anchor: "middle" });
  }

  iconCenter(scene, lucideIcons.alert, 1412, 40, 22, { stroke: colors.text, strokeWidth: 2.1 });
  scene.pill(spec.alertCount ?? "3", 1422, 16, 20, {
    fill: colors.gold,
    stroke: colors.gold,
    color: colors.bg,
    height: 20,
    radius: 10,
    size: 10,
  });
  iconCenter(scene, lucideIcons.help, 1454, 40, 22, { stroke: colors.text, strokeWidth: 2.1 });

  scene.rect(1480, 10, 176, 50, { fill: colors.bg, stroke: colors.line });
  scene.circle(1494, 16, 32, { fill: "#F8D582" });
  scene.text(spec.userInitials ?? "AR", 1510, 37, { size: 13, weight: 800, anchor: "middle" });
  const userTextWidth = 106;
  scene.text(truncateToWidth(spec.user ?? "Alex Ramirez", userTextWidth, 13), 1534, 31, { size: 13, weight: 700 });
  scene.text(truncateToWidth(spec.role, userTextWidth, 12), 1534, 49, { size: 12, fill: colors.faint });
  chevronDown(scene, 1636, 35, { stroke: colors.subtext, size: 8 });

  const nav = navSets[navKindFor(spec.role)] ?? navSets.it;
  nav.forEach((item, index) => {
    const y = 142 + index * 49;
    const rowCenter = y - 3;
    if (item === spec.activeNav) {
      scene.rect(12, y - 24, 236, 42, { fill: colors.goldSoft, stroke: colors.goldSoft, radius: 8 });
    }
    navGlyph(scene, item, 30, rowCenter);
    scene.text(item, 58, y, { size: 14, weight: item === spec.activeNav ? 700 : 500 });
    if (
      [
        "Staff Onboarding",
        "Offboarding",
        "Reports",
        "Admin",
        "Dashboard",
      ].includes(item) &&
      !["Dashboard", "My Profile", "Preferred Name"].includes(item)
    ) {
      chevronDown(scene, 228, rowCenter, { stroke: colors.subtext, size: 8 });
    }
  });

  iconCenter(scene, lucideIcons.help, 30, H - 62, 18, { stroke: colors.text, strokeWidth: 2 });
  scene.text("Support", 58, H - 58, { size: 14 });
  scene.text("Platform Status", 22, H - 26, { size: 14, weight: 700, fill: colors.subtext });
  scene.circle(22, H - 13, 10, { fill: colors.green });
  scene.text("All Systems Operational", 38, H - 8, { size: 12, fill: colors.faint });
}

function titleBlock(scene, spec) {
  const titleWidth = MAIN_W - 420;
  const titleLines = wrapLines([spec.title], titleWidth, 32, 2);
  scene.multiline(titleLines, MAIN_X, 132, {
    size: 32,
    weight: 800,
    lineHeight: 38,
    width: titleWidth,
    maxLines: 2,
  });
  let subtitleY = 160 + (titleLines.length - 1) * 38;
  if (spec.badge) {
    const badgeWidth = Math.max(86, textWidth(spec.badge, 12) + 26);
    const lastTitleWidth = Math.min(titleWidth, textWidth(titleLines.at(-1), 32));
    const sameLineX = MAIN_X + lastTitleWidth + 24;
    const titleRightLimit = W - 240;
    if (titleLines.length === 1 && sameLineX + badgeWidth < titleRightLimit) {
      scene.pill(spec.badge, sameLineX, 102, badgeWidth, {
        fill: colors.goldSoft,
        stroke: colors.goldSoft,
        color: colors.goldDeep,
        height: 28,
      });
    } else {
      const badgeY = 126 + titleLines.length * 38;
      scene.pill(spec.badge, MAIN_X, badgeY, badgeWidth, {
        fill: colors.goldSoft,
        stroke: colors.goldSoft,
        color: colors.goldDeep,
        height: 28,
      });
      subtitleY = badgeY + 44;
    }
  }
  scene.multiline(spec.subtitleLines, MAIN_X, subtitleY, {
    size: 13,
    fill: colors.subtext,
    lineHeight: 18,
    width: MAIN_W - 360,
    maxLines: 3,
  });
  scene.text(spec.lastRefreshed ?? "Last refreshed: May 2, 2025 9:05 AM PT", W - 310, 110, {
    size: 12,
    fill: colors.faint,
  });
  scene.rect(W - 126, 90, 106, 38, { fill: colors.bg, stroke: colors.line });
  scene.text(spec.refreshLabel ?? "Refresh", W - 73, 114, { size: 13, weight: 700, anchor: "middle" });
}

function metricCard(scene, x, y, w, title, value, subtitle, opts = {}) {
  const innerW = w - 36;
  const titleLines = wrapLines([title], innerW, 13, opts.titleMaxLines ?? 2);
  const valueSize = opts.valueSize ?? 24;
  const valueLines = wrapLines(Array.isArray(value) ? value : [value], innerW, valueSize, 3);
  const subtitleLines = wrapLines(Array.isArray(subtitle) ? subtitle : [subtitle], innerW, 12, 4);
  const titleH = titleLines.length * 18;
  const valueY = y + 24 + titleH + 18;
  const valueH = valueLines.length * (valueSize > 18 ? Math.round(valueSize * 1.18) : 20);
  const subtitleY = valueY + valueH + 14;
  const subtitleH = subtitleLines.length * 16;
  const linkSpace = opts.link ? 24 : 0;
  const h = Math.max(opts.height ?? 148, subtitleY - y + subtitleH + 20 + linkSpace);
  scene.rect(x, y, w, h, { fill: colors.bg, stroke: colors.line });
  scene.multiline(titleLines, x + 18, y + 26, {
    size: 13,
    weight: 700,
    lineHeight: 18,
    width: innerW,
    maxLines: 2,
  });
  scene.multiline(valueLines, x + 18, valueY, {
    size: valueSize,
    weight: 800,
    fill: opts.valueColor ?? colors.text,
    lineHeight: valueSize > 18 ? Math.round(valueSize * 1.18) : 20,
    width: innerW,
    maxLines: 3,
  });
  if (opts.valueSuffix && valueLines.length === 1) {
    scene.text(opts.valueSuffix, x + 18 + textWidth(valueLines[0], valueSize) + 10, valueY - 1, {
      size: 13,
      fill: colors.subtext,
    });
  }
  scene.multiline(subtitleLines, x + 18, subtitleY, {
    size: 12,
    fill: colors.subtext,
    lineHeight: 16,
    width: innerW,
    maxLines: 4,
  });
  if (opts.link) {
    scene.text(opts.link, x + 18, y + h - 20, { size: 12, weight: 700, fill: colors.goldDeep });
  }
  return h;
}

function metricGrid(scene, y, cards, opts = {}) {
  const columns = opts.columns ?? 4;
  const gapX = opts.gapX ?? 16;
  const gapY = opts.gapY ?? 16;
  const cardWidth = opts.cardWidth ?? Math.floor((MAIN_W - gapX * (columns - 1)) / columns);
  const cardHeight = opts.cardHeight ?? 118;
  cards.forEach((card, index) => {
    const col = index % columns;
    const row = Math.floor(index / columns);
    const x = MAIN_X + col * (cardWidth + gapX);
    const yy = y + row * (cardHeight + gapY);
    metricCard(scene, x, yy, cardWidth, card.title, card.value, card.subtitle, {
      valueColor: card.valueColor,
      valueSize: card.valueSize,
      height: cardHeight,
      link: card.link,
      titleMaxLines: opts.titleMaxLines ?? 1,
    });
  });
  const rows = Math.ceil(cards.length / columns);
  return y + rows * cardHeight + Math.max(0, rows - 1) * gapY;
}

function infoBanner(scene, x, y, w, lines, opts = {}) {
  const bodyLines = wrapLines(lines, w - 68, 13, opts.maxLines ?? 4);
  const h = Math.max(opts.height ?? 60, bodyLines.length * 20 + 18);
  scene.rect(x, y, w, h, { fill: opts.fill ?? colors.blueSoft, stroke: opts.stroke ?? "#D6E3FF" });
  iconCenter(scene, opts.icon ?? lucideIcons.info, x + 24, y + 24, 16, {
    stroke: opts.color ?? colors.blue,
    strokeWidth: 2,
  });
  scene.multiline(bodyLines, x + 48, y + 22, {
    size: 13,
    fill: opts.textColor ?? colors.blue,
    lineHeight: 20,
    width: w - 68,
    maxLines: opts.maxLines ?? 4,
  });
}

function tableCard(scene, x, y, w, title, columns, rows, opts = {}) {
  const innerLeft = 18;
  const innerRight = 18;
  const contentW = w - innerLeft - innerRight;
  const resolvedCols = columns.map((col, index) => {
    const next = columns[index + 1];
    const maxWidth = col.width ?? (next ? next.x - col.x - 18 : contentW - col.x);
    return { ...col, width: Math.max(44, maxWidth) };
  });
  const rowData = rows.map((row) => {
    const cells = {};
    let maxLines = 1;
    for (const col of resolvedCols) {
      const cell = row[col.key];
      if (cell && typeof cell === "object" && cell.pill) {
        cells[col.key] = { type: "pill", value: cell };
        maxLines = Math.max(maxLines, 1);
      } else {
        const wrapped = wrapLines(cell, col.width, 12, col.maxLines ?? 4);
        cells[col.key] = { type: "text", value: wrapped };
        maxLines = Math.max(maxLines, wrapped.length);
      }
    }
    return {
      row,
      cells,
      rowHeight: Math.max(42, maxLines * 18 + 18),
    };
  });
  const rightLabel = opts.rightLabel;
  const rightLabelWidth = rightLabel ? textWidth(rightLabel, 12) : 0;
  const sameLineTitleWidth = Math.max(120, contentW - (rightLabel ? rightLabelWidth + 18 : 0));
  let titleLines = wrapLines([title], sameLineTitleWidth, 13, 2);
  const stackedRightLabel = Boolean(rightLabel && titleLines.length > 1);
  if (stackedRightLabel) {
    titleLines = wrapLines([title], contentW, 13, 2);
  }
  const titleHeight = titleLines.length * 18;
  const metaHeight = stackedRightLabel ? titleHeight + 22 : Math.max(18, titleHeight);
  const headerY = y + 24 + metaHeight + 14;
  let footerRows = 0;
  if (opts.footerLeft || opts.footerRight) {
    footerRows = 1;
    if (opts.footerLeft && opts.footerRight) {
      const leftWidth = textWidth(opts.footerLeft, 12);
      const rightWidth = textWidth(opts.footerRight, 12);
      if (leftWidth + rightWidth + 36 > w - 36) {
        footerRows = 2;
      }
    }
  }
  const baseHeight = opts.height ?? 320;
  const bodyHeight = rowData.reduce((sum, row) => sum + row.rowHeight, 0);
  const footerSpace = footerRows === 2 ? 54 : footerRows === 1 ? 36 : 18;
  const bodyTop = headerY + 26 - y;
  const h = Math.max(baseHeight, bodyTop + bodyHeight + footerSpace);
  let cursorY = headerY + 26;
  scene.rect(x, y, w, h, { fill: colors.bg, stroke: colors.line });
  scene.multiline(titleLines, x + 18, y + 26, {
    size: 13,
    weight: 700,
    lineHeight: 18,
    width: stackedRightLabel ? contentW : sameLineTitleWidth,
    maxLines: 2,
  });
  if (rightLabel) {
    scene.text(rightLabel, x + w - 18, stackedRightLabel ? y + 26 + titleHeight + 4 : y + 26, {
      size: 12,
      weight: 700,
      fill: colors.goldDeep,
      anchor: "end",
    });
  }
  resolvedCols.forEach((col) => {
    scene.multiline([col.label], x + col.x, headerY, {
      size: 10,
      weight: 700,
      fill: colors.faint,
      lineHeight: 12,
      width: col.width,
      maxLines: 2,
    });
  });
  scene.line(x + 18, headerY + 16, x + w - 18, headerY + 16, { stroke: colors.lineSoft });

  for (const rowDataItem of rowData) {
    const rowHeight = rowDataItem.rowHeight;
    resolvedCols.forEach((col) => {
      const cell = rowDataItem.cells[col.key];
      const xx = x + col.x;
      const yy = cursorY;
      if (cell.type === "pill") {
        const pillWidth = cell.value.width ?? Math.max(76, textWidth(cell.value.pill, 12) + 24);
        const pillX =
          col.pillAlign === "end" || (col.pillAlign == null && /status|state|priority|blockers/i.test(col.key))
            ? xx + Math.max(0, col.width - pillWidth)
            : xx;
        scene.pill(cell.value.pill, pillX, yy - 15, pillWidth, {
          fill: cell.value.fill ?? colors.greenSoft,
          stroke: cell.value.stroke ?? cell.value.fill ?? colors.greenSoft,
          color: cell.value.color ?? colors.green,
          height: 24,
          radius: 7,
        });
      } else {
        scene.multiline(cell.value, xx, yy, {
          size: 12,
          weight: col.weight ?? 400,
          fill: rowDataItem.row[`${col.key}Color`] ?? col.color ?? colors.text,
          lineHeight: 18,
          width: col.width,
          maxLines: col.maxLines ?? 4,
        });
      }
    });
    cursorY += rowHeight;
    scene.line(x + 18, cursorY - 10, x + w - 18, cursorY - 10, { stroke: colors.lineSoft });
  }

  if (opts.footerLeft) {
    scene.text(opts.footerLeft, x + 18, footerRows === 2 ? y + h - 36 : y + h - 18, { size: 12, fill: colors.faint });
  }
  if (opts.footerRight) {
    scene.text(truncateToWidth(opts.footerRight, w - 80, 12), x + w - 18, y + h - 18, {
      size: 12,
      fill: colors.goldDeep,
      weight: 700,
      anchor: "end",
    });
  }
}

function railCard(scene, x, y, w, h, title, sections, opts = {}) {
  const innerW = w - 36;
  const valueX = Math.min(170, Math.max(132, Math.floor(w * 0.5)));
  const valueW = w - valueX - 26;
  const labelW = valueX - 32;
  const badgeWidth = opts.badge ? Math.max(80, textWidth(opts.badge, 12) + 24) : 0;
  const closeSpace = 34;
  const sameLineTitleWidth = Math.max(120, innerW - closeSpace - (badgeWidth ? badgeWidth + 12 : 0));
  let titleLines = wrapLines([title], sameLineTitleWidth, 16, 2);
  const stackedBadge = Boolean(opts.badge && titleLines.length > 1);
  if (stackedBadge) {
    titleLines = wrapLines([title], innerW - closeSpace, 16, 2);
  }
  const measured = [];
  let contentHeight = 0;
  for (const section of sections) {
    if (section.type === "kv") {
      const labelLines = wrapLines([section.label], labelW, 12, 2);
      const valueLines = wrapLines(section.value, valueW, 12, 4);
      const lineCount = Math.max(labelLines.length, valueLines.length);
      const height = Math.max(28, lineCount * 18 + 10);
      measured.push({ ...section, labelLines, valueLines, height });
      contentHeight += height;
    } else if (section.type === "members") {
      const items = section.items.map((item) => ({
        ...item,
        nameLines: wrapLines([item.name], w - 106, 12, 3),
      }));
      const itemsHeight = items.reduce((sum, item) => sum + Math.max(24, item.nameLines.length * 18), 0);
      const height = 28 + itemsHeight;
      measured.push({ ...section, items, height });
      contentHeight += height;
    } else if (section.type === "text") {
      const lines = wrapLines(section.lines, innerW, 12, 6);
      const height = lines.length * 18 + 8;
      measured.push({ ...section, wrappedLines: lines, height });
      contentHeight += height;
    } else if (section.type === "button") {
      const labelLines = wrapLines([section.label], w - 60, 13, 2);
      const buttonHeight = Math.max(section.height ?? 40, labelLines.length * 18 + 14);
      const height = buttonHeight + 12;
      measured.push({ ...section, labelLines, buttonHeight, height });
      contentHeight += height;
    } else if (section.type === "note") {
      const lines = wrapLines(section.lines, w - 68, 12, 6);
      const noteHeight = Math.max(section.height ?? 86, lines.length * 18 + 20);
      const height = noteHeight + 12;
      measured.push({ ...section, wrappedLines: lines, noteHeight, height });
      contentHeight += height;
    } else if (section.type === "divider") {
      measured.push({ ...section, height: 18 });
      contentHeight += 18;
    }
  }
  const titleHeight = titleLines.length * 20;
  const titleBlockHeight = stackedBadge ? titleHeight + 44 : Math.max(titleHeight, 24);
  const topOffset = 34 + titleBlockHeight;
  const actualHeight = Math.max(h, topOffset + contentHeight + 20);
  scene.rect(x, y, w, actualHeight, { fill: colors.bg, stroke: colors.line });
  scene.multiline(titleLines, x + 18, y + 30, {
    size: 16,
    weight: 700,
    lineHeight: 20,
    width: stackedBadge ? innerW - closeSpace : sameLineTitleWidth,
    maxLines: 2,
  });
  if (opts.badge) {
    const badgeX = stackedBadge ? x + 18 : x + w - badgeWidth - 34;
    const badgeY = stackedBadge ? y + 30 + titleHeight + 8 : y + 14;
    scene.pill(opts.badge, badgeX, badgeY, badgeWidth, {
      fill: opts.badgeFill ?? colors.greenSoft,
      stroke: opts.badgeFill ?? colors.greenSoft,
      color: opts.badgeColor ?? colors.green,
    });
  }
  iconCenter(scene, lucideIcons.close, x + w - 22, y + 26, 16, { stroke: colors.subtext, strokeWidth: 2 });
  let yy = y + topOffset;
  for (const section of measured) {
    if (section.type === "kv") {
      scene.multiline(section.labelLines, x + 18, yy, {
        size: 12,
        fill: colors.subtext,
        lineHeight: 18,
        width: labelW,
        maxLines: 2,
      });
      scene.multiline(section.valueLines, x + valueX, yy, {
        size: 12,
        fill: section.color ?? colors.text,
        lineHeight: 18,
        width: valueW,
        maxLines: 4,
      });
      yy += section.height;
    } else if (section.type === "members") {
      scene.text(section.label, x + 18, yy, { size: 12, weight: 700 });
      yy += 28;
      for (const member of section.items) {
        scene.multiline(member.nameLines, x + 32, yy, {
          size: 12,
          width: w - 106,
          lineHeight: 18,
          maxLines: 3,
        });
        if (member.meta) {
          scene.text(member.meta, x + w - 26, yy, { size: 12, fill: colors.subtext, anchor: "end" });
        }
        yy += Math.max(24, member.nameLines.length * 18);
      }
    } else if (section.type === "text") {
      scene.multiline(section.wrappedLines, x + 18, yy, {
        size: 12,
        fill: section.color ?? colors.subtext,
        lineHeight: 18,
        width: innerW,
        maxLines: 6,
      });
      yy += section.height;
    } else if (section.type === "button") {
      scene.rect(x + 18, yy, w - 36, section.buttonHeight, {
        fill: section.fill ?? colors.gold,
        stroke: section.stroke ?? section.fill ?? colors.gold,
      });
      scene.multiline(section.labelLines, x + 36, yy + 23, {
        size: 13,
        weight: 700,
        fill: section.color ?? colors.text,
        lineHeight: 16,
        width: w - 72,
        maxLines: 2,
      });
      yy += section.height;
    } else if (section.type === "note") {
      scene.rect(x + 18, yy, w - 36, section.noteHeight, {
        fill: section.fill ?? colors.orangeSoft,
        stroke: section.stroke ?? section.fill ?? colors.orangeSoft,
      });
      scene.multiline(section.wrappedLines, x + 32, yy + 22, {
        size: 12,
        fill: section.color ?? colors.subtext,
        lineHeight: 18,
        width: w - 68,
        maxLines: 6,
      });
      yy += section.height;
    } else if (section.type === "divider") {
      scene.line(x + 18, yy, x + w - 18, yy, { stroke: colors.lineSoft });
      yy += 18;
    }
  }
}

function footerNote(scene, textLines) {
  infoBanner(scene, MAIN_X, H - 84, MAIN_W, textLines, {
    height: 56,
    fill: colors.blueSoft,
    stroke: "#D6E3FF",
    color: colors.blue,
    textColor: colors.blue,
  });
}

function drawItOverview(scene) {
  const y = 182;
  const cardW = 204;
  metricCard(scene, MAIN_X, y, cardW, "Global Pause", "Not Paused", "All provisioning is active.", {
    valueColor: colors.green,
    link: "Pause all provisioning",
  });
  metricCard(scene, MAIN_X + 220, y, cardW, "Sync Health (All Providers)", "Healthy", [
    "Last successful sync: 9:03 AM PT",
    "Next scheduled sync: 9:15 AM PT",
  ], { valueColor: colors.green, link: "View sync health report →" });
  metricCard(scene, MAIN_X + 440, y, 240, "Provider Health", "Aeries SIS", [
    "Google Workspace  Healthy",
    "Zoom              Healthy",
    "IncidentIQ        Healthy",
    "InformedK12       Healthy",
  ], { valueColor: colors.blue, valueSize: 14, link: "View provider details →" });
  metricCard(scene, MAIN_X + 696, y, 180, "Onboarding (All Sites)", "186", "New hires not fully provisioned", {
    valueColor: colors.green,
    link: "View onboarding report →",
  });
  metricCard(scene, MAIN_X + 892, y, 180, "Offboarding (All Sites)", "142", "Users not fully deprovisioned", {
    valueColor: colors.red,
    link: "View offboarding report →",
  });
  metricCard(scene, MAIN_X + 1088, y, 180, "Room Moves (All Sites)", "64", "Moves not yet completed", {
    valueColor: colors.blue,
    link: "View room move report →",
  });

  const queueY = 352;
  const queueW = 300;
  const queueH = 304;
  tableCard(scene, MAIN_X, queueY, queueW, "Google-active / Aeries-inactive Queue", [
    { label: "Site", key: "site", x: 18 },
    { label: "Users", key: "users", x: 170 },
    { label: "Oldest", key: "oldest", x: 230 },
  ], [
    { site: "Clover High School (CLA)", users: "64", oldest: "Apr 18, 2025" },
    { site: "Desert View ES (DVE)", users: "41", oldest: "Apr 20, 2025" },
    { site: "Canyon Ridge MS (CRM)", users: "33", oldest: "Apr 21, 2025" },
    { site: "District Office (DO)", users: "24", oldest: "Apr 23, 2025" },
  ], {
    height: queueH,
    footerLeft: "Showing 1 to 4 of 18 sites",
    footerRight: "View all 18 sites →",
  });
  tableCard(scene, MAIN_X + 316, queueY, queueW, "Invalid Student Names Queue", [
    { label: "Site", key: "site", x: 18 },
    { label: "Issues", key: "issues", x: 170 },
    { label: "Oldest", key: "oldest", x: 230 },
  ], [
    { site: "Clover High School (CLA)", issues: "3", oldest: "Apr 28, 2025" },
    { site: "Desert View ES (DVE)", issues: "2", oldest: "Apr 29, 2025" },
    { site: "Canyon Ridge MS (CRM)", issues: "1", oldest: "Apr 30, 2025" },
    { site: "District Office (DO)", issues: "-", oldest: "-" },
  ], {
    height: queueH,
    footerLeft: "Showing 1 to 4 of 32 sites",
    footerRight: "View all 32 sites →",
  });
  tableCard(scene, MAIN_X + 632, queueY, queueW, "Frequent Fliers (All Sites)", [
    { label: "Site", key: "site", x: 18 },
    { label: "Students", key: "students", x: 170 },
    { label: "Incidents (90 Days)", key: "incidents", x: 230 },
  ], [
    { site: "Clover High School (CLA)", students: "42", incidents: "97" },
    { site: "Desert View ES (DVE)", students: "31", incidents: "64" },
    { site: "Canyon Ridge MS (CRM)", students: "27", incidents: "58" },
    { site: "District Office (DO)", students: "20", incidents: "36" },
  ], {
    height: queueH,
    footerLeft: "Showing 1 to 4 of 32 sites",
    footerRight: "View all 32 sites →",
  });
  tableCard(scene, MAIN_X + 948, queueY, 320, "Orphaned Zoom Cleanup", [
    { label: "Site", key: "site", x: 18 },
    { label: "Users", key: "users", x: 170 },
    { label: "Oldest", key: "oldest", x: 230 },
  ], [
    { site: "Clover High School (CLA)", users: "27", oldest: "Apr 16, 2025" },
    { site: "Desert View ES (DVE)", users: "18", oldest: "Apr 17, 2025" },
    { site: "Canyon Ridge MS (CRM)", users: "14", oldest: "Apr 19, 2025" },
    { site: "District Office (DO)", users: "9", oldest: "Apr 21, 2025" },
  ], {
    height: queueH,
    footerLeft: "Showing 1 to 4 of 21 sites",
    footerRight: "View all 21 sites →",
  });

  tableCard(scene, MAIN_X, 672, 780, "Recent Warning Summary", [
    { label: "Warning Type", key: "warning", x: 18 },
    { label: "Details", key: "details", x: 172 },
    { label: "First Detected", key: "first", x: 348 },
    { label: "Last Detected", key: "last", x: 504 },
    { label: "Affected", key: "affected", x: 642 },
  ], [
    { warning: "Slow Entra ID Convergence", details: "New user delay > 30 minutes", first: "May 2, 2025 8:31 AM PT", last: "May 2, 2025 9:01 AM PT", affected: ["Google Workspace", "Groups"] },
    { warning: "Schedule Overlap", details: "Overlapping sync windows detected", first: "May 2, 2025 8:50 AM PT", last: "May 2, 2025 9:05 AM PT", affected: ["Aeries SIS", "Onboarding"] },
    { warning: "Schedule Overlap", details: "Overlapping sync windows detected", first: "May 2, 2025 8:46 AM PT", last: "May 2, 2025 9:02 AM PT", affected: ["Google Workspace", "Groups"] },
  ], { height: 210, footerRight: "View all warnings →" });
  tableCard(scene, MAIN_X + 796, 672, 472, "Schedules Running Next", [
    { label: "Source / Job", key: "job", x: 18 },
    { label: "Next Run", key: "next", x: 206 },
    { label: "Cadence", key: "cadence", x: 330 },
    { label: "Status", key: "status", x: 414 },
  ], [
    { job: "Aeries SIS Sync", next: "May 2, 2025 9:15 AM PT", cadence: "30 minutes", status: { pill: "Scheduled", fill: colors.blueSoft, color: colors.blue } },
    { job: "Google Workspace Sync", next: "May 2, 2025 9:15 AM PT", cadence: "5 minutes", status: { pill: "Scheduled", fill: colors.blueSoft, color: colors.blue } },
    { job: "Zoom User Sync", next: "May 2, 2025 9:17 AM PT", cadence: "15 minutes", status: { pill: "Scheduled", fill: colors.blueSoft, color: colors.blue } },
    { job: "IncidentIQ Sync", next: "May 2, 2025 9:20 AM PT", cadence: "30 minutes", status: { pill: "Scheduled", fill: colors.blueSoft, color: colors.blue } },
  ], { height: 210, footerRight: "View all schedules and history →" });

  footerNote(scene, [
    "Times shown are in America / Los_Angeles (PT). Data is sourced from Aeries, Google Workspace, Zoom, IncidentIQ, and InformedK12.",
  ]);
}

function drawHrLifecycle(scene) {
  const metricsBottom = metricGrid(scene, 166, [
    { title: "Upcoming Onboarding", value: "42", subtitle: "Next 14 days", valueColor: colors.green },
    { title: "Upcoming Offboarding", value: "18", subtitle: "Next 14 days", valueColor: colors.red },
    { title: "Manual Intake Queue", value: "27", subtitle: "Needs review", valueColor: colors.purple },
    { title: "Preferred Name Requests", value: "6", subtitle: "Awaiting HR review", valueColor: colors.blue },
    { title: "Unmapped Title Blockers", value: "11", subtitle: "Require mapping", valueColor: colors.orange },
    { title: "Site Override Issues", value: "24", subtitle: "Temporary overrides", valueColor: colors.goldDeep },
    { title: "Orphan AD Accounts", value: "31", subtitle: "Need HR action", valueColor: colors.orange },
  ], {
    columns: 4,
    cardHeight: 110,
    gapX: 16,
    gapY: 14,
    titleMaxLines: 1,
  });

  const rowOneY = metricsBottom + 18;
  tableCard(scene, MAIN_X, rowOneY, 442, "Upcoming Onboarding", [
    { label: "Name", key: "name", x: 14 },
    { label: "Start", key: "start", x: 124 },
    { label: "Role", key: "role", x: 200 },
    { label: "Site", key: "site", x: 306 },
    { label: "Blockers", key: "blockers", x: 392, width: 32, pillAlign: "end" },
  ], [
    { name: "Alex Lee", start: "May 6", role: "Math Teacher", site: "Clover HS", blockers: { pill: "1", fill: colors.redSoft, color: colors.red, width: 30 } },
    { name: "Jordan Patel", start: "May 7", role: "Counselor", site: "District", blockers: { pill: "0", fill: colors.greenSoft, color: colors.green, width: 30 } },
  ], { height: 210, rightLabel: "View all", footerRight: "View all onboarding (42) →" });

  tableCard(scene, MAIN_X + 458, rowOneY, 442, "Upcoming Offboarding", [
    { label: "Name", key: "name", x: 14 },
    { label: "Last Day", key: "day", x: 122 },
    { label: "Role", key: "role", x: 202 },
    { label: "Site", key: "site", x: 312 },
    { label: "Exit", key: "exit", x: 388, width: 36 },
  ], [
    { name: "Sam Davis", day: "May 6", role: "3rd Grade", site: "Highland ES", exit: "Vol." },
    { name: "Riley Chen", day: "May 7", role: "Office Asst.", site: "Business", exit: "Ret." },
  ], { height: 210, rightLabel: "View all", footerRight: "View all offboarding (18) →" });

  tableCard(scene, MAIN_X + 916, rowOneY, 352, "Manual Intake Queue", [
    { label: "Name", key: "name", x: 14 },
    { label: "Type", key: "type", x: 138 },
    { label: "Role", key: "role", x: 206 },
    { label: "Status", key: "status", x: 274, width: 60, pillAlign: "end" },
  ], [
    { name: "Jamie O'Neil", type: "Contractor", role: "Technology", status: { pill: "New", fill: colors.blueSoft, color: colors.blue, width: 52 } },
    { name: "Jordan Kim", type: "Volunteer", role: "Library", status: { pill: "In Review", fill: colors.orangeSoft, color: colors.orange, width: 78 } },
  ], { height: 210, rightLabel: "Review all", footerRight: "View manual intake (27) →" });

  const rowTwoY = rowOneY + 226;
  tableCard(scene, MAIN_X, rowTwoY, 328, "Preferred Name Requests", [
    { label: "Name", key: "name", x: 14 },
    { label: "Requested", key: "requested", x: 138 },
    { label: "Priority", key: "priority", x: 252, width: 58, pillAlign: "end" },
  ], [
    { name: "Jordan Lee", requested: "Jordyn Lee", priority: { pill: "Normal", fill: colors.greenSoft, color: colors.green, width: 64 } },
    { name: "Taylor Smith", requested: "Tay Smith", priority: { pill: "Normal", fill: colors.greenSoft, color: colors.green, width: 64 } },
  ], { height: 246, rightLabel: "View all", footerRight: "View all requests (6) →" });

  tableCard(scene, MAIN_X + 344, rowTwoY, 328, "Unmapped Title Blockers", [
    { label: "Job Title", key: "title", x: 14 },
    { label: "Count", key: "count", x: 264, width: 34 },
  ], [
    { title: "Instructional Coach", count: "4" },
    { title: "Behavior Tech", count: "3" },
    { title: "Site Administrator", count: "2" },
  ], { height: 246, rightLabel: "View all", footerRight: "View mappings (11) →" });

  tableCard(scene, MAIN_X + 688, rowTwoY, 328, "Site Override Issues", [
    { label: "Issue", key: "issue", x: 14 },
    { label: "Count", key: "count", x: 264, width: 34 },
  ], [
    { issue: "Missing Manager", count: "9" },
    { issue: "Missing Work Email", count: "6" },
    { issue: "Invalid Location", count: "5" },
  ], { height: 246, rightLabel: "View all", footerRight: "View overrides (24) →" });

  tableCard(scene, MAIN_X + 1032, rowTwoY, 236, "Orphan AD Accounts", [
    { label: "Type", key: "type", x: 14 },
    { label: "Count", key: "count", x: 146, width: 34 },
    { label: "Oldest", key: "oldest", x: 184, width: 34 },
  ], [
    { type: "Retiree", count: "14", oldest: "Apr 18" },
    { type: "Service", count: "9", oldest: "Apr 19" },
    { type: "Student Teacher", count: "5", oldest: "Apr 21" },
  ], { height: 246, rightLabel: "View all", footerRight: "View accounts (31) →" });
}

function drawOnboarding(scene) {
  metricCard(scene, MAIN_X, 178, 220, "Ready to Provision", "64", "Common path", { valueColor: colors.green });
  metricCard(scene, MAIN_X + 236, 178, 220, "Blocked", "31", "Needs upstream fix", { valueColor: colors.orange });
  metricCard(scene, MAIN_X + 472, 178, 220, "IncidentIQ Follow-up", "22", "External status", { valueColor: colors.blue });
  metricCard(scene, MAIN_X + 708, 178, 220, "Manual Intake", "11", "Contractors / volunteers", { valueColor: colors.purple });
  metricCard(scene, MAIN_X + 944, 178, 324, "Current Rules", "AD → Google → Zoom", [
    "IncidentIQ user is polled hourly by email after the account exists.",
    "Earliest matching Aeries and Verkada tickets are linked when found.",
  ], { valueSize: 14, valueColor: colors.text });

  tableCard(scene, MAIN_X, 346, 820, "Upcoming Staff Onboarding", [
    { label: "Person", key: "person", x: 14 },
    { label: "Site", key: "site", x: 148 },
    { label: "Start", key: "start", x: 272 },
    { label: "Current Step", key: "step", x: 352 },
    { label: "Issue / Action", key: "issue", x: 506 },
    { label: "Workflow Status", key: "status", x: 690 },
  ], [
    { person: "Jordan Miles", site: "Clover HS", start: "May 6, 2025", step: "Google pending", issue: "Waiting Entra convergence", status: { pill: "In Progress", fill: colors.blueSoft, color: colors.blue, width: 92 } },
    { person: "Nia Brooks", site: "District Office", start: "May 8, 2025", step: "Sync dry-run", issue: "Room mapping required", status: { pill: "Needs Review", fill: colors.orangeSoft, color: colors.orange, width: 104 } },
    { person: "Evan Ruiz", site: "Franklin MS", start: "May 12, 2025", step: "HR intake", issue: "Missing mandatory field", status: { pill: "Blocked", fill: colors.redSoft, color: colors.red, width: 72 } },
    { person: "Mika Ito", site: "Desert View", start: "May 13, 2025", step: "Ready", issue: "No blockers", status: { pill: "Ready", fill: colors.greenSoft, color: colors.green, width: 64 } },
  ], {
    height: 328,
    footerLeft: "Showing 1 to 4 of 42 upcoming people",
    footerRight: "View all onboarding (42) →",
  });

  railCard(scene, MAIN_X + 840, 346, 428, 328, "Selected Workflow", [
    { type: "kv", label: "Person", value: "Jordan Miles" },
    { type: "kv", label: "Site", value: "Clover High School (CLA)" },
    { type: "kv", label: "Assigned Email", value: "jordan.miles@wusd.org" },
    { type: "kv", label: "Workflow State", value: "Waiting for Google completion" },
    { type: "kv", label: "IncidentIQ", value: ["No local write owned by this app", "User lookup retries at most once per hour"] },
    { type: "divider" },
    { type: "text", lines: ["Earliest matching Aeries ticket:", "IT-12904  Open"] },
    { type: "text", lines: ["Earliest matching Verkada ticket:", "MOT-4412  Waiting"] },
    { type: "divider" },
    { type: "note", fill: colors.blueSoft, color: colors.blue, lines: ["This dashboard surfaces external IIQ follow-up", "status after the user exists."] },
  ]);

  footerNote(scene, [
    "People are shown by person rather than spreadsheet row. Sensitive lifecycle data remains restricted to HR visibility.",
  ]);
}

function drawOffboarding(scene) {
  metricCard(scene, MAIN_X, 178, 220, "Scheduled Leaves", "58", "Next 30 days", { valueColor: colors.orange });
  metricCard(scene, MAIN_X + 236, 178, 220, "Immediate Terms", "9", "Needs approval", { valueColor: colors.red });
  metricCard(scene, MAIN_X + 472, 178, 220, "Asset Retrieval", "37", "Site tasks", { valueColor: colors.blue });
  metricCard(scene, MAIN_X + 708, 178, 220, "Security Risk", "6", "Recent Google activity", { valueColor: colors.red });
  metricCard(scene, MAIN_X + 944, 178, 324, "Deprovision Scope", "Accounts, licenses, assets", [
    "Room equipment remains room-assigned.",
    "Google-active / Aeries-inactive stays reviewable and separately visible.",
  ], { valueSize: 14, valueColor: colors.text });

  tableCard(scene, MAIN_X, 346, 820, "Upcoming Offboarding", [
    { label: "Person", key: "person", x: 14 },
    { label: "Site", key: "site", x: 146 },
    { label: "End", key: "end", x: 270 },
    { label: "Status", key: "status", x: 346 },
    { label: "Next Action", key: "action", x: 472 },
    { label: "Asset Work", key: "asset", x: 672 },
  ], [
    { person: "Chris Morgan", site: "Clover HS", end: "May 3, 2025", status: { pill: "Waiting Manual", fill: colors.orangeSoft, color: colors.orange, width: 110 }, action: "Retrieve laptop and badge", asset: "2 items" },
    { person: "Taylor Singh", site: "District Office", end: "May 9, 2025", status: { pill: "Scheduled", fill: colors.blueSoft, color: colors.blue, width: 80 }, action: "License reclaim queued", asset: "1 item" },
    { person: "Jamie Reed", site: "Desert View", end: "May 12, 2025", status: { pill: "Blocked", fill: colors.redSoft, color: colors.red, width: 72 }, action: "Exception review needed", asset: "0 items" },
    { person: "Robin Hall", site: "Franklin MS", end: "May 18, 2025", status: { pill: "Ready", fill: colors.greenSoft, color: colors.green, width: 64 }, action: "All provider checks passed", asset: "3 items" },
  ], { height: 328, footerRight: "View all offboarding (58) →" });

  railCard(scene, MAIN_X + 840, 346, 428, 328, "Queue Actions", [
    { type: "kv", label: "Scheduled leaves", value: "Site tasks and closeout tracking" },
    { type: "kv", label: "Immediate termination", value: "HR-only destructive action" },
    { type: "kv", label: "Exceptions", value: "Reason, owner, and review date required" },
    { type: "kv", label: "Senior exception", value: "Suppressed until configured cutoff day" },
    { type: "divider" },
    { type: "button", label: "Open Orphan Account Queue", fill: colors.goldSoft, stroke: colors.goldSoft, color: colors.goldDeep },
    { type: "button", label: "View Exception List", fill: colors.bg, stroke: colors.line, color: colors.text },
    { type: "note", lines: ["Accounts with recent Google activity after source-system inactivity", "must be treated as a security risk."], fill: colors.redSoft, color: colors.red },
  ]);

  footerNote(scene, [
    "Assets assigned directly to humans are tracked here. Phones, TVs, and other room equipment stay attached to the room.",
  ]);
}

function drawInvalidNames(scene) {
  infoBanner(scene, MAIN_X, 168, 960, [
    "These are active, unresolved student name issues detected during sync.",
    "Corrections must be made in Aeries. Changes will sync automatically.",
  ]);
  scene.rect(MAIN_X, 246, 960, 60, { fill: colors.bg, stroke: colors.line });
  iconCenter(scene, lucideIcons.warning, MAIN_X + 28, 282, 18, { stroke: colors.orange, strokeWidth: 2 });
  scene.text("7 active issues", MAIN_X + 58, 282, { size: 16, weight: 700 });
  scene.text("All must be corrected in Aeries", MAIN_X + 58, 302, { size: 12, fill: colors.subtext });
  scene.text("Last sync:  May 2, 2025 9:05 AM PT", MAIN_X + 554, 278, { size: 12, fill: colors.subtext });
  scene.text("Next sync: ~ in 55 minutes", MAIN_X + 554, 300, { size: 12, fill: colors.subtext });
  scene.rect(MAIN_X + 852, 262, 90, 30, { fill: colors.bg, stroke: colors.line });
  scene.text("Sync now", MAIN_X + 897, 282, { size: 12, weight: 700, anchor: "middle" });

  searchField(scene, MAIN_X, 328, 486, 42, "Search by student name, student ID, or issue type...");
  dropdownButton(scene, MAIN_X + 566, 328, 110, 42, "Issue Type", { align: "center" });
  dropdownButton(scene, MAIN_X + 692, 328, 94, 42, "Grade", { align: "center" });
  scene.rect(MAIN_X + 806, 328, 110, 42, { fill: colors.bg, stroke: colors.line });
  scene.text("Clear filters", MAIN_X + 861, 354, { size: 12, weight: 700, anchor: "middle" });

  tableCard(scene, MAIN_X, 386, 968, "Aeries Name Correction Queue", [
    { label: "Student ID", key: "id", x: 14 },
    { label: "Student Name", key: "name", x: 134 },
    { label: "FirstName (Aeries)", key: "first", x: 258 },
    { label: "LastName (Aeries)", key: "last", x: 436 },
    { label: "Issue Type", key: "issue", x: 618 },
    { label: "Grade", key: "grade", x: 764 },
    { label: "Submitted", key: "submitted", x: 842 },
  ], [
    { id: "0001021", name: "Carlos Nuno", first: ["Carlos", "Carlos"], last: ["Nuno", "Nuno"], issue: "Invalid character", grade: "11", submitted: "May 2, 2025 8:58 AM PT" },
    { id: "0001087", name: "Alex O'Neil", first: ["Alex", "Alex"], last: ["O'Neil", "ONeil"], issue: "Invalid character", grade: "10", submitted: "May 2, 2025 8:56 AM PT" },
    { id: "0001142", name: "Jose Martinez", first: ["Jose", "Jose"], last: ["Martinez", "Martinez"], issue: "Invalid character", grade: "12", submitted: "May 2, 2025 8:54 AM PT" },
    { id: "0001233", name: "Taylor Smith-Jones", first: ["Taylor", "Taylor"], last: ["Smith-Jones", "SmithJones"], issue: "Invalid character", grade: "9", submitted: "May 2, 2025 8:52 AM PT" },
  ], { height: 450, footerLeft: "Showing 1 to 4 of 7 issues" });

  railCard(scene, MAIN_X + 986, 156, 286, 680, "Student Details", [
    { type: "text", lines: ["Carlos Nuno", "Student ID: 0001021   Grade 11"], color: colors.text },
    { type: "kv", label: "FirstName (raw)", value: "Carlos" },
    { type: "kv", label: "LastName (raw)", value: "Nuño" },
    { type: "divider" },
    { type: "kv", label: "FirstName (normalized)", value: "Carlos" },
    { type: "kv", label: "LastName (normalized)", value: "Nuno" },
    { type: "button", label: "Copy Normalized First Name", fill: colors.bg, stroke: colors.line, color: colors.text },
    { type: "button", label: "Copy Normalized Last Name", fill: colors.bg, stroke: colors.line, color: colors.text },
    { type: "note", lines: ["Corrections must be made in Aeries.", "This dashboard cannot edit student data."], fill: colors.orangeSoft, color: colors.orange },
    { type: "button", label: "Open Student in Aeries", fill: colors.goldSoft, stroke: colors.goldSoft, color: colors.goldDeep },
    { type: "divider" },
    { type: "kv", label: "Issue Type", value: "Invalid character" },
    { type: "kv", label: "Detected", value: "May 2, 2025 8:58 AM PT" },
  ]);

  footerNote(scene, [
    "This dashboard is informational only. Student records cannot be edited here. All corrections must be made in Aeries.",
  ]);
}

function drawFrequentFliers(scene) {
  scene.rect(MAIN_X, 180, 920, 120, { fill: colors.bg, stroke: colors.line });
  scene.text("Show students with", MAIN_X + 20, 202, { size: 13, weight: 700 });
  scene.rect(MAIN_X + 20, 226, 74, 34, { fill: colors.bg, stroke: colors.line });
  scene.text(">=", MAIN_X + 40, 248, { size: 16, weight: 700, anchor: "middle" });
  scene.rect(MAIN_X + 110, 226, 58, 34, { fill: colors.bg, stroke: colors.line });
  scene.text("2", MAIN_X + 139, 248, { size: 16, anchor: "middle" });
  scene.rect(MAIN_X + 184, 226, 138, 34, { fill: colors.bg, stroke: colors.line });
  scene.text("assignments", MAIN_X + 253, 248, { size: 14, anchor: "middle" });
  scene.rect(MAIN_X + 426, 226, 58, 34, { fill: colors.bg, stroke: colors.line });
  scene.text("90", MAIN_X + 455, 248, { size: 16, anchor: "middle" });
  scene.text("days", MAIN_X + 500, 248, { size: 14 });
  scene.rect(MAIN_X + 768, 218, 108, 42, { fill: colors.gold, stroke: colors.gold });
  scene.text("Apply", MAIN_X + 822, 245, { size: 16, weight: 700, anchor: "middle" });
  scene.text("Counts include new assignments. Closed tickets linked to those devices are shown.", MAIN_X + 20, 282, {
    size: 12,
    fill: colors.subtext,
  });

  searchField(scene, MAIN_X, 314, 340, 42, "Search by student name or ID...");
  dropdownValueField(scene, MAIN_X + 360, 314, 110, 42, "Grade", "All");
  dropdownValueField(scene, MAIN_X + 488, 314, 110, 42, "Status", "All");
  scene.rect(MAIN_X + 616, 314, 112, 42, { fill: colors.bg, stroke: colors.line });
  scene.text("Clear filters", MAIN_X + 672, 340, { size: 12, weight: 700, anchor: "middle" });

  tableCard(scene, MAIN_X, 372, 902, "Frequent Fliers Queue", [
    { label: "Student ID", key: "id", x: 14 },
    { label: "Student Name", key: "name", x: 118 },
    { label: "Grade", key: "grade", x: 298 },
    { label: "Device Assignments", key: "assign", x: 360 },
    { label: "Linked Tickets", key: "tickets", x: 522 },
    { label: "Last Ticket", key: "last", x: 650 },
    { label: "Trend", key: "trend", x: 792 },
  ], [
    { id: "3504011", name: "Jason Rodriguez", grade: "10", assign: "4", tickets: "3", last: "May 1, 2025", trend: "/\\/\\", trendColor: colors.red },
    { id: "3502897", name: "Aisha Patel", grade: "11", assign: "3", tickets: "2", last: "Apr 24, 2025", trend: "/\\/\\", trendColor: colors.red },
    { id: "3503122", name: "Marcus Wu", grade: "9", assign: "3", tickets: "2", last: "Apr 18, 2025", trend: "/\\/\\", trendColor: colors.red },
    { id: "3502764", name: "Luis Martinez", grade: "10", assign: "2", tickets: "2", last: "Apr 15, 2025", trend: "/\\/\\", trendColor: colors.red },
    { id: "3503308", name: "Naomi Williams", grade: "11", assign: "2", tickets: "1", last: "Apr 10, 2025", trend: "/\\/\\", trendColor: colors.red },
  ], { height: 466, footerLeft: "Showing 1 to 5 of 8 students" });

  railCard(scene, MAIN_X + 922, 94, 350, 742, "Jason Rodriguez", [
    { type: "text", lines: ["ID: 3504011   Grade 10"], color: colors.subtext },
    { type: "kv", label: "Device Assignments", value: "4 (Last 90 Days)" },
    { type: "kv", label: "Linked Tickets", value: "3 (Last 90 Days)" },
    { type: "kv", label: "Days Since Last Ticket", value: "42" },
    { type: "divider" },
    { type: "members", label: "Device Assignment History", items: [
      { name: "Chromebook  •  CLA-24-27891", meta: "Active" },
      { name: "Chromebook  •  CLA-24-26412", meta: "Returned" },
      { name: "Chromebook  •  CLA-24-25103", meta: "Returned" },
      { name: "Chromebook  •  CLA-24-23987", meta: "Returned" },
    ] },
    { type: "divider" },
    { type: "members", label: "Recent Tickets (IncidentIQ)", items: [
      { name: "INC-1782345  Broken Screen", meta: "Closed" },
      { name: "INC-1758991  Broken Hinge", meta: "Closed" },
      { name: "INC-1741123  Keyboard Not Working", meta: "Closed" },
    ] },
    { type: "divider" },
    { type: "note", lines: ["Multiple physical damage incidents within 90 days.", "Tech Support  •  May 2, 2025"], fill: colors.orangeSoft, color: colors.subtext },
  ]);

  footerNote(scene, [
    "These students have multiple device assignments or incidents in the last 90 days. Use this information to plan interventions and support.",
  ]);
}

function drawSiteAdmin(scene) {
  metricCard(scene, MAIN_X, 182, 204, "Today's Onboarding", "5", "People starting today", {
    valueColor: colors.green,
    link: "View all onboarding →",
  });
  metricCard(scene, MAIN_X + 220, 182, 204, "Pending Offboarding", "3", "People exiting", {
    valueColor: colors.red,
    link: "View all offboarding →",
  });
  metricCard(scene, MAIN_X + 440, 182, 204, "Room Move Drafts", "4", "Drafts in progress", {
    valueColor: colors.blue,
    link: "View all drafts →",
  });
  metricCard(scene, MAIN_X + 660, 182, 204, "Room Corrections", "2", "Rooms need attention", {
    valueColor: colors.orange,
    link: "View corrections →",
  });
  metricCard(scene, MAIN_X + 880, 182, 204, "Phone Directory Sync", "98%", "Coverage", {
    valueColor: colors.purple,
    link: "View directory →",
  });
  railCard(scene, MAIN_X + 1104, 182, 164, 212, "Quick Actions", [
    { type: "text", lines: ["Get started with common tasks."], color: colors.subtext },
    { type: "button", label: "Create Room Move", fill: colors.gold, stroke: colors.gold },
    { type: "button", label: "View Phone Directory", fill: colors.bg, stroke: colors.line },
  ]);

  tableCard(scene, MAIN_X, 410, 392, "Onboarding Today (5)", [
    { label: "Name", key: "name", x: 14 },
    { label: "Start Date", key: "start", x: 96 },
    { label: "Role / Position", key: "role", x: 170 },
    { label: "Manager", key: "manager", x: 302 },
  ], [
    { name: "Alex Lee", start: "May 2, 2025", role: "Math Teacher", manager: "James Nguyen" },
    { name: "Jordan Patel", start: "May 2, 2025", role: "School Counselor", manager: "Lindsey Carter" },
    { name: "Morgan Rivera", start: "May 2, 2025", role: "Admin Assistant", manager: "Patricia Diaz" },
    { name: "Taylor Garcia", start: "May 2, 2025", role: "PE Teacher", manager: "Michael Johnson" },
  ], { height: 248, rightLabel: "View all", footerRight: "View all onboarding (12) →" });

  tableCard(scene, MAIN_X + 408, 410, 340, "Pending Offboarding (3)", [
    { label: "Name", key: "name", x: 14 },
    { label: "Last Day", key: "day", x: 84 },
    { label: "Department", key: "department", x: 154 },
    { label: "Status", key: "status", x: 274 },
  ], [
    { name: "Sam Davis", day: "May 6, 2025", department: "3rd Grade", status: { pill: "Pending", fill: colors.orangeSoft, color: colors.orange, width: 70 } },
    { name: "Riley Chen", day: "May 7, 2025", department: "Business Office", status: { pill: "Pending", fill: colors.orangeSoft, color: colors.orange, width: 70 } },
    { name: "Drew Martinez", day: "May 9, 2025", department: "Science", status: { pill: "Pending", fill: colors.orangeSoft, color: colors.orange, width: 70 } },
  ], { height: 248, rightLabel: "View all", footerRight: "View all offboarding (9) →" });

  tableCard(scene, MAIN_X + 764, 410, 340, "Room Move Drafts (4)", [
    { label: "Draft Name", key: "name", x: 14 },
    { label: "Moves", key: "moves", x: 180 },
    { label: "Status", key: "status", x: 236 },
  ], [
    { name: "Classroom Updates - May", moves: "18", status: { pill: "In Progress", fill: colors.blueSoft, color: colors.blue, width: 90 } },
    { name: "Science Dept Move", moves: "7", status: { pill: "In Progress", fill: colors.blueSoft, color: colors.blue, width: 90 } },
    { name: "End of Year Clean Up", moves: "12", status: { pill: "In Progress", fill: colors.blueSoft, color: colors.blue, width: 90 } },
    { name: "PE Equipment Rooms", moves: "4", status: { pill: "In Progress", fill: colors.blueSoft, color: colors.blue, width: 90 } },
  ], { height: 248, rightLabel: "View all", footerRight: "View all drafts (12) →" });

  railCard(scene, MAIN_X + 1120, 412, 148, 498, "Site Scope", [
    { type: "text", lines: ["You are viewing data for", "Clover High School (CLA) only.", "District-wide views and IT", "administration tools are not", "available in this role."], color: colors.subtext },
    { type: "divider" },
    { type: "text", lines: ["Data is sourced from Aeries SIS,", "AD, and Telephony."], color: colors.subtext },
    { type: "text", lines: ["Updates reflect the last completed", "sync for this site."], color: colors.subtext },
  ]);

  tableCard(scene, MAIN_X, 674, 392, "Room Corrections Needed (2)", [
    { label: "Room", key: "room", x: 14 },
    { label: "Issue", key: "issue", x: 84 },
    { label: "Details", key: "details", x: 178 },
    { label: "Identified", key: "identified", x: 300 },
  ], [
    { room: "B204", issue: "Invalid Capacity", details: "Capacity is 0", identified: "May 1, 2025" },
    { room: "GYM", issue: "Missing Room Type", details: "Room type not set", identified: "Apr 30, 2025" },
  ], { height: 236, rightLabel: "View all", footerRight: "View all corrections (5) →" });
  tableCard(scene, MAIN_X + 408, 674, 340, "Phone Directory Sync Status", [
    { label: "Metric", key: "metric", x: 14 },
    { label: "Value", key: "value", x: 172 },
  ], [
    { metric: "Status", value: "Synced" },
    { metric: "Last successful sync", value: "May 2, 2025 9:03 AM PT" },
    { metric: "Source Systems", value: "Aeries SIS, AD, Telephony" },
    { metric: "Directory Entries", value: "284 people, 132 rooms" },
  ], { height: 236, rightLabel: "Synced", footerRight: "View phone directory →" });
  tableCard(scene, MAIN_X + 764, 674, 340, "Frequent Fliers (This Site)", [
    { label: "Name", key: "name", x: 14 },
    { label: "Incidents", key: "incidents", x: 146 },
    { label: "Linked Tickets", key: "tickets", x: 230 },
    { label: "Trend", key: "trend", x: 314 },
  ], [
    { name: "Jason Rodriguez", incidents: "4", tickets: "3", trend: "/\\", trendColor: colors.red },
    { name: "Aisha Patel", incidents: "3", tickets: "2", trend: "/\\", trendColor: colors.red },
    { name: "Marcus Wu", incidents: "3", tickets: "2", trend: "/\\", trendColor: colors.red },
    { name: "Luis Martinez", incidents: "2", tickets: "1", trend: "/\\", trendColor: colors.orange },
  ], { height: 236, rightLabel: "View report", footerRight: "View full report (42 students) →" });
}

function drawMyProfile(scene) {
  scene.rect(MAIN_X, 146, 804, 628, { fill: colors.bg, stroke: colors.line });
  scene.text("My Information", MAIN_X + 82, 190, { size: 18, weight: 700 });
  scene.circle(MAIN_X + 26, 164, 40, { fill: colors.goldSoft });
  const rows = [
    ["Legal Name", "Maria Elena Torres"],
    ["Preferred / Display Name", "Maria Torres"],
    ["WUSD Email", "maria.torres@wusd.org"],
    ["Site", "Clover High School (CLA)"],
    ["Department", "Mathematics"],
    ["Manager", "James Nguyen"],
    ["Room", "208B"],
    ["Phone / Extension", "(702) 555-1000 x1200"],
    ["Account Status", "Active"],
  ];
  let y = 244;
  for (const [label, value] of rows) {
    scene.line(MAIN_X + 24, y + 26, MAIN_X + 780, y + 26, { stroke: colors.lineSoft });
    scene.text(label, MAIN_X + 78, y, { size: 14, fill: colors.subtext });
    if (label === "Account Status") {
      scene.pill(value, MAIN_X + 350, y - 18, 66, { fill: colors.greenSoft, stroke: colors.greenSoft, color: colors.green });
      scene.text("Your account is active and in good standing.", MAIN_X + 350, y + 34, { size: 12, fill: colors.subtext });
    } else {
      scene.text(value, MAIN_X + 318, y, { size: 14 });
    }
    y += 54;
  }
  scene.rect(MAIN_X + 588, 720, 166, 40, { fill: colors.bg, stroke: colors.gold });
  scene.text("View in Phone Directory", MAIN_X + 671, 745, { size: 13, weight: 700, anchor: "middle" });

  railCard(scene, MAIN_X + 826, 146, 442, 756, "Request a Preferred Name", [
    { type: "text", lines: ["Your preferred name is used in directories, email display,", "and other district systems where allowed."], color: colors.subtext },
    { type: "note", lines: ["Your legal name will continue to be used for official", "documents, payroll, and compliance purposes."], fill: colors.blueSoft, color: colors.blue },
    { type: "kv", label: "Your Current Display Name", value: "Maria Torres" },
    { type: "kv", label: "Preferred Name", value: "e.g., Maria Torres" },
    { type: "text", lines: ["50 characters max"], color: colors.faint },
    { type: "kv", label: "Why are you requesting a preferred name? (Optional)", value: "Provide any additional context that may help." },
    { type: "text", lines: ["200 characters max"], color: colors.faint },
    { type: "kv", label: "Pronouns (optional)", value: "She / Her" },
    { type: "note", lines: ["By submitting, you confirm this name is how you", "would like to be identified at work."], fill: colors.goldSoft, color: colors.goldDeep },
    { type: "button", label: "Submit Preferred Name Request", fill: colors.gold, stroke: colors.gold },
  ]);

  scene.rect(MAIN_X, 790, 804, 82, { fill: colors.bg, stroke: colors.line });
  scene.text("Need to find someone?", MAIN_X + 82, 822, { size: 18, weight: 700 });
  scene.text("Search the shared phone directory for staff, rooms, and extensions.", MAIN_X + 82, 846, { size: 13, fill: colors.subtext });
  scene.rect(MAIN_X + 588, 810, 166, 40, { fill: colors.bg, stroke: colors.gold });
  scene.text("Go to Phone Directory", MAIN_X + 671, 835, { size: 13, weight: 700, anchor: "middle" });
}

function drawReports(scene) {
  const cards = [
    ["Onboarding", "186", "Pending", colors.green],
    ["Offboarding", "142", "Pending", colors.red],
    ["Room Moves", "64", "In Progress", colors.blue],
    ["Phone Directory", "98%", "Coverage", colors.purple],
    ["Invalid Student Names", "7", "Active Issues", colors.orange],
    ["Frequent Fliers", "142", "This Site", colors.blue],
    ["Google-active / Aeries-inactive", "237", "Users", colors.orange],
    ["Orphaned Zoom Cleanup", "89", "Orphaned Users", colors.blue],
    ["Sync Health", "Healthy", "All Providers", colors.cyan],
  ];
  cards.forEach((card, index) => {
    const row = index < 6 ? 0 : 1;
    const col = row === 0 ? index : index - 6;
    const x = MAIN_X + col * 205;
    const y = row === 0 ? 160 : 290;
    metricCard(scene, x, y, row === 0 ? 190 : 286, card[0], card[1], card[2], {
      valueColor: card[3],
      height: 116,
      link: "View report →",
      valueSize: card[1] === "Healthy" ? 20 : 18,
    });
  });

  tableCard(scene, MAIN_X, 420, 982, "Available Reports", [
    { label: "Report", key: "report", x: 14 },
    { label: "Scope", key: "scope", x: 214 },
    { label: "Source Systems", key: "source", x: 320 },
    { label: "Last Refreshed", key: "refresh", x: 508 },
    { label: "Open Items", key: "items", x: 670 },
    { label: "Status", key: "status", x: 776 },
    { label: "Actions", key: "action", x: 888 },
  ], [
    { report: "Onboarding Status Report", scope: "District-wide", source: "Aeries SIS", refresh: "May 2, 2025 9:05 AM PT", items: "186", status: { pill: "Up to date", fill: colors.greenSoft, color: colors.green, width: 92 }, action: "Open" },
    { report: "Offboarding Status Report", scope: "District-wide", source: "Aeries SIS", refresh: "May 2, 2025 9:05 AM PT", items: "142", status: { pill: "Up to date", fill: colors.greenSoft, color: colors.green, width: 92 }, action: "Open" },
    { report: "Room Move Status Report", scope: "District-wide", source: "Aeries SIS", refresh: "May 2, 2025 9:05 AM PT", items: "64", status: { pill: "Up to date", fill: colors.greenSoft, color: colors.green, width: 92 }, action: "Open" },
    { report: "Phone Directory Coverage Report", scope: "District-wide", source: "Aeries, AD, Telephony", refresh: "May 2, 2025 9:05 AM PT", items: "2,146", status: { pill: "Up to date", fill: colors.greenSoft, color: colors.green, width: 92 }, action: "Open" },
    { report: "Invalid Student Names Queue Report", scope: "District-wide", source: "Aeries SIS", refresh: "May 2, 2025 9:05 AM PT", items: "7", status: { pill: "Up to date", fill: colors.greenSoft, color: colors.green, width: 92 }, action: "Open" },
  ], { height: 308, footerLeft: "Showing 1 to 5 of 9 reports" });

  tableCard(scene, MAIN_X, 744, 982, "Recent Refreshes", [
    { label: "Source System", key: "system", x: 14 },
    { label: "Last Refresh", key: "time", x: 200 },
    { label: "Status", key: "status", x: 454 },
  ], [
    { system: "Aeries SIS", time: "May 2, 2025 9:05 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
    { system: "Google Workspace", time: "May 2, 2025 9:04 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
    { system: "Zoom", time: "May 2, 2025 9:02 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
    { system: "IncidentIQ", time: "May 2, 2025 8:59 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
  ], { height: 192 });
  railCard(scene, MAIN_X + 998, 290, 270, 614, "Onboarding Status Report", [
    { type: "text", lines: ["Shows all new hires pending provision and", "their progress through the employee reprovisioning workflow."], color: colors.subtext },
    { type: "kv", label: "Scope", value: "District-wide (All Sites)" },
    { type: "kv", label: "Source System", value: "Aeries SIS" },
    { type: "text", lines: ["Data Included", "New hires not yet fully provisioned", "License and account provisioning status", "Devices staged but not assigned"], color: colors.subtext },
    { type: "divider" },
    { type: "kv", label: "Open Items", value: ["186", "Pending onboarding"] },
    { type: "kv", label: "Last Refreshed", value: "May 2, 2025 9:05 AM PT" },
    { type: "kv", label: "Refresh Frequency", value: "Every 15 minutes" },
    { type: "button", label: "Open Report", fill: colors.gold, stroke: colors.gold },
  ]);
}

function drawAdmin(scene) {
  metricCard(scene, MAIN_X, 168, 288, "Global Pause", "Not Paused", ["All provisioning is active.", "Emergency cutoff: None set"], {
    valueColor: colors.green,
    link: "Pause all provisioning",
  });
  metricCard(scene, MAIN_X + 304, 168, 288, "Application Timezone", "America / Los_Angeles", "PT (UTC-07:00)", {
    valueSize: 20,
  });
  metricCard(scene, MAIN_X + 608, 168, 356, "Admin Warnings", "2 Active", [
    "Slow Entra ID convergence  •  User sync delay > 30 minutes",
    "Repeated schedule overlap  •  Google Workspace - Groups",
  ], { valueColor: colors.red, valueSize: 20, link: "View all warnings →" });
  metricCard(scene, MAIN_X + 980, 168, 288, "Sync Health (All Providers)", "Healthy", "Last successful sync: 9:03 AM PT", {
    valueColor: colors.green,
    link: "View sync health report →",
  });

  tableCard(scene, MAIN_X, 392, 570, "Per-Job Sync Cadence", [
    { label: "Source / Job", key: "job", x: 14 },
    { label: "Cadence", key: "cadence", x: 152 },
    { label: "Next Run", key: "next", x: 244 },
    { label: "Last Successful Run", key: "last", x: 362 },
    { label: "Status", key: "status", x: 502 },
  ], [
    { job: "Escape SIS Sync", cadence: "60 minutes", next: "10:05 AM PT", last: "9:05 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
    { job: "Aeries SIS Sync", cadence: "30 minutes", next: "9:35 AM PT", last: "9:05 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
    { job: "Zoom User Sync", cadence: "15 minutes", next: "9:20 AM PT", last: "9:07 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
    { job: "Google Workspace Sync", cadence: "5 minutes", next: "9:10 AM PT", last: "9:05 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
    { job: "IncidentIQ Sync", cadence: "30 minutes", next: "9:20 AM PT", last: "8:50 AM PT", status: { pill: "Healthy", fill: colors.greenSoft, color: colors.green, width: 74 } },
  ], { height: 294, footerRight: "View sync schedules and history →" });

  railCard(scene, MAIN_X + 586, 392, 330, 294, "Google-active / Aeries-inactive Defaults", [
    { type: "text", lines: ["Staff   |   Students"], color: colors.goldDeep },
    { type: "kv", label: "Default Action", value: "Disable Google Access" },
    { type: "kv", label: "Delay Before Action", value: "3 days" },
    { type: "text", lines: ["Send notification email to manager", "Log and notify security operations"], color: colors.subtext },
    { type: "button", label: "View queue report", fill: colors.bg, stroke: colors.line, color: colors.text },
  ]);

  tableCard(scene, MAIN_X + 932, 392, 336, "Deprovisioning Exceptions", [
    { label: "Name", key: "name", x: 14 },
    { label: "Type", key: "type", x: 142, width: 62 },
    { label: "Scope", key: "scope", x: 196, width: 72 },
    { label: "Expires", key: "expires", x: 280, width: 42, maxLines: 2 },
  ], [
    { name: "Substitute Accounts", type: "User", scope: "District", expires: "Permanent" },
    { name: "Retiree Board Members", type: "User", scope: "District", expires: "Permanent" },
    { name: "Service Accounts", type: "Service", scope: "District", expires: "Permanent" },
    { name: "Student Teachers", type: "User", scope: "District", expires: "6/30/2025" },
  ], { height: 294, rightLabel: "View all" });

  tableCard(scene, MAIN_X, 704, 290, "Zoom SAML Mapping", [
    { label: "Area", key: "area", x: 14 },
    { label: "Status", key: "status", x: 186 },
  ], [
    { area: "Department Mapping", status: "35 mapped" },
    { area: "License Mapping", status: "6 mapped" },
    { area: "Cohort / Group Mapping", status: "28 mapped" },
    { area: "SCIM Deprovisioning", status: "Enabled" },
  ], { height: 166, rightLabel: "View details" });
  tableCard(scene, MAIN_X + 306, 704, 290, "Provisioning Profiles", [
    { label: "Profile", key: "profile", x: 14 },
    { label: "Users", key: "users", x: 212 },
  ], [
    { profile: "Standard Teacher", users: "1,245" },
    { profile: "Staff - Limited", users: "612" },
    { profile: "Student", users: "9,842" },
    { profile: "Substitute", users: "418" },
  ], { height: 166, rightLabel: "View all" });
  tableCard(scene, MAIN_X + 612, 704, 306, "Title & Category Mappings", [
    { label: "Mapping", key: "mapping", x: 14 },
    { label: "Count", key: "count", x: 240 },
  ], [
    { mapping: "Job Titles", count: "1,278" },
    { mapping: "Employee Categories", count: "18" },
    { mapping: "Locations (Sites)", count: "112" },
    { mapping: "Departments", count: "189" },
  ], { height: 166, rightLabel: "Manage" });
  tableCard(scene, MAIN_X + 934, 704, 334, "Invalid-Name Settings", [
    { label: "Setting", key: "setting", x: 14 },
    { label: "Value", key: "value", x: 228 },
  ], [
    { setting: "Transliteration Rules", value: "Active" },
    { setting: "Apostrophe Cleanup", value: "Enabled" },
    { setting: "Hyphen Cleanup", value: "Enabled" },
    { setting: "IT-Managed Allowlist", value: "312" },
  ], { height: 166, rightLabel: "Manage" });
}

function drawDataQuality(scene) {
  metricCard(scene, MAIN_X, 190, 204, "Title Mapping", "18", "Unmapped", { valueColor: colors.orange });
  metricCard(scene, MAIN_X + 220, 190, 204, "Room Mapping", "23", "Required", { valueColor: colors.orange });
  metricCard(scene, MAIN_X + 440, 190, 204, "Source Conflicts", "41", "Open", { valueColor: colors.red });
  metricCard(scene, MAIN_X + 660, 190, 204, "Resolved Today", "29", "Cleared", { valueColor: colors.green });

  tableCard(scene, MAIN_X, 374, 864, "Data Quality Queue", [
    { label: "Issue", key: "issue", x: 14 },
    { label: "Source", key: "source", x: 236 },
    { label: "Owner", key: "owner", x: 360 },
    { label: "Impact", key: "impact", x: 478 },
    { label: "Next Action", key: "action", x: 646 },
  ], [
    { issue: "Unmapped job title", source: "Escape / SFTP", owner: "HR + IT", impact: "Blocks access bundle", action: "Map title" },
    { issue: "Room mismatch", source: "Aeries", owner: "Site Secretary", impact: "Blocks sync", action: "Confirm room" },
    { issue: "Google-active / Aeries-inactive", source: "Google + Aeries", owner: "IT", impact: "Security review", action: "Schedule deprovision" },
    { issue: "Missing mandatory field", source: "HR intake", owner: "HR", impact: "Blocks onboarding", action: "Update record" },
    { issue: "Site mismatch", source: "Escape / Aeries", owner: "HR", impact: "Blocks baseline site selection", action: "Apply temporary override" },
  ], { height: 520, footerRight: "View full queue →" });
}

function drawSyncTransparency(scene) {
  scene.rect(MAIN_X, 182, 540, 48, { fill: colors.bg, stroke: colors.line });
  const tabs = [
    ["Pending", 124],
    ["In Progress / Manual", 196],
    ["Completed", 128],
    ["History", 92],
  ];
  let tx = MAIN_X + 10;
  tabs.forEach(([label, width], index) => {
    scene.rect(tx, 190, width, 32, {
      fill: index === 1 ? colors.goldSoft : colors.bg,
      stroke: index === 1 ? colors.goldSoft : colors.lineSoft,
      radius: 7,
    });
    scene.text(label, tx + width / 2, 211, { size: 12, weight: 700, anchor: "middle" });
    tx += width + 10;
  });

  metricCard(scene, MAIN_X + 570, 182, 160, "Pending", "18", "Queued", { valueColor: colors.orange, height: 112 });
  metricCard(scene, MAIN_X + 746, 182, 160, "Manual Action", "7", "Needs human review", { valueColor: colors.red, height: 112 });
  metricCard(scene, MAIN_X + 922, 182, 160, "Completed", "44", "Current year", { valueColor: colors.green, height: 112 });
  metricCard(scene, MAIN_X + 1098, 182, 170, "Archived", "312", "Previous school year", { valueColor: colors.blue, height: 112 });

  tableCard(scene, MAIN_X, 262, 872, "In Progress / Manual Actions", [
    { label: "User", key: "user", x: 14 },
    { label: "Type", key: "type", x: 170 },
    { label: "Current Step", key: "step", x: 248 },
    { label: "Issue / Action", key: "issue", x: 404 },
    { label: "Date", key: "date", x: 668 },
    { label: "Action", key: "action", x: 754, width: 90, maxLines: 2 },
  ], [
    { user: "Alex Ramirez", type: "Staff", step: "room_mapped", issue: "Primary phone conflict", date: "Apr 29", action: "Open mapping" },
    { user: "Marisol Vega", type: "Student", step: "iiq_matched", issue: "Missing asset", date: "Apr 29", action: "Override" },
    { user: "Mika Ito", type: "Staff", step: "photo_processed", issue: "Rollover wait", date: "Apr 28", action: "Recheck" },
    { user: "Nia Brooks", type: "Staff", step: "ingested", issue: "Room mapping required", date: "Apr 28", action: "Open mapping" },
  ], { height: 520, footerRight: "Open sync dashboard mappings →" });

  railCard(scene, MAIN_X + 888, 262, 380, 520, "Selected Sync Item", [
    { type: "kv", label: "User", value: "Alex Ramirez" },
    { type: "kv", label: "Type", value: "Staff" },
    { type: "kv", label: "Current Phase", value: "room_mapped" },
    { type: "kv", label: "Overall Status", value: "manual_action" },
    { type: "kv", label: "Queued At", value: "Apr 29, 2026 8:44 AM PT" },
    { type: "divider" },
    { type: "text", lines: ["Errors / Warnings", "primary_conflict", "Phone assignment needs a primary owner selection"], color: colors.subtext },
    { type: "note", lines: ["rollover_wait is an in-progress warning,", "not a manual-action block."], fill: colors.blueSoft, color: colors.blue },
    { type: "button", label: "Open Mapping Tool", fill: colors.gold, stroke: colors.gold },
  ]);
}

function drawTicketingHumanWork(scene) {
  metricCard(scene, MAIN_X, 182, 220, "Waiting User Match", "14", "Poll hourly", { valueColor: colors.orange });
  metricCard(scene, MAIN_X + 236, 182, 220, "Linked Tickets", "63", "Earliest match", { valueColor: colors.blue });
  metricCard(scene, MAIN_X + 472, 182, 220, "Manual Fallbacks", "11", "Needs owner", { valueColor: colors.red });
  metricCard(scene, MAIN_X + 708, 182, 220, "Closed", "38", "This week", { valueColor: colors.green });
  metricCard(scene, MAIN_X + 944, 182, 324, "Ticket Matching", "Requestor email + ticket category", [
    "If the earliest matching ticket is inaccessible, the link is hidden silently.",
    "Ticket number is shown as a raw link with raw status value.",
  ], { valueSize: 14 });

  tableCard(scene, MAIN_X, 348, 920, "Human Work Queue", [
    { label: "Parent Flow", key: "flow", x: 14 },
    { label: "Affected User", key: "user", x: 132 },
    { label: "Ticket", key: "ticket", x: 286 },
    { label: "Status", key: "status", x: 400 },
    { label: "Category", key: "category", x: 522 },
    { label: "Rule", key: "rule", x: 758 },
  ], [
    { flow: "Onboarding", user: "Jordan Miles", ticket: "IT-12904", status: "Open", category: "Aeries Add User", rule: "Earliest matching ticket" },
    { flow: "Onboarding", user: "Nia Brooks", ticket: "MOT-4412", status: "Waiting", category: "Alarm Code", rule: "External IIQ config" },
    { flow: "Room Move", user: "Morgan Lee", ticket: "IT-13012", status: "Open", category: "Phone conflict", rule: "Manual fallback" },
    { flow: "Offboarding", user: "Chris Morgan", ticket: "IT-13044", status: "Closed", category: "Asset retrieval", rule: "Linked to lifecycle" },
  ], { height: 500, footerRight: "View full queue →" });

  railCard(scene, MAIN_X + 936, 348, 332, 500, "Selected Ticket Context", [
    { type: "kv", label: "Affected User", value: "Jordan Miles" },
    { type: "kv", label: "Current Workflow", value: "Onboarding" },
    { type: "kv", label: "Matching Rule", value: "Requestor email + category" },
    { type: "kv", label: "Displayed Ticket", value: "IT-12904" },
    { type: "kv", label: "Current Status", value: "Open" },
    { type: "divider" },
    { type: "text", lines: ["Aeries (Asset Tag: AERIES) → User Rights → Add User", "Security Systems → Alarm Codes → Add Alarm Code"], color: colors.subtext },
    { type: "note", lines: ["Only the earliest matching ticket is linked.", "If it disappears later, the dashboard shows no link silently."], fill: colors.blueSoft, color: colors.blue },
    { type: "button", label: "Open Linked Ticket", fill: colors.gold, stroke: colors.gold },
  ]);
}

function drawRoomMoves(scene) {
  metricCard(scene, MAIN_X, 182, 220, "Draft Moves", "31", "Saved", { valueColor: colors.blue });
  metricCard(scene, MAIN_X + 236, 182, 220, "Warnings", "8", "Need review", { valueColor: colors.orange });
  metricCard(scene, MAIN_X + 472, 182, 220, "Immediate", "5", "Allowed", { valueColor: colors.green });
  metricCard(scene, MAIN_X + 708, 182, 220, "Batch Cutovers", "3", "Scheduled", { valueColor: colors.blue });
  metricCard(scene, MAIN_X + 944, 182, 324, "Execution Rules", [
    "5 or fewer moves may execute immediately after final review.",
    "Non-IT batches over 5 must use scheduled off-hours cutover windows.",
  ][0], ["5 or fewer moves may execute immediately after final review.", "Non-IT batches over 5 must use scheduled off-hours cutover windows."], { valueSize: 14 });

  tableCard(scene, MAIN_X, 348, 840, "Move Set Review", [
    { label: "Person", key: "person", x: 14 },
    { label: "Current", key: "current", x: 172 },
    { label: "Target", key: "target", x: 292 },
    { label: "Phone", key: "phone", x: 414 },
    { label: "Warning", key: "warning", x: 590 },
    { label: "State", key: "state", x: 742 },
  ], [
    { person: "Alex Ramirez", current: "A-104", target: "A-108", phone: "Move ext 51042", warning: "Ready", state: { pill: "Ready", fill: colors.greenSoft, color: colors.green, width: 64 } },
    { person: "Morgan Lee", current: "B-210", target: "B-204", phone: "Manual ticket", warning: "Primary conflict", state: { pill: "Review", fill: colors.orangeSoft, color: colors.orange, width: 72 } },
    { person: "Jamie Reed", current: "C-118", target: "No room", phone: "Remove phone", warning: "Null-room outcome", state: { pill: "Review", fill: colors.orangeSoft, color: colors.orange, width: 72 } },
    { person: "Nia Brooks", current: "D-102", target: "D-112", phone: "Assign line", warning: "Ready", state: { pill: "Ready", fill: colors.greenSoft, color: colors.green, width: 64 } },
  ], { height: 480, footerRight: "View full draft set →" });

  railCard(scene, MAIN_X + 856, 348, 412, 480, "Cutover Controls", [
    { type: "kv", label: "5 or fewer moves", value: "Allowed immediately after final review" },
    { type: "kv", label: "More than 5", value: "Requires a batch cutover window" },
    { type: "kv", label: "Non-IT off hours", value: "20:00 to 04:00 Pacific time" },
    { type: "kv", label: "IT authored batches", value: "May span multiple sites and broader times" },
    { type: "divider" },
    { type: "button", label: "Schedule Cutover", fill: colors.gold, stroke: colors.gold },
    { type: "button", label: "Create One-Person Correction", fill: colors.bg, stroke: colors.line },
    { type: "note", lines: ["Site staff can launch a one-person correction directly", "when the directory room or phone assignment is wrong."], fill: colors.blueSoft, color: colors.blue },
  ]);
}

function drawPhoneDirectory(scene, mode) {
  const modeIndex = { person: 0, room: 1, department: 2 }[mode];
  scene.rect(MAIN_X, 182, 560, 42, { fill: colors.bg, stroke: colors.line });
  ["By Person", "By Room", "By Department"].forEach((label, index) => {
    const widths = [174, 174, 194];
    const x = MAIN_X + 10 + widths.slice(0, index).reduce((a, b) => a + b + 10, 0);
    scene.rect(x, 188, widths[index], 30, {
      fill: index === modeIndex ? colors.goldSoft : colors.bg,
      stroke: index === modeIndex ? colors.goldSoft : colors.lineSoft,
      radius: 7,
    });
    scene.text(label, x + widths[index] / 2, 208, { size: 12, weight: 700, anchor: "middle" });
  });
  if (mode === "department") {
    infoBanner(scene, MAIN_X, 236, 990, [
      "By Department groups shared-service extensions and Zoom call queues.",
      "Extensions are displayed as numeric values only.",
    ], {
      height: 40,
      fill: colors.blueSoft,
      stroke: "#D6E3FF",
      color: colors.blue,
      textColor: colors.blue,
    });
  }

  const filterY = mode === "department" ? 292 : 250;
  searchField(
    scene,
    MAIN_X,
    filterY,
    376,
    42,
    mode === "room" ? "Search rooms, department, or staff..." : "Search by name, classification, or extension..."
  );
  const filterLabels = mode === "department"
    ? ["Classification", "Provider Source", "State", "Clear filters"]
    : mode === "person"
      ? ["Department", "Room", "State", "Clear filters"]
      : ["Department", "SLG", "State", "Clear filters"];
  let fx = MAIN_X + 402;
  for (const label of filterLabels) {
    const width = label === "Provider Source" ? 160 : label === "Clear filters" ? 120 : 110;
    if (label === "Clear filters") {
      scene.rect(fx, filterY, width, 42, { fill: colors.bg, stroke: colors.line });
      scene.text(label, fx + width / 2, filterY + 26, { size: 12, weight: 700, anchor: "middle" });
    } else {
      dropdownButton(scene, fx, filterY, width, 42, label, { align: "center" });
    }
    fx += width + 16;
  }

  let columns;
  let rows;
  let title;
  let railTitle;
  let railSections;
  if (mode === "person") {
    title = "Directory Results - By Person";
    columns = [
      { label: "Person", key: "person", x: 14 },
      { label: "Email", key: "email", x: 188 },
      { label: "Room", key: "room", x: 414 },
      { label: "Extension", key: "extension", x: 500 },
      { label: "Phone", key: "phone", x: 596 },
      { label: "State", key: "state", x: 744 },
      { label: "Action", key: "action", x: 858 },
    ];
    rows = [
      { person: "Maria Torres", email: "maria.torres@wusd.org", room: "A-104", extension: "350001", phone: "(707) 555-1000", state: { pill: "Assigned", fill: colors.greenSoft, color: colors.green, width: 82 }, action: "Create Room Move" },
      { person: "James Nguyen", email: "james.nguyen@wusd.org", room: "B-210", extension: "350002", phone: "(707) 555-1200", state: { pill: "Pending Change", fill: colors.orangeSoft, color: colors.orange, width: 110 }, action: "Review Change" },
      { person: "Patricia Diaz", email: "patricia.diaz@wusd.org", room: "Office", extension: "350003", phone: "(707) 555-1003", state: { pill: "Assigned", fill: colors.greenSoft, color: colors.green, width: 82 }, action: "Open Details" },
      { person: "Rebecca Lee", email: "rebecca.lee@wusd.org", room: "220", extension: "350004", phone: "(707) 555-2200", state: { pill: "Conflict", fill: colors.redSoft, color: colors.red, width: 72 }, action: "Review Conflict" },
    ];
    railTitle = "Maria Torres";
    railSections = [
      { type: "text", lines: ["Site Admin  •  Clover High School (CLA)"], color: colors.subtext },
      { type: "kv", label: "Room", value: "A-104" },
      { type: "kv", label: "Extension", value: "350001" },
      { type: "kv", label: "Phone", value: "(707) 555-1000" },
      { type: "kv", label: "Current State", value: "Assigned" },
      { type: "divider" },
      { type: "button", label: "Create Room Move", fill: colors.gold, stroke: colors.gold },
      { type: "button", label: "View Assignment History", fill: colors.bg, stroke: colors.line },
      { type: "note", lines: ["Use this action when the current room or phone assignment", "is wrong and a one-person correction is needed."], fill: colors.blueSoft, color: colors.blue },
    ];
  } else if (mode === "room") {
    title = "Directory Results - By Room";
    columns = [
      { label: "Room", key: "room", x: 14 },
      { label: "Extension", key: "extension", x: 198 },
      { label: "Phone", key: "phone", x: 292 },
      { label: "Assigned To", key: "assigned", x: 474 },
      { label: "State", key: "state", x: 742 },
      { label: "Action", key: "action", x: 858 },
    ];
    rows = [
      { room: "100A - Main Office", extension: "350000", phone: "(707) 555-1000 x1000", assigned: "Maria Torres", state: { pill: "Assigned", fill: colors.greenSoft, color: colors.green, width: 82 }, action: "Create Room Move" },
      { room: "120 - Library", extension: "350120", phone: "(707) 555-1000 x1200", assigned: "James Nguyen", state: { pill: "Assigned", fill: colors.greenSoft, color: colors.green, width: 82 }, action: "Open Details" },
      { room: "220 - Nurse Office", extension: "350220", phone: "(707) 555-1000 x2200", assigned: "<empty>", state: { pill: "Unassigned", fill: colors.blueSoft, color: colors.blue, width: 94 }, action: "Assign" },
      { room: "300 - Gymnasium", extension: "350300", phone: "(707) 555-1000 x3000", assigned: "Michael Johnson", state: { pill: "Pending Change", fill: colors.orangeSoft, color: colors.orange, width: 110 }, action: "Review Change" },
    ];
    railTitle = "100A - Main Office";
    railSections = [
      { type: "kv", label: "Display Name", value: "Main Office" },
      { type: "kv", label: "Phone / Extension", value: "(707) 555-1000 x1000" },
      { type: "kv", label: "Assigned To", value: "Maria Torres" },
      { type: "kv", label: "Current State", value: "Assigned" },
      { type: "divider" },
      { type: "button", label: "Create Room Move", fill: colors.gold, stroke: colors.gold },
      { type: "button", label: "View Assignment History", fill: colors.bg, stroke: colors.line },
      { type: "note", lines: ["For one-person corrections, site staff can launch a", "prefilled room-move workflow directly from this panel."], fill: colors.blueSoft, color: colors.blue },
    ];
  } else {
    title = "Directory Results - By Department";
    columns = [
      { label: "Name", key: "name", x: 14 },
      { label: "Classification", key: "class", x: 204 },
      { label: "Extension", key: "extension", x: 334 },
      { label: "Provider Source", key: "source", x: 438 },
      { label: "Assigned To / Destination", key: "dest", x: 620 },
      { label: "State", key: "state", x: 890 },
    ];
    rows = [
      { name: "CLA Main Office", class: "Main Office", extension: "350000", source: "Zoom Shared Line Group", dest: ["Maria Torres (350001)", "James Nguyen (350002)", "Patricia Diaz (350003)"], state: { pill: "Assigned", fill: colors.greenSoft, color: colors.green, width: 82 } },
      { name: "CLA Attendance", class: "Call Queue", extension: "350004", source: "Zoom Call Queue", dest: ["Lindsey Carter (350101)", "Rebecca Lee (350102)", "Patricia Diaz (350103)", "James Nguyen (350104)"], state: { pill: "Assigned", fill: colors.greenSoft, color: colors.green, width: 82 } },
      { name: "CLA Main Auto Receptionist", class: "Main Line", extension: "350010", source: "Zoom Auto Attendant", dest: "Auto Receptionist Routing", state: { pill: "Common Area", fill: colors.blueSoft, color: colors.blue, width: 106 } },
      { name: "CLA 2FA", class: "Department", extension: "350022", source: "Zoom Shared Line Group", dest: ["Security Team (Group)", "Kathy Clark (350301)", "Miguel Ramirez (350302)", "Deanna Wu (350304)"], state: { pill: "Unassigned", fill: colors.blueSoft, color: colors.blue, width: 94 } },
    ];
    railTitle = "CLA Attendance";
    railSections = [
      { type: "kv", label: "Classification", value: "Call Queue" },
      { type: "kv", label: "Extension", value: "350004" },
      { type: "kv", label: "Provider Source", value: "Zoom Call Queue" },
      { type: "members", label: "Assigned To / Destination (4 members)", items: [
        { name: "Lindsey Carter", meta: "350101" },
        { name: "Rebecca Lee", meta: "350102" },
        { name: "Patricia Diaz", meta: "350103" },
        { name: "James Nguyen", meta: "350104" },
      ] },
      { type: "divider" },
      { type: "kv", label: "Current State", value: "Assigned" },
      { type: "text", lines: ["Queue is active and in use."], color: colors.subtext },
      { type: "divider" },
      { type: "button", label: "View Assignment History", fill: colors.gold, stroke: colors.gold },
      { type: "button", label: "Test Queue (Dial 350004)", fill: colors.bg, stroke: colors.line },
      { type: "note", lines: ["To correct an individual assignment or start a room move,", "use By Person or By Room."], fill: colors.blueSoft, color: colors.blue },
    ];
  }

  tableCard(scene, MAIN_X, filterY + 58, 980, title, columns, rows, {
    height: 520,
    footerLeft: "Showing 1 to 4 entries",
  });
  railCard(scene, MAIN_X + 1000, 160, 268, 676, railTitle, railSections, {
    badge: mode === "department" ? "Assigned" : undefined,
  });
}

const screens = [
  {
    file: "wireframe-it-admin-overview",
    title: "Dashboard Overview",
    badge: "District-wide",
    role: "IT Admin",
    user: "Alex Ramirez",
    userInitials: "AR",
    activeNav: "Dashboard",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["Operational overview of provisioning, data quality, and sync health across all sites."],
    draw: drawItOverview,
  },
  {
    file: "wireframe-hr-lifecycle-overview",
    title: "HR Lifecycle Overview",
    badge: "District-wide",
    role: "Human Resources",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Dashboard",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["Monitor and manage the employee lifecycle from onboarding to offboarding."],
    draw: drawHrLifecycle,
  },
  {
    file: "wireframe-onboarding-dashboard",
    title: "Onboarding Dashboard",
    badge: "District-wide",
    role: "Human Resources",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Staff Onboarding",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["Upcoming onboarding processes by person, with blockers, workflow state, and external IIQ follow-up status."],
    draw: drawOnboarding,
  },
  {
    file: "wireframe-offboarding-dashboard",
    title: "Offboarding Dashboard",
    badge: "District-wide",
    role: "Human Resources",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Offboarding",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["Offboarding status by person across accounts, licenses, assets, and security review queues."],
    draw: drawOffboarding,
  },
  {
    file: "wireframe-site-secretary-invalid-names",
    title: "Invalid Student Names",
    role: "Site Secretary",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Invalid Student Names",
    scopeLabel: "Site (read-only)",
    scopeSub: "Clover High School (CLA)",
    searchPlaceholder: "Search by student name or ID...",
    subtitleLines: ["Review student name field issues that must be corrected in Aeries."],
    draw: drawInvalidNames,
  },
  {
    file: "wireframe-device-wrangler-frequent-fliers",
    title: "Frequent Fliers",
    role: "Device Wrangler",
    user: "Maria Torres",
    userInitials: "DW",
    activeNav: "Frequent Fliers",
    scopeLabel: "Site (read-only)",
    scopeSub: "Clover High School (CLA)",
    searchPlaceholder: "Search by student name or ID...",
    subtitleLines: ["Students with repeated device damage or support incidents."],
    draw: drawFrequentFliers,
  },
  {
    file: "wireframe-site-admin-dashboard",
    title: "Dashboard",
    badge: "Site-level view",
    role: "Site Admin",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Dashboard",
    scopeLabel: "Site (read-only)",
    scopeSub: "Clover High School (CLA)",
    subtitleLines: ["Monitor and manage Clover High School (CLA) operations."],
    draw: drawSiteAdmin,
  },
  {
    file: "wireframe-faculty-staff-my-profile",
    title: "My Profile",
    role: "Faculty / Staff",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "My Profile",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["View your account information and manage your preferred name."],
    draw: drawMyProfile,
  },
  {
    file: "wireframe-it-admin-reports",
    title: "Reports",
    badge: "District-wide",
    role: "IT Admin",
    user: "Alex Ramirez",
    userInitials: "AR",
    activeNav: "Reports",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["Operational reports and queue summaries across all systems and workflows."],
    draw: drawReports,
  },
  {
    file: "wireframe-it-admin-admin-controls",
    title: "Admin",
    badge: "District-wide",
    role: "IT Admin",
    user: "Alex Ramirez",
    userInitials: "AR",
    activeNav: "Admin",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["Manage system controls, integrations, mappings, and provisioning behavior."],
    draw: drawAdmin,
  },
  {
    file: "wireframe-data-quality-dashboard",
    title: "Data Quality",
    badge: "District-wide",
    role: "IT Admin",
    user: "Alex Ramirez",
    userInitials: "AR",
    activeNav: "Data Quality",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: [],
    draw: drawDataQuality,
  },
  {
    file: "wireframe-sync-transparency-dashboard",
    title: "Sync Transparency",
    badge: "District-wide",
    role: "IT Admin",
    user: "Alex Ramirez",
    userInitials: "AR",
    activeNav: "Reports",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["First-class workflow projection for pending, in-progress, completed, and historical sync state."],
    draw: drawSyncTransparency,
  },
  {
    file: "wireframe-ticketing-human-work",
    title: "Ticketing and Human Work",
    badge: "District-wide",
    role: "IT Admin",
    user: "Alex Ramirez",
    userInitials: "AR",
    activeNav: "Reports",
    scopeLabel: "District-wide",
    scopeSub: "All Sites",
    subtitleLines: ["IncidentIQ-linked work surface for automation fallbacks, related work, and human handoffs."],
    draw: drawTicketingHumanWork,
  },
  {
    file: "wireframe-room-moves",
    title: "Room Moves",
    role: "Site Secretary",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Room Moves",
    scopeLabel: "Site (read-only)",
    scopeSub: "Clover High School (CLA)",
    subtitleLines: ["Draft, review, schedule, and commit room and phone changes with warnings before automation writes."],
    draw: drawRoomMoves,
  },
  {
    file: "wireframe-phone-directory-by-person",
    title: "Phone Directory",
    role: "Site Admin",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Phone Directory",
    scopeLabel: "Site (read-only)",
    scopeSub: "Clover High School (CLA)",
    subtitleLines: ["Provider-synced directory for one site."],
    draw: (scene) => drawPhoneDirectory(scene, "person"),
  },
  {
    file: "wireframe-phone-directory-by-room",
    title: "Phone Directory",
    role: "Site Admin",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Phone Directory",
    scopeLabel: "Site (read-only)",
    scopeSub: "Clover High School (CLA)",
    subtitleLines: ["Provider-synced directory for one site."],
    draw: (scene) => drawPhoneDirectory(scene, "room"),
  },
  {
    file: "wireframe-phone-directory-by-department",
    title: "Phone Directory",
    role: "Site Admin",
    user: "Maria Torres",
    userInitials: "MT",
    activeNav: "Phone Directory",
    scopeLabel: "Site (read-only)",
    scopeSub: "Clover High School (CLA)",
    subtitleLines: ["Provider-synced directory for one site."],
    draw: (scene) => drawPhoneDirectory(scene, "department"),
  },
];

function buildScene(spec) {
  const scene = new Scene(spec.title);
  shell(scene, spec);
  titleBlock(scene, spec);
  spec.draw(scene);
  return scene;
}

fs.mkdirSync(outDir, { recursive: true });

const legacyGeneratorScreens = screens.filter((spec) => !implementedPageFiles.has(spec.file));

for (const spec of legacyGeneratorScreens) {
  const scene = buildScene(spec);
  fs.writeFileSync(path.join(outDir, `${spec.file}.svg`), scene.svg());
  fs.writeFileSync(path.join(outDir, `${spec.file}.pen`), scene.pen());
}

console.log(
  `Generated ${legacyGeneratorScreens.length} legacy wireframe SVGs and PEN files in ${path.relative(process.cwd(), outDir)}`
);
