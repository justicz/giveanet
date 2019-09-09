var cnv = document.getElementById("cnv");
var celsz = 25;
var startedDrawing = false;

(function () {
  // Load finger touch image
  var img = new Image();
  img.onload = function() {
    if (!startedDrawing) {
      var ctx = cnv.getContext("2d");
      ctx.drawImage(img, 0, 0, cnv.width, cnv.height);
    }
  }
  img.src = "/static/image/finger.png";
})();

function initializeDrawingPage() {
  // Initialize color picker
  var wrap = document.getElementsByClassName("colorpickerwrapper")[0];
  var drawwrap = document.getElementsByClassName("drawwrapper")[0];
  var picker = new CP(document.getElementById("selcol"), false, wrap);
  picker.fit = function() {
    this.self.style.left = this.self.style.top = "";
  };
  picker.self.classList.add('static');
  picker.set("#9B60FE");
  picker.enter();
  picker.on("change", function(color) {
    this.source.value = '#' + color;
    picker.self.style.borderColor = this.source.value;
    drawwrap.style.borderColor = this.source.value;
  });

  // Initialize canvas
  function clearCanvas() {
    var ctx = cnv.getContext("2d");
    ctx.beginPath();
    ctx.rect(0, 0, cnv.width, cnv.height);
    ctx.fillStyle = "#FFFFFF";
    ctx.fill();
  }

  // Register event handlers
  cnv.addEventListener("mousedown",   draw);
  cnv.addEventListener("mouseup",     draw);
  cnv.addEventListener("mousemove",   draw);
  cnv.addEventListener("mouseout",    draw);
  cnv.addEventListener("touchstart",  draw, { passive: false });
  cnv.addEventListener("touchend",    draw, { passive: false });
  cnv.addEventListener("touchcancel", draw, { passive: false });
  cnv.addEventListener("touchmove",   draw, { passive: false });

  // Initialize state
  var holding = false;

  function draw(e) {
    e.preventDefault();

    // Start drawing
    if (e.type == "mousedown") {
      holding = true;
    }

    // Stop drawing
    if (e.type == "mouseup" || e.type == "mouseout") {
      holding = false;
    }

    // Fill in "touches" property on desktop so that we can use the same
    // drawing code for both
    if (!e.touches) {
      e.touches = [{ clientX: e.clientX, clientY: e.clientY }]
    }

    // Iterate over all touches
    for (var i = 0; i < e.touches.length; i++) {
      // Compute position on canvas
      var boundingBox = cnv.getBoundingClientRect();
      var x = e.touches[i].clientX - boundingBox.left;
      var y = e.touches[i].clientY - boundingBox.top;

      // If we're outside the canvas, bail out (can happen with touch)
      if (x < 0 || x > cnv.width || y < 0 || y > cnv.height) {
        continue
      }

      // Round to celsz
      rx = parseInt(x / celsz) * celsz;
      ry = parseInt(y / celsz) * celsz;

      // If holding down, fill in square
      if (holding || e.type == "touchmove" || e.type == "touchstart") {
        if (!startedDrawing) {
          startedDrawing = true;
          cnv.setAttribute("touched", "touched");
          clearCanvas();
        }

        // Actually draw
        var ctx = cnv.getContext("2d");
        ctx.beginPath();
        ctx.rect(rx, ry, celsz, celsz);
        ctx.fillStyle = picker.source.value;
        ctx.fill();
      }
    }
  }
}

// Downsample canvas for form submission
function sampleCanvas() {
  // If we haven't touched the canvas, leave canvasdata blank
  if (!cnv.getAttribute("touched")) {
    return "";
  }

  // Downsample image
  var ctx = cnv.getContext("2d");
  var area = Math.floor(cnv.width / celsz) * Math.floor(cnv.height / celsz);
  var samples = new Uint8Array(area * 3);
  var i = 0;
  for (var y = 0; y < cnv.height; y += celsz) {
    for (var x = 0; x < cnv.width; x += celsz) {
      var sample = ctx.getImageData(x, y, 1, 1).data;
      samples[i++] = sample[0]
      samples[i++] = sample[1]
      samples[i++] = sample[2]
    }
  }

  // Serialize bytes
  var bytes = [];
  for (var i = 0; i < samples.byteLength; i++) {
    bytes.push(String.fromCharCode(samples[i]));
  }

  return btoa(bytes.join(''));
}
