const snapshotElement = document.getElementById("snapshot");
let snapshot = null;
if (snapshotElement) {
  snapshot = JSON.parse(snapshotElement.textContent);
  window.__SNAPSHOT__ = snapshot;
}

/**
 * @type {HTMLCanvasElement}
 */
const canvas = document.getElementById("board");
/**
 * @type {CanvasRenderingContext2D}
 */
const canvasContext = canvas.getContext("2d");
const patchCountElement = document.getElementById("patchCount");
const toggleDoorButton = document.getElementById("toggleDoor");

let patchCount = 0;
let revealedRegions = new Set(Array.isArray(snapshot?.revealedRegionIds) ? snapshot.revealedRegionIds : []);
const doorStateById = new Map();
if (Array.isArray(snapshot?.doorSockets)) {
  for (const d of snapshot.doorSockets) doorStateById.set(d.id, d.state || "closed");
}

function resizeCanvas() {
  const rect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  canvas.width = Math.floor(rect.width * dpr);
  canvas.height = Math.floor(rect.height * dpr);
  canvasContext.setTransform(dpr, 0, 0, dpr, 0, 0);
}

function drawBoard() {
  const rect = canvas.getBoundingClientRect();
  const cols = snapshot?.mapWidth ?? 26;
  const rows = snapshot?.mapHeight ?? 19;

  const tileSize = Math.floor(Math.min(rect.width / cols, rect.height / rows));
  const gridWidth = tileSize * cols;
  const gridHeight = tileSize * rows;
  const offsetX = Math.floor((rect.width - gridWidth) / 2);
  const offsetY = Math.floor((rect.height - gridHeight) / 2);

  const colorSurface = getComputedStyle(document.documentElement).getPropertyValue("--color-surface").trim();
  const colorSurface2 = getComputedStyle(document.documentElement).getPropertyValue("--color-surface-2").trim();
  const colorBorder = getComputedStyle(document.documentElement).getPropertyValue("--color-border-rgb").trim();

  canvasContext.clearRect(0, 0, rect.width, rect.height);
  canvasContext.fillStyle = `rgb(${colorSurface})`;
  canvasContext.fillRect(0, 0, rect.width, rect.height);

  if (Array.isArray(snapshot?.tileRegionIds)) {
    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < cols; x++) {
        const idx = y * cols + x;
        const regionId = snapshot.tileRegionIds[idx];
        const visible = revealedRegions.has(regionId);
        canvasContext.fillStyle = visible ? `rgb(${colorSurface2})` : `rgb(${colorSurface})`;
        canvasContext.fillRect(offsetX + x * tileSize, offsetY + y * tileSize, tileSize, tileSize);
      }
    }
  }

  canvasContext.strokeStyle = `rgb(${colorBorder})`;
  canvasContext.globalAlpha = 0.25;
  for (let y = 0; y < rows; y++) {
    for (let x = 0; x < cols; x++) {
      canvasContext.strokeRect(offsetX + x * tileSize, offsetY + y * tileSize, tileSize, tileSize);
    }
  }
  canvasContext.globalAlpha = 1;

  if (Array.isArray(snapshot?.doorSockets)) {
    for (const d of snapshot.doorSockets) {
      const state = doorStateById.get(d.id) || "closed";
      const open = state === "open";
      canvasContext.strokeStyle = open ? "rgb(74 222 128)" : "rgb(248 113 113)";
      canvasContext.lineWidth = 3;
      if (d.orientation === "vertical") {
        const x = offsetX + d.x * tileSize;
        const y = offsetY + d.y * tileSize;
        canvasContext.beginPath();
        canvasContext.moveTo(x + tileSize, y + 2);
        canvasContext.lineTo(x + tileSize, y + tileSize - 2);
        canvasContext.stroke();
      } else {
        const x = offsetX + d.x * tileSize;
        const y = offsetY + d.y * tileSize;
        canvasContext.beginPath();
        canvasContext.moveTo(x + 2, y + tileSize);
        canvasContext.lineTo(x + tileSize - 2, y + tileSize);
        canvasContext.stroke();
      }
    }
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
  } else if (patch.type === "DoorStateChanged" && patch.payload && patch.payload.thresholdId) {
    doorStateById.set(patch.payload.thresholdId, patch.payload.state);
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  }
}

function openStream() {
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  const url = `${scheme}://${location.host}/stream`;
  const socket = new WebSocket(url);
  socket.onopen = () => {};
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

if (toggleDoorButton) {
  toggleDoorButton.addEventListener("click", async () => {
    try { await fetch("/dev/toggle-door", { method: "POST" }); } catch {}
  });
}

window.addEventListener("resize", () => {
  resizeCanvas();
  drawBoard();
});

resizeCanvas();
drawBoard();
openStream();
