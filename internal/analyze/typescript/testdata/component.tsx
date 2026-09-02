// JSX only parses with the TSX grammar.
import { Button } from "./button";

export function Panel(): JSX.Element {
  return <Button label="ok" />;
}
