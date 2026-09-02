// Every unit kind, and every declaration that is not a unit.
import { helper } from "./helper";

export class Exported {}

class Internal {}

export abstract class Abstract {}

export interface Shape {
  area(): number;
}

export enum Color {
  Red,
}

export type Alias = string;

export function exported(): void {
  // A nested function is not a unit: it belongs to the unit around it.
  function nested(): void {}
  nested();
}

function plain(): void {}

export function* generated(): Generator<number> {
  yield 1;
}

export const arrow = (): void => {};

export const expr = function (): void {};

export const first = (): void => {},
  second = function (): void {};

// Not a unit: not exported.
const notExported = (): void => {};

// Not a unit: exported, but not a function.
export const value = 42;

// Not units: overload signatures have no body.
function overloaded(a: string): void;
function overloaded(a: number): void;
// A unit: the implementation.
function overloaded(a: unknown): void {
  void a;
}

// Not units: ambient declarations.
declare function ambient(a: number): void;
declare const ambientConst: number;
declare module "side" {}

// Not a unit: a re-export.
export { helper };

// A unit named "default".
export default class {}

void notExported;
void value;
void plain;
void overloaded;
void ambient;
void ambientConst;
void first;
void second;
void expr;
void arrow;
void generated;
void Internal;
