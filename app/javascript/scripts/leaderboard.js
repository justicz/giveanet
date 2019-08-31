var tableId = "leaderboardtable";
var requestBase = "/rankings/";
var stateBase = "/leaderboard/";
var errorMessage = "Error fetching leaderboard";
var noMoreMessage = "No more entries"
var minEntries = 11;
var rowFromEntry = function(entry) {
  return makeLeaderRow(entry.rank, entry.name, entry.link, entry.points);
};
var entriesFromMessage = function(parsed) {
  return parsed.lr.l;
}
