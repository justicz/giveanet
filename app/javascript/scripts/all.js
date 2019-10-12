var tableId = "msgtable";
var requestBase = "/messages/";
var stateBase = "/all/";
var errorMessage = "Error fetching messages";
var noMoreMessage = "No more messages"
var minEntries = 7;
var rowFromEntry = function(entry) {
  return makeMessageRow(entry.icon, entry.name, entry.link, entry.nets, entry.msg, entry.country);
};
var entriesFromMessage = function(parsed) {
  return parsed.qr.q;
}
