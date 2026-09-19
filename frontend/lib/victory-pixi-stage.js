(function () {
  function resolveMount(target) {
    if (!target) return null;
    if (typeof target === "string") {
      return document.querySelector(target);
    }
    if (target instanceof Element || target instanceof HTMLCanvasElement) {
      return target;
    }
    if (window.PIXI && target instanceof PIXI.Container) {
      return target;
    }
    return null;
  }

  function clamp(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function fitSpriteToBounds(sprite, width, height, options = {}) {
    if (!sprite || !sprite.texture) return;
    const texture = sprite.texture;
    const sourceWidth = Number(texture.orig?.width || texture.width || 0);
    const sourceHeight = Number(texture.orig?.height || texture.height || 0);
    if (!sourceWidth || !sourceHeight || !width || !height) return;

    const fit = String(options.fit || "contain").trim() || "contain";
    const originX = Number(options.x || 0);
    const originY = Number(options.y || 0);
    const cropX = clamp(options.cropX ?? options.crop_x, 0, 1, 0.5);
    const cropY = clamp(options.cropY ?? options.crop_y, 0, 1, 0.5);

    const userScale = clamp(options.scale, 0.25, 4, 1);
    const scaleX = width / sourceWidth;
    const scaleY = height / sourceHeight;
    const fitScale = fit === "cover" ? Math.max(scaleX, scaleY) : Math.min(scaleX, scaleY);
    const scale = fitScale * userScale;

    sprite.scale.set(scale);
    const renderedWidth = sourceWidth * scale;
    const renderedHeight = sourceHeight * scale;
    const extraWidth = Math.max(0, renderedWidth - width);
    const extraHeight = Math.max(0, renderedHeight - height);
    sprite.position.set(
      originX + (width / 2) + ((0.5 - cropX) * extraWidth),
      originY + (height / 2) + ((0.5 - cropY) * extraHeight),
    );
  }

  function mountBackdrop(options = {}) {
    if (!window.PIXI) {
      return null;
    }

    const mount = resolveMount(options.mount || options.container);
    const imageUrl = String(options.imageUrl || "").trim();
    const fit = String(options.fit || "contain").trim() || "contain";
    if (!mount || !imageUrl) {
      return null;
    }

    const alpha = clamp(options.alpha, 0, 1, 1);
    const backgroundColor = options.backgroundColor ?? 0x000000;
    const app = mount instanceof Element ? new PIXI.Application({
      width: Math.max(1, clamp(options.width, 1, 4096, mount.clientWidth || 1)),
      height: Math.max(1, clamp(options.height, 1, 4096, mount.clientHeight || 1)),
      backgroundAlpha: 0,
      backgroundColor,
      antialias: true,
      resolution: window.devicePixelRatio || 1,
      autoDensity: true,
    }) : null;

    const host = app ? app.stage : mount;
    let sprite = null;
    let disposed = false;
    let resizeHandle = null;
    let lastWidth = Math.max(1, Number(options.width || 0));
    let lastHeight = Math.max(1, Number(options.height || 0));

    const state = {
      app,
      host,
      sprite: null,
      resize(width, height) {
        if (disposed) return;
        const nextWidth = Math.max(1, Number(width || lastWidth || 0));
        const nextHeight = Math.max(1, Number(height || lastHeight || 0));
        lastWidth = nextWidth;
        lastHeight = nextHeight;
        if (app) {
          app.renderer.resize(nextWidth, nextHeight);
        }
        fitSpriteToBounds(sprite, nextWidth, nextHeight, { fit });
      },
      destroy() {
        disposed = true;
        if (resizeHandle) {
          window.removeEventListener("resize", resizeHandle);
        }
        if (sprite && sprite.parent) {
          sprite.parent.removeChild(sprite);
        }
        if (app) {
          app.destroy(true, { children: true, texture: false, baseTexture: false });
        }
      },
    };

    const attachSprite = (texture) => {
      if (disposed) return;
      sprite = new PIXI.Sprite(texture);
      sprite.alpha = alpha;
      sprite.anchor.set(0.5);
      sprite.eventMode = "none";
      sprite.interactive = false;
      state.sprite = sprite;
      if (host && host.addChild) {
        host.addChild(sprite);
      } else if (app) {
        app.stage.addChild(sprite);
      }
      const bounds = mount instanceof Element
        ? { width: mount.clientWidth, height: mount.clientHeight }
        : { width: lastWidth, height: lastHeight };
      state.resize(bounds.width, bounds.height);
    };

    const loadTexture = () =>
      window.PIXI.Assets?.load
        ? window.PIXI.Assets.load(imageUrl)
        : Promise.resolve(window.PIXI.Texture.from(imageUrl));

    // Kernel 101 (101-10): a decorative backdrop failing to load is
    // genuinely low-stakes (the stage still works, nothing is lost) and
    // stays a console.warn on purpose -- a visible error/toast for a
    // missing background image would be a worse experience than the
    // blank backdrop itself. The one real, cheap improvement: most real
    // failures of a same-origin image load are a transient network
    // blip, not a permanently broken URL, so one retry after a short
    // delay before giving up meaningfully reduces how often this shows
    // up at all, with no new UI needed.
    loadTexture()
      .then((texture) => {
        const resolvedTexture = texture?.texture || texture || window.PIXI.Texture.from(imageUrl);
        attachSprite(resolvedTexture);
      })
      .catch(() => {
        if (disposed) return;
        new Promise((resolve) => setTimeout(resolve, 750))
          .then(loadTexture)
          .then((texture) => {
            if (disposed) return;
            const resolvedTexture = texture?.texture || texture || window.PIXI.Texture.from(imageUrl);
            attachSprite(resolvedTexture);
          })
          .catch((error) => {
            console.warn("VictoryPixiStage backdrop load failed after retry", error);
          });
      });

    if (app && mount instanceof Element) {
      mount.innerHTML = "";
      mount.appendChild(app.view);
      resizeHandle = () => state.resize(mount.clientWidth, mount.clientHeight);
      window.addEventListener("resize", resizeHandle, { passive: true });
    }

    return state;
  }

  window.VictoryPixiStage = {
    fitSpriteToBounds,
    mountBackdrop,
    resolveMount,
  };
})();
