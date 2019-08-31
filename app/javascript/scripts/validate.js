var alertBox = document.getElementById("alertbox");
function addAlert(msg) {
  var alertMessage = document.createElement("h4");
  alertMessage.innerText = msg;
  alertBox.appendChild(alertMessage);
  alertBox.style.display = "block";
}

function clearAlerts() {
  alertBox.style.display = "none";
  while (alertBox.firstChild) {
    alertBox.removeChild(alertBox.firstChild);
  }
}
