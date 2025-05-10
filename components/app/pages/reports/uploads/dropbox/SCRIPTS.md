# Scripts

Script that accepts drag and drop with preview before uploading.

```html
        <div class="mt-6 flex items-center justify-end gap-x-6">
            <a href="/dashboard" class="text-sm font-semibold leading-6 text-gray-900">Cancel</a>
            <button type="button" id="report-upload-button"
                    hx-post="/reports/dropbox/scrub" hx-trigger="click" hx-target="#notifications-panel"
                    class="rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600">
                Upload
            </button>
        </div>
```

```javascript
<script>
//----------------------------------------------------------------
// display the parsed content code
const targetContentId = "report-file-content";

//----------------------------------------------------------------
// file input element code
const fileInputId = "report-file-input";
const fileInput = document.getElementById(fileInputId);

function onGetFile(el) {
    const files = el.files;
    const file = files[0];
    const fileName = file.name.toLowerCase();
    if (fileName.endsWith('.docx')) {
        previewWordFile(files);
    } else if (fileName.endsWith('.txt')) {
        previewTextFile(files);
    } else {
        alert('Unsupported file type. Please drop a .docx or .txt file.');
    }
}

//----------------------------------------------------------------
// drag and drop code
const dropZoneId = "drop_zone";
const dropZone = document.getElementById(dropZoneId);

// Prevent default behaviors for drag-and-drop events
['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
    dropZone.addEventListener(eventName, preventDefaults, false);
});

function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
}

// Add highlighting effect when a file is dragged over the drop zone
['dragenter', 'dragover'].forEach(eventName => {
    dropZone.addEventListener(eventName, () => dropZone.classList.add('highlight'), false);
});

['dragleave', 'drop'].forEach(eventName => {
    dropZone.addEventListener(eventName, () => dropZone.classList.remove('highlight'), false);
});

// Handle the file drop event
dropZone.addEventListener('drop', handleDrop, false);

function handleDrop(e) {
    const dt = e.dataTransfer;
    const files = dt.files;

    // do nothing if no files were dropped
    if (files.length === 0) {
        return;
    } else if (files.length > 1) {
        alert('Only one file can be dropped at a time');
        return;
    }

    const file = files[0];
    const fileName = file.name.toLowerCase();
    if (fileName.endsWith('.docx')) {
        previewWordFile(files);
    } else if (fileName.endsWith('.txt')) {
        previewTextFile(files);
    } else {
        alert('Unsupported file type. Please drop a .docx or .txt file.');
    }
}

//----------------------------------------------------------------
// code to preview text and word files
function previewTextFile(files) {
    const file = files[0];

    // Basic file validation (size and type)
    if (file.size > 1048576) {
        alert('File size must be less than 1MB');
        return;
    }

    // stuff the file into the file input element
    fileInput.files = files;

    // copy the file content into the target div
    const targetDiv = document.getElementById(targetContentId);
    if (!targetDiv) {
        alert(`Element with id '${targetContentId}' not found.`);
        return;
    }

    // Create a FileReader to read the file
    const reader = new FileReader();

    // Define the onload event handler
    reader.onload = function(event) {
        // Create <pre> and <code> elements
        const preElement = document.createElement('pre');
        const codeElement = document.createElement('code');

        // Set the text content of the <code> element
        codeElement.textContent = event.target.result;

        // Append the <code> element to the <pre> element
        preElement.appendChild(codeElement);

        // Clear any existing content in the target div
        targetDiv.innerHTML = '';

        // Append the <pre> element to the target div
        targetDiv.appendChild(preElement);
    };

    // Define the onerror event handler
    reader.onerror = function() {
        alert('Error reading file.');
    };

    // Read the file as text
    reader.readAsText(file);

    // after we've updated the div, scroll to the upload button
    document.getElementById("report-upload-button").scrollIntoView({
        behavior: 'smooth', // This makes the scroll smooth
        block: 'start'      // Aligns it to the top of the visible area
    });
}

function previewWordFile(files) {
    const file = files[0];

    // Basic file validation (size and type)
    if (file.size > 1048576) {
        alert('File size must be less than 1MB');
        return;
    }

    // stuff the file into the file input element
    fileInput.files = files;

    // now we to render it and stuff it into the content div
    docx.renderAsync(file, document.getElementById(targetContentId), null, {
        inWrapper: true, // was false
        ignoreWidth: true,
        ignoreHeight: true
    })
        .then(x => {
            // after we've updated the div, scroll to the upload button
            document.getElementById("report-upload-button").scrollIntoView({
                behavior: 'smooth', // This makes the scroll smooth
                block: 'start'      // Aligns it to the top of the visible area
            });
        });
}
</script>
```