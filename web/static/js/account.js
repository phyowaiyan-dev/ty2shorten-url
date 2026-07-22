(function () {
  function togglePassword(button) {
    var wrapper = button.closest(".password-input");
    if (!wrapper) return;
    var input = wrapper.querySelector("input");
    if (!input) return;

    var showing = input.type === "text";
    input.type = showing ? "password" : "text";
    button.setAttribute("aria-label", (showing ? "Show " : "Hide ") + passwordLabel(input.name));
    button.setAttribute("aria-pressed", showing ? "false" : "true");
  }

  function passwordLabel(name) {
    if (name === "current_password") return "current password";
    if (name === "confirm_password") return "confirm password";
    return "new password";
  }

  function setRule(rules, name, valid) {
    var rule = rules.querySelector('[data-rule="' + name + '"]');
    if (!rule) return;
    rule.dataset.valid = valid ? "true" : "false";
  }

  function updateRules() {
    var rules = document.querySelector("[data-password-rules]");
    var password = document.querySelector("[data-new-password]");
    var confirm = document.querySelector("[data-confirm-password]");
    if (!rules || !password || !confirm) return;

    setRule(rules, "length", password.value.length >= 10);
    setRule(rules, "match", password.value.length > 0 && password.value === confirm.value);
  }

  document.addEventListener("click", function (event) {
    var button = event.target.closest("[data-password-toggle]");
    if (button) togglePassword(button);
  });

  document.addEventListener("input", function (event) {
    if (event.target.matches("[data-new-password], [data-confirm-password]")) {
      updateRules();
    }
  });

  updateRules();
})();
