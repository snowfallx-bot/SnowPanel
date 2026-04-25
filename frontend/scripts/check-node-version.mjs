const MIN_NODE_VERSION = [20, 19, 0];

function parseVersion(version) {
  const [major = "0", minor = "0", patch = "0"] = version.split(".");
  return [Number(major), Number(minor), Number(patch)];
}

function isLowerThanMin(current, min) {
  for (let index = 0; index < min.length; index += 1) {
    if (current[index] < min[index]) {
      return true;
    }
    if (current[index] > min[index]) {
      return false;
    }
  }
  return false;
}

const currentNodeVersion = process.versions.node;
const currentParts = parseVersion(currentNodeVersion);

if (isLowerThanMin(currentParts, MIN_NODE_VERSION)) {
  const expected = MIN_NODE_VERSION.join(".");
  console.error(
    `[snowpanel/frontend] Node.js ${expected}+ is required for frontend test tooling. Current: ${currentNodeVersion}.`
  );
  console.error(
    "Please upgrade Node.js (22+ recommended) and retry `npm run test`."
  );
  process.exit(1);
}
