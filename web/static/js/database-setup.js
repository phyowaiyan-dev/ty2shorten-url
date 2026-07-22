(function () {
  const form = document.querySelector("[data-database-setup-form]");
  if (!form) return;

  const driverSelect = form.querySelector("[data-driver-select]");
  const sections = Array.from(form.querySelectorAll("[data-driver-section]"));
  if (!driverSelect || sections.length === 0) return;

  function syncDriverSections() {
    const driver = driverSelect.value;
    sections.forEach((section) => {
      const active = section.dataset.driverSection === driver;
      section.hidden = !active;
      section.querySelectorAll("input, select, textarea").forEach((field) => {
        field.disabled = !active;
      });
    });
  }

  driverSelect.addEventListener("change", syncDriverSections);
  syncDriverSections();
})();
