#!/usr/bin/env node

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const {
  appIconSvg,
  lockupSvg,
  markSvg,
  monoLockupSvg,
  monoMarkSvg,
  nativeReadme,
  overviewSvg,
  printReadme,
  readme,
  socialSvg,
  stackedSvg,
  symbolSvg,
  wordmarkSvg,
} = require("./lib/areaflow-brand-svg.cjs");
const { expandRasterJobs, loadBrandManifest, repoRoot } = require("./brand-assets.cjs");

let sharp;
try {
  sharp = require("sharp");
} catch (error) {
  console.error("sharp is required to export the PNG brand assets.");
  console.error("Install sharp or expose it through NODE_PATH before running this script.");
  process.exit(1);
}

const manifest = loadBrandManifest();
const finalDir = path.join(repoRoot, manifest.packageRoot);
const printDpiPerMillimetre = manifest.native.printDpi / 25.4;
const refresh = process.argv.includes("--refresh");

function write(relativePath, content) {
  const target = path.join(finalDir, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, content);
}

async function render(svg, relativePath, width, height = width) {
  const target = path.join(finalDir, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  await sharp(Buffer.from(svg), { density: 300 })
    .resize(width, height, { fit: "fill" })
    .png({ compressionLevel: 9 })
    .toFile(target);
  return fs.readFileSync(target);
}

async function renderOpaque(svg, relativePath, width, height = width, background = "#07191D") {
  const target = path.join(finalDir, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  await sharp(Buffer.from(svg), { density: 384 })
    .resize(width, height, { fit: "fill" })
    .flatten({ background })
    .removeAlpha()
    .png({ compressionLevel: 9 })
    .toFile(target);
  return fs.readFileSync(target);
}

function createIco(images) {
  const header = Buffer.alloc(6 + images.length * 16);
  header.writeUInt16LE(0, 0);
  header.writeUInt16LE(1, 2);
  header.writeUInt16LE(images.length, 4);
  let offset = header.length;
  images.forEach(({ size, data }, index) => {
    const entry = 6 + index * 16;
    header.writeUInt8(size >= 256 ? 0 : size, entry);
    header.writeUInt8(size >= 256 ? 0 : size, entry + 1);
    header.writeUInt8(0, entry + 2);
    header.writeUInt8(0, entry + 3);
    header.writeUInt16LE(1, entry + 4);
    header.writeUInt16LE(32, entry + 6);
    header.writeUInt32LE(data.length, entry + 8);
    header.writeUInt32LE(offset, entry + 12);
    offset += data.length;
  });
  return Buffer.concat([header, ...images.map(({ data }) => data)]);
}

async function main() {
  fs.mkdirSync(finalDir, { recursive: true });
  writeCanonicalSources();
  await exportRasterJobs();
  exportFavicon();
  await exportOverview();
  await exportMacos();
  await exportIos();
  await exportAndroid();
  await exportWindows();
  await exportPrint();

  console.log(
    `AreaFlow brand export complete: ${expandRasterJobs(manifest).length} raster contracts (refresh=${refresh})`,
  );
}

function writeCanonicalSources() {
  const svgAssets = {
    "areaflow-app-icon-dark.svg": appIconSvg("dark"),
    "areaflow-app-icon-light.svg": appIconSvg("light"),
    "areaflow-app-icon-small-dark.svg": appIconSvg("dark", "small"),
    "areaflow-app-icon-small-light.svg": appIconSvg("light", "small"),
    "areaflow-app-icon-opaque-dark.svg": appIconSvg("dark", "opaque"),
    "areaflow-app-icon-opaque-light.svg": appIconSvg("light", "opaque"),
    "areaflow-app-icon-maskable-dark.svg": appIconSvg("dark", "maskable"),
    "areaflow-app-icon-maskable-light.svg": appIconSvg("light", "maskable"),
    "areaflow-logo-mark-dark.svg": markSvg("dark"),
    "areaflow-logo-mark-light.svg": markSvg("light"),
    "areaflow-logo-mark-mono-dark.svg": monoMarkSvg("dark"),
    "areaflow-logo-mark-mono-light.svg": monoMarkSvg("light"),
    "areaflow-logo-symbol-dark.svg": symbolSvg("dark"),
    "areaflow-logo-symbol-light.svg": symbolSvg("light"),
    "areaflow-logo-lockup.svg": lockupSvg("default"),
    "areaflow-logo-lockup-dark.svg": lockupSvg("dark"),
    "areaflow-logo-lockup-light.svg": lockupSvg("light"),
    "areaflow-logo-lockup-outlined.svg": lockupSvg("default", true),
    "areaflow-logo-lockup-outlined-dark.svg": lockupSvg("dark", true),
    "areaflow-logo-lockup-outlined-light.svg": lockupSvg("light", true),
    "areaflow-logo-lockup-mono-dark.svg": monoLockupSvg("dark"),
    "areaflow-logo-lockup-mono-light.svg": monoLockupSvg("light"),
    "areaflow-wordmark-dark.svg": wordmarkSvg("dark"),
    "areaflow-wordmark-light.svg": wordmarkSvg("light"),
    "areaflow-logo-stacked-dark.svg": stackedSvg("dark"),
    "areaflow-logo-stacked-light.svg": stackedSvg("light"),
    "social/areaflow-social-preview.svg": socialSvg("light"),
    "social/areaflow-social-preview-light.svg": socialSvg("light"),
    "social/areaflow-social-preview-dark.svg": socialSvg("dark"),
  };

  Object.entries(svgAssets).forEach(([file, svg]) => write(file, svg));
  write("README.md", readme);
  write("native/README.md", nativeReadme);
  write("print/README.md", printReadme);
}

async function exportRasterJobs() {
  for (const job of expandRasterJobs(manifest)) {
    if (!refresh && fs.existsSync(path.join(finalDir, job.output))) continue;
    const source = fs.readFileSync(path.join(finalDir, job.source));
    if (job.alpha) {
      await render(source, job.output, job.width, job.height);
    } else {
      await renderOpaque(source, job.output, job.width, job.height, manifest.native.androidBackground);
    }
  }
}

function exportFavicon() {
  const faviconImages = manifest.favicon.sizes.map((size) => ({
    size,
    data: fs.readFileSync(path.join(finalDir, manifest.favicon.output.replace("{size}", String(size)))),
  }));
  if (refresh || !fs.existsSync(path.join(finalDir, manifest.favicon.ico))) {
    write(manifest.favicon.ico, createIco(faviconImages));
  }
}

async function exportOverview() {
  if (refresh || !fs.existsSync(path.join(finalDir, manifest.overview.output))) {
    await render(overviewSvg(), manifest.overview.output, manifest.overview.width, manifest.overview.height);
  }
}

async function exportMacos() {
  const outputDirectory = path.join(finalDir, "native", "macos");
  const iconset = path.join(outputDirectory, "AreaFlow.iconset");
  const icns = path.join(outputDirectory, "AreaFlow.icns");
  if (!refresh && fs.existsSync(icns)) return;
  fs.rmSync(iconset, { recursive: true, force: true });
  fs.mkdirSync(iconset, { recursive: true });
  const source = fs.readFileSync(path.join(finalDir, manifest.native.macosSource));
  for (const size of [16, 32, 128, 256, 512]) {
    await renderOpaque(source, path.relative(finalDir, path.join(iconset, `icon_${size}x${size}.png`)), size);
    await renderOpaque(source, path.relative(finalDir, path.join(iconset, `icon_${size}x${size}@2x.png`)), size * 2);
  }
  if (process.platform === "darwin") {
    execFileSync("iconutil", ["-c", "icns", iconset, "-o", icns]);
  }
}

async function exportIos() {
  const outputDirectory = path.join(finalDir, "native", "ios", "AreaFlowAppIcon.appiconset");
  const contentsPath = path.join(outputDirectory, "Contents.json");
  if (!refresh && fs.existsSync(contentsPath)) return;
  fs.mkdirSync(outputDirectory, { recursive: true });
  const source = fs.readFileSync(path.join(finalDir, manifest.native.iosSource));
  const specs = [
    ["iphone", "20x20", "2x", 40], ["iphone", "20x20", "3x", 60],
    ["iphone", "29x29", "2x", 58], ["iphone", "29x29", "3x", 87],
    ["iphone", "40x40", "2x", 80], ["iphone", "40x40", "3x", 120],
    ["iphone", "60x60", "2x", 120], ["iphone", "60x60", "3x", 180],
    ["ipad", "20x20", "1x", 20], ["ipad", "20x20", "2x", 40],
    ["ipad", "29x29", "1x", 29], ["ipad", "29x29", "2x", 58],
    ["ipad", "40x40", "1x", 40], ["ipad", "40x40", "2x", 80],
    ["ipad", "76x76", "1x", 76], ["ipad", "76x76", "2x", 152],
    ["ipad", "83.5x83.5", "2x", 167], ["ios-marketing", "1024x1024", "1x", 1024],
  ];
  const images = [];
  for (const [idiom, size, scale, pixels] of specs) {
    const filename = `areaflow-${idiom}-${size.replaceAll(".", "_")}-${scale}.png`;
    await renderOpaque(source, path.relative(finalDir, path.join(outputDirectory, filename)), pixels);
    images.push({ idiom, size, scale, filename });
  }
  const contents = `${JSON.stringify({ images, info: { author: "xcode", version: 1 } }, null, 2)}\n`;
  write(path.relative(finalDir, path.join(outputDirectory, "Contents.json")), contents);
}

async function exportAndroid() {
  const base = path.join("native", "android", "res");
  const foregroundPath = path.join(base, "drawable-nodpi", "areaflow_adaptive_foreground.png");
  const backgroundPath = path.join(base, "drawable-nodpi", "areaflow_adaptive_background.png");
  if (!refresh && fs.existsSync(path.join(finalDir, foregroundPath))) return;
  const foreground = await sharp(path.join(finalDir, manifest.native.androidForegroundSource), { density: 384 })
    .resize(286, 286, { fit: "contain" })
    .png()
    .toBuffer();
  const foregroundTarget = path.join(finalDir, foregroundPath);
  fs.mkdirSync(path.dirname(foregroundTarget), { recursive: true });
  await sharp({
    create: {
      width: 432,
      height: 432,
      channels: 4,
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    },
  })
    .composite([{ input: foreground, left: 73, top: 73 }])
    .png({ compressionLevel: 9 })
    .toFile(foregroundTarget);
  const backgroundTarget = path.join(finalDir, backgroundPath);
  await sharp({
    create: { width: 432, height: 432, channels: 3, background: manifest.native.androidBackground },
  })
    .png({ compressionLevel: 9 })
    .toFile(backgroundTarget);
  const colors = `<?xml version="1.0" encoding="utf-8"?>
<resources>
  <color name="areaflow_icon_background">${manifest.native.androidBackground}</color>
</resources>
`;
  write(path.join(base, "values", "colors.xml"), colors);
  const adaptiveIcon = `<?xml version="1.0" encoding="utf-8"?>
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
  <background android:drawable="@color/areaflow_icon_background" />
  <foreground android:drawable="@drawable/areaflow_adaptive_foreground" />
</adaptive-icon>
`;
  write(path.join(base, "mipmap-anydpi-v26", "ic_launcher.xml"), adaptiveIcon);
  write(path.join(base, "mipmap-anydpi-v26", "ic_launcher_round.xml"), adaptiveIcon);
}

async function exportWindows() {
  const output = "native/windows/AreaFlow.ico";
  if (!refresh && fs.existsSync(path.join(finalDir, output))) return;
  const images = [];
  const source = path.join(finalDir, manifest.native.windowsSource);
  for (const size of [16, 24, 32, 48, 64, 128, 256]) {
    const data = await sharp(source, { density: 384 })
      .resize(size, size, { fit: "fill" })
      .flatten({ background: manifest.native.androidBackground })
      .removeAlpha()
      .png({ compressionLevel: 9 })
      .toBuffer();
    images.push({ size, data });
  }
  write(output, createIco(images));
}

async function exportPrint() {
  const light = fs.readFileSync(path.join(finalDir, manifest.native.printLightSource), "utf8");
  const dark = fs.readFileSync(path.join(finalDir, manifest.native.printDarkSource), "utf8");
  const printDir = path.join(finalDir, "print");
  fs.mkdirSync(printDir, { recursive: true });
  write("print/areaflow-logo-light-background.svg", light);
  write("print/areaflow-logo-dark-background.svg", dark);
  const lightTiff = path.join(printDir, "areaflow-logo-light-background-cmyk.tiff");
  const darkTiff = path.join(printDir, "areaflow-logo-dark-background-cmyk.tiff");
  if (refresh || !fs.existsSync(lightTiff)) {
    await sharp(Buffer.from(light), { density: manifest.native.printDpi })
      .resize(3600, 1170, { fit: "fill" })
      .flatten({ background: "#FFFFFF" })
      .toColourspace("cmyk")
      .tiff({ compression: "lzw", resolutionUnit: "inch", xres: printDpiPerMillimetre, yres: printDpiPerMillimetre })
      .toFile(lightTiff);
  }
  if (refresh || !fs.existsSync(darkTiff)) {
    await sharp(Buffer.from(dark), { density: manifest.native.printDpi })
      .resize(3600, 1170, { fit: "fill" })
      .flatten({ background: manifest.native.androidBackground })
      .toColourspace("cmyk")
      .tiff({ compression: "lzw", resolutionUnit: "inch", xres: printDpiPerMillimetre, yres: printDpiPerMillimetre })
      .toFile(darkTiff);
  }
  const lightPdf = path.join(printDir, "areaflow-logo-light-background.pdf");
  const darkPdf = path.join(printDir, "areaflow-logo-dark-background.pdf");
  if (process.platform === "darwin" && (refresh || !fs.existsSync(lightPdf) || !fs.existsSync(darkPdf))) {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "areaflow-brand-print-"));
    const lightPdfSource = path.join(tempDir, "light.svg");
    const darkPdfSource = path.join(tempDir, "dark.svg");
    fs.writeFileSync(lightPdfSource, addSvgBackground(light, "#FFFFFF"));
    fs.writeFileSync(darkPdfSource, addSvgBackground(dark, manifest.native.androidBackground));
    execFileSync("sips", ["-s", "format", "pdf", lightPdfSource, "--out", lightPdf]);
    execFileSync("sips", ["-s", "format", "pdf", darkPdfSource, "--out", darkPdf]);
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
}

function addSvgBackground(svg, color) {
  return svg.replace(/(<desc id="desc">[^<]*<\/desc>)/, `$1\n  <rect width="1600" height="520" fill="${color}"/>`);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
