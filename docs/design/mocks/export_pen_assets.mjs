import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";

const pencilBin = "/opt/homebrew/bin/pencil";

function fail(message) {
  throw new Error(message);
}

function esc(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function renderImage({ href, x, y, w, h, radius = 0, stroke, strokeWidth = 1, fill = null }, defs, clipCounter) {
  const parts = [];
  if (radius > 0) {
    const clipId = `clip-${clipCounter.value++}`;
    defs.push(`<clipPath id="${clipId}"><rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${radius}"/></clipPath>`);
    if (fill && typeof fill === "string") {
      parts.push(
        `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${radius}" fill="${fill}"${stroke ? ` stroke="${stroke}" stroke-width="${strokeWidth}"` : ""}/>`
      );
    } else if (stroke) {
      parts.push(
        `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${radius}" fill="none" stroke="${stroke}" stroke-width="${strokeWidth}"/>`
      );
    }
    parts.push(
      `<image x="${x}" y="${y}" width="${w}" height="${h}" href="${esc(href)}" clip-path="url(#${clipId})" preserveAspectRatio="xMidYMid meet"/>`
    );
    return parts.join("\n");
  }

  if (fill && typeof fill === "string") {
    parts.push(
      `<rect x="${x}" y="${y}" width="${w}" height="${h}" fill="${fill}"${stroke ? ` stroke="${stroke}" stroke-width="${strokeWidth}"` : ""}/>`
    );
  } else if (stroke) {
    parts.push(
      `<rect x="${x}" y="${y}" width="${w}" height="${h}" fill="none" stroke="${stroke}" stroke-width="${strokeWidth}"/>`
    );
  }
  parts.push(`<image x="${x}" y="${y}" width="${w}" height="${h}" href="${esc(href)}" preserveAspectRatio="xMidYMid meet"/>`);
  return parts.join("\n");
}

function renderText(node, x, y) {
  const fontSize = Number(node.fontSize ?? 14);
  const lines = String(node.content ?? "").split("\n");
  const family = esc(node.fontFamily ?? "Arial");
  const weight = esc(node.fontWeight ?? "400");
  const fill = esc(node.fill ?? "#111827");

  if (lines.length === 1) {
    return `<text x="${x}" y="${y + fontSize}" font-family="${family}" font-size="${fontSize}" font-weight="${weight}" fill="${fill}">${esc(lines[0])}</text>`;
  }

  return [
    `<text x="${x}" y="${y + fontSize}" font-family="${family}" font-size="${fontSize}" font-weight="${weight}" fill="${fill}">`,
    ...lines.map((line, index) =>
      `<tspan x="${x}" dy="${index === 0 ? 0 : Math.round(fontSize * 1.35)}">${esc(line)}</tspan>`
    ),
    `</text>`,
  ].join("\n");
}

function renderNode(node, offsetX, offsetY, defs, clipCounter) {
  const x = Number(node.x ?? 0) + offsetX;
  const y = Number(node.y ?? 0) + offsetY;
  const type = node.type;

  if (type === "frame") {
    const width = Number(node.width ?? 0);
    const height = Number(node.height ?? 0);
    const radius = Number(node.cornerRadius ?? 0);
    const stroke = node.stroke?.fill ? esc(node.stroke.fill) : null;
    const strokeWidth = Number(node.stroke?.thickness ?? 1);
    const parts = [];

    if (node.fill && typeof node.fill === "object" && node.fill.type === "image" && node.fill.url) {
      parts.push(
        renderImage(
          {
            href: node.fill.url,
            x,
            y,
            w: width,
            h: height,
            radius,
            stroke,
            strokeWidth,
          },
          defs,
          clipCounter
        )
      );
    } else {
      const fill = typeof node.fill === "string" ? esc(node.fill) : "none";
      parts.push(
        `<rect x="${x}" y="${y}" width="${width}" height="${height}"${radius ? ` rx="${radius}"` : ""} fill="${fill}"${
          stroke ? ` stroke="${stroke}" stroke-width="${strokeWidth}"` : ""
        }/>`
      );
    }

    if (Array.isArray(node.children) && node.children.length > 0) {
      for (const child of node.children) {
        parts.push(renderNode(child, x, y, defs, clipCounter));
      }
    }
    return parts.join("\n");
  }

  if (type === "text") {
    return renderText(node, x, y);
  }

  if (type === "ellipse") {
    const width = Number(node.width ?? 0);
    const height = Number(node.height ?? 0);
    const cx = x + width / 2;
    const cy = y + height / 2;
    const rx = width / 2;
    const ry = height / 2;
    const stroke = node.stroke?.fill ? esc(node.stroke.fill) : null;
    const strokeWidth = Number(node.stroke?.thickness ?? 1);
    const fill = typeof node.fill === "string" ? esc(node.fill) : "none";
    return `<ellipse cx="${cx}" cy="${cy}" rx="${rx}" ry="${ry}" fill="${fill}"${
      stroke ? ` stroke="${stroke}" stroke-width="${strokeWidth}"` : ""
    }/>`;
  }

  if (type === "path") {
    const width = Number(node.width ?? 0);
    const height = Number(node.height ?? 0);
    const viewBox = Array.isArray(node.viewBox) ? node.viewBox.join(" ") : "0 0 24 24";
    const stroke = node.stroke?.fill ? esc(node.stroke.fill) : null;
    const strokeWidth = Number(node.stroke?.thickness ?? 2);
    const linejoin = esc(node.stroke?.join ?? "round");
    const linecap = esc(node.stroke?.cap ?? "round");
    const fill = typeof node.fill === "string" ? esc(node.fill) : "none";
    return `<svg x="${x}" y="${y}" width="${width}" height="${height}" viewBox="${viewBox}"><path d="${esc(
      node.geometry ?? ""
    )}" fill="${fill}"${stroke ? ` stroke="${stroke}" stroke-width="${strokeWidth}" stroke-linejoin="${linejoin}" stroke-linecap="${linecap}"` : ""}/></svg>`;
  }

  if (type === "image") {
    return renderImage(
      {
        href: node.url ?? "",
        x,
        y,
        w: Number(node.width ?? 0),
        h: Number(node.height ?? 0),
        radius: Number(node.cornerRadius ?? 0),
      },
      defs,
      clipCounter
    );
  }

  return "";
}

function penToSvg(penPath, svgPath) {
  const pen = JSON.parse(fs.readFileSync(penPath, "utf8"));
  const root = pen.children?.[0];
  if (!root || root.type !== "frame") {
    fail(`Expected root frame in ${penPath}`);
  }

  const defs = [];
  const clipCounter = { value: 1 };
  const body = [];

  if (Array.isArray(root.children)) {
    for (const child of root.children) {
      body.push(renderNode(child, 0, 0, defs, clipCounter));
    }
  }

  const parts = [
    `<?xml version="1.0" encoding="UTF-8"?>`,
    `<svg xmlns="http://www.w3.org/2000/svg" width="${root.width}" height="${root.height}" viewBox="0 0 ${root.width} ${root.height}">`,
  ];
  if (defs.length > 0) {
    parts.push(`<defs>`);
    parts.push(...defs);
    parts.push(`</defs>`);
  }
  parts.push(...body);
  parts.push(`</svg>`);

  fs.writeFileSync(svgPath, `${parts.join("\n")}\n`, "utf8");
}

function penToPng(penPath, pngPath) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "accountsdot-pencil-single-"));
  const input = `export_nodes({ nodeIds: ["root"], outputDir: ${JSON.stringify(
    tmpDir
  )}, scale: 2, format: "png" })\nexit()\n`;

  const result = spawnSync(pencilBin, ["interactive", "-i", penPath, "-o", penPath], {
    input,
    encoding: "utf8",
    maxBuffer: 20 * 1024 * 1024,
  });

  const exported = path.join(tmpDir, "root.png");
  const exportedExists = fs.existsSync(exported);
  const benignCloseBug =
    result.status !== 0 &&
    exportedExists &&
    (result.stderr ?? "").includes("ERR_USE_AFTER_CLOSE");

  if ((!exportedExists || result.status !== 0) && !benignCloseBug) {
    fail(
      [
        `Pencil export failed for ${path.basename(penPath)}`,
        `exit=${result.status}`,
        result.stdout?.trim() ? `stdout:\n${result.stdout.trim()}` : "",
        result.stderr?.trim() ? `stderr:\n${result.stderr.trim()}` : "",
      ]
        .filter(Boolean)
        .join("\n\n")
    );
  }

  fs.copyFileSync(exported, pngPath);
  fs.rmSync(tmpDir, { recursive: true, force: true });
}

const inputPath = process.argv[2];
if (!inputPath) {
  fail("Usage: node export_pen_assets.mjs <wireframe.pen>");
}

const penPath = path.resolve(inputPath);
if (!fs.existsSync(penPath)) {
  fail(`File not found: ${penPath}`);
}
if (!penPath.endsWith(".pen")) {
  fail(`Expected a .pen file: ${penPath}`);
}

const pngPath = penPath.replace(/\.pen$/, ".png");
const svgPath = penPath.replace(/\.pen$/, ".svg");

penToSvg(penPath, svgPath);
penToPng(penPath, pngPath);

console.log(`Exported ${path.basename(svgPath)}`);
console.log(`Exported ${path.basename(pngPath)}`);
