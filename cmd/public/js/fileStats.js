import { humanizeBytes } from './utils.js';

export async function loadFileStats() {
    const filesStatsElement = document.getElementById('files-stats');
    if (!filesStatsElement) return;

    const response = await fetch('/api/account/files/stats', {
        method: 'GET',
    });

    if (!response.ok) {
        filesStatsElement.textContent = 'Failed to load stats';
        return;
    }

    const data = await response.json();
    const count = data.count || 0;
    const sizeTotal = data.size_total || 0;

    const filesText = count === 1 ? '1 file' : `${count} files`;
    filesStatsElement.textContent = `${filesText} • ${humanizeBytes(sizeTotal)}`;
}

document.addEventListener('DOMContentLoaded', loadFileStats);
