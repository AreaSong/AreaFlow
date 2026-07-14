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
  overviewSvg,
  readme,
  socialSvg,
  stackedSvg,
  symbolSvg,
  wordmarkSvg,
} = require("./lib/areaflow-brand-svg.cjs");

let sharp;
try {
  sharp = require("sharp");
} catch (error) {
  console.error("sharp is required to export the PNG brand assets.");
  console.error("Install sharp or expose it through NODE_PATH before running this script.");
  process.exit(1);
}

const root = path.resolve(__dirname, "..");
const finalDir = path.join(root, "assets", "brand", "final");
const dirs = ["app-icon", "favicon", "lockup", "mark", "social", "stacked", "symbol", "wordmark"];

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
  dirs.forEach((dir) => fs.mkdirSync(path.join(finalDir, dir), { recursive: true }));

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

  const normalSizes = [16, 32, 48, 64, 128, 180, 192, 256, 512, 1024];
  for (const theme of ["dark", "light"]) {
    for (const size of normalSizes) {
      const source = size <= 48 ? appIconSvg(theme, "small") : appIconSvg(theme);
      await render(source, `app-icon/areaflow-app-icon-${theme}-${size}.png`, size);
    }
    for (const size of [180, 1024]) {
      await render(appIconSvg(theme, "opaque"), `app-icon/areaflow-app-icon-opaque-${theme}-${size}.png`, size);
    }
    for (const size of [192, 512]) {
      await render(appIconSvg(theme, "maskable"), `app-icon/areaflow-app-icon-maskable-${theme}-${size}.png`, size);
    }
  }

  for (const theme of ["dark", "light"]) {
    for (const size of [256, 512, 1024]) {
      await render(markSvg(theme), `mark/areaflow-logo-mark-${theme}-${size}.png`, size);
      await render(monoMarkSvg(theme), `mark/areaflow-logo-mark-mono-${theme}-${size}.png`, size);
      await render(symbolSvg(theme), `symbol/areaflow-logo-symbol-${theme}-${size}.png`, size);
    }
  }

  const lockups = {
    "areaflow-logo-lockup-1600x520.png": lockupSvg("default"),
    "areaflow-logo-lockup-dark-1600x520.png": lockupSvg("dark"),
    "areaflow-logo-lockup-light-1600x520.png": lockupSvg("light"),
    "areaflow-logo-lockup-outlined-1600x520.png": lockupSvg("default", true),
    "areaflow-logo-lockup-outlined-dark-1600x520.png": lockupSvg("dark", true),
    "areaflow-logo-lockup-outlined-light-1600x520.png": lockupSvg("light", true),
    "areaflow-logo-lockup-mono-dark-1600x520.png": monoLockupSvg("dark"),
    "areaflow-logo-lockup-mono-light-1600x520.png": monoLockupSvg("light"),
  };
  for (const [file, svg] of Object.entries(lockups)) {
    await render(svg, `lockup/${file}`, 1600, 520);
  }

  await render(wordmarkSvg("dark"), "wordmark/areaflow-wordmark-dark-1200x336.png", 1200, 336);
  await render(wordmarkSvg("light"), "wordmark/areaflow-wordmark-light-1200x336.png", 1200, 336);
  await render(stackedSvg("dark"), "stacked/areaflow-logo-stacked-dark-1024.png", 1024);
  await render(stackedSvg("light"), "stacked/areaflow-logo-stacked-light-1024.png", 1024);

  const faviconImages = [];
  for (const size of [16, 32, 48]) {
    const data = await render(appIconSvg("light", "small"), `favicon/areaflow-favicon-${size}.png`, size);
    faviconImages.push({ size, data });
  }
  write("favicon/areaflow-favicon.ico", createIco(faviconImages));

  await render(socialSvg("light"), "social/areaflow-social-preview.png", 1200, 630);
  await render(socialSvg("light"), "social/areaflow-social-preview-light.png", 1200, 630);
  await render(socialSvg("dark"), "social/areaflow-social-preview-dark.png", 1200, 630);
  await render(overviewSvg(), "areaflow-brand-overview.png", 1600, 1200);

  await exportMacos();
  await exportIos();
  await exportAndroid();
  await exportWindows();
  await exportPrint();

  console.log(`Generated AreaFlow brand assets in ${finalDir}`);
}

async function exportMacos() {
  const outputDirectory = path.join(finalDir, "native", "macos");
  const iconset = path.join(outputDirectory, "AreaFlow.iconset");
  const icns = path.join(outputDirectory, "AreaFlow.icns");
  fs.rmSync(iconset, { recursive: true, force: true });
  fs.mkdirSync(iconset, { recursive: true });
  for (const size of [16, 32, 128, 256, 512]) {
    await renderOpaque(appIconSvg("dark", "opaque"), path.relative(finalDir, path.join(iconset, `icon_${size}x${size}.png`)), size);
    await renderOpaque(appIconSvg("dark", "opaque"), path.relative(finalDir, path.join(iconset, `icon_${size}x${size}@2x.png`)), size * 2);
  }
  if (process.platform === "darwin") {
    execFileSync("iconutil", ["-c", "icns", iconset, "-o", icns]);
  }
}

async function exportIos() {
  const outputDirectory = path.join(finalDir, "native", "ios", "AreaFlowAppIcon.appiconset");
  fs.mkdirSync(outputDirectory, { recursive: true });
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
    await renderOpaque(appIconSvg("dark", "opaque"), path.relative(finalDir, path.join(outputDirectory, filename)), pixels);
    images.push({ idiom, size, scale, filename });
  }
  write(path.relative(finalDir, path.join(outputDirectory, "Contents.json")), `${JSON.stringify({ images, info: { author: "xcode", version: 1 } }, null, 2)}\n`);
}

async function exportAndroid() {
  const base = path.join("native", "android", "res");
  const foregroundPath = path.join(base, "drawable-nodpi", "areaflow_adaptive_foreground.png");
  const backgroundPath = path.join(base, "drawable-nodpi", "areaflow_adaptive_background.png");
  const foreground = await sharp(Buffer.from(symbolSvg("light")), { density: 384 })
    .resize(286, 286, { fit: "contain" })
    .png()
    .toBuffer();
  const foregroundTarget = path.join(finalDir, foregroundPath);
  fs.mkdirSync(path.dirname(foregroundTarget), { recursive: true });
  await sharp({ create: { width: 432, height: 432, channels: 4, background: { r: 0, g: 0, b: 0, alpha: 0 } } })
    .composite([{ input: foreground, left: 73, top: 73 }])
    .png({ compressionLevel: 9 })
    .toFile(foregroundTarget);
  const backgroundTarget = path.join(finalDir, backgroundPath);
  await sharp({ create: { width: 432, height: 432, channels: 3, background: "#07191D" } })
    .png({ compressionLevel: 9 })
    .toFile(backgroundTarget);
  write(path.join(base, "values", "colors.xml"), `<?xml version="1.0" encoding="utf-8"?>\n<resources>\n  <color name="areaflow_icon_background">#07191D</color>\n</resources>\n`);
  const adaptiveIcon = `<?xml version="1.0" encoding="utf-8"?>\n<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">\n  <background android:drawable="@color/areaflow_icon_background" />\n  <foreground android:drawable="@drawable/areaflow_adaptive_foreground" />\n</adaptive-icon>\n`;
  write(path.join(base, "mipmap-anydpi-v26", "ic_launcher.xml"), adaptiveIcon);
  write(path.join(base, "mipmap-anydpi-v26", "ic_launcher_round.xml"), adaptiveIcon);
}

async function exportWindows() {
  const images = [];
  for (const size of [16, 24, 32, 48, 64, 128, 256]) {
    const data = await sharp(Buffer.from(appIconSvg("dark", "opaque")), { density: 384 })
      .resize(size, size, { fit: "fill" })
      .flatten({ background: "#07191D" })
      .removeAlpha()
      .png({ compressionLevel: 9 })
      .toBuffer();
    images.push({ size, data });
  }
  write("native/windows/AreaFlow.ico", createIco(images));
}

async function exportPrint() {
  const light = lockupSvg("light", true);
  const dark = lockupSvg("dark", true);
  const printDir = path.join(finalDir, "print");
  fs.mkdirSync(printDir, { recursive: true });
  write("print/areaflow-logo-light-background.svg", light);
  write("print/areaflow-logo-dark-background.svg", dark);
  await sharp(Buffer.from(light), { density: 300 })
    .resize(3600, 1170, { fit: "fill" })
    .flatten({ background: "#FFFFFF" })
    .toColourspace("cmyk")
    .tiff({ compression: "lzw", resolutionUnit: "inch", xres: 300, yres: 300 })
    .toFile(path.join(printDir, "areaflow-logo-light-background-cmyk.tiff"));
  await sharp(Buffer.from(dark), { density: 300 })
    .resize(3600, 1170, { fit: "fill" })
    .flatten({ background: "#07191D" })
    .toColourspace("cmyk")
    .tiff({ compression: "lzw", resolutionUnit: "inch", xres: 300, yres: 300 })
    .toFile(path.join(printDir, "areaflow-logo-dark-background-cmyk.tiff"));
  if (process.platform === "darwin") {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "areaflow-brand-print-"));
    const lightPdfSource = path.join(tempDir, "light.svg");
    const darkPdfSource = path.join(tempDir, "dark.svg");
    fs.writeFileSync(lightPdfSource, addSvgBackground(light, "#FFFFFF"));
    fs.writeFileSync(darkPdfSource, addSvgBackground(dark, "#07191D"));
    execFileSync("sips", ["-s", "format", "pdf", lightPdfSource, "--out", path.join(printDir, "areaflow-logo-light-background.pdf")]);
    execFileSync("sips", ["-s", "format", "pdf", darkPdfSource, "--out", path.join(printDir, "areaflow-logo-dark-background.pdf")]);
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
