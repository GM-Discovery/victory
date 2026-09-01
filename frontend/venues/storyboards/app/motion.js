// Kernel 94 Pass 4: shared reduced-motion gate for the imperative (JS-driven)
// motion in this venue -- pure-CSS motion is gated per-rule via its own
// @media (prefers-reduced-motion: reduce) block instead, since that's
// cheaper and doesn't need JS at all.
export function prefersReducedMotion() {
  return !!(window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches);
}
