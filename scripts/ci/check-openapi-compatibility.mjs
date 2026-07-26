import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import process from "node:process";

const baseRef = process.env.CONTRACT_BASE_REF;
if (!baseRef) {
  console.log("CONTRACT_BASE_REF is unset; compatibility comparison skipped locally");
  process.exit(0);
}
const directory = await mkdtemp(join(tmpdir(), "openrevenue-contract-"));
const source = join(directory, "openapi.yaml");
const bundled = join(directory, "openapi.json");
await writeFile(source, execFileSync("git", ["show", `${baseRef}:contracts/openapi/openapi.yaml`]));
execFileSync("node_modules/.bin/redocly", ["bundle", source, "--output", bundled], { stdio: "inherit" });
const previous = JSON.parse(await readFile(bundled, "utf8"));
const current = JSON.parse(await readFile(".contract-openapi.json", "utf8"));
const breaks = [];

for (const [path, item] of Object.entries(previous.paths ?? {})) {
  if (!current.paths?.[path]) {
    breaks.push(`removed path ${path}`);
    continue;
  }
  for (const method of ["get", "post", "put", "patch", "delete"]) {
    if (item[method] && !current.paths[path][method]) breaks.push(`removed operation ${method.toUpperCase()} ${path}`);
  }
}
for (const [name, schema] of Object.entries(previous.components?.schemas ?? {})) {
  const next = current.components?.schemas?.[name];
  if (!next) {
    breaks.push(`removed schema ${name}`);
    continue;
  }
  for (const property of Object.keys(schema.properties ?? {})) {
    if (!next.properties?.[property]) breaks.push(`removed property ${name}.${property}`);
  }
  for (const required of next.required ?? []) {
    if (!(schema.required ?? []).includes(required)) breaks.push(`new required property ${name}.${required}`);
  }
}
const previousEnvelope = JSON.parse(execFileSync("git", [
  "show", `${baseRef}:contracts/events/envelope.schema.json`,
], { encoding: "utf8" }));
const currentEnvelope = JSON.parse(await readFile("contracts/events/envelope.schema.json", "utf8"));
for (const property of Object.keys(previousEnvelope.properties ?? {})) {
  if (!currentEnvelope.properties?.[property]) breaks.push(`removed event envelope property ${property}`);
}
for (const required of currentEnvelope.required ?? []) {
  if (!(previousEnvelope.required ?? []).includes(required)) {
    breaks.push(`new required event envelope property ${required}`);
  }
}
for (const eventType of previousEnvelope.properties?.eventType?.enum ?? []) {
  if (!(currentEnvelope.properties?.eventType?.enum ?? []).includes(eventType)) {
    breaks.push(`removed event type ${eventType}`);
  }
}
if (breaks.length) {
  console.error(`Breaking OpenAPI changes require an explicit versioned migration decision:\n${breaks.join("\n")}`);
  process.exit(1);
}
console.log(`no breaking OpenAPI changes relative to ${baseRef}`);
