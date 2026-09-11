// Consent bar for Google Analytics, included by consentScript in cmd/page.go.
// Nothing is requested from Google until the visitor allows it.
// Any element with the reset attribute withdraws the choice, clears the analytics cookies and asks again.
(function () {
  var STORAGE_KEY = "analytics-consent";
  var GRANTED = "granted";
  var DENIED = "denied";
  var BAR_ID = "consent"; // styled as div#consent in css/site.css
  var RESET_ATTRIBUTE = "data-consent-reset"; // consentResetAttr in cmd/names.go
  var ANALYTICS_COOKIE_PREFIX = "_ga";

  var measurementID = document.currentScript.dataset.ga4;
  var privacyHref = document.currentScript.dataset.privacy;

  function storedChoice() {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch (e) {
      return null;
    }
  }

  function storeChoice(choice) {
    try {
      localStorage.setItem(STORAGE_KEY, choice);
    } catch (e) {}
  }

  function clearChoice() {
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (e) {}
  }

  function loadAnalytics() {
    window.dataLayer = window.dataLayer || [];
    window.gtag = function () {
      dataLayer.push(arguments);
    };
    gtag("js", new Date());
    gtag("config", measurementID);
    var tag = document.createElement("script");
    tag.async = true;
    tag.src = "https://www.googletagmanager.com/gtag/js?id=" + encodeURIComponent(measurementID);
    document.head.appendChild(tag);
  }

  function deleteAnalyticsCookies() {
    document.cookie.split(";").forEach(function (cookie) {
      var name = cookie.split("=")[0].trim();
      if (name.indexOf(ANALYTICS_COOKIE_PREFIX) !== 0) return;
      document.cookie = name + "=; Max-Age=0; path=/";
      document.cookie = name + "=; Max-Age=0; path=/; domain=" + location.hostname;
    });
  }

  function askForConsent() {
    var bar = document.createElement("div");
    bar.id = BAR_ID;
    bar.innerHTML =
      "<p>May this site count your visit with Google Analytics? It sets cookies. <a>Privacy</a></p>" +
      '<button type="button" value="' + GRANTED + '">Allow</button>' +
      '<button type="button" value="' + DENIED + '">Decline</button>';
    bar.querySelector("a").href = privacyHref;
    bar.addEventListener("click", function (event) {
      if (event.target.tagName !== "BUTTON") return;
      storeChoice(event.target.value);
      bar.remove();
      if (event.target.value === GRANTED) loadAnalytics();
    });
    document.body.appendChild(bar);
  }

  function withdrawConsent(event) {
    event.preventDefault();
    clearChoice();
    deleteAnalyticsCookies();
    location.reload();
  }

  var choice = storedChoice();
  if (choice === GRANTED) loadAnalytics();
  if (!choice) askForConsent();
  document.querySelectorAll("[" + RESET_ATTRIBUTE + "]").forEach(function (element) {
    element.addEventListener("click", withdrawConsent);
  });
})();
