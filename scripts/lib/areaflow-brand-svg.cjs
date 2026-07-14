const AREA_PATH = "M48.34 31.83L80.45 112L62.84 112L55.84 93.79L23.79 93.79L17.17 112L0 112L31.23 31.83L48.34 31.83ZM28.77 80.28L50.64 80.28L39.59 50.53L28.77 80.28ZM103.63 94.06L103.63 112L88.27 112L88.27 53.92L102.54 53.92L102.54 62.18Q106.20 56.33 109.13 54.47Q112.05 52.61 115.77 52.61Q121.02 52.61 125.89 55.51L121.13 68.91Q117.25 66.39 113.91 66.39Q110.69 66.39 108.45 68.17Q106.20 69.95 104.92 74.59Q103.63 79.24 103.63 94.06ZM166.14 93.52L181.45 96.09Q178.50 104.51 172.13 108.91Q165.76 113.31 156.19 113.31Q141.04 113.31 133.77 103.41Q128.02 95.48 128.02 83.40Q128.02 68.96 135.57 60.79Q143.12 52.61 154.66 52.61Q167.62 52.61 175.11 61.17Q182.60 69.73 182.27 87.39L143.77 87.39Q143.94 94.23 147.49 98.03Q151.05 101.83 156.35 101.83Q159.96 101.83 162.42 99.86Q164.88 97.89 166.14 93.52ZM144.05 77.98L167.02 77.98Q166.85 71.31 163.57 67.84Q160.29 64.37 155.59 64.37Q150.55 64.37 147.27 68.03Q143.99 71.70 144.05 77.98ZM206.28 71.64L192.34 69.13Q194.69 60.70 200.43 56.66Q206.17 52.61 217.49 52.61Q227.77 52.61 232.80 55.04Q237.84 57.48 239.89 61.22Q241.94 64.97 241.94 74.98L241.77 92.91Q241.77 100.57 242.51 104.21Q243.25 107.84 245.27 112L230.07 112Q229.47 110.47 228.59 107.46Q228.21 106.09 228.05 105.66Q224.11 109.48 219.63 111.40Q215.14 113.31 210.05 113.31Q201.09 113.31 195.92 108.45Q190.75 103.58 190.75 96.14Q190.75 91.22 193.10 87.36Q195.45 83.51 199.69 81.46Q203.93 79.41 211.91 77.88Q222.69 75.85 226.84 74.10L226.84 72.57Q226.84 68.14 224.66 66.25Q222.47 64.37 216.40 64.37Q212.30 64.37 210 65.98Q207.70 67.59 206.28 71.64ZM226.84 87.17L226.84 84.11Q223.89 85.09 217.49 86.46Q211.09 87.83 209.13 89.14Q206.12 91.27 206.12 94.55Q206.12 97.78 208.52 100.13Q210.93 102.48 214.65 102.48Q218.80 102.48 222.58 99.75Q225.37 97.67 226.24 94.66Q226.84 92.70 226.84 87.17Z";

const FLOW_PATH = "M24.45 112L8.26 112L8.26 31.83L63.22 31.83L63.22 45.39L24.45 45.39L24.45 64.37L57.91 64.37L57.91 77.93L24.45 77.93L24.45 112ZM91.82 112L76.45 112L76.45 31.83L91.82 31.83L91.82 112ZM104.02 82.14Q104.02 74.48 107.79 67.32Q111.56 60.16 118.48 56.38Q125.40 52.61 133.93 52.61Q147.11 52.61 155.53 61.17Q163.95 69.73 163.95 82.80Q163.95 95.98 155.45 104.64Q146.95 113.31 134.04 113.31Q126.05 113.31 118.81 109.70Q111.56 106.09 107.79 99.12Q104.02 92.15 104.02 82.14ZM119.77 82.96Q119.77 91.60 123.87 96.20Q127.97 100.79 133.98 100.79Q140 100.79 144.07 96.20Q148.15 91.60 148.15 82.85Q148.15 74.32 144.07 69.73Q140 65.13 133.98 65.13Q127.97 65.13 123.87 69.73Q119.77 74.32 119.77 82.96ZM201.74 112L186.81 112L168.44 53.92L183.37 53.92L194.25 91.98L204.26 53.92L219.08 53.92L228.76 91.98L239.86 53.92L255.01 53.92L236.36 112L221.59 112L211.59 74.65L201.74 112Z";

const COLORS = {
  darkBg: "#07191D",
  darkSurface: "#0D2D31",
  lightBg: "#F1FAF7",
  lightSurface: "#FFFFFF",
  ink: "#09272D",
  inkMuted: "#587477",
  white: "#F4FBF8",
  mint: "#36D9A6",
  cyan: "#18BFC7",
  amber: "#F5B02E",
  coral: "#F46D5E",
};

function doc(viewBox, title, desc, body, extra = "") {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${viewBox}" role="img" aria-labelledby="title desc" ${extra}>
  <title id="title">${title}</title>
  <desc id="desc">${desc}</desc>
${body}
</svg>\n`;
}

function palette(theme) {
  if (theme === "dark") {
    return {
      tile: COLORS.darkBg,
      tileStroke: "none",
      outer: "#E7FFF8",
      middle: "#BDF3EA",
      ticks: "#8EA7AA",
      base: "#94ADB0",
      taskLine: "#FFFFFF",
    };
  }
  return {
    tile: COLORS.lightBg,
    tileStroke: "#CDE5DF",
    outer: "#E1F8F1",
    middle: "#B8EBE2",
    ticks: "#789396",
    base: "#6E8A8D",
    taskLine: "#FFFFFF",
  };
}

function gradientDefs(prefix, theme) {
  const light = theme === "light";
  const mint = light ? "#20C99A" : COLORS.mint;
  const cyan = light ? "#0EACB4" : COLORS.cyan;
  const amber = light ? "#E9A524" : COLORS.amber;
  const coral = light ? "#E95E52" : COLORS.coral;
  return `<defs>
    <linearGradient id="${prefix}-flow" x1="90" y1="360" x2="410" y2="120" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="${mint}"/><stop offset="0.48" stop-color="${cyan}"/><stop offset="1" stop-color="${amber}"/>
    </linearGradient>
    <linearGradient id="${prefix}-orbit" x1="120" y1="330" x2="390" y2="130" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="${mint}"/><stop offset="0.5" stop-color="${amber}"/><stop offset="1" stop-color="${coral}"/>
    </linearGradient>
  </defs>`;
}

function fullSymbol(prefix, theme) {
  const p = palette(theme);
  return `${gradientDefs(prefix, theme)}
    <circle cx="256" cy="246" r="126" fill="${p.outer}"/>
    <circle cx="256" cy="246" r="91" fill="${p.middle}"/>
    <circle cx="256" cy="246" r="46" fill="${COLORS.cyan}"/>
    <path d="M256 82V146M256 346V410M92 246H156M356 246H420M140 130L185 175M372 130L327 175M140 362L185 317M372 362L327 317" stroke="${p.ticks}" stroke-width="21" stroke-linecap="round"/>
    <path d="M116 395H396" stroke="${p.base}" stroke-width="15" stroke-linecap="round" opacity="0.38"/>
    <path d="M112 352C171 381 263 387 333 337C395 292 415 219 382 159" stroke="url(#${prefix}-orbit)" stroke-width="31" stroke-linecap="round" fill="none"/>
    <path d="M120 352C181 345 232 315 275 266C315 220 351 183 395 157" stroke="url(#${prefix}-flow)" stroke-width="26" stroke-linecap="round" fill="none"/>
    <circle cx="120" cy="352" r="34" fill="${theme === "light" ? "#20C99A" : COLORS.mint}"/>
    <circle cx="256" cy="246" r="25" fill="${COLORS.cyan}" stroke="${p.outer}" stroke-width="10"/>
    <circle cx="334" cy="337" r="25" fill="${theme === "light" ? "#E9A524" : COLORS.amber}"/>
    <circle cx="395" cy="157" r="30" fill="${theme === "light" ? "#E95E52" : COLORS.coral}"/>
    <rect x="361" y="122" width="70" height="70" rx="22" fill="${theme === "light" ? "#E95E52" : COLORS.coral}"/>
    <path d="M376 157H416" stroke="${p.taskLine}" stroke-width="14" stroke-linecap="round"/>`;
}

function smallSymbol(prefix, theme) {
  const p = palette(theme);
  return `${gradientDefs(prefix, theme)}
    <circle cx="256" cy="246" r="118" fill="${p.outer}"/>
    <circle cx="256" cy="246" r="76" fill="${p.middle}"/>
    <circle cx="256" cy="246" r="37" fill="${COLORS.cyan}"/>
    <path d="M256 94V143M256 350V399M104 246H153M359 246H408" stroke="${p.ticks}" stroke-width="24" stroke-linecap="round"/>
    <path d="M126 390H386" stroke="${p.base}" stroke-width="17" stroke-linecap="round" opacity="0.42"/>
    <path d="M116 352C186 385 270 383 335 334C386 295 410 224 382 159" stroke="url(#${prefix}-orbit)" stroke-width="36" stroke-linecap="round" fill="none"/>
    <path d="M120 352C191 341 238 305 279 258C318 214 353 181 395 157" stroke="url(#${prefix}-flow)" stroke-width="31" stroke-linecap="round" fill="none"/>
    <circle cx="120" cy="352" r="38" fill="${theme === "light" ? "#20C99A" : COLORS.mint}"/>
    <rect x="356" y="118" width="78" height="78" rx="25" fill="${theme === "light" ? "#E95E52" : COLORS.coral}"/>
    <path d="M374 157H416" stroke="#FFFFFF" stroke-width="17" stroke-linecap="round"/>`;
}

function iconInner(theme, mode = "regular", prefix = `icon-${theme}-${mode}`) {
  const p = palette(theme);
  const isSmall = mode === "small";
  const symbol = isSmall ? smallSymbol(prefix, theme) : fullSymbol(prefix, theme);
  if (mode === "maskable") {
    return `<rect width="512" height="512" fill="${p.tile}"/>\n  <g transform="translate(61.44 61.44) scale(0.76)">${symbol}</g>`;
  }
  if (mode === "opaque") {
    return `<rect width="512" height="512" fill="${p.tile}"/>\n  <g transform="translate(25.6 25.6) scale(0.9)">${symbol}</g>`;
  }
  return `<rect width="512" height="512" rx="112" fill="${p.tile}"${p.tileStroke === "none" ? "" : ` stroke="${p.tileStroke}" stroke-width="6"`}/>\n  ${symbol}`;
}

function appIconSvg(theme, mode = "regular") {
  const suffix = mode === "regular" ? "" : ` ${mode[0].toUpperCase()}${mode.slice(1)}`;
  return doc(
    "0 0 512 512",
    `AreaFlow App Icon ${theme === "dark" ? "Dark" : "Light"}${suffix}`,
    "AreaFlow scheduler dial with governed workflow paths, execution nodes, and a subtle shared baseline.",
    `  ${iconInner(theme, mode)}`,
  );
}

function symbolSvg(theme) {
  return doc(
    "0 0 512 512",
    `AreaFlow Logo Symbol ${theme === "dark" ? "Dark" : "Light"}`,
    "The transparent AreaFlow workflow scheduler symbol adapted for the target background.",
    `  ${fullSymbol(`symbol-${theme}`, theme)}`,
  );
}

function markSvg(theme) {
  return doc(
    "0 0 512 512",
    `AreaFlow Logo Mark ${theme === "dark" ? "Dark" : "Light"}`,
    "The AreaFlow logo mark with the canonical scheduler dial and governed flow paths.",
    `  ${iconInner(theme, "regular", `mark-${theme}`)}`,
  );
}

function monoMarkBody(color) {
  return `<g fill="none" stroke="${color}" stroke-linecap="round" stroke-linejoin="round">
    <rect x="48" y="48" width="416" height="416" rx="96" stroke-width="24"/>
    <circle cx="256" cy="246" r="112" stroke-width="22" opacity="0.45"/>
    <path d="M256 92V142M256 350V400M104 246H154M358 246H408" stroke-width="22" opacity="0.62"/>
    <path d="M124 391H388" stroke-width="18" opacity="0.58"/>
    <path d="M116 352C186 385 270 383 335 334C386 295 410 224 382 159" stroke-width="31"/>
    <path d="M120 352C191 341 238 305 279 258C318 214 353 181 395 157" stroke-width="25"/>
    <circle cx="256" cy="246" r="29" stroke-width="16"/>
    <rect x="360" y="121" width="73" height="73" rx="23" stroke-width="16"/>
    <path d="M378 157H415" stroke-width="14"/>
  </g>
  <g fill="${color}"><circle cx="120" cy="352" r="27"/><circle cx="334" cy="337" r="20"/></g>`;
}

function monoMarkSvg(tone) {
  const color = tone === "dark" ? COLORS.ink : COLORS.white;
  return doc(
    "0 0 512 512",
    `AreaFlow Logo Mark Mono ${tone === "dark" ? "Dark" : "Light"}`,
    `A single-color ${tone} AreaFlow mark for constrained brand applications.`,
    `  ${monoMarkBody(color)}`,
  );
}

function wordGradient(id) {
  return `<linearGradient id="${id}" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#18BFC7"/><stop offset="0.52" stop-color="#36D9A6"/><stop offset="0.82" stop-color="#F5B02E"/><stop offset="1" stop-color="#F46D5E"/></linearGradient>`;
}

function outlinedWordPaths(areaFill, gradientId) {
  return `<path d="${AREA_PATH}" fill="${areaFill}"/><path d="${FLOW_PATH}" transform="translate(249.05 0)" fill="url(#${gradientId})"/>`;
}

function wordmarkUnderline(color, endColor = COLORS.ink, y = 166, width = 548) {
  return `<path d="M4 ${y}H${width}" stroke="${color}" stroke-width="11" stroke-linecap="round" opacity="0.58"/>
    <path d="M4 ${y}C126 ${y} 204 ${y - 15} 292 ${y - 20}C390 ${y - 26} 474 ${y - 10} ${width} ${y}" fill="none" stroke="${COLORS.cyan}" stroke-width="11" stroke-linecap="round"/>
    <circle cx="222" cy="${y - 3}" r="10" fill="${COLORS.amber}"/><circle cx="${width}" cy="${y}" r="10" fill="${endColor}"/>`;
}

function lockupSvg(variant = "default", outlined = false) {
  const target = variant === "dark" ? "dark" : "light";
  const areaFill = target === "dark" ? COLORS.white : COLORS.ink;
  const iconTheme = target;
  const gradientId = `lockup-flow-${variant}-${outlined ? "outlined" : "text"}`;
  const titleSuffix = `${variant === "default" ? "Default" : variant[0].toUpperCase() + variant.slice(1)}${outlined ? " Outlined" : ""}`;
  const word = outlined
    ? `<g transform="translate(588 140)">${outlinedWordPaths(areaFill, gradientId)}${wordmarkUnderline(target === "dark" ? "#557579" : "#A9C8C4", target === "dark" ? COLORS.white : COLORS.ink)}</g>`
    : `<g transform="translate(588 0)">
    <text x="0" y="252" font-family="Arial, Helvetica Neue, sans-serif" font-size="112" font-weight="700" letter-spacing="0" fill="${areaFill}">Area</text>
    <text x="249" y="252" font-family="Arial, Helvetica Neue, sans-serif" font-size="112" font-weight="700" letter-spacing="0" fill="url(#${gradientId})">Flow</text>
    ${wordmarkUnderline(target === "dark" ? "#557579" : "#A9C8C4", target === "dark" ? COLORS.white : COLORS.ink, 306)}
  </g>`;
  return doc(
    "0 0 1600 520",
    `AreaFlow Logo Lockup ${titleSuffix}`,
    "AreaFlow horizontal logo with the canonical workflow scheduler icon and emphasized Flow wordmark.",
    `  <defs>${wordGradient(gradientId)}</defs>
  <svg x="90" y="50" width="420" height="420" viewBox="0 0 512 512">${iconInner(iconTheme, "regular", `lockup-icon-${variant}-${outlined}`)}</svg>
  ${word}`,
  );
}

function monoLockupSvg(tone) {
  const color = tone === "dark" ? COLORS.ink : COLORS.white;
  return doc(
    "0 0 1600 520",
    `AreaFlow Logo Lockup Mono ${tone === "dark" ? "Dark" : "Light"}`,
    `A single-color ${tone} horizontal AreaFlow logo lockup.`,
    `  <svg x="90" y="50" width="420" height="420" viewBox="0 0 512 512">${monoMarkBody(color)}</svg>
  <g transform="translate(588 140)"><path d="${AREA_PATH}" fill="${color}"/><path d="${FLOW_PATH}" transform="translate(249.05 0)" fill="${color}"/>${wordmarkUnderline(color, color)}</g>`,
  );
}

function wordmarkSvg(target) {
  const areaFill = target === "dark" ? COLORS.white : COLORS.ink;
  const endColor = target === "dark" ? COLORS.white : COLORS.ink;
  const gradientId = `wordmark-flow-${target}`;
  return doc(
    "0 0 1000 280",
    `AreaFlow Wordmark ${target === "dark" ? "Dark" : "Light"} Background`,
    `AreaFlow wordmark adapted for a ${target} background.`,
    `  <defs>${wordGradient(gradientId)}</defs>
  <g transform="translate(92 18.2) scale(1.4)">${outlinedWordPaths(areaFill, gradientId)}</g>
  <path d="M92 220H820" stroke="${target === "dark" ? "#557579" : "#A9C8C4"}" stroke-width="11" stroke-linecap="round" opacity="0.62"/>
  <path d="M92 220C250 220 345 202 450 198C590 193 700 205 820 220" fill="none" stroke="${COLORS.cyan}" stroke-width="11" stroke-linecap="round"/>
  <circle cx="404" cy="210" r="10" fill="${COLORS.amber}"/><circle cx="820" cy="220" r="10" fill="${endColor}"/>`,
  );
}

function stackedSvg(target) {
  const areaFill = target === "dark" ? COLORS.white : COLORS.ink;
  const endColor = target === "dark" ? COLORS.white : COLORS.ink;
  const gradientId = `stacked-flow-${target}`;
  return doc(
    "0 0 1024 1024",
    `AreaFlow Stacked Logo ${target === "dark" ? "Dark" : "Light"} Background`,
    "Stacked AreaFlow logo with the canonical app icon and outlined wordmark.",
    `  <defs>${wordGradient(gradientId)}</defs>
  <svg x="256" y="80" width="512" height="512" viewBox="0 0 512 512">${iconInner(target, "regular", `stacked-icon-${target}`)}</svg>
  <g transform="translate(247 636) scale(1.05)">${outlinedWordPaths(areaFill, gradientId)}</g>
  <path d="M230 825H794" stroke="${target === "dark" ? "#557579" : "#A9C8C4"}" stroke-width="11" stroke-linecap="round" opacity="0.62"/>
  <path d="M230 825C375 825 465 807 560 803C680 798 775 810 794 825" fill="none" stroke="${COLORS.cyan}" stroke-width="11" stroke-linecap="round"/>
  <circle cx="495" cy="821" r="10" fill="${COLORS.amber}"/><circle cx="794" cy="825" r="10" fill="${endColor}"/>`,
  );
}

function socialSvg(target) {
  const dark = target === "dark";
  const bg = dark ? COLORS.darkBg : COLORS.lightBg;
  const graphBg = dark ? "#0D3034" : "#E7F4F0";
  const text = dark ? COLORS.white : COLORS.ink;
  const muted = dark ? "#9FC4C3" : COLORS.inkMuted;
  const pillBg = dark ? COLORS.white : COLORS.darkBg;
  const pillText = dark ? COLORS.ink : COLORS.white;
  const gradientId = `social-flow-${target}`;
  return doc(
    "0 0 1200 630",
    `AreaFlow Social Preview ${dark ? "Dark" : "Light"}`,
    "AreaFlow social preview describing AI development orchestration and auditable delivery.",
    `  <defs>${wordGradient(gradientId)}<pattern id="grid-${target}" width="36" height="36" patternUnits="userSpaceOnUse"><path d="M36 0H0V36" fill="none" stroke="${dark ? "#21464A" : "#CFE5E0"}" stroke-width="1"/></pattern></defs>
  <rect width="1200" height="630" fill="${bg}"/>
  <rect x="650" width="550" height="630" fill="${graphBg}"/>
  <rect x="650" width="550" height="630" fill="url(#grid-${target})"/>
  <svg x="90" y="72" width="150" height="150" viewBox="0 0 512 512">${iconInner(target, "regular", `social-icon-${target}`)}</svg>
  <g transform="translate(280 78) scale(0.52)">${outlinedWordPaths(text, gradientId)}</g>
  <path d="M280 168H565" stroke="${dark ? "#557579" : "#A9C8C4"}" stroke-width="5" stroke-linecap="round"/>
  <path d="M280 168C350 168 415 154 478 157C520 159 548 164 565 168" fill="none" stroke="${COLORS.cyan}" stroke-width="5" stroke-linecap="round"/>
  <circle cx="390" cy="164" r="6" fill="${COLORS.amber}"/><circle cx="565" cy="168" r="6" fill="${text}"/>
  <text x="90" y="284" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif" font-size="24" font-weight="700" fill="${COLORS.cyan}">AI 开发执行治理平台</text>
  <text x="90" y="350" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif" font-size="40" font-weight="700" fill="${text}">把需求编排成可审计的软件交付</text>
  <text x="90" y="405" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif" font-size="22" fill="${muted}">项目 · Workflow · Run · Worker · Artifact · 审批 · 审计</text>
  <rect x="90" y="510" width="350" height="56" rx="28" fill="${pillBg}"/>
  <circle cx="120" cy="538" r="8" fill="${COLORS.mint}"/>
  <text x="143" y="546" font-family="Arial, sans-serif" font-size="18" font-weight="700" fill="${pillText}">AI Development Control Plane</text>
  <path d="M708 514C762 480 795 422 837 378C884 329 930 334 972 268C1003 219 1037 165 1090 118" fill="none" stroke="${dark ? "#244F53" : "#FFFFFF"}" stroke-width="26" stroke-linecap="round"/>
  <path d="M708 514C762 480 795 422 837 378C884 329 930 334 972 268C1003 219 1037 165 1090 118" fill="none" stroke="url(#${gradientId})" stroke-width="12" stroke-linecap="round"/>
  <circle cx="708" cy="514" r="18" fill="${COLORS.mint}" stroke="${dark ? "#12393D" : "#FFFFFF"}" stroke-width="7"/><circle cx="837" cy="378" r="16" fill="${text}" stroke="${graphBg}" stroke-width="7"/><circle cx="972" cy="268" r="18" fill="${COLORS.amber}" stroke="${dark ? "#12393D" : "#FFFFFF"}" stroke-width="7"/><circle cx="1090" cy="118" r="20" fill="${COLORS.coral}" stroke="${dark ? "#12393D" : "#FFFFFF"}" stroke-width="7"/>
  <g fill="none" stroke="${dark ? "#315D61" : "#8DBEB6"}" stroke-width="2"><rect x="734" y="112" width="86" height="86"/><rect x="780" y="156" width="86" height="86"/><rect x="1028" y="430" width="92" height="92"/><rect x="1070" y="474" width="92" height="92"/></g>`,
  );
}

function overviewSvg() {
  return doc(
    "0 0 1600 1200",
    "AreaFlow Brand System Overview",
    "Overview of AreaFlow app icons, symbols, lockups, mono marks, stacked logos, and social preview.",
    `  <rect width="1600" height="1200" fill="#EAF5F1"/>
  <rect width="1600" height="126" fill="${COLORS.darkBg}"/>
  <text x="62" y="88" font-family="Arial, sans-serif" font-size="58" font-weight="700" fill="${COLORS.white}">AreaFlow Brand System</text>
  <text x="1176" y="70" font-family="Arial, sans-serif" font-size="20" fill="#8EDBD1">DIGITAL ASSET KIT</text>
  <g font-family="Arial, sans-serif" font-weight="700" font-size="25" fill="${COLORS.ink}">
    <text x="76" y="203">APP ICONS &amp; SYMBOLS</text><text x="826" y="203" fill="${COLORS.white}">DARK BACKGROUND</text>
    <text x="76" y="492">HORIZONTAL LOCKUPS</text><text x="1076" y="492" fill="${COLORS.white}">MONO</text>
    <text x="76" y="790">STACKED</text><text x="826" y="790" fill="${COLORS.white}">SOCIAL PREVIEW</text>
  </g>
  <rect x="50" y="160" width="700" height="252" rx="10" fill="#FFFFFF"/>
  <rect x="800" y="160" width="750" height="252" rx="10" fill="${COLORS.darkBg}"/>
  <rect x="50" y="450" width="960" height="274" rx="10" fill="#FFFFFF"/>
  <rect x="1050" y="450" width="500" height="274" rx="10" fill="${COLORS.darkBg}"/>
  <rect x="50" y="750" width="700" height="390" rx="10" fill="#FFFFFF"/>
  <rect x="800" y="750" width="750" height="390" rx="10" fill="${COLORS.darkBg}"/>
  <svg x="92" y="226" width="160" height="160" viewBox="0 0 512 512">${iconInner("light", "regular", "overview-app-light")}</svg>
  <svg x="278" y="226" width="160" height="160" viewBox="0 0 512 512">${iconInner("dark", "regular", "overview-app-dark")}</svg>
  <svg x="470" y="226" width="160" height="160" viewBox="0 0 512 512">${fullSymbol("overview-symbol-light", "light")}</svg>
  <text x="82" y="405" font-family="Arial, sans-serif" font-size="20" fill="${COLORS.inkMuted}">Light / Dark / Transparent Symbol</text>
  <svg x="940" y="228" width="150" height="150" viewBox="0 0 512 512">${fullSymbol("overview-symbol-dark", "dark")}</svg>
  <svg x="1164" y="228" width="150" height="150" viewBox="0 0 512 512">${iconInner("dark", "regular", "overview-dark-tile")}</svg>
  <text x="826" y="405" font-family="Arial, sans-serif" font-size="20" fill="#9FC4C3">Dark-background symbol and app tile</text>
  <svg x="100" y="510" width="820" height="190" viewBox="0 0 1600 520">${lockupSvg("light", true).replace(/^[\s\S]*?<desc[^>]*>[\s\S]*?<\/desc>/, "").replace(/<\/svg>\s*$/, "")}</svg>
  <text x="82" y="706" font-family="Arial, sans-serif" font-size="20" fill="${COLORS.inkMuted}">Full-color outlined horizontal lockup</text>
  <svg x="1088" y="506" width="164" height="164" viewBox="0 0 512 512">${monoMarkBody(COLORS.white)}</svg>
  <g transform="translate(1276 512) scale(0.42)"><path d="${AREA_PATH}" fill="${COLORS.white}"/><path d="${FLOW_PATH}" transform="translate(249.05 0)" fill="${COLORS.white}"/></g>
  <path d="M1276 590H1480" stroke="${COLORS.white}" stroke-width="5" stroke-linecap="round"/>
  <text x="1076" y="706" font-family="Arial, sans-serif" font-size="20" fill="#9FC4C3">Single-color lockup</text>
  <svg x="92" y="814" width="280" height="280" viewBox="0 0 1024 1024">${stackedSvg("light").replace(/^[\s\S]*?<desc[^>]*>[\s\S]*?<\/desc>/, "").replace(/<\/svg>\s*$/, "")}</svg>
  <svg x="410" y="814" width="280" height="280" viewBox="0 0 1024 1024"><rect width="1024" height="1024" rx="40" fill="${COLORS.darkBg}"/>${stackedSvg("dark").replace(/^[\s\S]*?<desc[^>]*>[\s\S]*?<\/desc>/, "").replace(/<\/svg>\s*$/, "")}</svg>
  <svg x="880" y="810" width="590" height="310" viewBox="0 0 1200 630">${socialSvg("dark").replace(/^[\s\S]*?<desc[^>]*>[\s\S]*?<\/desc>/, "").replace(/<\/svg>\s*$/, "")}</svg>`,
  );
}

const readme = `# AreaFlow Brand Assets

本目录保存 AreaFlow 当前品牌素材包。机器可读规格位于上一级 \`brand-manifest.json\`，生成、验证与持续门禁见上一级 \`README.md\`。

## 目录

- \`areaflow-app-icon-dark.svg\` / \`areaflow-app-icon-light.svg\`：完整 App/PWA 图标源。
- \`areaflow-app-icon-small-dark.svg\` / \`areaflow-app-icon-small-light.svg\`：16px、32px 和 48px 小尺寸简化源。
- \`areaflow-app-icon-opaque-*.svg\`：Apple Touch Icon 和 App Store 使用的不透明全出血源。
- \`areaflow-app-icon-maskable-*.svg\`：PWA maskable 全出血源，主体位于安全区内。
- \`areaflow-logo-mark-*.svg\`：独立标志源；\`mono\` 文件为单色版本。
- \`areaflow-logo-symbol-*.svg\`：无底板透明 Symbol，针对目标背景适配对比。
- \`areaflow-logo-lockup*.svg\`：横向 Logo，包含默认、深浅背景、轮廓化和单色版本。
- \`areaflow-wordmark-*.svg\`：不带图标的纯字标源。
- \`areaflow-logo-stacked-*.svg\`：竖向堆叠 Logo 源。
- \`app-icon/\`：常规 App icon PNG 为 \`16/32/48/64/128/180/192/256/512/1024\` 深浅两套，并包含 opaque 与 maskable 导出。
- \`mark/\`、\`symbol/\`、\`lockup/\`、\`wordmark/\`、\`stacked/\`：常用 PNG 导出。
- \`favicon/\`：\`16/32/48\` PNG 与多尺寸 ICO。
- \`social/\`：\`1200x630\` 深浅社交预览图；无后缀文件为浅色兼容入口。
- \`native/macos/\`：\`AreaFlow.icns\` 与完整 \`.iconset\`。
- \`native/ios/\`：iPhone、iPad 和 App Store marketing 尺寸的 \`AreaFlowAppIcon.appiconset\`。
- \`native/android/res/\`：Android adaptive icon 前景、背景色和 v26 XML。
- \`native/windows/AreaFlow.ico\`：包含 \`16/24/32/48/64/128/256\` 的 Windows 应用图标。
- \`print/\`：浅色/深色背景的 outlined SVG、矢量 PDF 与 300 DPI CMYK TIFF。
- \`areaflow-brand-overview.png\`：完整数字品牌素材总览。

## 使用边界

- 主体结构固定为调度盘、双 Flow 轨迹、执行节点、完成节点和弱底线，不再改变核心构图。
- 深浅版本保持同一结构，只做背景、灰度和对比适配。
- 横向字标中 \`Area\` 使用中性色，\`Flow\` 使用青绿、青色、琥珀和珊瑚红渐变。
- 16px、32px 和 48px 必须使用 small 源，避免调度刻度与中心细节糊成一团。
- 常规 App icon 保留圆角透明边；opaque 与 maskable 必须全画布不透明。
- \`lockup/wordmark/stacked/symbol\` 的 \`dark/light\` 表示目标背景；\`mono\` 的 \`dark/light\` 表示墨线明暗。
- 对外直接分发横向 SVG 时优先使用 \`outlined\` 版本，避免字体替换。
- 社交预览图定位为“AI 开发执行治理平台”，主标题为“把需求编排成可审计的软件交付”。
- 横向 Logo 最小屏幕宽度为 120px，印刷最小宽度为 25mm；低于 48px 使用 small icon，最低不得小于 16px。
- Apple 与 Windows 原生包使用 opaque dark 源；Android 使用透明 Symbol 和品牌深色背景。
- 印刷 CMYK 文件是从 sRGB 品牌色换算的通用起点，正式生产仍需印厂打样。

## 标准色

| 名称 | HEX | 用途 |
|---|---|---|
| Flow Ink 950 | \`#07191D\` | 深色背景、App icon 底板 |
| Flow Ink 900 | \`#09272D\` | 浅色背景字标、描边 |
| Scheduler Mint | \`#36D9A6\` | 起始节点与 Flow 轨迹 |
| Control Cyan | \`#18BFC7\` | 调度核心与主强调 |
| Evidence Amber | \`#F5B02E\` | 执行证据节点 |
| Completion Coral | \`#F46D5E\` | 完成节点与风险强调 |
| Control Mist | \`#F4FBF8\` | 深色背景文字 |
| Surface Mist | \`#F1FAF7\` | 浅色图标底板 |

## 生成与校验

\`\`\`bash
npm ci
npm run brand:export
npm run brand:validate
\`\`\`

生成器读取 \`brand-manifest.json\` 的尺寸、透明度、平台来源和印刷 DPI；校验器覆盖全部清单输出、原生包、印刷包和目录卫生。

默认导出只补齐缺失文件；需要从当前 SVG 全量重建时运行 \`npm run brand:export -- --refresh\`。
`;

const nativeReadme = `# AreaFlow Native Icons

本目录保存从品牌 opaque icon 和透明 Symbol 生成的原生平台交付文件。

- \`macos/AreaFlow.icns\`：直接设置为 macOS 应用图标；\`.iconset\` 保留 Xcode 和重新打包所需的源尺寸。
- \`ios/AreaFlowAppIcon.appiconset/\`：复制到 Xcode asset catalog，包含 iPhone、iPad 与 App Store marketing 槽位。
- \`android/res/\`：把目录内容合并到 Android 工程 \`app/src/main/res/\`；\`ic_launcher.xml\` 和 \`ic_launcher_round.xml\` 使用同一安全区前景。
- \`windows/AreaFlow.ico\`：包含 \`16/24/32/48/64/128/256\`，用于 Windows 可执行文件和快捷方式。

这些文件由 \`npm run brand:export\` 从当前品牌源和机器清单重建。更新品牌源后运行生成器，再运行 \`npm run brand:validate\`。
`;

const printReadme = `# AreaFlow Print Assets

## 文件选择

- \`*.svg\`：轮廓化矢量源，适合排版、缩放和专业设计工具。
- \`*.pdf\`：可直接预览和交换的单页文件；dark 版本包含品牌深色背景，保证反白字标可见。
- \`*-cmyk.tiff\`：\`3600x1170\`、300 DPI、LZW 压缩的 CMYK 位图，用于不接收 SVG/PDF 的印刷流程。

CMYK 文件是从 sRGB 品牌色转换的通用起点，正式大批量印刷仍需由印厂按纸张、油墨和设备打样。

本包不提供 EPS。EPS 无法可靠保留当前 Logo 的透明度和渐变。需要旧版设计软件兼容时，应优先导入 outlined SVG 或 PDF，再由目标软件按自身色彩配置导出。
`;

module.exports = {
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
};
