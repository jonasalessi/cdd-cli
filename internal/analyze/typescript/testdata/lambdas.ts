// lambda fixture: arrow functions and function expressions.

// Expected total for `lambdas`: 3.
export function lambdas(xs: number[]): number[] {
  const double = (n: number): number => n * 2; // +1  arrow
  const triple = function (n: number): number {
    // +1  function expression
    return n * 3;
  };
  return xs.map((n) => double(n) + triple(n)); // +1  callback arrow
}

// Expected total for `unitArrow`: 1 — the unit's own arrow is the unit, not
// one of its lambdas; only the nested callback counts.
export const unitArrow = (xs: number[]): number[] => xs.map((n) => n + 1);

// Expected total for `Methods`: 0 — a method, a constructor and a getter
// are not lambdas.
export class Methods {
  constructor(private readonly n: number) {}

  get value(): number {
    return this.n;
  }

  method(): number {
    return this.n;
  }
}
