;(function () {
  "use strict";
  document.querySelectorAll("a.aws-console").forEach(function (element) {
    element.addEventListener("click", function (event) {
      event.stopPropagation();
      setTimeout(function () {
        window.open("https://signin.aws.amazon.com/oauth?Action=logout", event.target.target);
        setTimeout(function () {
          window.open(event.target.href, event.target.target);
        }, 1000); // 99th percentile latency for <https://signin.aws.amazon.com/oauth?Action=logout> is about 500ms
      }, 0);
    });
  });
})();
