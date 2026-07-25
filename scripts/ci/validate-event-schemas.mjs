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

for (const file of schemaFiles) {
  const path = join(schemaDirectory, file);
  const schema = JSON.parse(await readFile(path, "utf8"));

  if (!ajv.validateSchema(schema)) {
    console.error(ajv.errorsText(ajv.errors, { separator: "\n" }));
    process.exitCode = 1;
    continue;
  }

  ajv.compile(schema);
  console.log(`schema ${path} is valid`);
}
