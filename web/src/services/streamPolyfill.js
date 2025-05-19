/**
 * StreamSaver.js polyfill setup for browsers without native Streams support
 * This file should be loaded before the main application to ensure compatibility
 */

// Check if we need the polyfill
const needsPolyfill = !window.WritableStream || !window.ReadableStream || !window.TransformStream;

if (needsPolyfill) {
    console.log('Loading streams polyfill for browser compatibility');

    // Load the polyfill from CDN as fallback
    // In production, you might want to bundle this locally
    const script = document.createElement('script');
    script.src = 'https://cdn.jsdelivr.net/npm/web-streams-polyfill@3.2.1/dist/ponyfill.min.js';
    script.onload = () => {
        // Apply the polyfill
        if (window.WebStreamsPolyfill) {
            if (!window.ReadableStream) {
                window.ReadableStream = window.WebStreamsPolyfill.ReadableStream;
            }
            if (!window.WritableStream) {
                window.WritableStream = window.WebStreamsPolyfill.WritableStream;
            }
            if (!window.TransformStream) {
                window.TransformStream = window.WebStreamsPolyfill.TransformStream;
            }
            console.log('Streams polyfill loaded successfully');
        }
    };
    script.onerror = () => {
        console.warn('Failed to load streams polyfill from CDN');
    };
    document.head.appendChild(script);
} else {
    console.log('Native streams support detected - no polyfill needed');
}

// Log browser compatibility information
console.log('Stream compatibility check:', {
    isSecureContext: window.isSecureContext,
    hasServiceWorker: 'serviceWorker' in navigator,
    hasReadableStream: 'ReadableStream' in window,
    hasWritableStream: 'WritableStream' in window,
    hasTransformStream: 'TransformStream' in window,
    userAgent: navigator.userAgent
});

export { };
