
/**
 * Display an error message.
 * If the message contains newlines, it will display them in multiple messages (one per line). 
 * 
 * @param {Error} error - The error object containing the error message.
 * @returns {void}
 */
export function displayMultilineErrorMessage(error) {
    // Workaround to display the error message in multiple lines.
    // Unfortunately, the ElMessage does not support multiline messages.
    const lines = error.message.split('\n');
    lines.forEach(line => {
        ElMessage({
            message: line,
            type: 'error',
            duration: 30000,
            showClose: true,
        });
    });
    console.error(error);
}

/**
 * Display an error message and log the error to the console.
 * If the message contains a <br> tag, it will split the lines of the error message.
 * 
 * @param {Error} error - The error object to display and log.
 * @returns {void}
 */
export function displayErrorMessage(error) {
    ElMessage({
        message: error.message,
        type: 'error',
        duration: 30000,
        showClose: true,
        // it will split the lines of the error message if it contains a <br> tag
        dangerouslyUseHTMLString: true
    });
    console.error(error);
}

/**
 * Joins two paths together.
 *
 * @param {string} base - The base path.
 * @param {string} relative - The relative path.
 * @returns {string} - The joined path.
 */
export const joinPaths = (base, relative) => {
    return `${base.replace(/\/$/, '')}/${relative.replace(/^\//, '')}`;
};