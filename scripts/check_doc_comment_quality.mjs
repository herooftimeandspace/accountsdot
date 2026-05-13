import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(__filename), "..");

const scannedRoots = ["cmd", "internal", path.join("frontend", "src")];
const scannedExtensions = new Set([".go", ".js", ".jsx", ".ts", ".tsx", ".css"]);
const templateExpressionExtensions = new Set([".js", ".jsx", ".ts", ".tsx"]);
const excludedPathSegments = new Set(["generated", "dist", "node_modules", ".cache"]);

const placeholderPatterns = [
  {
    id: "generic-data-flow",
    pattern: /\bdocuments (?:the )?(?:runtime )?data flow for\b/i,
  },
  {
    id: "generic-ui-surface",
    pattern: /\brenders the UI surface for\b/i,
  },
  {
    id: "generic-derived-data",
    pattern: /\bbuilds derived data for\b/i,
  },
  {
    id: "generic-load-decode",
    pattern: /\bloads or decodes data for\b/i,
  },
  {
    id: "generic-decision-data",
    pattern: /\bresolves decision data for\b/i,
  },
  {
    id: "generic-build-value",
    pattern: /\bbuilds the value used by\b/i,
  },
  {
    id: "generic-response-payload",
    pattern: /\bwrites the response payload for\b/i,
  },
  {
    id: "generic-request-path",
    pattern: /\bhandles the request path for\b/i,
  },
  {
    id: "generic-user-network-event",
    pattern: /\bhandles the user or network event for\b/i,
  },
  {
    id: "signature-input-output-placeholder",
    pattern: /\bInputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller\b/i,
  },
  {
    id: "generic-side-effect-warning",
    pattern: /\bPay special attention to side effects: this path may (?:update|mutate) .*docs\/external-write-inventory\.md\b/i,
  },
];

const baselinePath = path.join(repoRoot, "scripts", "doc_comment_quality_baseline.json");

function loadAllowedPlaceholderComments() {
  if (!fs.existsSync(baselinePath)) {
    return new Set();
  }
  const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf8"));
  if (!Array.isArray(baseline.allowedPlaceholderComments)) {
    throw new Error(`${path.relative(repoRoot, baselinePath)} must contain allowedPlaceholderComments`);
  }
  return new Set(baseline.allowedPlaceholderComments);
}

function normalizeComment(value) {
  return value
    .split(/\r?\n/)
    .map((line) =>
      line
        .replace(/^\s*\/{2,}\s?/, "")
        .replace(/^\s*\/\*\*?\s?/, "")
        .replace(/\s*\*\/\s*$/, "")
        .replace(/^\s*\*\s?/, "")
        .trim()
    )
    .filter(Boolean)
    .join(" ")
    .replace(/\s+/g, " ")
    .trim();
}

function shouldSkip(relativePath) {
  return relativePath
    .split(path.sep)
    .some((segment) => excludedPathSegments.has(segment));
}

function walkFiles(root) {
  const files = [];

  function walk(current) {
    const entries = fs.readdirSync(current, { withFileTypes: true });
    for (const entry of entries) {
      const absolutePath = path.join(current, entry.name);
      const relativePath = path.relative(repoRoot, absolutePath);
      if (shouldSkip(relativePath)) {
        continue;
      }
      if (entry.isDirectory()) {
        walk(absolutePath);
        continue;
      }
      if (entry.isFile() && scannedExtensions.has(path.extname(entry.name))) {
        files.push(absolutePath);
      }
    }
  }

  walk(root);
  return files;
}

function extractComments(source, { parseTemplateExpressions = true } = {}) {
  const comments = [];
  let index = 0;
  let line = 1;
  let state = "code";
  let quote = "";
  const stateStack = [];
  let templateExpressionDepth = 0;

  function advance(char) {
    index += 1;
    if (char === "\n") {
      line += 1;
    }
  }

  function enterState(nextState) {
    stateStack.push({ state, quote, templateExpressionDepth });
    state = nextState;
    quote = "";
  }

  function leaveState() {
    const previous = stateStack.pop();
    if (!previous) {
      state = "code";
      quote = "";
      templateExpressionDepth = 0;
      return;
    }
    state = previous.state;
    quote = previous.quote;
    templateExpressionDepth = previous.templateExpressionDepth;
  }

  while (index < source.length) {
    const char = source[index];
    const next = source[index + 1];

    if (state === "line-comment") {
      const startLine = line;
      const start = index;
      while (index < source.length && source[index] !== "\n") {
        index += 1;
      }
      comments.push({ line: startLine, text: source.slice(start, index) });
      leaveState();
      continue;
    }

    if (state === "block-comment") {
      const startLine = line;
      const start = index;
      while (index < source.length && !(source[index] === "*" && source[index + 1] === "/")) {
        advance(source[index]);
      }
      if (index < source.length) {
        advance(source[index]);
        advance(source[index]);
      }
      comments.push({ line: startLine, text: source.slice(start, index) });
      leaveState();
      continue;
    }

    if (state === "string") {
      if (char === "\\") {
        advance(char);
        if (index < source.length) {
          advance(source[index]);
        }
        continue;
      }
      if (parseTemplateExpressions && quote === "`" && char === "$" && next === "{") {
        enterState("template-expression");
        templateExpressionDepth = 1;
        advance(char);
        advance(next);
        continue;
      }
      if (char === quote) {
        leaveState();
      }
      advance(char);
      continue;
    }

    if (state === "template-expression") {
      if (char === "/" && next === "/") {
        enterState("line-comment");
        continue;
      }
      if (char === "/" && next === "*") {
        enterState("block-comment");
        continue;
      }
      if (char === "\"" || char === "'" || char === "`") {
        enterState("string");
        quote = char;
        advance(char);
        continue;
      }
      if (char === "{") {
        templateExpressionDepth += 1;
        advance(char);
        continue;
      }
      if (char === "}") {
        templateExpressionDepth -= 1;
        advance(char);
        if (templateExpressionDepth === 0) {
          leaveState();
        }
        continue;
      }
      advance(char);
      continue;
    }

    if (char === "/" && next === "/") {
      enterState("line-comment");
      continue;
    }
    if (char === "/" && next === "*") {
      enterState("block-comment");
      continue;
    }
    if (char === "\"" || char === "'" || char === "`") {
      enterState("string");
      quote = char;
      advance(char);
      continue;
    }
    advance(char);
  }

  return comments;
}

function findPlaceholderMatches({ allowedComments = new Set() } = {}) {
  const matches = [];
  for (const root of scannedRoots) {
    const absoluteRoot = path.join(repoRoot, root);
    if (!fs.existsSync(absoluteRoot)) {
      continue;
    }
    for (const absolutePath of walkFiles(absoluteRoot)) {
      const relativePath = path.relative(repoRoot, absolutePath).split(path.sep).join("/");
      const extension = path.extname(relativePath);
      const source = fs.readFileSync(absolutePath, "utf8");
      for (const comment of extractComments(source, {
        parseTemplateExpressions: templateExpressionExtensions.has(extension),
      })) {
        const normalized = normalizeComment(comment.text);
        if (!normalized) {
          continue;
        }
        const patternIds = placeholderPatterns
          .filter((entry) => entry.pattern.test(normalized))
          .map((entry) => entry.id);
        if (patternIds.length === 0) {
          continue;
        }
        const key = `${relativePath}:${comment.line}:${normalized}`;
        if (allowedComments.has(key)) {
          continue;
        }
        matches.push({
          relativePath,
          line: comment.line,
          patternIds,
          text: normalized,
          key,
        });
      }
    }
  }
  return matches;
}

function writeBaseline() {
  const matches = findPlaceholderMatches();
  const baseline = {
    note:
      "Temporary baseline for placeholder comments inherited from issue #3 documentation work. Do not add entries for new code; rewrite the comment with specific caller, data-flow, output, and side-effect details instead.",
    allowedPlaceholderComments: [...new Set(matches.map((match) => match.key))].sort(),
  };
  fs.writeFileSync(`${baselinePath}.tmp`, `${JSON.stringify(baseline, null, 2)}\n`);
  fs.renameSync(`${baselinePath}.tmp`, baselinePath);
  console.log(
    `wrote ${baseline.allowedPlaceholderComments.length} baseline entries to ${path.relative(repoRoot, baselinePath)}`
  );
}

function runSelfTest() {
  const sample = `
const notAComment = "handleSave handles the user or network event for frontend/src/pages/Example.jsx.";
// handleSave handles the user or network event for frontend/src/pages/Example.jsx. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
/* handleRoute handles the request path for internal/web/example.go. */
/** ExamplePage renders the UI surface for frontend/src/pages/ExamplePage.jsx. */
`;
  const comments = extractComments(sample).map((comment) => normalizeComment(comment.text));
  if (comments.length !== 3) {
    throw new Error(`expected 3 comments, found ${comments.length}`);
  }
  const requestPathPattern = placeholderPatterns.find((entry) => entry.id === "generic-request-path");
  const eventPattern = placeholderPatterns.find((entry) => entry.id === "generic-user-network-event");
  if (!requestPathPattern.pattern.test(comments[1]) || !eventPattern.pattern.test(comments[0])) {
    throw new Error("request or event placeholder patterns did not match extracted comments");
  }
  const uiSurfacePattern = placeholderPatterns.find((entry) => entry.id === "generic-ui-surface");
  if (!uiSurfacePattern.pattern.test(comments[2])) {
    throw new Error("generic UI placeholder pattern did not match extracted comments");
  }
  const goRawStringSample = "const raw = `${`;\n// goRaw documents the data flow for internal/example.go.";
  const goRawComments = extractComments(goRawStringSample, { parseTemplateExpressions: false });
  if (goRawComments.length !== 1) {
    throw new Error("Go raw strings containing ${ must not hide later comments");
  }
  const jsTemplateSample = "const value = `prefix ${\"${\"} suffix`;\n// jsTemplate documents the data flow for frontend/src/example.js.";
  const jsTemplateComments = extractComments(jsTemplateSample, { parseTemplateExpressions: true });
  if (jsTemplateComments.length !== 1) {
    throw new Error("quoted ${ text inside JS template expressions must not hide later comments");
  }
  const duplicatePlaceholder = [
    "// duplicate documents the data flow for frontend/src/example.js.",
    "// duplicate documents the data flow for frontend/src/example.js.",
  ].join("\n");
  const duplicateMatches = [];
  for (const comment of extractComments(duplicatePlaceholder)) {
    const normalized = normalizeComment(comment.text);
    duplicateMatches.push(`frontend/src/example.js:${comment.line}:${normalized}`);
  }
  if (new Set(duplicateMatches).size !== 2) {
    throw new Error("baseline keys must distinguish copied placeholder comments by line");
  }
  console.log("doc-comment-quality self-test passed");
}

if (process.argv.includes("--self-test")) {
  runSelfTest();
  process.exit(0);
}

if (process.argv.includes("--update-baseline")) {
  writeBaseline();
  process.exit(0);
}

const matches = findPlaceholderMatches({ allowedComments: loadAllowedPlaceholderComments() });
if (matches.length > 0) {
  console.error("Boilerplate documentation comments found.");
  console.error("Rewrite these comments with specific caller, data-flow, output, and side-effect details.");
  for (const match of matches) {
    console.error(
      `${match.relativePath}:${match.line} [${match.patternIds.join(", ")}] ${match.text}`
    );
  }
  process.exit(1);
}

console.log("doc-comment-quality check passed");
