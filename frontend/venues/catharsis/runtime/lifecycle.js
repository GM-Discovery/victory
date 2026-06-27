(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryCatharsisLifecycle = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createLifecycleBag() {
    const disposers = [];
    let disposed = false;

    function track(dispose) {
      if (disposed) {
        if (typeof dispose === "function") {
          try {
            dispose();
          } catch (error) {
            console.warn("lifecycle dispose failed", error);
          }
        }
        return () => {};
      }
      const disposer = typeof dispose === "function" ? dispose : () => {};
      disposers.push(disposer);
      return disposer;
    }

    function listen(target, type, handler, options) {
      if (!target || typeof target.addEventListener !== "function") {
        return () => {};
      }
      target.addEventListener(type, handler, options);
      return track(() => {
        target.removeEventListener(type, handler, options);
      });
    }

    function timeout(callback, delay) {
      const id = window.setTimeout(callback, delay);
      return track(() => window.clearTimeout(id));
    }

    function interval(callback, delay) {
      const id = window.setInterval(callback, delay);
      return track(() => window.clearInterval(id));
    }

    function animationFrame(callback) {
      const id = window.requestAnimationFrame(callback);
      return track(() => window.cancelAnimationFrame(id));
    }

    function dispose() {
      if (disposed) return;
      disposed = true;
      while (disposers.length) {
        const disposer = disposers.pop();
        try {
          disposer();
        } catch (error) {
          console.warn("lifecycle cleanup failed", error);
        }
      }
    }

    return {
      track,
      listen,
      timeout,
      interval,
      animationFrame,
      dispose,
    };
  }

  return {
    createLifecycleBag,
  };
});
