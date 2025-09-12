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
let socketRef = null;

function resizeCanvas() {
  const clientRect = canvas.getBoundingClientRect();
  const devicePixelRatioValue = window.devicePixelRatio || 1;
  canvas.width = Math.floor(clientRect.width * devicePixelRatioValue);
  canvas.height = Math.floor(clientRect.height * devicePixelRatioValue);
  canvasContext.setTransform(devicePixelRatioValue, 0, 0, devicePixelRatioValue, 0, 0);
}

function drawBoard() {
  const clientRect = canvas.getBoundingClientRect();
  const cols = snapshot?.mapWidth ?? 26;
  const rows = snapshot?.mapHeight ?? 19;

  const tileSize = Math.floor(Math.min(clientRect.width / cols, clientRect.height / rows));
  const gridWidth = tileSize * cols;
  const gridHeight = tileSize * rows;
  const offsetX = Math.floor((clientRect.width - gridWidth) / 2);
  const offsetY = Math.floor((clientRect.height - gridHeight) / 2);

  const colorSurface = getComputedStyle(document.documentElement).getPropertyValue("--color-surface");
  const colorSurface2 = getComputedStyle(document.documentElement).getPropertyValue("--color-surface-2");
  const colorBorder = getComputedStyle(document.documentElement).getPropertyValue("--color-border-rgb");

  canvasContext.clearRect(0, 0, clientRect.width, clientRect.height);
  canvasContext.fillStyle = `rgb(${colorSurface.trim()})`;
  canvasContext.fillRect(0, 0, clientRect.width, clientRect.height);

  if (Array.isArray(snapshot?.tileRegionIds)) {
    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < cols; x++) {
        const idx = y * cols + x;
        const regionId = snapshot.tileRegionIds[idx];
        const visible = revealedRegions.has(regionId);
        canvasContext.fillStyle = visible ? `rgb(${colorSurface2.trim()})` : `rgb(${colorSurface.trim()})`;
        canvasContext.fillRect(offsetX + x * tileSize, offsetY + y * tileSize, tileSize, tileSize);
      }
    }
  }

  canvasContext.strokeStyle = `rgb(${colorBorder.trim()})`;
  canvasContext.globalAlpha = 0.25;
  for (let y = 0; y < rows; y++) {
    for (let x = 0; x < cols; x++) {
      canvasContext.strokeRect(offsetX + x * tileSize, offsetY + y * tileSize, tileSize, tileSize);
    }
  }
  canvasContext.globalAlpha = 1;
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
  }
}

function openStream() {
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  const url = `${scheme}://${location.host}/stream`;
  const socket = new WebSocket(url);
  socketRef = socket;
  socket.onopen = () => {};
  socket.onmessage = (event) => {
    try {
      const patch = JSON.parse(event.data);
      applyPatch(patch);
    } catch {}
  };
  socket.onerror = () => {};
  socket.onclose = () => {
    setTimeout(openStream, 2000);
  };
}

if (toggleDoorButton) {
  toggleDoorButton.addEventListener("click", () => {
    if (!socketRef || socketRef.readyState !== WebSocket.OPEN) return;
    const message = {
      type: "RequestToggleDoor",
      payload: { thresholdId: "dev-seg-0:12:9:vertical" }
    };
    console.log("Sending", message);
    socketRef.send(JSON.stringify(message));
  });
}

window.addEventListener("resize", () => {
  resizeCanvas();
  drawBoard();
});

resizeCanvas();
drawBoard();
openStream();