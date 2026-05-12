import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const wireframeDir = path.join(__dirname, "wireframes");
const pencilBin = "/opt/homebrew/bin/pencil";

const penFiles = fs
  .readdirSync(wireframeDir)
  .filter((name) => name.endsWith(".pen"))
  .sort();

if (penFiles.length === 0) {
  throw new Error(`No .pen files found in ${wireframeDir}`);
}

for (const file of penFiles) {
  const penPath = path.join(wireframeDir, file);
  const pngPath = penPath.replace(/\.pen$/, ".png");
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "accountsdot-pencil-"));
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
    throw new Error(
      [
        `Pencil export failed for ${file}`,
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
  console.log(`Exported ${path.basename(pngPath)}${benignCloseBug ? " (ignored readline close bug)" : ""}`);
}
