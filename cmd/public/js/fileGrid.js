import { reloadCurrentPage, decrementTotalFiles } from './fileRenderer.js';
import { loadFileStats } from './fileStats.js';

function deleteFileGrid(elem) {
    const filename = elem.parentElement.dataset.filename;

    if (confirm(`Are you sure you want to delete "${filename}"?`)) {
        deleteFileByName(filename);
    }
}

window.deleteFileGrid = deleteFileGrid;

export async function deleteFileByName(filename) {
    const formData = new FormData();
    formData.append('file_name', filename);

    const response = await fetch('/api/account/file', {
        method: 'DELETE',
        body: formData
    });

    if (response.ok) {
        decrementTotalFiles();
        reloadCurrentPage();
        loadFileStats();
        return true;
    } else {
        alert('Failed to delete file');
        return false;
    }
}

export function setVisibilityGrid(filename, isPublic) {
    const fileElements = document.getElementsByClassName('file-entry');

    for (const fileElement of fileElements) {
        if (fileElement.dataset.filename === filename) {
            fileElement.dataset.public = isPublic.toString();

            const visibilityStatus = fileElement.querySelector('.visbility-status');
            if (visibilityStatus) {
                const icon = visibilityStatus.querySelector('use');
                const text = visibilityStatus.querySelector('span');

                if (isPublic) {
                    icon.href.baseVal = '/public/assets/lucide-sprite.svg#lock-open';
                    text.textContent = 'Public';
                } else {
                    icon.href.baseVal = '/public/assets/lucide-sprite.svg#lock';
                    text.textContent = 'Private';
                }
            }
            break;
        }
    }
}