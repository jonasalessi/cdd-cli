// The worked examples of docs/cdd.md, verbatim.

// "if (a > b && c < d) scores 3 ICPs: 1 for the if and 1 for each Boolean
// condition." -> code_branch 1, condition 2.
export function docCondition(a: number, b: number, c: number, d: number): boolean {
  if (a > b && c < d) {
    return true;
  }
  return false;
}

// "1.0 ICP per branch (if = 1 …)".
export function docIf(a: number): number {
  if (a > 0) {
    return 1;
  }
  return 0;
}

// "… if-else = 2".
export function docIfElse(a: number): number {
  if (a > 0) {
    return 1;
  } else {
    return 0;
  }
}

// "A complete try-catch-finally block counts as 3 ICPs (1 for each block)."
export function docTryCatchFinally(v: string): number {
  try {
    return JSON.parse(v);
  } catch {
    return -1;
  } finally {
    void 0;
  }
}
