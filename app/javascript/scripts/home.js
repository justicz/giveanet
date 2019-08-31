function Messages(table) {
  this.table = table;
  this.maxIdx = 0;
}

Messages.prototype.push = function(entries, animated) {
  if (!Array.isArray(entries)) {
    return;
  }

  // Inserting from the top, so iterate backwards
  for (var i = entries.length - 1; i >= 0; i--) {
    var entry = entries[i];

    // Don't reinsert old entries (they should come sorted by timeline idx)
    if (entry.idx <= this.maxIdx) {
      continue;
    }
    this.maxIdx = entry.idx;

    // Build the new row
    var newRow = makeMessageRow(entry.icon, entry.name, entry.link, entry.nets, entry.msg);
    newRow.classList.toggle("noanimate", !animated);

    // Add the row to the table
    this.table.insertBefore(newRow, this.table.firstChild);
  }
};

function Leaderboard(table) {
  this.table = table;
}

Leaderboard.prototype.update = function(entries) {
  if (!Array.isArray(entries)) {
    return;
  }

  clearTable(this.table);

  for (var i = 0; i < entries.length; i++) {
    var entry = entries[i];

    // Build the new row
    var newRow = makeLeaderRow(entry.rank, entry.name, entry.link, entry.points);
    // Add the row to the table
    this.table.appendChild(newRow);
  }
};

function clearTable(t) {
  var child = t.firstChild;
  while (child) {
    t.removeChild(child);
    child = t.firstChild;
  }
}

(function init() {
  var counter = document.getElementById("counter");
  var wrapper = document.getElementById("wrapper");
  var goalType = document.getElementById("goaltype");
  var plural = document.getElementById("plural");
  var goals = [{ num: 10000,   name: "Base Goal" },
               { num: 50000,   name: "Stretch Goal" },
               { num: 100000,  name: "Mega Goal" },
               { num: 500000,  name: "Ultra Goal" },
               { num: 1000000, name: "Million Goal" }];

  // Websocket request URL
  var wsurl = wrapper.dataset.wsOrigin;
  wsurl += "/ws";

  function connect() {
    var ws = new WebSocket(wsurl);
    var firstMessages = { "q": true, "l": true };

    var msgTable = document.getElementById("msgtable");
    var messages = new Messages(msgTable);

    var leaderboardTable = document.getElementById("leaderboardtable");
    var leaderboard = new Leaderboard(leaderboardTable);

    var pong = JSON.stringify({"type": "p"});

    ws.onmessage = function (e) {
      var msg = e.data;
      var parsedMessage;
      try {
        parsedMessage = JSON.parse(msg);
      } catch (err) {
        return
      }

      switch(parsedMessage.type) {
        // ping
        case "p":
          ws.send(pong);
          break;
        // nets given
        case "n":
          var netsGiven = parsedMessage.nr.ng;
          var goal = null;
          for (var i = 0; i < goals.length; i++) {
            goal = goals[i];
            if (netsGiven < goals[i].num) {
              break;
            }
          }
          goalType.innerText = goal.name;
          var remaining = Math.max(goal.num - netsGiven, 0);
          counter.innerText = remaining;
          if (remaining == 1) {
            plural.style.display = "none";
          } else {
            plural.style.display = "inline";
          }
          break;
        // queue entries
        case "q":
          if (firstMessages["q"]) { clearTable(msgTable); }
          messages.push(parsedMessage.qr.q, !firstMessages["q"]);
          firstMessages["q"] = false;
          break;
        // leaderboard entries
        case "l":
          if (firstMessages["l"]) { clearTable(leaderboardTable); }
          leaderboard.update(parsedMessage.lr.l);
          firstMessages["l"] = false;
          break;
      }
    }

    // Try to reconnect if we get dropped for some reason
    ws.onclose = function(e) {
      setTimeout(connect, 4000);
    }
  }

  // Connect to websocket
  connect();
})();
