(function () {
  const target = document.querySelector("[data-countdown]");
  if (!target) return;

  let remaining = Number.parseInt(target.dataset.countdown || target.textContent || "5", 10);
  if (!Number.isFinite(remaining) || remaining <= 0) return;

  const timer = window.setInterval(function () {
    remaining -= 1;
    target.textContent = String(Math.max(remaining, 0));
    if (remaining <= 0) {
      window.clearInterval(timer);
      window.location.assign("/");
    }
  }, 1000);
})();
