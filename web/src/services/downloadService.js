import { getBatchDownloadUrl, getStreamingDownloadUrl } from '@/api/api.js'

/**
 * Service for handling file downloads via the browser's native download
 * engine.
 *
 * Both backend endpoints respond with `Content-Disposition: attachment`, so
 * navigating to them hands the response straight to the browser's download
 * manager, which streams it to disk as bytes arrive — constant memory use
 * regardless of file size, with the browser's own progress UI. No JavaScript
 * touches the byte stream.
 *
 * History: downloads previously went through StreamSaver.js (fetch piped into
 * a service worker registered by a third-party MITM iframe). Safari blocks
 * service workers in third-party iframes, and StreamSaver's fallback
 * detection no longer recognizes modern Safari, so downloads there silently
 * did nothing. StreamSaver is also unmaintained and routed every download
 * through an externally hosted page — the native engine needs none of that.
 */
export class DownloadService {
  /**
   * Download a single file.
   *
   * @param {string} filePath - Full path to the file on XRootD
   * @param {string} fileName - Display name (used for logging only; the
   *   server's Content-Disposition header names the saved file)
   * @returns {{success: boolean, native: true}}
   */
  static downloadFile(filePath, fileName) {
    console.log(`Starting native browser download: ${fileName}`)
    getDownloadSinkFrame().src = getStreamingDownloadUrl(filePath)
    return { success: true, native: true }
  }

  /**
   * Download multiple files as a tar (optionally gzipped) archive.
   *
   * The batch endpoint takes a POST, so the navigation is driven by a hidden
   * form submission (`basePath` + repeated `files` fields — the backend
   * accepts this alongside JSON).
   *
   * @param {string} basePath - Current directory path on XRootD
   * @param {string[]} files - File names to include in the archive
   * @returns {{success: boolean, native: true}}
   */
  static downloadBatch(basePath, files) {
    console.log(`Starting native browser batch download: ${files.length} files from ${basePath}`)
    const form = document.createElement('form')
    form.method = 'POST'
    form.action = getBatchDownloadUrl()
    form.target = getDownloadSinkFrame().name
    form.hidden = true
    const addField = (name, value) => {
      const input = document.createElement('input')
      input.type = 'hidden'
      input.name = name
      input.value = value
      form.appendChild(input)
    }
    addField('basePath', basePath)
    for (const f of files) addField('files', f)
    document.body.appendChild(form)
    form.submit()
    form.remove()
    return { success: true, native: true }
  }
}

/**
 * Returns a persistent hidden iframe used as the navigation target for
 * downloads. Navigating an iframe (instead of the top-level page) means a
 * server error response cannot blow away the SPA — a successful response is
 * turned into a download by its Content-Disposition: attachment header.
 */
function getDownloadSinkFrame() {
  let frame = document.querySelector('iframe[name="dh-download-sink"]')
  if (!frame) {
    frame = document.createElement('iframe')
    frame.name = 'dh-download-sink'
    frame.hidden = true
    document.body.appendChild(frame)
  }
  return frame
}

export default DownloadService
