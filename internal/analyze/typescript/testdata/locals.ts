// local_variable fixture: one ICP per declarator, plus class fields.

// Expected total for `locals`: 6.
export function locals(input: number[]): number {
  const a = 1; // +1
  let b = 2,
    c = 3; // +2  one per declarator
  var d = 4; // +1
  const { x, y } = { x: 1, y: 2 }; // +1  a destructuring declarator is one
  for (let i = 0; i < 1; i++) {
    // +1  the loop initialiser is a declarator too
    b += i;
  }
  for (const item of input) {
    // +0  a for…of binding is not a declarator in the grammar
    b += item;
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
