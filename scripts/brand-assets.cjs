const fs = require("node:fs");
const path = require("node:path");

const repoRoot = path.resolve(__dirname, "..");
const manifestPath = path.join(repoRoot, "assets", "brand", "brand-manifest.json");

function loadBrandManifest() {
  return JSON.parse(fs.readFileSync(manifestPath, "utf8"));
}

function expandRasterJobs(manifest) {
  const jobs = [];
  for (const family of manifest.themedSquareFamilies) {
    for (const theme of manifest.themes) {
      for (const size of family.sizes) {
        const useSmallSource = family.smallSource && family.smallSizes?.includes(size);
        jobs.push({
          source: replaceTokens(useSmallSource ? family.smallSource : family.source, theme, size),
          output: replaceTokens(family.output, theme, size),
          width: size,
          height: size,
          alpha: family.alpha,
        });
      }
    }
  }
  jobs.push(...manifest.fixedRasterExports);
  for (const size of manifest.favicon.sizes) {
    jobs.push({
      source: manifest.favicon.source,
      output: manifest.favicon.output.replace("{size}", String(size)),
      width: size,
      height: size,
      alpha: true,
    });
  }
  return jobs;
}

function replaceTokens(value, theme, size) {
  return value.replace("{theme}", theme).replace("{size}", String(size));
}

module.exports = { expandRasterJobs, loadBrandManifest, manifestPath, repoRoot };
