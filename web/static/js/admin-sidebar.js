(function () {
  const shell = document.querySelector("[data-sidebar-provider]");
  if (!shell) return;

  const triggers = Array.from(shell.querySelectorAll("[data-sidebar-trigger]"));
  const backdrop = shell.querySelector("[data-sidebar-backdrop]");

  function setMobileOpen(open) {
    if (open) {
      shell.dataset.mobileOpen = "true";
      return;
    }
    delete shell.dataset.mobileOpen;
  }

  triggers.forEach((trigger) => {
    trigger.addEventListener("click", () => {
      setMobileOpen(shell.dataset.mobileOpen !== "true");
    });
  });

  if (backdrop) {
    backdrop.addEventListener("click", () => setMobileOpen(false));
  }

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      setMobileOpen(false);
    }
  });
})();
