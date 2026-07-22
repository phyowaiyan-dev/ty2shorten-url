(function () {
  const forms = document.querySelectorAll("[data-settings-form], [data-branding-form]");
  if (!forms.length) {
    return;
  }

  function updateCharacterCounter(input) {
    const target = document.getElementById(input.dataset.countTarget);
    if (!target) {
      return;
    }
    const max = Number(input.getAttribute("maxlength")) || 0;
    const count = Array.from(input.value || "").length;
    target.textContent = max > 0 ? `${count}/${max}` : String(count);
    target.dataset.state = max > 0 && count > max ? "invalid" : "valid";
  }

  function wordCount(value) {
    const words = value.trim().split(/\s+/).filter(Boolean);
    return words.length;
  }

  function updateWordCounter(input) {
    const target = document.getElementById(input.dataset.wordTarget);
    if (!target) {
      return;
    }
    const min = Number(input.dataset.wordMin) || 0;
    const max = Number(input.dataset.wordMax) || 0;
    const count = wordCount(input.value || "");
    target.textContent = `${count}/${min}-${max} words`;
    target.dataset.state = count >= min && (max === 0 || count <= max) ? "valid" : "invalid";
  }

  function validStoreURL(value, kind) {
    try {
      const parsed = new URL(value);
      if (kind === "google-play") {
        return parsed.protocol === "https:" && parsed.hostname === "play.google.com" && parsed.pathname.startsWith("/store/apps/");
      }
      if (kind === "apple-store") {
        return parsed.protocol === "https:" && parsed.hostname === "apps.apple.com" && parsed.pathname.includes("/app/");
      }
    } catch (_) {
      return false;
    }
    return false;
  }

  function updateStoreStatus(input) {
    const status = input.form.querySelector(`[data-url-status="${input.dataset.urlKind}"]`);
    if (!status) {
      return;
    }
    if (!input.value.trim()) {
      status.textContent = "Waiting for URL";
      status.dataset.state = "neutral";
      return;
    }
    const isValid = validStoreURL(input.value.trim(), input.dataset.urlKind);
    status.textContent = isValid ? "Valid store URL" : "Check store URL";
    status.dataset.state = isValid ? "valid" : "invalid";
  }

  forms.forEach((form) => {
    form.querySelectorAll("[data-count-chars]").forEach((input) => {
      updateCharacterCounter(input);
      input.addEventListener("input", () => updateCharacterCounter(input));
    });

    form.querySelectorAll("[data-word-count]").forEach((input) => {
      updateWordCounter(input);
      input.addEventListener("input", () => updateWordCounter(input));
    });

    form.querySelectorAll("[data-url-kind]").forEach((input) => {
      updateStoreStatus(input);
      input.addEventListener("input", () => updateStoreStatus(input));
      input.addEventListener("blur", () => updateStoreStatus(input));
    });

  });
})();
