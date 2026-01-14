import { deleteFileByName, setVisibilityGrid } from './fileGrid.js';

const modal = document.getElementById('file-modal');

const togglePublicButton = document.getElementById('file-modal-toggle-public-button');

const fileModalDeleteButton = document.getElementById('file-modal-delete-button');

const filePreviewImage = document.getElementById('file-preview-image');
const filePreviewVideo = document.getElementById('file-preview-video');
const filePreviewAudio = document.getElementById('file-preview-audio');
const filePreviewGeneric = document.getElementById('file-preview-generic');

const fileModalFilename = document.getElementById('file-modal-filename');
const fileModalFilenameUrl = document.getElementById('file-modal-filename-url');
const fileModalOriginalFilename = document.getElementById('file-modal-original-filename');

const fileModalViews = document.getElementById('file-modal-views');

const fileModalFilesize = document.getElementById('file-modal-filesize');
const fileModalFilesizeWrapper = document.getElementById('file-modal-filesize-wrapper');

const fileModalCreatedAt = document.getElementById('file-modal-createdat');
const fileModalCreatedAtWrapper = document.getElementById('file-modal-createdat-wrapper');

const fileModalExpiryDate = document.getElementById('file-modal-expirydate');
const fileModalExpiryDateWrapper = document.getElementById('file-modal-expirydate-wrapper');

const fileModalVisibility = document.getElementById('file-modal-visibility');
const fileModalVisibilityIcon = document.getElementById('file-modal-visibility-icon');

function showModal(elem) {
    modal.classList.remove('file-modal-hidden');
    modal.classList.add('file-modal-visible');

    document.body.classList.add('no-scroll');

    const isimage = elem.parentElement.dataset.isimage === 'true';
    const isvideo = elem.parentElement.dataset.isvideo === 'true';
    const isaudio = elem.parentElement.dataset.isaudio === 'true';

    filePreviewImage.style.display = 'none';
    filePreviewVideo.style.display = 'none';
    filePreviewAudio.style.display = 'none';
    filePreviewGeneric.style.display = 'none';

    if (isimage) {
        filePreviewImage.src = `/${elem.parentElement.dataset.filename}`;
        filePreviewImage.style.display = 'block';
    } else if (isvideo) {
        filePreviewVideo.src = `/${elem.parentElement.dataset.filename}`;
        filePreviewVideo.style.display = 'block';
    } else if (isaudio) {
        filePreviewAudio.src = `/${elem.parentElement.dataset.filename}`;
        filePreviewAudio.style.display = 'block';
    } else {
        filePreviewGeneric.href = `/${elem.parentElement.dataset.filename}`;
        filePreviewGeneric.style.display = 'block';
    }

    fileModalFilename.textContent = elem.parentElement.dataset.filename;
    fileModalFilenameUrl.href = `/${elem.parentElement.dataset.filename}`;

    if (elem.parentElement.dataset.originalfilename !== '') {
        fileModalOriginalFilename.parentElement.style.display = 'block';
        fileModalOriginalFilename.textContent = elem.parentElement.dataset.originalfilename;
    } else {
        fileModalOriginalFilename.parentElement.style.display = 'none';
    }

    let views = parseInt(elem.parentElement.dataset.views, 10);

    fileModalViews.textContent = `${views} view`;
    if (views > 1) {
        fileModalViews.textContent += 's';
    }

    fileModalFilesize.textContent = elem.parentElement.dataset.filesize;
    fileModalFilesizeWrapper.title = `${elem.parentElement.dataset.filesizebytes} bytes`;

    fileModalCreatedAt.textContent = elem.parentElement.dataset.createdat;
    fileModalCreatedAtWrapper.title = `Uploaded ${elem.parentElement.dataset.createdatrelative}`;

    if (elem.parentElement.dataset.expirydate !== '') {
        fileModalExpiryDateWrapper.style.display = 'flex';
        fileModalExpiryDate.textContent = `Expires ${elem.parentElement.dataset.expiryrelative}`;
        fileModalExpiryDateWrapper.title = elem.parentElement.dataset.expirydate;
    } else {
        fileModalExpiryDateWrapper.style.display = 'none';
    }

    let isPublic = elem.parentElement.dataset.public === 'true';

    togglePublicButton.onclick = async function () {
        const filename = elem.parentElement.dataset.filename;

        const formData = new FormData();
        formData.append('file_name', filename);

        const response = await fetch('/api/account/file/public', {
            method: 'POST',
            body: formData
        });

        if (response.ok) {
            isPublic = !isPublic;
            setVisibility(isPublic);
            setVisibilityGrid(filename, isPublic);
        } else {
            alert('Failed to make file private');
        }
    }

    setVisibility(isPublic);

    fileModalDeleteButton.onclick = async function () {
        const filename = elem.parentElement.dataset.filename;

        if (confirm(`Are you sure you want to delete "${filename}"?`)) {
            if (deleteFileByName(filename)) {
                closeModal();
            }
        }
    }
}

window.showModal = showModal;

function closeModal() {
    modal.classList.add('file-modal-hidden');
    modal.classList.remove('file-modal-visible');

    document.body.classList.remove('no-scroll');
}

window.closeModal = closeModal;

function setVisibility(isPublic) {
    if (isPublic) {
        fileModalVisibility.textContent = 'Public';
        fileModalVisibilityIcon.href.baseVal = '/public/assets/lucide-sprite.svg#lock-open';
        togglePublicButton.textContent = 'Make Private';
    } else {
        fileModalVisibility.textContent = 'Private';
        fileModalVisibilityIcon.href.baseVal = '/public/assets/lucide-sprite.svg#lock';
        togglePublicButton.textContent = 'Make Public';
    }
}

