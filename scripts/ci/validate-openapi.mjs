import { readFile } from "node:fs/promises";
import process from "node:process";

const document = JSON.parse(await readFile(process.argv[2], "utf8"));
const operations = [];
const operationIds = new Set();
const errors = [];

for (const [path, item] of Object.entries(document.paths ?? {})) {
  for (const method of ["get", "post", "put", "patch", "delete"]) {
    const operation = item[method];
    if (!operation) continue;
    operations.push(`${method.toUpperCase()} ${path}`);
    if (!operation.operationId) errors.push(`${method.toUpperCase()} ${path} has no operationId`);
    else if (operationIds.has(operation.operationId)) errors.push(`duplicate operationId ${operation.operationId}`);
    else operationIds.add(operation.operationId);

    for (const [status, response] of Object.entries(operation.responses ?? {})) {
      if ((status === "default" || Number(status) >= 400) && status !== "401") {
        const resolved = response?.$ref === "#/components/responses/Problem"
          ? document.components.responses.Problem
          : response;
        const schema = resolved?.content?.["application/problem+json"]?.schema;
        if (schema?.$ref !== "#/components/schemas/Problem") {
          errors.push(`${method.toUpperCase()} ${path} response ${status} must use application/problem+json Problem`);
        }
      }
    }
  }
}

const router = await readFile("apps/api/http.go", "utf8");
for (const match of router.matchAll(/r\.(Get|Post|Put|Patch|Delete)\("([^"]+)"/g)) {
  if (!match[2].startsWith("/") || ["/health", "/ready", "/metrics"].includes(match[2])) continue;
  const normalized = match[2].replaceAll(/\{([^}]+)ID\}/g, (_, name) => `{${name}Id}`);
  const key = `${match[1].toUpperCase()} ${normalized}`;
  if (!operations.includes(key)) errors.push(`implemented route ${key} is missing from OpenAPI`);
}

if (errors.length) {
  console.error(errors.join("\n"));
  process.exit(1);
}
console.log(`validated ${operations.length} operations, unique operation IDs, routes, and error shapes`);
