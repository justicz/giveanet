(function() {
  // Set referral cookie if there's an r param
  var u = new URL(window.location.href);
  var referralCode = u.searchParams.get("r");
  if (typeof(referralCode) === "string") {
    if (/^[A-Z2-7]{13}$/.test(referralCode)) {
      document.cookie = "referral=" + referralCode;
    }
  }
})();

function submissionButtons(enabled) {
  var buttons = document.getElementsByClassName("submissionbutton");
  for (var i = 0; i < buttons.length; i++) {
    buttons[i].style.opacity = enabled ? "1.0" : "0.5";
    buttons[i].style.pointerEvents = enabled ? "auto" : "none";
  }
}

function pushMessage(table, message) {
  var newRow = document.createElement("tr");
  var msgCol = document.createElement("td");
  newRow.classList.add("noanimate");
  msgCol.setAttribute("colspan", "3");
  msgCol.innerText = message;
  newRow.appendChild(msgCol);
  table.appendChild(newRow);
}
