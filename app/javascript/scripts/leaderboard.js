var tableId = "leaderboardtable";
var requestBase = "/rankings/";
var stateBase = "/leaderboard/";
var errorMessage = "Error fetching leaderboard";
var noMoreMessage = "No more entries"
var minEntries = 11;
var rowFromEntry = function(entry) {
  return makeLeaderRow(entry.rank, entry.name, entry.link, entry.points, entry.country);
};
var entriesFromMessage = function(parsed) {
  return parsed.lr.l;
}
var isCountryLeaderboard = (new URL(location)).searchParams.get("t") === "country";

var leaderboardBy = document.getElementById("leaderboardby");
var leaderboardTitle = document.getElementById("leaderboardtitle");
var leaderboardNameType = document.getElementById("leaderboardnametype");
if (isCountryLeaderboard) {
  leaderboardBy.innerText = "by individual donation.";
  leaderboardTitle.innerText = "Leaderboard by Country";
  leaderboardNameType.innerText = "Country";
  leaderboardBy.href = "/leaderboard";
}
