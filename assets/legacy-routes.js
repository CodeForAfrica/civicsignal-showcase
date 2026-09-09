// Old React routes belong to the existing portal; ordinary page anchors stay here.
(function () {
  function forwardLegacyRoute() {
    if (window.location.hash.startsWith('#/')) {
      window.location.replace('https://tools.civicsignal.africa/' +
        window.location.search + window.location.hash);
    }
  }
  forwardLegacyRoute();
  window.addEventListener('hashchange', forwardLegacyRoute);
}());
