/**
 * artboardHasSharedShell identifies generated artboards that actually contain
 * the logged-in shared shell. The renderer uses short Pencil ids such as `t6`
 * and `t7`; logged-out pages can reuse those ids for unrelated page text, so
 * sticky shell behavior must be gated by the shell frame instead of by child id
 * alone.
 */
export function artboardHasSharedShell(artboard) {
  const topLevelIds = new Set((artboard?.children || []).map((child) => child.id));
  return topLevelIds.has("f3") && topLevelIds.has("f4");
}
