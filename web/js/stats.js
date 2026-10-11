// Keeps the counts on the home page current between builds and counts them up when the page opens. The build writes
// the same numbers into the HTML, so the page reads correctly without this script.
(() => {
  const DAY = 24 * 60 * 60 * 1000;

  function parse(value) {
    const [year, month, day] = (value || "").split("-").map(Number);
    return year && month && day ? new Date(Date.UTC(year, month - 1, day)) : null;
  }

  function today() {
    const now = new Date();
    return new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate()));
  }

  // Weekdays from start to end, both included.
  function workdays(start, end) {
    const days = Math.round((end - start) / DAY) + 1;
    const weeks = Math.floor(days / 7);
    let count = weeks * 5;
    for (let i = 0; i < days % 7; i++) {
      const weekday = new Date(start.getTime() + (weeks * 7 + i) * DAY).getUTCDay();
      if (weekday !== 0 && weekday !== 6) {
        count++;
      }
    }
    return count;
  }

  // Whole years from start to end.
  function years(start, end) {
    let count = end.getUTCFullYear() - start.getUTCFullYear();
    if (end.getUTCMonth() < start.getUTCMonth() || (end.getUTCMonth() === start.getUTCMonth() && end.getUTCDate() < start.getUTCDate())) {
      count--;
    }
    return count;
  }

  const DURATION = 2200;
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  // Counts element up from zero to value, then leaves the final text in place.
  function countUp(element, value, format) {
    if (reduceMotion) {
      element.textContent = format(value);
      return;
    }
    const started = performance.now();
    function frame(now) {
      const progress = Math.min((now - started) / DURATION, 1);
      const eased = 1 - Math.pow(1 - progress, 3);
      element.textContent = format(Math.round(value * eased));
      if (progress < 1) {
        requestAnimationFrame(frame);
      }
    }
    element.textContent = format(0);
    requestAnimationFrame(frame);
  }

  const end = today();
  document.querySelectorAll("[data-workdays-since]").forEach((element) => {
    const start = parse(element.dataset.workdaysSince);
    if (start && start <= end) {
      countUp(element, workdays(start, end), (n) => n.toLocaleString("en-US"));
    }
  });
  document.querySelectorAll("[data-years-since]").forEach((element) => {
    const start = parse(element.dataset.yearsSince);
    if (start && start <= end) {
      countUp(element, years(start, end), (n) => n + "+");
    }
  });
  document.querySelectorAll("[data-count-to]").forEach((element) => {
    const value = Number(element.dataset.countTo);
    if (value > 0) {
      countUp(element, value, (n) => n + "+");
    }
  });
})();
