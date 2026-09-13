import { createHash } from "node:crypto";
import { homedir } from "node:os";
import { join } from "node:path";
import { ClientTransaction, fetchXDocument } from "x-client-transaction-id";

type Contract = {
  host: string;
  path: string;
  method: string;
  encoding: "query" | "json";
  body?: Record<string, unknown>;
  variables: Record<string, unknown>;
  features: Record<string, unknown>;
  fieldToggles: Record<string, unknown>;
};

type Authentication = {
  cookies: Array<{ name: string; value: string }>;
  authorization: string;
};

const configDirectory =
  process.env.XDG_CONFIG_HOME ?? join(homedir(), ".config");
const contractsPath =
  process.env.TWT_CONTRACT_FILE ??
  join(configDirectory, "x-twitter-cli3", "contracts.json");
const authenticationPath = join(
  configDirectory,
  "x-twitter-cli3",
  "authentication.json",
);
const attempts = Number.parseInt(process.env.ATTEMPTS ?? "5", 10);
const query = process.env.QUERY ?? "x";

if (!Number.isInteger(attempts) || attempts < 1 || attempts > 20) {
  throw new Error("ATTEMPTS must be an integer between 1 and 20");
}

const contractsFile = await Bun.file(contractsPath).json();
const authentication =
  (await Bun.file(authenticationPath).json()) as Authentication;
const contract = contractsFile.operations.SearchTimeline as Contract;

if (contract.encoding !== "query") {
  throw new Error(`This probe expects query encoding, got ${contract.encoding}`);
}

const variables = { ...contract.variables, rawQuery: query };
const searchURL = new URL(`https://${contract.host}${contract.path}`);
searchURL.searchParams.set("variables", JSON.stringify(variables));
searchURL.searchParams.set("features", JSON.stringify(contract.features));
searchURL.searchParams.set(
  "fieldToggles",
  JSON.stringify(contract.fieldToggles),
);

const headers: Record<string, string> = {
  accept: "*/*",
  authorization: authentication.authorization,
  cookie: authentication.cookies
    .map(({ name, value }) => `${name}=${value}`)
    .join("; "),
  referer: `https://x.com/search?q=${encodeURIComponent(query)}&src=typed_query`,
  "x-twitter-active-user": "yes",
  "x-twitter-auth-type": "OAuth2Session",
  "x-twitter-client-language": "en",
};
const csrf = authentication.cookies.find(({ name }) => name === "ct0")?.value;
if (csrf) headers["x-csrf-token"] = csrf;

console.log(`contract: ${contract.method} ${contract.path}`);
console.log(`attempts: ${attempts}`);
console.log("initializing generator from live X frontend data...");
const initializationStarted = performance.now();
const document = await fetchXDocument();
const transaction = await ClientTransaction.create(document);
console.log(
  `generator initialized in ${Math.round(performance.now() - initializationStarted)}ms`,
);

const fingerprints = new Set<string>();
let successes = 0;

for (let attempt = 1; attempt <= attempts; attempt += 1) {
  const generationStarted = performance.now();
  const transactionId = await transaction.generateTransactionId(
    contract.method,
    contract.path,
  );
  const generationMs = performance.now() - generationStarted;
  const fingerprint = createHash("sha256")
    .update(transactionId)
    .digest("hex")
    .slice(0, 12);
  fingerprints.add(fingerprint);

  const requestStarted = performance.now();
  const response = await fetch(searchURL, {
    method: contract.method,
    headers: { ...headers, "x-client-transaction-id": transactionId },
  });
  const body = await response.text();
  const requestMs = performance.now() - requestStarted;
  const contentType = response.headers.get("content-type") ?? "missing";
  const graphqlShaped = body.startsWith("{") && body.includes('"data"');
  if (response.status === 200 && graphqlShaped) successes += 1;

  console.log(
    `attempt ${attempt}: status=${response.status} graphql=${graphqlShaped} ` +
      `idLength=${transactionId.length} idSha256=${fingerprint} ` +
      `generate=${generationMs.toFixed(2)}ms request=${requestMs.toFixed(0)}ms ` +
      `contentType=${contentType}`,
  );
  if (response.status !== 200) {
    console.log(`  failure body: ${body.slice(0, 300)}`);
  }
}

console.log(
  `verdict: ${successes}/${attempts} accepted; ${fingerprints.size}/${attempts} IDs unique`,
);
if (successes !== attempts || fingerprints.size !== attempts) process.exitCode = 1;
