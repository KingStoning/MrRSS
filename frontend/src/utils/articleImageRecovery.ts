import { getProxiedMediaUrl, isMediaProxyFallbackEnabled } from './mediaProxy';

// Own retries per DOM node. Reattaching reader interactions must not restart downloads.
export function createArticleImageRecovery() {
  const bindings = new Map<HTMLImageElement, () => void>();

  function clear() {
    bindings.forEach((dispose) => dispose());
    bindings.clear();
  }

  function attach(img: HTMLImageElement, referer?: string) {
    for (const [node, dispose] of bindings) {
      if (!node.isConnected) {
        dispose();
        bindings.delete(node);
      }
    }
    if (bindings.has(img)) return;
    let attempts = 0;
    let pending = false;
    let disposed = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const onError = async () => {
      if (disposed || pending || attempts >= 2 || !img.isConnected) return;
      const source = img.currentSrc || img.src;
      if (!source || /^(data|blob):/i.test(source)) return;
      pending = true;
      attempts += 1;
      const useProxy = await isMediaProxyFallbackEnabled();
      if (disposed || !img.isConnected) return;
      timer = setTimeout(() => {
        pending = false;
        if (disposed || !img.isConnected) return;
        // Use the same signed source URL; never append cache-busters to a CDN URL.
        const target = useProxy ? getProxiedMediaUrl(source, referer) : source;
        img.removeAttribute('srcset');
        img.removeAttribute('src');
        img.src = target;
      }, attempts * 300);
    };
    img.addEventListener('error', onError);
    bindings.set(img, () => {
      disposed = true;
      clearTimeout(timer);
      img.removeEventListener('error', onError);
    });
    // An image can fail before Vue has finished attaching its reader handlers.
    if (img.complete && img.naturalWidth === 0 && img.getAttribute('src')) void onError();
  }

  return { attach, clear };
}
