// inheritance fixture: 1.0 ICP per inherited base or implemented contract.
export class Base<T = unknown> {
  value?: T;
}
export interface First {}
export interface Second {}
export interface One {}
export interface Two {}
export interface Three {}
export type Generic = string;

// Expected total for `Both`: 3 — one extends plus two implements.
export class Both extends Base<Generic> implements First, Second {}

// Expected total for `OnlyExtends`: 1 — the type argument is not a parent.
export class OnlyExtends extends Base<Generic> {}

// Expected total for `Wide`: 3 — one per listed parent interface.
export interface Wide extends One, Two, Three {}

// Expected total for `None`: 0.
export class None {}
