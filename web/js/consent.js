/**
 * consent.js shows the consent bar for Google Analytics. It reads the measurement ID and the privacy page from the data
 * attributes of the script tag that scriptTag in app/gen/layout.go writes; without an ID there is nothing to ask about
 * and it does nothing. Nothing is requested from Google until the visitor allows it. Any element with the reset
 * attribute withdraws the choice, clears the analytics cookies and asks again.
 */
(() => {
  const STORAGE_KEY = "analytics-consent";
  const GRANTED = "granted";
  const DENIED = "denied";
  const BAR_ID = "consent"; // styled as #consent in css/main.css
  const RESET_ATTRIBUTE = "data-consent-reset"; // written by the footer in app/gen/templates/chrome.html
  const ANALYTICS_COOKIE_PREFIX = "_ga";

  // currentScript is null once the bundle runs as a module or after it has loaded, so the tag is also found by its data.
  const ownTag = document.currentScript || document.querySelector("script[data-ga4]");
  const measurementID = ownTag && ownTag.dataset.ga4;
  const privacyHref = ownTag && ownTag.dataset.privacy;
  if (!measurementID) {
    return;
  }

  // Storage may be blocked, so every access is allowed to fail.
  function withStorage(action, fallback) {
    try {
      return action(localStorage);
    } catch {
      return fallback;
    }
  }

  function storedChoice() {
    return withStorage((storage) => storage.getItem(STORAGE_KEY), null);
  }

  function storeChoice(choice) {
    withStorage((storage) => storage.setItem(STORAGE_KEY, choice));
  }

  function clearChoice() {
    withStorage((storage) => storage.removeItem(STORAGE_KEY));
  }

  function loadAnalytics() {
    window.dataLayer = window.dataLayer || [];
    window.gtag = function () {
      dataLayer.push(arguments);
    };
    // The visitor allowed analytics only, so the advertising flags of Consent Mode v2 are denied explicitly.
    gtag("consent", "default", {
      analytics_storage: "granted",
      ad_storage: "denied",
      ad_user_data: "denied",
      ad_personalization: "denied",
    });
    gtag("js", new Date());
    gtag("config", measurementID);
    const tag = document.createElement("script");
    tag.async = true;
    tag.src = "https://www.googletagmanager.com/gtag/js?id=" + encodeURIComponent(measurementID);
    document.head.appendChild(tag);
  }

  function deleteAnalyticsCookies() {
    document.cookie.split(";").forEach((cookie) => {
      const name = cookie.split("=")[0].trim();
      if (!name.startsWith(ANALYTICS_COOKIE_PREFIX)) {
        return;
      }
      document.cookie = name + "=; Max-Age=0; path=/";
      document.cookie = name + "=; Max-Age=0; path=/; domain=" + location.hostname;
    });
  }

  function element(tag, text, attributes = {}) {
    const node = document.createElement(tag);
    node.textContent = text;
    Object.assign(node, attributes);
    return node;
  }

  function consentBar() {
    const question = element("p", "May this site count your visit with Google Analytics? It sets cookies. ");
    question.appendChild(element("a", "Privacy", { href: privacyHref }));
    const bar = element("div", "", { id: BAR_ID });
    bar.append(
      question,
      element("button", "Allow", { type: "button", value: GRANTED }),
      element("button", "Decline", { type: "button", value: DENIED }),
    );
    return bar;
  }

  function askForConsent() {
    const bar = consentBar();
    bar.addEventListener("click", (event) => {
      if (event.target.tagName !== "BUTTON") {
        return;
      }
      storeChoice(event.target.value);
      bar.remove();
      if (event.target.value === GRANTED) {
        loadAnalytics();
      }
    });
    document.body.appendChild(bar);
  }

  function withdrawConsent(event) {
    event.preventDefault();
    clearChoice();
    deleteAnalyticsCookies();
    location.reload();
  }

  const choice = storedChoice();
  if (choice === GRANTED) {
    loadAnalytics();
  }
  if (!choice) {
    askForConsent();
  }
  document.querySelectorAll("[" + RESET_ATTRIBUTE + "]").forEach((resetter) => {
    resetter.addEventListener("click", withdrawConsent);
  });
})();
