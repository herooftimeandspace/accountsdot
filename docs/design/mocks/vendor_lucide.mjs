import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.join(__dirname, "..", "..");
const packageRoot = path.join(repoRoot, "node_modules", "lucide-static");
const vendorRoot = path.join(repoRoot, "docs", "vendor", "lucide-static");
const vendorIconsDir = path.join(vendorRoot, "icons");

const ICONS = [
  "arrow-left-right",
  "bell",
  "building-2",
  "chart-column",
  "chevron-down",
  "circle-help",
  "contact-round",
  "files",
  "info",
  "layout-grid",
  "phone",
  "refresh-cw",
  "search",
  "settings-2",
  "sparkles",
  "square-pen",
  "star",
  "trash-2",
  "triangle-alert",
  "user-minus",
  "user-plus",
  "x"
];

if (!fs.existsSync(packageRoot)) {
  throw new Error(`lucide-static is not installed at ${packageRoot}`);
}

fs.mkdirSync(vendorIconsDir, { recursive: true });

for (const icon of ICONS) {
  const src = path.join(packageRoot, "icons", `${icon}.svg`);
  const dest = path.join(vendorIconsDir, `${icon}.svg`);
  if (!fs.existsSync(src)) {
    throw new Error(`Missing Lucide icon: ${icon}`);
  }
  fs.copyFileSync(src, dest);
}

for (const asset of ["LICENSE", "README.md", "package.json"]) {
  fs.copyFileSync(path.join(packageRoot, asset), path.join(vendorRoot, asset));
}

const pkg = JSON.parse(fs.readFileSync(path.join(packageRoot, "package.json"), "utf8"));
const manifest = {
  package: pkg.name,
  version: pkg.version,
  license: pkg.license,
  icons: ICONS
};
fs.writeFileSync(path.join(vendorRoot, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);

console.log(`Vendored ${ICONS.length} Lucide SVGs into ${path.relative(repoRoot, vendorRoot)}`);
