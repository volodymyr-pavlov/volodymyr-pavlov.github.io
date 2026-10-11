/**
 * contact.js keeps the contact form from posting before hCaptcha is solved, since Web3Forms would reject the message.
 * It acts only on a page that holds the form of app/gen/templates/contact.html.
 */
(() => {
  const form = document.querySelector("form.contact");
  const note = document.getElementById("captcha-missing");
  if (!form || !note) {
    return;
  }

  form.addEventListener("submit", (event) => {
    const response = form.querySelector('[name="h-captcha-response"]');
    if (response && response.value) {
      return;
    }
    event.preventDefault();
    note.hidden = false;
  });
})();
