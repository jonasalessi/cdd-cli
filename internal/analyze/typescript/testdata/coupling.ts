// coupling fixture, analyzed with InternalPrefixes = ["@app/"].
//
// internal modules: "./repo", "@app/services", "@app/config", "./polyfill"
// external modules: "node:fs/promises", "lodash/fp", "react", "ink",
//                   "reflect-metadata"
import { Repo } from "./repo";
import { Other } from "./repo"; // same module, counted once per unit
import { Service } from "@app/services";
import type { Config } from "@app/config";
import { readFile } from "node:fs/promises";
import * as lodash from "lodash/fp";
import React from "react";
import { render as draw } from "ink";
import "./polyfill"; // internal side effect: charged to every unit
import "reflect-metadata"; // external side effect: charged to every unit

// internal 2: ./repo, ./polyfill
// external 1: reflect-metadata
export class UsesInternal {
  private readonly repo = new Repo();

  get value(): Repo {
    return this.repo;
  }
}

// internal 1: ./polyfill
// external 3: node:fs/promises, lodash/fp, reflect-metadata
export function usesExternal(): void {
  void readFile;
  void lodash;
}

// internal 2: ./repo (once, for both bindings and both statements),
//             ./polyfill
// external 1: reflect-metadata
export function usesBoth(): void {
  void Repo;
  void Other;
}

// internal 3: @app/config, @app/services, ./polyfill
// external 1: reflect-metadata
export type Wrapper = {
  cfg: Config;
  svc: Service;
};

// internal 1: ./polyfill
// external 3: ink, react, reflect-metadata
export const renderer = (): unknown => draw(React.createElement("div"));

// internal 1: ./polyfill
// external 1: reflect-metadata
export interface Untouched {
  n: number;
}
