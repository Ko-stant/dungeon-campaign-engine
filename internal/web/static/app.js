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

  canvasContext.strokeStyle = `rgb(${colorBorder})`;
  canvasContext.globalAlpha = 0.25;
  for (let y = 0; y < m.rows; y++) {
    for (let x = 0; x < m.cols; x++) {
      const r = tileRect(x, y, m.tile, m.ox, m.oy);
      canvasContext.strokeRect(r.x, r.y, r.w, r.h);
    }
  }
  canvasContext.globalAlpha = 1;

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
  } else if (patch.type === "EntityUpdated" && patch.payload && patch.payload.tile) {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    const p = patch.payload;
    entityPositions.set(p.id, p.tile);
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
    } catch {}
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
  if (!step) return;
  if (!socketRef || socketRef.readyState !== WebSocket.OPEN) return;
  const msg = {
    type: "RequestMove",
    payload: { entityId: "hero-1", dx: step.dx, dy: step.dy }
  };
  socketRef.send(JSON.stringify(msg));
  ev.preventDefault();
});

window.addEventListener("resize", () => {
  resizeCanvas();
  drawBoard();
});

resizeCanvas();
drawBoard();
openStream();
