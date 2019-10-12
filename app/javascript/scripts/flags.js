(function() {
  // Initialize country picker                    
  var countryDropdown = document.getElementById("countrydropdown");

  var select = document.createElement("select");
  select.setAttribute("name", "country");
  select.setAttribute("id", "countryselect");
  countryDropdown.appendChild(select);

  for (var i = 0; i < allCountries.length; i++) {
    var countryName = document.createElement("span");
    countryName.innerText = allCountries[i][0];

    var row = document.createElement("option");
    row.appendChild(countryName);
    row.setAttribute("value", allCountries[i][1]);

    // Set none as default
    if (allCountries[i][1] == "none") {
      row.setAttribute("selected", "selected");
    }

    select.appendChild(row);
  }
})();
