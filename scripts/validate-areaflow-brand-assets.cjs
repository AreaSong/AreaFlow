#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const sharp = require("sharp");
const { expandRasterJobs, loadBrandManifest, repoRoot } = require("./brand-assets.cjs");

async function main() {
  const manifest = loadBrandManifest();
  const packageRoot = path.join(repoRoot, manifest.packageRoot);
  const errors = [];

  validateManifest(manifest, errors);
  for (const job of expandRasterJobs(manifest)) {
    requireFile(path.join(packageRoot, job.source), errors);
    await validateRaster(path.join(packageRoot, job.output), job.width, job.height, job.alpha, errors);
  }
  await validateRaster(
    path.join(packageRoot, manifest.overview.output),
    manifest.overview.width,
    manifest.overview.height,
    true,
    errors,
  );
  validateIco(path.join(packageRoot, manifest.favicon.ico), manifest.favicon.sizes, errors);
  await validateNative(packageRoot, manifest, errors);
  validateSvgSources(packageRoot, manifest, errors);
  validateDocumentation(packageRoot, errors);
  validateMetadataFiles(path.join(repoRoot, "assets", "brand"), errors);

  if (errors.length > 0) {
    console.error(errors.map((error) => `FAIL ${error}`).join("\n"));
    process.exitCode = 1;
    return;
  }
  console.log(
    `brand assets valid: ${expandRasterJobs(manifest).length} raster exports, native and print deliverables, docs, and CI contract`,
  );
}

function validateManifest(manifest, errors) {
  if (manifest.schemaVersion !== 1 || manifest.brand !== "AreaFlow") {
    errors.push("brand manifest identity or schema version is invalid");
  }
  if (manifest.native?.printDpi !== 300) {
    errors.push(`brand manifest printDpi must be 300, got ${manifest.native?.printDpi}`);
  }
}

async function validateRaster(file, width, height, alpha, errors) {
  if (!fs.existsSync(file)) {
    errors.push(`missing raster: ${relative(file)}`);
    return;
  }
  const metadata = await sharp(file).metadata();
  if (metadata.width !== width || metadata.height !== height) {
    errors.push(
      `wrong dimensions: ${relative(file)} expected ${width}x${height}, got ${metadata.width}x${metadata.height}`,
    );
  }
  if (Boolean(metadata.hasAlpha) !== alpha) {
    errors.push(`wrong alpha mode: ${relative(file)} expected alpha=${alpha}, got ${Boolean(metadata.hasAlpha)}`);
  }
}

async function validateNative(packageRoot, manifest, errors) {
  validateNativeFiles(packageRoot, errors);
  await validateMacos(packageRoot, errors);
  await validateIosAppIconSet(path.join(packageRoot, "native/ios/AreaFlowAppIcon.appiconset"), errors);
  await validateAndroid(packageRoot, errors);
  validateIco(path.join(packageRoot, "native/windows/AreaFlow.ico"), [16, 24, 32, 48, 64, 128, 256], errors);
  await validatePrint(packageRoot, manifest.native.printDpi, errors);
}

function validateNativeFiles(packageRoot, errors) {
  const required = [
    "native/README.md",
    "native/macos/AreaFlow.icns",
    "native/ios/AreaFlowAppIcon.appiconset/Contents.json",
    "native/android/res/drawable-nodpi/areaflow_adaptive_foreground.png",
    "native/android/res/drawable-nodpi/areaflow_adaptive_background.png",
    "native/android/res/values/colors.xml",
    "native/android/res/mipmap-anydpi-v26/ic_launcher.xml",
    "native/android/res/mipmap-anydpi-v26/ic_launcher_round.xml",
    "native/windows/AreaFlow.ico",
    "print/README.md",
    "print/areaflow-logo-light-background.svg",
    "print/areaflow-logo-dark-background.svg",
    "print/areaflow-logo-light-background.pdf",
    "print/areaflow-logo-dark-background.pdf",
    "print/areaflow-logo-light-background-cmyk.tiff",
    "print/areaflow-logo-dark-background-cmyk.tiff",
  ];
  for (const file of required) requireFile(path.join(packageRoot, file), errors);
}

async function validateMacos(packageRoot, errors) {
  for (const size of [16, 32, 128, 256, 512]) {
    await validateRaster(path.join(packageRoot, `native/macos/AreaFlow.iconset/icon_${size}x${size}.png`), size, size, false, errors);
    await validateRaster(
      path.join(packageRoot, `native/macos/AreaFlow.iconset/icon_${size}x${size}@2x.png`),
      size * 2,
      size * 2,
      false,
      errors,
    );
  }
  validateMagic(path.join(packageRoot, "native/macos/AreaFlow.icns"), "icns", errors);
}

async function validateAndroid(packageRoot, errors) {
  await validateRaster(
    path.join(packageRoot, "native/android/res/drawable-nodpi/areaflow_adaptive_foreground.png"),
    432,
    432,
    true,
    errors,
  );
  await validateRaster(
    path.join(packageRoot, "native/android/res/drawable-nodpi/areaflow_adaptive_background.png"),
    432,
    432,
    false,
    errors,
  );
}

async function validatePrint(packageRoot, printDpi, errors) {
  validateMagic(path.join(packageRoot, "print/areaflow-logo-light-background.pdf"), "%PDF", errors);
  validateMagic(path.join(packageRoot, "print/areaflow-logo-dark-background.pdf"), "%PDF", errors);
  await validatePrintTiff(
    path.join(packageRoot, "print/areaflow-logo-light-background-cmyk.tiff"),
    printDpi,
    errors,
  );
  await validatePrintTiff(
    path.join(packageRoot, "print/areaflow-logo-dark-background-cmyk.tiff"),
    printDpi,
    errors,
  );
}

async function validateIosAppIconSet(directory, errors) {
  const contentsPath = path.join(directory, "Contents.json");
  if (!fs.existsSync(contentsPath)) return;
  const contents = JSON.parse(fs.readFileSync(contentsPath, "utf8"));
  if (!contents.images || contents.images.length !== 18) {
    errors.push(`iOS AppIcon expected 18 slots, got ${contents.images?.length ?? 0}`);
    return;
  }
  for (const image of contents.images) {
    if (!image.filename || !image.size || !image.scale) {
      errors.push("iOS AppIcon entry is incomplete");
      continue;
    }
    const points = Number.parseFloat(image.size.split("x")[0]);
    const scale = Number.parseInt(image.scale, 10);
    const pixels = Math.round(points * scale);
    await validateRaster(path.join(directory, image.filename), pixels, pixels, false, errors);
  }
}

async function validatePrintTiff(file, expectedDpi, errors) {
  if (!fs.existsSync(file)) return;
  const metadata = await sharp(file).metadata();
  if (metadata.width !== 3600 || metadata.height !== 1170 || metadata.space !== "cmyk") {
    errors.push(
      `invalid print TIFF: ${relative(file)} expected 3600x1170 CMYK, got ${metadata.width}x${metadata.height} ${metadata.space}`,
    );
  }
  if (metadata.density !== expectedDpi || metadata.resolutionUnit !== "inch") {
    errors.push(
      `wrong print TIFF density: ${relative(file)} expected ${expectedDpi} DPI/inch, got ${metadata.density} ${metadata.resolutionUnit}`,
    );
  }
}

function validateIco(file, expectedSizes, errors) {
  if (!fs.existsSync(file)) {
    errors.push(`missing ICO: ${relative(file)}`);
    return;
  }
  const buffer = fs.readFileSync(file);
  if (buffer.length < 6 || buffer.readUInt16LE(2) !== 1) {
    errors.push(`invalid ICO header: ${relative(file)}`);
    return;
  }
  const count = buffer.readUInt16LE(4);
  const sizes = Array.from({ length: count }, (_, index) => buffer.readUInt8(6 + index * 16) || 256);
  if (sizes.join(",") !== expectedSizes.join(",")) {
    errors.push(`wrong ICO sizes: ${relative(file)} expected ${expectedSizes.join(",")}, got ${sizes.join(",")}`);
  }
}

function validateSvgSources(packageRoot, manifest, errors) {
  const sources = new Set(expandRasterJobs(manifest).map((job) => job.source).filter((file) => file.endsWith(".svg")));
  sources.add(manifest.native.printLightSource);
  sources.add(manifest.native.printDarkSource);
  for (const source of sources) {
    const file = path.join(packageRoot, source);
    if (!fs.existsSync(file)) continue;
    const content = fs.readFileSync(file, "utf8");
    if (!content.includes("<svg") || !content.includes("viewBox=")) {
      errors.push(`invalid SVG root or viewBox: ${relative(file)}`);
    }
    if (source.includes("outlined") && /<text\b/i.test(content)) {
      errors.push(`outlined SVG contains text: ${relative(file)}`);
    }
  }
}

function validateDocumentation(packageRoot, errors) {
  const required = [
    path.join(repoRoot, "assets/brand/README.md"),
    path.join(repoRoot, "assets/brand/brand-manifest.json"),
    path.join(packageRoot, "README.md"),
    path.join(repoRoot, "scripts/brand-assets.cjs"),
    path.join(repoRoot, "scripts/generate-areaflow-brand-assets.cjs"),
    path.join(repoRoot, "scripts/validate-areaflow-brand-assets.cjs"),
    path.join(repoRoot, ".github/workflows/brand-assets.yml"),
    path.join(repoRoot, "package.json"),
  ];
  for (const file of required) requireFile(file, errors);
  requireText(path.join(repoRoot, "package.json"), '"brand:export"', errors);
  requireText(path.join(repoRoot, "package.json"), '"brand:validate"', errors);
  requireText(
    path.join(repoRoot, "Makefile"),
    "check: fmt-check test build web-build desktop-build docs-check governance-check contract-check brand-validate",
    errors,
  );
  requireText(
    path.join(repoRoot, ".github/workflows/brand-assets.yml"),
    "npm run brand:validate",
    errors,
  );
  requireText(path.join(packageRoot, "README.md"), "npm run brand:validate", errors);
}

function validateMetadataFiles(directory, errors) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) validateMetadataFiles(file, errors);
    if ([".DS_Store", "Thumbs.db"].includes(entry.name) || entry.name.endsWith("~")) {
      errors.push(`metadata file present: ${relative(file)}`);
    }
  }
}

function validateMagic(file, magic, errors) {
  if (!fs.existsSync(file)) return;
  const buffer = fs.readFileSync(file);
  if (buffer.subarray(0, magic.length).toString("ascii") !== magic) {
    errors.push(`wrong file signature: ${relative(file)}`);
  }
}

function requireFile(file, errors) {
  if (!fs.existsSync(file) || !fs.statSync(file).isFile()) errors.push(`missing file: ${relative(file)}`);
}

function requireText(file, expected, errors) {
  if (!fs.existsSync(file)) return;
  if (!fs.readFileSync(file, "utf8").includes(expected)) {
    errors.push(`missing required text in ${relative(file)}: ${expected}`);
  }
}

function relative(file) {
  return path.relative(repoRoot, file);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
