const canvas = document.getElementById("board");
const canvasContext = canvas.getContext("2d");

function resizeCanvas() {
  const clientRect = canvas.getBoundingClientRect();
  const devicePixelRatioValue = window.devicePixelRatio || 1;
  canvas.width = Math.floor(clientRect.width * devicePixelRatioValue);
  canvas.height = Math.floor(clientRect.height * devicePixelRatioValue);
  canvasContext.scale(devicePixelRatioValue, devicePixelRatioValue);
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
window.addEventListener("resize", () => {
  resizeCanvas();
  drawPlaceholder();
});
resizeCanvas();
drawPlaceholder();
