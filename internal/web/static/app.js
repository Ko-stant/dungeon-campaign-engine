const snapshotElement = document.getElementById("snapshot");
let snapshot = null;
if (snapshotElement) {
  snapshot = JSON.parse(snapshotElement.textContent);
  window.__SNAPSHOT__ = snapshot;
}

const canvas = document.getElementById("board");
const canvasContext = canvas.getContext("2d");
const patchCountElement = document.getElementById("patchCount");

let patchCount = 0;

function resizeCanvas() {
  const clientRect = canvas.getBoundingClientRect();
  const devicePixelRatioValue = window.devicePixelRatio || 1;
  canvas.width = Math.floor(clientRect.width * devicePixelRatioValue);
  canvas.height = Math.floor(clientRect.height * devicePixelRatioValue);
  canvasContext.setTransform(devicePixelRatioValue, 0, 0, devicePixelRatioValue, 0, 0);
}

function drawPlaceholder() {
  const clientRect = canvas.getBoundingClientRect();
  canvasContext.clearRect(0, 0, clientRect.width, clientRect.height);
  canvasContext.fillStyle = getComputedStyle(document.documentElement).getPropertyValue("--color-surface-2");
  canvasContext.fillRect(0, 0, clientRect.width, clientRect.height);
  canvasContext.strokeStyle = getComputedStyle(document.documentElement).getPropertyValue("--color-border-rgb");
  canvasContext.globalAlpha = 0.25;
  const tileSize = Math.floor(clientRect.width / 26);
  for (let y = 0; y < 19; y++) {
    for (let x = 0; x < 26; x++) {
      canvasContext.strokeRect(x * tileSize, y * tileSize, tileSize, tileSize);
    }
  }
  canvasContext.globalAlpha = 1;
}

function applyPatch(patch) {
  if (patch.type === "VariablesChanged" && patch.payload && patch.payload.entries) {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
  }
}

function openStream() {
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  const url = `${scheme}://${location.host}/stream`;
  const socket = new WebSocket(url);
  socket.onmessage = (event) => {
    try {
      const patch = JSON.parse(event.data);
      applyPatch(patch);
    } catch (_e) {}
  };
  socket.onclose = () => {
    setTimeout(openStream, 2000);
  };
}

window.addEventListener("resize", () => {
  resizeCanvas();
  drawPlaceholder();
});

resizeCanvas();
drawPlaceholder();
openStream();
