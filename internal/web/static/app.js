/**
 * @typedef {Object} TileAddress
 * @property {string} segmentId
 * @property {number} x
 * @property {number} y
 */
/**
 * @typedef {Object} EntityLite
 * @property {string} id
 * @property {string} kind
 * @property {TileAddress} tile
 */
/**
 * @typedef {Object} Snapshot
 * @property {string} mapId
 * @property {string} packId
 * @property {number} turn
 * @property {number} lastEventId
 * @property {number} mapWidth
 * @property {number} mapHeight
 * @property {number} regionsCount
 * @property {number[]} tileRegionIds
 * @property {number[]} revealedRegionIds
 * @property {EntityLite[]} entities
 * @property {string} protocolVersion
 * @property {ThresholdLite[]} thresholds
 * @property {number[]} visibleRegionIds
 * @property {number} corridorRegionId
 * @property {number[]} knownRegionIds
 */

/**
 * @typedef {Object} ThresholdLite
 * @property {string} id
 * @property {number} x
 * @property {number} y
 * @property {"vertical"|"horizontal"} orientation
 * @property {"DoorSocket"} kind
 * @property {"open"|"closed"} [state]
 */

const snapshotElement = document.getElementById("snapshot");
/** @type {Snapshot|null} */
let snapshot = null;
if (snapshotElement) {
  snapshot = JSON.parse(snapshotElement.textContent);
  window.__SNAPSHOT__ = snapshot;
}

/** @type {HTMLCanvasElement} */
const canvas = document.getElementById("board");
/** @type {CanvasRenderingContext2D} */
const canvasContext = canvas.getContext("2d");
/** @type {HTMLElement} */
const patchCountElement = document.getElementById("patchCount");
/** @type {HTMLButtonElement|null} */
const toggleDoorButton = document.getElementById("toggleDoor");

/** @type {string|null} */
let selectedDoorId = null;

let patchCount = 0;
/** @type {Set<number>} */
let revealedRegions = new Set(Array.isArray(snapshot?.revealedRegionIds) ? snapshot.revealedRegionIds : []);
/** @type {WebSocket|null} */
let socketRef = null;
/** @type {Map<string, TileAddress>} */
let entityPositions = new Map();
if (Array.isArray(snapshot?.entities)) {
  for (const e of snapshot.entities) entityPositions.set(e.id, structuredClone(e.tile));
}

/** @type {ThresholdLite[]} */
let thresholds = Array.isArray(snapshot?.thresholds) ? snapshot.thresholds.slice() : [];
/** @type {Set<number>} */
let visibleNow = new Set(Array.isArray(snapshot?.visibleRegionIds) ? snapshot.visibleRegionIds : []);
/** @type {number} */
const corridorRegionId = typeof snapshot?.corridorRegionId === "number" ? snapshot.corridorRegionId : 0;
/** @type {Set<number>} */
let knownRegions = new Set(Array.isArray(snapshot?.knownRegionIds) ? snapshot.knownRegionIds : []);


function resizeCanvas() {
  const clientRect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  canvas.width = Math.floor(clientRect.width * dpr);
  canvas.height = Math.floor(clientRect.height * dpr);
  canvasContext.setTransform(dpr, 0, 0, dpr, 0, 0);
}

function gridMetrics() {
  const clientRect = canvas.getBoundingClientRect();
  const cols = snapshot?.mapWidth ?? 26;
  const rows = snapshot?.mapHeight ?? 19;
  const tile = Math.floor(Math.min(clientRect.width / cols, clientRect.height / rows));
  const gridW = tile * cols;
  const gridH = tile * rows;
  const ox = Math.floor((clientRect.width - gridW) / 2);
  const oy = Math.floor((clientRect.height - gridH) / 2);
  return { cols, rows, tile, ox, oy, width: clientRect.width, height: clientRect.height };
}

/**
 * @param {number} x
 * @param {number} y
 * @param {number} tile
 * @param {number} ox
 * @param {number} oy
 * @returns {{x:number,y:number,w:number,h:number,cx:number,cy:number}}
 */
function tileRect(x, y, tile, ox, oy) {
  const px = ox + x * tile;
  const py = oy + y * tile;
  const cx = px + tile / 2;
  const cy = py + tile / 2;
  return { x: px, y: py, w: tile, h: tile, cx, cy };
}

function drawDoorOverlays() {
  const m = gridMetrics();
  if (!Array.isArray(thresholds) || thresholds.length === 0) return;

  const cs = getComputedStyle(document.documentElement);
  const closedRGB = cs.getPropertyValue("--color-brand").trim();
  const openRGB = cs.getPropertyValue("--color-positive").trim();

  canvasContext.save();
  canvasContext.lineWidth = 2;
  for (const t of thresholds) {
    const cs = getComputedStyle(document.documentElement);
    const closedRGB = cs.getPropertyValue("--color-brand").trim();
    const openRGB = cs.getPropertyValue("--color-positive").trim();
    const accentRGB = cs.getPropertyValue("--color-accent").trim();

    const isSelected = selectedDoorId === t.id;
    const base = t.state === "open" ? openRGB : closedRGB;
    canvasContext.strokeStyle = `rgb(${isSelected ? accentRGB : base})`;
    canvasContext.lineWidth = isSelected ? 3 : 2;

    const m = gridMetrics();
    const gap = Math.max(2, Math.floor(m.tile * 0.15));
    if (t.orientation === "vertical") {
      const xLine = m.ox + (t.x + 1) * m.tile + 0.5;
      const y1 = m.oy + t.y * m.tile + gap + 0.5;
      const y2 = y1 + (m.tile - 2 * gap);
      canvasContext.beginPath();
      canvasContext.moveTo(xLine, y1);
      canvasContext.lineTo(xLine, y2);
      canvasContext.stroke();
    } else {
      const yLine = m.oy + (t.y + 1) * m.tile + 0.5;
      const x1 = m.ox + t.x * m.tile + gap + 0.5;
      const x2 = x1 + (m.tile - 2 * gap);
      canvasContext.beginPath();
      canvasContext.moveTo(x1, yLine);
      canvasContext.lineTo(x2, yLine);
      canvasContext.stroke();
    }
  }

  canvasContext.restore();
}

/**
 * @param {string} entityId
 * @param {number} dx
 * @param {number} dy
 * @returns {string|null}
 */
function findAdjacentDoorIdFromDirection(entityId, dx, dy) {
  const pos = entityPositions.get(entityId);
  if (!pos) return null;
  /** @type {{x:number,y:number,orientation:"vertical"|"horizontal"}|null} */
  let edge = null;
  if (dx === 1 && dy === 0) edge = { x: pos.x, y: pos.y, orientation: "vertical" };
  else if (dx === -1 && dy === 0) edge = { x: pos.x - 1, y: pos.y, orientation: "vertical" };
  else if (dx === 0 && dy === 1) edge = { x: pos.x, y: pos.y, orientation: "horizontal" };
  else if (dx === 0 && dy === -1) edge = { x: pos.x, y: pos.y - 1, orientation: "horizontal" };
  if (!edge) return null;
  const t = thresholds.find(d => d.orientation === edge.orientation && d.x === edge.x && d.y === edge.y);
  return t ? t.id : null;
}

/**
 * @param {number} dx
 * @param {number} dy
 */
function selectDoorInDirection(dx, dy) {
  selectedDoorId = findAdjacentDoorIdFromDirection("hero-1", dx, dy);
  drawBoard();
}

function drawBoard() {
  const m = gridMetrics();

  const cs = getComputedStyle(document.documentElement);
  const colorSurface = cs.getPropertyValue("--color-surface").trim();
  const colorSurface2 = cs.getPropertyValue("--color-surface-2").trim();
  const colorBorder = cs.getPropertyValue("--color-border-rgb").trim();

  canvasContext.clearRect(0, 0, m.width, m.height);
  canvasContext.fillStyle = `rgb(${colorSurface})`;
  canvasContext.fillRect(0, 0, m.width, m.height);

  if (Array.isArray(snapshot?.tileRegionIds)) {
    for (let y = 0; y < m.rows; y++) {
      for (let x = 0; x < m.cols; x++) {
        const idx = y * m.cols + x;
        const rid = snapshot.tileRegionIds[idx];
        const visible = revealedRegions.has(rid);
        canvasContext.fillStyle = visible ? `rgb(${colorSurface2})` : `rgb(${colorSurface})`;
        const r = tileRect(x, y, m.tile, m.ox, m.oy);
        canvasContext.fillRect(r.x, r.y, r.w, r.h);
      }
    }
  }

  drawGrid();          // subtle per-cell grid for counting
  drawRegionBorders(); // bold room/corridor outlines
  drawDoorOverlays();  // door overlays on top of regions

  for (const [id, t] of entityPositions.entries()) {
    if (!t) continue;
    const r = tileRect(t.x, t.y, m.tile, m.ox, m.oy);
    canvasContext.beginPath();
    canvasContext.arc(r.cx, r.cy, Math.max(2, Math.floor(m.tile * 0.35)), 0, Math.PI * 2);
    canvasContext.closePath();
    canvasContext.fillStyle = `rgb(${cs.getPropertyValue("--color-accent").trim()})`;
    canvasContext.fill();
  }
}

/**
 * Draw a subtle per-cell grid across the whole board for movement counting.
 * Uses half-pixel alignment for crisp 1px lines.
 */
function drawGrid() {
  const m = gridMetrics();
  const cs = getComputedStyle(document.documentElement);
  const borderRGB = cs.getPropertyValue("--color-border").trim(); // "r g b"

  canvasContext.save();
  canvasContext.strokeStyle = `rgb(${borderRGB})`;
  canvasContext.lineWidth = 1;
  canvasContext.globalAlpha = 0.18;

  // Vertical grid lines (x = 0..cols)
  canvasContext.beginPath();
  for (let x = 0; x <= m.cols; x++) {
    const xLine = m.ox + x * m.tile + 0.5;
    canvasContext.moveTo(xLine, m.oy);
    canvasContext.lineTo(xLine, m.oy + m.tile * m.rows);
  }
  canvasContext.stroke();

  // Horizontal grid lines (y = 0..rows)
  canvasContext.beginPath();
  for (let y = 0; y <= m.rows; y++) {
    const yLine = m.oy + y * m.tile + 0.5;
    canvasContext.moveTo(m.ox, yLine);
    canvasContext.lineTo(m.ox + m.tile * m.cols, yLine);
  }
  canvasContext.stroke();

  canvasContext.restore();
}

/**
 * Draw thin region borders only where neighbor tiles belong to different regions.
 * Uses a stronger color so rooms/corridors remain visually distinct when revealed.
 */
function drawRegionBorders() {
  const m = gridMetrics();
  const ids = snapshot?.tileRegionIds;
  if (!Array.isArray(ids)) return;

  const cs = getComputedStyle(document.documentElement);
  const brandRGB = cs.getPropertyValue("--color-brand").trim();

  canvasContext.save();
  canvasContext.strokeStyle = `rgb(${brandRGB})`;
  canvasContext.lineWidth = 1;
  canvasContext.globalAlpha = 0.85;

  // Vertical borders: between (x,y) and (x+1,y)
  for (let y = 0; y < m.rows; y++) {
    for (let x = 0; x < m.cols - 1; x++) {
      const a = ids[y * m.cols + x];
      const b = ids[y * m.cols + (x + 1)];
      if (a !== b) {
        const roomSideA = (a !== corridorRegionId) ? a : null;
        const roomSideB = (b !== corridorRegionId) ? b : null;

        const show =
          (roomSideA !== null && (knownRegions.has(roomSideA) || revealedRegions.has(roomSideA))) ||
          (roomSideB !== null && (knownRegions.has(roomSideB) || revealedRegions.has(roomSideB)));
        if (show) {
          const r = tileRect(x, y, m.tile, m.ox, m.oy);
          const xLine = r.x + r.w + 0.5;
          canvasContext.beginPath();
          canvasContext.moveTo(xLine, r.y);
          canvasContext.lineTo(xLine, r.y + r.h);
          canvasContext.stroke();
        }
      }
    }
  }
  // Horizontal borders: between (x,y) and (x,y+1)
  for (let y = 0; y < m.rows - 1; y++) {
    for (let x = 0; x < m.cols; x++) {
      const a = ids[y * m.cols + x];
      const b = ids[(y + 1) * m.cols + x];
      if (a !== b) {
        const roomSideA = (a !== corridorRegionId) ? a : null;
        const roomSideB = (b !== corridorRegionId) ? b : null;

        const show =
          (roomSideA !== null && (knownRegions.has(roomSideA) || revealedRegions.has(roomSideA))) ||
          (roomSideB !== null && (knownRegions.has(roomSideB) || revealedRegions.has(roomSideB)));
        if (show) {
          const r = tileRect(x, y, m.tile, m.ox, m.oy);
          const yLine = r.y + r.h + 0.5;
          canvasContext.beginPath();
          canvasContext.moveTo(r.x, yLine);
          canvasContext.lineTo(r.x + r.w, yLine);
          canvasContext.stroke();
        }
      }
    }
  }
  canvasContext.restore();
}


function applyPatch(patch) {
  if (patch.type === "VariablesChanged" && patch.payload && patch.payload.entries) {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
  } else if (patch.type === "RegionsRevealed" && patch.payload && Array.isArray(patch.payload.ids)) {
    for (const id of patch.payload.ids) revealedRegions.add(id);
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "DoorStateChanged") {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    const p = patch.payload;
    const idx = thresholds.findIndex(d => d.id === p.thresholdId);
    if (idx !== -1) {
      thresholds[idx] = { ...thresholds[idx], state: p.state };
      drawBoard();
    }
  } else if (patch.type === "EntityUpdated" && patch.payload && patch.payload.tile) {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    const p = patch.payload;
    entityPositions.set(p.id, p.tile);
    drawBoard();
  } else if (patch.type === "VisibleNow" && patch.payload && Array.isArray(patch.payload.ids)) {
    visibleNow = new Set(patch.payload.ids);
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "RegionsKnown" && patch.payload && Array.isArray(patch.payload.ids)) {
    for (const id of patch.payload.ids) knownRegions.add(id);
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  }
}

function openStream() {
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  const url = `${scheme}://${location.host}/stream`;
  const socket = new WebSocket(url);
  socketRef = socket;

  socket.onmessage = (event) => {
    try {
      const patch = JSON.parse(event.data);
      applyPatch(patch);
    } catch { }
  };
  socket.onclose = () => {
    setTimeout(openStream, 2000);
  };
}

/**
 * @param {KeyboardEvent} ev
 * @returns {{dx:number, dy:number}|null}
 */
function keyToStep(ev) {
  switch (ev.key) {
    case "ArrowLeft":
    case "a":
    case "A":
      return { dx: -1, dy: 0 };
    case "ArrowRight":
    case "d":
    case "D":
      return { dx: 1, dy: 0 };
    case "ArrowUp":
    case "w":
    case "W":
      return { dx: 0, dy: -1 };
    case "ArrowDown":
    case "s":
    case "S":
      return { dx: 0, dy: 1 };
    default:
      return null;
  }
}

window.addEventListener("keydown", (ev) => {
  const step = keyToStep(ev);
  if (step) {
    selectDoorInDirection(step.dx, step.dy); // always update selection
    if (socketRef && socketRef.readyState === WebSocket.OPEN) {
      const msg = { type: "RequestMove", payload: { entityId: "hero-1", dx: step.dx, dy: step.dy } };
      socketRef.send(JSON.stringify(msg));
    }
    ev.preventDefault();
    return;
  }
  if (ev.key === "e" || ev.key === "E") {
    if (selectedDoorId && socketRef && socketRef.readyState === WebSocket.OPEN) {
      const msg = { type: "RequestToggleDoor", payload: { thresholdId: selectedDoorId } };
      socketRef.send(JSON.stringify(msg));
    }
    ev.preventDefault();
  }
});


window.addEventListener("resize", () => {
  resizeCanvas();
  drawBoard();
});

resizeCanvas();
drawBoard();
openStream();
