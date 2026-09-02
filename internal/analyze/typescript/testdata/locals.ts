// local_variable fixture: one ICP per declarator, plus class fields and the
// binding a `for…of` or `for…in` declares.

// Expected total for `locals`: 10.
export function locals(input: number[], pairs: [number, number][]): number {
  const a = 1; // +1
  let b = 2,
    c = 3; // +2  one per declarator
  var d = 4; // +1
  const { x, y } = { x: 1, y: 2 }; // +1  a destructuring declarator is one
  let seen = 0; // +1
  for (let i = 0; i < 1; i++) {
    // +1  the loop initialiser is a declarator too
    b += i;
  }
  for (const item of input) {
    // +1  a for…of that declares its binding
    b += item;
  }
  for (const k in input) {
    // +1  a for…in that declares its binding
    b += Number(k);
  }
  for (const [p, q] of pairs) {
    // +1  one per statement, even when the binding destructures
    b += p + q;
  }
  for (seen of input) {
    // +0  no `kind` keyword: it assigns to an existing variable
    b += seen;
  }
  return a + b + c + d + x + y + input.length;
}

// Expected total for `Fields`: 5 — four fields plus one method local.
// A `#private` field is a public_field_definition in the grammar, so it
// counts like any other field; methods and getters do not.
export class Fields {
  public a = 1; // +1
  #b = 2; // +1
  static c = 3; // +1
  private d = 4; // +1

  method(): number {
    const local = 1; // +1
    return local + this.a + this.d + Fields.c + this.#b;
  }
}

// Expected total for `Props`: 0 — property signatures declare a shape, not
// variables.
export interface Props {
  a: number;
  b: string;
}
