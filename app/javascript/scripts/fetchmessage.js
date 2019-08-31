(function() {
  var xhr = new XMLHttpRequest();
  var previewTable = document.getElementById("previewtable");
  xhr.onload = function() {
    var msg = JSON.parse(xhr.responseText).card;

    // Generate preview row
    var previewRow = makeMessageRow(msg.icon, msg.name, msg.link, msg.nets, msg.msg);

    // Clear out old preview
    var child = previewTable.firstChild;
    while (child) {
      previewTable.removeChild(child);
      child = previewTable.firstChild;
    }

    // Fill preview row
    previewTable.appendChild(previewRow);
  }
  xhr.open("GET", "/card/" + previewTable.dataset.messageToken + "/data");
  xhr.send();
})();
