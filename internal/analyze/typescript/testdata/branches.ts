// code_branch fixture. Expected total for `branches`: 14.
export function branches(v: number, o: { a?: { b: number } }): number {
  if (v === 0) {
    // +1  if
    return 0;
  } else if (v === 1) {
    // +1  the chained if; its `else` adds nothing, so if/else-if/else is 3
    return 1;
  } else {
    // +1  else
    return 2;
  }

  switch (v) {
    // the switch itself adds nothing
    case 1:
      // +1  case
      break;
    case 2:
      // +1  case
      break;
    default:
      // +0  default
      break;
  }

  const t = v > 0 ? 1 : -1; // +1  ternary

  for (let i = 0; i < 1; i++) {
    // +1  for
    void i;
  }
  for (const k in o) {
    // +1  for…in
    void k;
  }
  for (const n of [1]) {
    // +1  for…of, which is a for_in_statement in the grammar
    void n;
  }
  while (v > 5) {
    // +1  while
    break;
  }
  do {
    // +1  do…while
    break;
  } while (v > 5);

  const shallow = o?.a; // +1  optional chain
  const deep = o?.a?.b; // +2  two optional chains

  return t + Number(shallow) + Number(deep);
}

// Expected total for `optionalCalls`: 4.
//
// The grammar gives `?.` an `optional_chain` node in a member access
// (`a?.b`) and in a subscript access (`a?.[0]`) only. In an optional call
// the `?.` is an anonymous token of `call_expression`, whose rule is
// seq(field("function", …), "?.", field("arguments", …)), so it produces no
// node at all and adds nothing. Every `optional_chain` node is charged
// whatever its parent; there is simply none to charge for the call itself.
export function optionalCalls(f: any, o: any): unknown {
  const a = f?.(); // +0  optional call, no node in the grammar
  const b = o.f?.(); // +0  plain member access, then an optional call
  const c = o?.f?.(); // +1  the `?.f` member access only
  const d = o?.[0]?.(); // +1  the `?.[0]` subscript access only
  const e = o?.b?.(1)?.[2]; // +2  `?.b` and `?.[2]`, not the `?.(`
  return [a, b, c, d, e];
}
