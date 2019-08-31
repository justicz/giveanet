(function() {
  var retryButton = document.getElementById("retrybutton");
  var previewTable = document.getElementById("previewtable");

  if (retryButton) {
    retryButton.addEventListener("click", function() {
      // Disable submission buttons
      submissionButtons(false);
      var form = document.getElementById("retryform");
      var fd = new FormData(form);
      var xhr = new XMLHttpRequest();

      // Attempt repayment
      xhr.onload = function() {
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

      xhr.open("POST", "/pay/" + previewTable.dataset.messageToken + "/retry");
      xhr.send(fd);
    });
  }
})();
