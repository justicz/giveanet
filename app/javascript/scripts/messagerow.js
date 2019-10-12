function generateIconURLAndBorder(iconData) {
  var dataDimensions = [10, 10];
  var blockDimensions = [10, 10];
  var requiredLength = dataDimensions[0] * dataDimensions[1] * 3;

  // Decode icon data if it exists available
  var data = ""
  if (iconData) {
    data = atob(iconData);
  }

  // If we weren't passed valid image data, return a transparent image and a
  // transparent border
  if (data.length != requiredLength) {
    var url = "data:image/gif;base64,R0lGODlhAQABAA" +
              "AAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==";
    var border = "rgba(255,255,255,0)";
    return [url, border, false];
  }

  // Create a temporary canvas
  var tmp = document.createElement("canvas");
  tmp.width = dataDimensions[0] * blockDimensions[0];
  tmp.height = dataDimensions[1] * blockDimensions[1];

  // Fill it in with the data
  var ctx = tmp.getContext("2d");
  var tots = [0, 0, 0];
  for (var y = 0; y < dataDimensions[1]; y++) {
    for (var x = 0; x < dataDimensions[0]; x++) {
      var off = (3 * y * dataDimensions[1]) + (3 * x);
      var r = data[off + 0].charCodeAt(0);
      var g = data[off + 1].charCodeAt(0);
      var b = data[off + 2].charCodeAt(0);
      ctx.beginPath();
      ctx.rect(x * blockDimensions[0], y * blockDimensions[1],
               blockDimensions[0], blockDimensions[1]);
      ctx.fillStyle = "rgb(" + r + ", " + g + ", " + b + ")";
      ctx.fill();
      // Tally color so we can take the average for the border
      tots[0] += r;
      tots[1] += g;
      tots[2] += b;
    }
  }

  var samples = dataDimensions[0] * dataDimensions[1];
  var avgs = [0, 0, 0];
  var darken = 0.9;
  avgs[0] = darken * tots[0] / samples;
  avgs[1] = darken * tots[1] / samples;
  avgs[2] = darken * tots[2] / samples;
  var border = "rgb(" + avgs[0] + ", " + avgs[1] + ", " + avgs[2] + ")";
  var url = tmp.toDataURL();
  return [url, border, true];
}

function makeMessageRow(iconData, name, linkURL, nets, message, country) {
  var newRow = document.createElement("tr");

  var imgCol = document.createElement("td");
  var icon = document.createElement("img");
  imgCol.appendChild(icon);

  var nameCol = document.createElement("td");
  var nameLink = document.createElement("a");
  var actionPhrase = document.createElement("span");
  nameCol.appendChild(nameLink);
  nameCol.appendChild(actionPhrase);

  var flag;
  if (country && country != "none") {
    flag = document.createElement("div");
    flag.classList.add("iti__flag");
    flag.classList.add("iti__" + country);
    flag.setAttribute("title", countryCodes[country]);
  }

  var quoteCol = document.createElement("td");
  quoteCol.classList.add("quoted");

  // Generate icon and icon border and conditionally append col
  var nameColspan = 1;
  var iconURLBorder = generateIconURLAndBorder(iconData);
  var hasIcon = iconURLBorder[2];
  if (hasIcon) {
    newRow.appendChild(imgCol);
  } else {
    nameColspan += 1;
  }

  // Always add name column
  newRow.appendChild(nameCol);

  // Conditionally append message column
  if (!message || message.length === 0) {
    nameColspan += 1;
  } else {
    newRow.appendChild(quoteCol);
  }

  // Make name take up appropriate width
  nameCol.setAttribute("colspan", nameColspan);

  // Fill in icon
  icon.src = iconURLBorder[0];
  icon.style.borderColor = iconURLBorder[1];

  // Fill in name
  nameLink.innerText = name || "Anonymous";

  // Fill in link
  nameLink.classList.toggle("unclickable", !linkURL);
  nameLink.setAttribute("target", "_blank");
  nameLink.href = linkURL;

  // Fill the number of nets sent action phrase
  var numNetsWithCommas = nets.toLocaleString();
  var plural = nets != 1;
  var action = "sent ";
  action += numNetsWithCommas + " net" + (plural ? "s" : "") + (flag ? " from" : "");
  actionPhrase.innerText = action;
  if (flag) {
    actionPhrase.appendChild(flag);
  }

  // Fill in block quote
  quoteCol.innerText = message;

  return newRow;
}

function makeLeaderRow(rank, name, linkURL, score, country) {
  var newRow = document.createElement("tr");
  var rankCol = document.createElement("td");
  var nameCol = document.createElement("td");
  var nameLink = document.createElement("a");
  nameCol.appendChild(nameLink);

  var scoreCol = document.createElement("td");

  newRow.appendChild(rankCol);
  newRow.appendChild(nameCol);
  newRow.appendChild(scoreCol);

  // Fill in rank
  var rankWithCommas = rank.toLocaleString();
  rankCol.innerText = rankWithCommas;

  // Fill in name
  nameLink.innerText = name || "Anonymous";

  // Fill in link
  nameLink.classList.toggle("unclickable", !linkURL);
  nameLink.href = linkURL;

  // Potentially fill in country
  if (country && country != "none") {
    var flag = document.createElement("div");
    flag.classList.add("iti__flag");
    flag.classList.add("iti__" + country);
    flag.setAttribute("title", countryCodes[country]);
    nameCol.appendChild(flag);
  }

  // Fill in score
  var scoreWithCommas = score.toLocaleString();
  scoreCol.innerText = scoreWithCommas;

  return newRow;
}
