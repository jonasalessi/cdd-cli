// condition fixture: 1.0 ICP per Boolean clause (docs/cdd.md section 2),
// not per operator. Expected total for `conditions`: 32.
export function conditions(a: number, b: number, c: number, d: number): number {
  let n = 0;
  if (a > b && c < d) n = 1; // +2  a > b, c < d
  if (a && b || c) n = 2; // +3  a, b, c
  if ((a && b) || c) n = 3; // +3  parentheses change nothing
  if (!(a && b)) n = 4; // +2  a, b
  if (a > 1) n = 5; // +0  no logical operator, only the branch
  if (!(a || b) && c) n = 6; // +3  De Morgan: !a && !b && c
  if (!a && b) n = 7; // +2  a, b
  if (!!a || b) n = 8; // +2  a, b
  if (a && b && c && d) n = 9; // +4  one clause per leaf

  const x = a ?? b; // +2  ?? joins clauses like && and ||

  let y = a;
  y ||= b; // +2  sugar for y = y || b
  y &&= b; // +2
  y ??= b; // +2
  y ||= b && c; // +3  the left-hand side plus the two clauses on the right

  return n + x + y;
}
