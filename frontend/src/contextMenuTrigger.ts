import { onCleanup } from "solid-js";

const LONG_PRESS_MS = 500;
const LONG_PRESS_MOVE_TOLERANCE = 10;

export function createContextMenuTrigger(
  open: (x: number, y: number) => void,
  ignore?: string,
) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let fired = false;
  let touching = false;
  let startX = 0;
  let startY = 0;
  onCleanup(() => clearTimeout(timer));

  return {
    onContextMenu(e: MouseEvent) {
      e.preventDefault();
      clearTimeout(timer);
      if (touching) {
        if (fired) return;
        fired = true;
      }
      open(e.clientX, e.clientY);
    },
    onTouchStart(e: TouchEvent) {
      fired = false;
      if (ignore && (e.target as HTMLElement).closest(ignore)) return;
      touching = true;
      startX = e.touches[0].clientX;
      startY = e.touches[0].clientY;
      clearTimeout(timer);
      timer = setTimeout(() => {
        fired = true;
        open(startX, startY);
      }, LONG_PRESS_MS);
    },
    onTouchMove(e: TouchEvent) {
      const dx = e.touches[0].clientX - startX;
      const dy = e.touches[0].clientY - startY;
      if (Math.hypot(dx, dy) > LONG_PRESS_MOVE_TOLERANCE) clearTimeout(timer);
    },
    onTouchEnd(e: TouchEvent) {
      clearTimeout(timer);
      touching = false;
      if (fired) e.preventDefault();
    },
    onTouchCancel() {
      clearTimeout(timer);
      touching = false;
    },
  };
}
