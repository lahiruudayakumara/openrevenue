import { readdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import process from "node:process";

import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";

const schemaDirectory = "contracts/events";
const schemaFiles = (await readdir(schemaDirectory))
  .filter((file) => file.endsWith(".json"))
  .sort();

if (schemaFiles.length === 0) {
  throw new Error(`No event schemas found in ${schemaDirectory}`);
}

const ajv = new Ajv2020({ allErrors: true, strict: true });
addFormats(ajv);
let envelopeValidator;

for (const file of schemaFiles) {
  const path = join(schemaDirectory, file);
  const schema = JSON.parse(await readFile(path, "utf8"));

  if (!ajv.validateSchema(schema)) {
    console.error(ajv.errorsText(ajv.errors, { separator: "\n" }));
    process.exitCode = 1;
    continue;
  }

  const validator = ajv.compile(schema);
  if (file === "envelope.schema.json") envelopeValidator = validator;
  console.log(`schema ${path} is valid`);
}

const exampleDirectory = join(schemaDirectory, "examples");
const exampleFiles = (await readdir(exampleDirectory))
  .filter((file) => file.endsWith(".json"))
  .sort();
if (!envelopeValidator || exampleFiles.length === 0) {
  throw new Error("The event envelope and at least one example are required");
}
for (const file of exampleFiles) {
  const path = join(exampleDirectory, file);
  const example = JSON.parse(await readFile(path, "utf8"));
  if (!envelopeValidator(example)) {
    console.error(`${path}\n${ajv.errorsText(envelopeValidator.errors, { separator: "\n" })}`);
    process.exitCode = 1;
  } else {
    console.log(`example ${path} is valid`);
  }
}

const serviceSource = await readFile("internal/administration/application/service.go", "utf8");
const declaredTypes = new Set(
  JSON.parse(await readFile(join(schemaDirectory, "envelope.schema.json"), "utf8"))
    .properties.eventType.enum,
);
for (const match of serviceSource.matchAll(/s\.emit\(scope,\s*"([^"]+)"/g)) {
  if (!declaredTypes.has(match[1])) {
    console.error(`implemented event ${match[1]} is missing from the canonical envelope`);
    process.exitCode = 1;
  }
}
