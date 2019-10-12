var pageOne = document.getElementById("pageone");
var pageTwo = document.getElementById("pagetwo");
var pageThree = document.getElementById("pagethree");
var pageFour = document.getElementById("pagefour");
var pageFive = document.getElementById("pagefive");

var minNets = 1;
var maxNets = 25000;

// Update display text when net slider changes
var netSlider = document.getElementById("netslider");
var numNetsText = document.getElementById("numnetstext");
netSlider.addEventListener("input", function() {
  var numNets = parseInt(netSlider.value);
  var plural = numNets != 1;
  var dollars = (numNets * 2).toLocaleString();
  numNetsText.innerText = numNets + " net" + (plural ? "s" : "") + " - " + "$" + dollars;
});

// When the first page's next button is clicked, display the next page
// depending on whether or not they want a homepage message
var wantMsgCheckbox = document.getElementById("wantmsgcheckbox");
var pageOneNextButton = document.getElementById("pageonenextbutton");
pageOneNextButton.addEventListener("click", function() {
  if (!validatePageOne()) {
    return;
  }
  if (!wantMsgCheckbox.checked) {
    submitForm();
    return;
  }
  pageOne.style.display = "none";
  pageTwo.style.display = "block";
  window.scrollTo(0, 0);
});

// Display number entry field instead of slider
var toggleLotsNetsButton = document.getElementById("togglelotsnetsbutton");
var regNetsBox = document.getElementById("regnetsbox");
var lotsNetsBox = document.getElementById("lotsnetsbox");
var netQuantityType = document.getElementById("netquantitytype");
var moreFewer = document.getElementById("morefewer");
var sendingLots = false;
toggleLotsNetsButton.addEventListener("click", function(e) {
  sendingLots = !sendingLots;
  lotsNetsBox.style.display = sendingLots ? "block" : "none";
  regNetsBox.style.display = sendingLots ? "none" : "block";
  netQuantityType.value = sendingLots ? "lots" : "reg";
  moreFewer.innerText = sendingLots ? "75 nets or fewer" : "more than 75 nets";
  e.preventDefault();
});

var netNumField = document.getElementById("netnumfield");
var lotsNetsText = document.getElementById("lotsnetstext");
netNumField.addEventListener("input", function() {
  var fieldVal = netNumField.value;
  var numNets = parseInt(fieldVal);
  if (fieldVal.length == 0 || isNaN(numNets) || numNets < minNets) {
    numNets = 0;
  }

  if (numNets > maxNets) {
    numNets = maxNets;
    netNumField.value = numNets;
  }

  var plural = numNets != 1;
  var dollars = (numNets * 2).toLocaleString();
  lotsNetsText.innerText = "net" + (plural ? "s" : "") + " - " + "$" + dollars;
});

// Get the next sibling input element
function getNextInput(e) {
  while (e = e.nextElementSibling) {
    if (e.nodeName.toLowerCase() === "input") {
      return e
    }
  }
}

function validatePageOne() {
  // Clear existing alerts
  clearAlerts();

  // Check that number of nets is reasonable
  var numNets = getPreviewNumNets();
  if (isNaN(numNets) || numNets < minNets || numNets > maxNets) {
    addAlert("Please select between " + minNets + " and " + maxNets + " nets");
    window.scrollTo(0, 0);
    return false;
  }

  return true;
}

function validatePageTwo() {
  // Clear existing alerts
  clearAlerts();

  // Radio buttons
  var twitterRadio = document.getElementById("twitterradio");
  var instagramRadio = document.getElementById("instagramradio");
  var customLinkRadio = document.getElementById("customlinkradio");

  // Validate custom/social links
  var invalid = false;
  switch (true) {
    case twitterRadio.checked:
      var username = getNextInput(twitterRadio).value || "";
      username = username.replace("@", "");
      if (username.length == 0) {
        addAlert("Twitter username must be at least 1 character");
        invalid = true;
      }
      break;
    case instagramRadio.checked:
      var username = getNextInput(instagramRadio).value || "";
      username = username.replace("@", "");
      if (username.length == 0) {
        addAlert("Instagram username must be at least 1 character");
        invalid = true;
      }
      break;
    case customLinkRadio.checked:
      var customLinkText = getNextInput(customLinkRadio);
      var link = customLinkText.value;
      if (!((link.substring(0, 8) === "https://") ||
            (link.substring(0, 7) === "http://"))) {
        addAlert("Custom link must begin with https:// or http://");
        customLinkText.focus();
        invalid = true;
      }
      break;
  }

  // Scroll to see alerts
  if (invalid) {
    window.scrollTo(0, 0);
  }

  return !invalid;
}
// Update whether or not social text box is enabled/disabled based on selection
var linkCheckboxes = document.getElementsByClassName("linktype");
for (var i = 0; i < linkCheckboxes.length; i++) {
  linkCheckboxes[i].addEventListener("input", function() {
    for (var i = 0; i < linkCheckboxes.length; i++) {
      var elem = linkCheckboxes[i];
      var n = getNextInput(elem);
      if (n) {
        n.disabled = !elem.checked;
        if (elem.checked) {
          n.focus();
        }
      }
    }
  });
}

// Prevent typing newlines in message on client
var pageTwoTextArea = document.getElementById("msgarea");
pageTwoTextArea.addEventListener("keydown", function(e) {
  // Prevent newline
  if (e.keyCode == 13) {
    e.preventDefault();
    return false;
  }
});

// Go back to first page
var pageTwoBackButton = document.getElementById("pagetwobackbutton");
pageTwoBackButton.addEventListener("click", function() {
  clearAlerts();
  pageOne.style.display = "block";
  pageTwo.style.display = "none";
  window.scrollTo(0, 0);
});

// Go to third page
var drawingPageInitialized = false;
var pageTwoNextButton = document.getElementById("pagetwonextbutton");
pageTwoNextButton.addEventListener("click", function() {
  if (!validatePageTwo()) {
    return;
  }
  pageTwo.style.display = "none";
  pageThree.style.display = "block";
  // We need do this once page three is visible due to some kind of browser
  // bug where the picker doesn't work if it's made visible after
  // initialization
  if (!drawingPageInitialized) {
    initializeDrawingPage();
    drawingPageInitialized = true;
  }
  window.scrollTo(0, 0);
});

// Go back to second page
var pageThreeBackButton = document.getElementById("pagethreebackbutton");
pageThreeBackButton.addEventListener("click", function() {
  pageTwo.style.display = "block";
  pageThree.style.display = "none";
  window.scrollTo(0, 0);
});

// Go to fourth page
var pageThreeNextButton = document.getElementById("pagethreenextbutton");
pageThreeNextButton.addEventListener("click", function() {
  // Display next page
  pageFour.style.display = "block";
  pageThree.style.display = "none";

  // Scroll to top
  window.scrollTo(0, 0);
});

// Go back to third page
var pageFourBackButton = document.getElementById("pagefourbackbutton");
pageFourBackButton.addEventListener("click", function() {
  pageThree.style.display = "block";
  pageFour.style.display = "none";
  window.scrollTo(0, 0);
});

// Go to fifth page
var pageFourNextButton = document.getElementById("pagefournextbutton");
pageFourNextButton.addEventListener("click", function() {
  // Get details for preview
  var iconData = sampleCanvas();
  var linkURL = getLinkPreviewURL();
  var nets = getPreviewNumNets();
  var name = document.getElementById("displaynameinput").value;
  var message = document.getElementById("msgarea").value;
  var countryDropdown = document.getElementById("countryselect");
  var country = countryDropdown.options[countryDropdown.selectedIndex].value;

  // Generate preview row
  var previewRow = makeMessageRow(iconData, name, linkURL, nets, message, country);
  var previewTable = document.getElementById("previewtable");

  // Clear out old preview
  var child = previewTable.firstChild;
  while (child) {
    previewTable.removeChild(child);
    child = previewTable.firstChild;
  }

  // Fill preview row
  previewTable.appendChild(previewRow);

  // Display next page
  pageFive.style.display = "block";
  pageFour.style.display = "none";

  // Scroll to top
  window.scrollTo(0, 0);
});


function getLinkPreviewURL() {
  var nowhereRadio = document.getElementById("nowhereradio");
  var twitterRadio = document.getElementById("twitterradio");
  var instagramRadio = document.getElementById("instagramradio");
  var customLinkRadio = document.getElementById("customlinkradio");

  switch (true) {
    case nowhereRadio.checked:
      return null;
    case twitterRadio.checked:
      var username = getNextInput(twitterRadio).value || "";
      username = username.replace("@", "");
      return "https://twitter.com/" + username;
    case instagramRadio.checked:
      var username = getNextInput(instagramRadio).value || "";
      username = username.replace("@", "");
      return "https://www.instagram.com/" + username;
    case customLinkRadio.checked:
      return getNextInput(customLinkRadio).value;
  }
}

function getPreviewNumNets() {
  if (sendingLots) {
    return parseInt(netNumField.value);
  }
  return parseInt(netSlider.value);
}

// Go back to fourth page
var pageFiveBackButton = document.getElementById("pagefivebackbutton");
pageFiveBackButton.addEventListener("click", function() {
  pageFour.style.display = "block";
  pageFive.style.display = "none";
  window.scrollTo(0, 0);
});

// Submit form
var pageFiveNextButton = document.getElementById("pagefivenextbutton");
pageFiveNextButton.addEventListener("click", function() {
  submitForm();
});

function submitForm() {
  // Sample the canvas to be submitted as part of the form
  document.getElementById("canvasdata").value = sampleCanvas();

  // Disable submission buttons
  submissionButtons(false);

  var form = document.getElementById("netform");
  var fd = new FormData(form);
  var xhr = new XMLHttpRequest();

  // If not displaying a personalized message, clear all fields except number
  // of nets and CSRF token
  if (!wantMsgCheckbox.checked) {
    var keys = Array.from(fd.keys());
    var keep = { "netnumbox": true, "netslider": true,
                 "netquantitytype": true, "wantmsg": true, "tok": true };
    for (var i = 0; i < keys.length; i++) {
      var key = keys[i];
      if (!keep[key]) {
        fd.delete(key);
      }
    }
  }

  xhr.onload = function() {
    // Parse response
    var resp = JSON.parse(xhr.responseText);

    // Check for and display errors
    if (resp.errors && resp.errors.length > 0) {
      clearAlerts();
      for (var i = 0; i < resp.errors.length; i++) {
        addAlert(resp.errors[i]);
      }
      window.scrollTo(0, 0);
      // Reenable submission buttons
      submissionButtons(true);
      return;
    }

    // Check for dev environment
    if (resp.stripe && resp.stripe.startsWith("mn_test")) {
      clearAlerts();
      submissionButtons(true);
      prompt("dev token", resp.stripe);
      return;
    }

    // Success! Redirect to stripe
    if (resp.stripe) {
      stripe.redirectToCheckout({
        sessionId: resp.stripe
      }).then(function (result) {
        clearAlerts();
        addAlert("Couldn't redirect to Stripe. Please check your internet connection and try again.");
        window.scrollTo(0, 0);
        // Reenable submission buttons
        submissionButtons(true);
      });
    }
  };

  xhr.onerror = function() {
    submissionButtons(true);
    clearAlerts();
    addAlert("Couldn't submit form. Please check your internet connection and try again.");
    window.scrollTo(0, 0);
  };

  xhr.open("POST", "/pay");
  xhr.send(fd);
}
