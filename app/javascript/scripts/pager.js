(function() {
function setEnabled(button, enabled) {
  button.style.opacity = enabled ? "1.0" : "0.5";
  button.style.pointerEvents = enabled ? "auto" : "none";
}

var backButton = document.getElementById("backbutton");
var nextButton = document.getElementById("nextbutton");

function clearEntries(table) {
  var child = table.firstChild;
  while (child) {
    table.removeChild(child);
    child = table.firstChild;
  }
}

function displayMessage(table, message) {
  // Pad out entries (minEntries is odd)
  fillEntries(table, Math.floor(minEntries/2));
  pushMessage(table, message);
  fillEntries(table, Math.floor(minEntries/2));
}

function fillEntries(table, n) {
  for (var i = 0; i < n; i++) {
    pushMessage(table, "");
  }
}

function getPageNumber() {
  // Figure out what page to request from the URL
  var u = new URL(window.location.href);
  var parts = u.pathname.split("/");
  var page = parseInt(parts[parts.length - 1]);
  if (isNaN(page)) {
    page = 1;
  }
  return page;
}

function fetchPage(page, pushState) {
  // Disable submission buttons
  setEnabled(backButton, false);
  setEnabled(nextButton, false);
  var params = isCountryLeaderboard ? "?t=country" : "";

  // Update page URL (if not first fetch)
  if (pushState) {
    window.history.pushState({}, null, stateBase + page + params);
  }

  var xhr = new XMLHttpRequest();
  var table = document.getElementById(tableId);
  xhr.onerror = function() {
    // Reenable next button
    setEnabled(nextButton, true);

    // Reenable back button as long as we're not on page 1
    if (page != 1) {
      setEnabled(backButton, true);
    }

    // Clear out old results
    clearEntries(table);

    // Display an error message
    displayMessage(table, errorMessage);
  }
  xhr.onload = function() {
    // Reenable back button as long as we're not on page 1
    if (page != 1) {
      setEnabled(backButton, true);
    }

    var parsedMessage = JSON.parse(xhr.responseText);
    var entries = entriesFromMessage(parsedMessage);
    if (!Array.isArray(entries)) {
      entries = [];
    }

    // Clear out old results
    clearEntries(table);

    for (var i = 0; i < entries.length; i++) {
      // Build the new row
      var entry = entries[i];
      table.appendChild(rowFromEntry(entry));
    }

    // If there are no results, add special entry
    if (entries.length == 0) {
      displayMessage(table, noMoreMessage);
    } else {
      // Reenable next button if there are still entries
      // and pad out list to minEntries
      setEnabled(nextButton, true);
      if (entries.length < minEntries) {
        fillEntries(table, minEntries - entries.length);
      }
    }

  }
  xhr.open("GET", requestBase + page + params);
  xhr.send();
}

// Back button pressed
window.addEventListener("popstate", function(e) {
  fetchPage(getPageNumber(), false);
});

// Set up handlers for next and back
nextButton.addEventListener("click", function() {
  fetchPage(getPageNumber() + 1, true);
});

backButton.addEventListener("click", function() {
  fetchPage(getPageNumber() - 1, true);
});

// Load initial results
fetchPage(getPageNumber(), false);
})();
