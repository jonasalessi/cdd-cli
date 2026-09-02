// A legacy <Type>expr cast only parses with the plain grammar; the TSX
// grammar reads the angle bracket as a JSX element.
export function cast(value: unknown): string {
  return <string>value;
}
